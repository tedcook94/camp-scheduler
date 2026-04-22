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

const runTypeCamperCabin = "camper_cabin"

func StoreCamperSolutions(ctx context.Context, pool *pgxpool.Pool, campID, sessionID string, snapshot CamperCabinSnapshot, solutions []CamperSolution) (string, error) {
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
		RunType:   runTypeCamperCabin,
		Status:    "completed",
	})
	if err != nil {
		return "", fmt.Errorf("error creating assignment run: %w", err)
	}

	for i, solution := range solutions {
		if err := storeCamperSolution(ctx, qtx, campUUID, run.ID, i, snapshot, solution); err != nil {
			return "", fmt.Errorf("error storing camper solution %d: %w", i, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("error committing transaction: %w", err)
	}

	return api.UUIDToString(run.ID), nil
}

func storeCamperSolution(ctx context.Context, qtx *db.Queries, campID, runID pgtype.UUID, index int, snapshot CamperCabinSnapshot, solution CamperSolution) error {
	breakdownJSON, err := json.Marshal(solution.Score.Breakdown)
	if err != nil {
		return fmt.Errorf("error marshaling score breakdown: %w", err)
	}

	sol, err := qtx.CreateCamperCabinSolution(ctx, db.CreateCamperCabinSolutionParams{
		CampID:          campID,
		AssignmentRunID: runID,
		SolutionIndex:   int32(index),
		Score:           solution.Score.Total,
		ScoreBreakdown:  breakdownJSON,
	})
	if err != nil {
		return fmt.Errorf("error creating camper solution row: %w", err)
	}

	explanation := ExplainCamper(snapshot, solution)

	if err := storeCamperAssignments(ctx, qtx, campID, sol.ID, snapshot, solution); err != nil {
		return err
	}

	if err := storeCamperExplanations(ctx, qtx, campID, sol.ID, explanation); err != nil {
		return err
	}

	return nil
}

func storeCamperAssignments(ctx context.Context, qtx *db.Queries, campID, solutionID pgtype.UUID, snapshot CamperCabinSnapshot, solution CamperSolution) error {
	sagcByCabinID := make(map[string]string, len(snapshot.Cabins))
	for _, c := range snapshot.Cabins {
		sagcByCabinID[c.ID] = c.SessionAgeGroupCabinID
	}

	for cabinID, camperIDs := range solution.Assignment.CabinCampers {
		sagcID, ok := sagcByCabinID[cabinID]
		if !ok || sagcID == "" {
			return fmt.Errorf("error resolving session_age_group_cabin_id for cabin %s", cabinID)
		}
		sagcUUID, err := api.ParseUUID(sagcID)
		if err != nil {
			return err
		}
		for _, camperID := range camperIDs {
			camperUUID, err := api.ParseUUID(camperID)
			if err != nil {
				return err
			}

			_, err = qtx.CreateCamperCabinAssignment(ctx, db.CreateCamperCabinAssignmentParams{
				CampID:                 campID,
				SolutionID:             solutionID,
				CamperID:               camperUUID,
				SessionAgeGroupCabinID: sagcUUID,
			})
			if err != nil {
				return fmt.Errorf("error creating camper assignment for camper %s: %w", camperID, err)
			}
		}
	}
	return nil
}

func storeCamperExplanations(ctx context.Context, qtx *db.Queries, campID, solutionID pgtype.UUID, explanation CamperExplanation) error {
	for _, ae := range explanation.Assignments {
		camperUUID, err := api.ParseUUID(ae.CamperID)
		if err != nil {
			return err
		}

		for _, reason := range ae.Reasons {
			_, err := qtx.CreateCamperCabinExplanation(ctx, db.CreateCamperCabinExplanationParams{
				CampID:          campID,
				SolutionID:      solutionID,
				CamperID:        camperUUID,
				ExplanationType: "reason",
				ConstraintName:  pgtype.Text{String: reason.Constraint, Valid: reason.Constraint != ""},
				Rank:            pgtype.Int4{Int32: int32(reason.Rank), Valid: reason.Rank > 0},
				Message:         reason.Message,
			})
			if err != nil {
				return fmt.Errorf("error creating reason explanation for camper %s: %w", ae.CamperID, err)
			}
		}
	}

	for _, up := range explanation.UnmetPreferences {
		camperUUID, err := api.ParseUUID(up.CamperID)
		if err != nil {
			return err
		}

		_, err = qtx.CreateCamperCabinExplanation(ctx, db.CreateCamperCabinExplanationParams{
			CampID:          campID,
			SolutionID:      solutionID,
			CamperID:        camperUUID,
			ExplanationType: "unmet_preference",
			ConstraintName:  pgtype.Text{String: up.Constraint, Valid: true},
			Rank:            pgtype.Int4{Int32: int32(up.Rank), Valid: up.Rank > 0},
			Message:         up.Message,
		})
		if err != nil {
			return fmt.Errorf("error creating unmet preference explanation for camper %s: %w", up.CamperID, err)
		}
	}

	return nil
}
