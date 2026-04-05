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

const runTypeCounselorCabin = "counselor_cabin"

func StoreSolutions(ctx context.Context, pool *pgxpool.Pool, campID, sessionID string, snapshot SessionSnapshot, solutions []Solution) (string, error) {
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
		RunType:   runTypeCounselorCabin,
		Status:    "completed",
	})
	if err != nil {
		return "", fmt.Errorf("error creating assignment run: %w", err)
	}

	for i, solution := range solutions {
		if err := storeSolution(ctx, qtx, campUUID, run.ID, i, snapshot, solution); err != nil {
			return "", fmt.Errorf("error storing solution %d: %w", i, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("error committing transaction: %w", err)
	}

	return api.UUIDToString(run.ID), nil
}

func storeSolution(ctx context.Context, qtx *db.Queries, campID, runID pgtype.UUID, index int, snapshot SessionSnapshot, solution Solution) error {
	breakdownJSON, err := json.Marshal(solution.Score.Breakdown)
	if err != nil {
		return fmt.Errorf("error marshaling score breakdown: %w", err)
	}

	sol, err := qtx.CreateCounselorCabinSolution(ctx, db.CreateCounselorCabinSolutionParams{
		CampID:          campID,
		AssignmentRunID: runID,
		SolutionIndex:   int32(index),
		Score:           solution.Score.Total,
		ScoreBreakdown:  breakdownJSON,
	})
	if err != nil {
		return fmt.Errorf("error creating solution row: %w", err)
	}

	explanation := Explain(snapshot, solution)

	if err := storeAssignments(ctx, qtx, campID, sol.ID, explanation, solution); err != nil {
		return err
	}

	if err := storeExplanations(ctx, qtx, campID, sol.ID, explanation); err != nil {
		return err
	}

	return nil
}

func storeAssignments(ctx context.Context, qtx *db.Queries, campID, solutionID pgtype.UUID, explanation Explanation, solution Solution) error {
	for _, ae := range explanation.Assignments {
		counselorUUID, err := api.ParseUUID(ae.CounselorID)
		if err != nil {
			return err
		}

		cabinUUID, err := api.ParseUUID(ae.CabinID)
		if err != nil {
			return err
		}

		_, err = qtx.CreateCounselorCabinAssignment(ctx, db.CreateCounselorCabinAssignmentParams{
			CampID:      campID,
			SolutionID:  solutionID,
			CounselorID: counselorUUID,
			CabinID:     cabinUUID,
		})
		if err != nil {
			return fmt.Errorf("error creating assignment for counselor %s: %w", ae.CounselorID, err)
		}
	}
	return nil
}

func storeExplanations(ctx context.Context, qtx *db.Queries, campID, solutionID pgtype.UUID, explanation Explanation) error {
	for _, ae := range explanation.Assignments {
		counselorUUID, err := api.ParseUUID(ae.CounselorID)
		if err != nil {
			return err
		}

		for _, reason := range ae.Reasons {
			_, err := qtx.CreateCounselorCabinExplanation(ctx, db.CreateCounselorCabinExplanationParams{
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

		_, err = qtx.CreateCounselorCabinExplanation(ctx, db.CreateCounselorCabinExplanationParams{
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
