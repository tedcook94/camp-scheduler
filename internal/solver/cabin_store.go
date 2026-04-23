package solver

import (
	"context"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

const RunTypeCabin = "cabin"

// StoreCabinSolutions persists a combined cabin run. Each pair becomes one
// row in counselor_cabin_solutions AND one row in camper_cabin_solutions
// sharing the same assignment_run_id and solution_index. The two halves are
// paired by index when read back.
//
// Before inserting, any existing run for (session, 'cabin') is deleted so
// the unique (session_id, run_type) constraint holds and the new run
// replaces the old one.
func StoreCabinSolutions(ctx context.Context, pool *pgxpool.Pool, campID, sessionID string, snapshot CabinSnapshot, solutions []CabinSolution) (string, error) {
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
		RunType:   RunTypeCabin,
	}); err != nil {
		return "", fmt.Errorf("error deleting prior cabin runs: %w", err)
	}

	run, err := qtx.CreateAssignmentRun(ctx, db.CreateAssignmentRunParams{
		CampID:    campUUID,
		SessionID: sessionUUID,
		RunType:   RunTypeCabin,
		Status:    "completed",
	})
	if err != nil {
		return "", fmt.Errorf("error creating assignment run: %w", err)
	}

	for i, pair := range solutions {
		if err := storeSolution(ctx, qtx, campUUID, run.ID, i, snapshot.Counselor, pair.Counselor); err != nil {
			return "", fmt.Errorf("error storing counselor half of solution %d: %w", i, err)
		}
		if err := storeCamperSolution(ctx, qtx, campUUID, run.ID, i, snapshot.Camper, pair.Camper); err != nil {
			return "", fmt.Errorf("error storing camper half of solution %d: %w", i, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("error committing transaction: %w", err)
	}

	return api.UUIDToString(run.ID), nil
}
