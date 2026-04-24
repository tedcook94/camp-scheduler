package solver

import (
	"context"
	"encoding/json"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const RunTypeActivitySchedule = "activity_schedule"

func StoreActivitySolutions(ctx context.Context, pool *pgxpool.Pool, campID, sessionID string, snapshot ActivitySnapshot, solutions []ActivitySolution) (string, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return "", err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return "", err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("error beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := db.New(tx)

	if _, err := qtx.DeleteAssignmentRunsBySessionAndType(ctx, db.DeleteAssignmentRunsBySessionAndTypeParams{
		CampID:    campUUID,
		SessionID: sessionUUID,
		RunType:   RunTypeActivitySchedule,
	}); err != nil {
		return "", fmt.Errorf("error deleting prior activity runs: %w", err)
	}

	run, err := qtx.CreateAssignmentRun(ctx, db.CreateAssignmentRunParams{
		CampID:    campUUID,
		SessionID: sessionUUID,
		RunType:   RunTypeActivitySchedule,
		Status:    "completed",
	})
	if err != nil {
		return "", fmt.Errorf("error creating assignment run: %w", err)
	}

	for i, solution := range solutions {
		if err := storeActivitySolution(ctx, qtx, campUUID, run.ID, i, snapshot, solution); err != nil {
			return "", fmt.Errorf("error storing activity solution %d: %w", i, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("error committing transaction: %w", err)
	}

	return api.UUIDToString(run.ID), nil
}

func storeActivitySolution(ctx context.Context, qtx *db.Queries, campID, runID pgtype.UUID, index int, snapshot ActivitySnapshot, solution ActivitySolution) error {
	breakdownJSON, err := json.Marshal(solution.Score.Breakdown)
	if err != nil {
		return fmt.Errorf("error marshaling score breakdown: %w", err)
	}

	sol, err := qtx.CreateActivitySolution(ctx, db.CreateActivitySolutionParams{
		CampID:          campID,
		AssignmentRunID: runID,
		SolutionIndex:   int32(index),
		Score:           solution.Score.Total,
		ScoreBreakdown:  breakdownJSON,
	})
	if err != nil {
		return fmt.Errorf("error creating activity solution row: %w", err)
	}

	explanation := ExplainActivity(snapshot, solution)

	if err := storeActivityAssignments(ctx, qtx, campID, sol.ID, solution); err != nil {
		return err
	}

	if err := storeActivityUnassignedCounselors(ctx, qtx, campID, sol.ID, solution); err != nil {
		return err
	}

	if err := storeActivityExplanations(ctx, qtx, campID, sol.ID, explanation); err != nil {
		return err
	}

	return nil
}

// storeActivityUnassignedCounselors persists per-counselor missing time
// slots. ActivitySlot.TimeSlotID is the session_time_slot.id, so it's used
// directly as the FK target.
func storeActivityUnassignedCounselors(ctx context.Context, qtx *db.Queries, campID, solutionID pgtype.UUID, solution ActivitySolution) error {
	for _, uc := range solution.UnassignedCounselors {
		counselorUUID, err := api.ParseUUID(uc.CounselorID)
		if err != nil {
			return err
		}
		parent, err := qtx.CreateCounselorActivityUnassigned(ctx, db.CreateCounselorActivityUnassignedParams{
			CampID:      campID,
			SolutionID:  solutionID,
			CounselorID: counselorUUID,
		})
		if err != nil {
			return fmt.Errorf("error recording unassigned counselor %s: %w", uc.CounselorID, err)
		}
		for _, tsID := range uc.MissingTimeSlotIDs {
			tsUUID, err := api.ParseUUID(tsID)
			if err != nil {
				return err
			}
			if _, err := qtx.CreateCounselorActivityUnassignedSlot(ctx, db.CreateCounselorActivityUnassignedSlotParams{
				CampID:            campID,
				UnassignedID:      parent.ID,
				SessionTimeSlotID: tsUUID,
			}); err != nil {
				return fmt.Errorf("error recording missing time slot %s for counselor %s: %w", tsID, uc.CounselorID, err)
			}
		}
	}
	return nil
}

func storeActivityAssignments(ctx context.Context, qtx *db.Queries, campID, solutionID pgtype.UUID, solution ActivitySolution) error {
	for slotID, counselorIDs := range solution.Assignment.SlotCounselors {
		slotUUID, err := api.ParseUUID(slotID)
		if err != nil {
			return err
		}
		for _, counselorID := range counselorIDs {
			counselorUUID, err := api.ParseUUID(counselorID)
			if err != nil {
				return err
			}

			_, err = qtx.CreateActivityAssignment(ctx, db.CreateActivityAssignmentParams{
				CampID:            campID,
				SolutionID:        solutionID,
				CounselorID:       counselorUUID,
				SessionActivityID: slotUUID,
			})
			if err != nil {
				return fmt.Errorf("error creating activity assignment for counselor %s: %w", counselorID, err)
			}
		}
	}
	return nil
}

func storeActivityExplanations(ctx context.Context, qtx *db.Queries, campID, solutionID pgtype.UUID, explanation ActivityExplanation) error {
	for _, ae := range explanation.Assignments {
		counselorUUID, err := api.ParseUUID(ae.CounselorID)
		if err != nil {
			return err
		}

		for _, reason := range ae.Reasons {
			var slotUUID pgtype.UUID
			if reason.SlotID != "" {
				parsed, err := api.ParseUUID(reason.SlotID)
				if err != nil {
					return err
				}
				slotUUID = parsed
			}

			_, err := qtx.CreateActivityExplanation(ctx, db.CreateActivityExplanationParams{
				CampID:            campID,
				SolutionID:        solutionID,
				CounselorID:       counselorUUID,
				SessionActivityID: slotUUID,
				ExplanationType:   "reason",
				ConstraintName:    pgtype.Text{String: reason.Constraint, Valid: reason.Constraint != ""},
				Rank:              pgtype.Int4{Int32: int32(reason.Rank), Valid: reason.Rank > 0},
				Message:           reason.Message,
			})
			if err != nil {
				return fmt.Errorf("error creating reason explanation for counselor %s: %w", ae.CounselorID, err)
			}
		}
	}

	for _, up := range explanation.UnmetPreferences {
		counselorUUID, err := api.ParseUUID(up.CounselorID)
		if err != nil {
			return err
		}

		_, err = qtx.CreateActivityExplanation(ctx, db.CreateActivityExplanationParams{
			CampID:          campID,
			SolutionID:      solutionID,
			CounselorID:     counselorUUID,
			ExplanationType: "unmet_preference",
			ConstraintName:  pgtype.Text{String: up.Constraint, Valid: true},
			Rank:            pgtype.Int4{Int32: int32(up.Rank), Valid: up.Rank > 0},
			Message:         up.Message,
		})
		if err != nil {
			return fmt.Errorf("error creating unmet preference explanation for counselor %s: %w", up.CounselorID, err)
		}
	}

	for _, ip := range explanation.IneligiblePreferences {
		counselorUUID, err := api.ParseUUID(ip.CounselorID)
		if err != nil {
			return err
		}

		_, err = qtx.CreateActivityExplanation(ctx, db.CreateActivityExplanationParams{
			CampID:          campID,
			SolutionID:      solutionID,
			CounselorID:     counselorUUID,
			ExplanationType: "ineligible_preference",
			ConstraintName:  pgtype.Text{String: ip.Constraint, Valid: true},
			Rank:            pgtype.Int4{Int32: int32(ip.Rank), Valid: ip.Rank > 0},
			Message:         ip.Message,
		})
		if err != nil {
			return fmt.Errorf("error creating ineligible preference explanation for counselor %s: %w", ip.CounselorID, err)
		}
	}

	return nil
}
