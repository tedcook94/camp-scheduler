package solver

import (
	"context"
	"encoding/json"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
)

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

	if err := storeAssignments(ctx, qtx, campID, sol.ID, snapshot, solution); err != nil {
		return err
	}

	if err := storeUnassignedCounselors(ctx, qtx, campID, sol.ID, solution); err != nil {
		return err
	}

	if err := storeExplanations(ctx, qtx, campID, sol.ID, explanation); err != nil {
		return err
	}

	return nil
}

func storeUnassignedCounselors(ctx context.Context, qtx *db.Queries, campID, solutionID pgtype.UUID, solution Solution) error {
	for _, counselorID := range solution.UnassignedCounselors {
		counselorUUID, err := api.ParseUUID(counselorID)
		if err != nil {
			return err
		}
		if _, err := qtx.CreateCounselorCabinUnassigned(ctx, db.CreateCounselorCabinUnassignedParams{
			CampID:      campID,
			SolutionID:  solutionID,
			CounselorID: counselorUUID,
		}); err != nil {
			return fmt.Errorf("error recording unassigned counselor %s: %w", counselorID, err)
		}
	}
	return nil
}

func storeAssignments(ctx context.Context, qtx *db.Queries, campID, solutionID pgtype.UUID, snapshot SessionSnapshot, solution Solution) error {
	sagcByCabinID := make(map[string]string, len(snapshot.Cabins))
	for _, c := range snapshot.Cabins {
		sagcByCabinID[c.ID] = c.SessionCabinID
	}

	for cabinID, counselorIDs := range solution.Assignment.CabinCounselors {
		sagcID, ok := sagcByCabinID[cabinID]
		if !ok || sagcID == "" {
			return fmt.Errorf("error resolving session_cabin_id for cabin %s", cabinID)
		}
		sagcUUID, err := api.ParseUUID(sagcID)
		if err != nil {
			return err
		}
		for _, counselorID := range counselorIDs {
			counselorUUID, err := api.ParseUUID(counselorID)
			if err != nil {
				return err
			}

			_, err = qtx.CreateCounselorCabinAssignment(ctx, db.CreateCounselorCabinAssignmentParams{
				CampID:                 campID,
				SolutionID:             solutionID,
				CounselorID:            counselorUUID,
				SessionCabinID: sagcUUID,
			})
			if err != nil {
				return fmt.Errorf("error creating assignment for counselor %s: %w", counselorID, err)
			}
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
				ConstraintName:  pgtype.Text{String: reason.Constraint, Valid: reason.Constraint != ""},
				Rank:            pgtype.Int4{Int32: int32(reason.Rank), Valid: reason.Rank > 0},
				Message:         reason.Message,
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

		_, err = qtx.CreateCounselorCabinExplanation(ctx, db.CreateCounselorCabinExplanationParams{
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
