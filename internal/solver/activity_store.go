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

const runTypeActivitySchedule = "activity_schedule"

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

	run, err := qtx.CreateAssignmentRun(ctx, db.CreateAssignmentRunParams{
		CampID:    campUUID,
		SessionID: sessionUUID,
		RunType:   runTypeActivitySchedule,
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

	if err := storeActivityAssignments(ctx, qtx, campID, sol.ID, explanation); err != nil {
		return err
	}

	if err := storeActivityExplanations(ctx, qtx, campID, sol.ID, explanation); err != nil {
		return err
	}

	return nil
}

func storeActivityAssignments(ctx context.Context, qtx *db.Queries, campID, solutionID pgtype.UUID, explanation ActivityExplanation) error {
	for _, ae := range explanation.Assignments {
		counselorUUID, err := api.ParseUUID(ae.CounselorID)
		if err != nil {
			return err
		}

		slotUUID, err := api.ParseUUID(ae.SlotID)
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
			return fmt.Errorf("error creating activity assignment for counselor %s: %w", ae.CounselorID, err)
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
			_, err := qtx.CreateActivityExplanation(ctx, db.CreateActivityExplanationParams{
				CampID:          campID,
				SolutionID:      solutionID,
				CounselorID:     counselorUUID,
				ExplanationType: "reason",
				Message:         reason,
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
			Message:         up.Message,
		})
		if err != nil {
			return fmt.Errorf("error creating unmet preference explanation for counselor %s: %w", up.CounselorID, err)
		}
	}

	return nil
}
