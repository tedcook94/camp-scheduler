package assignment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
	"camp-scheduler/internal/solver"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRunNotFound      = errors.New("assignment run not found")
	ErrSolutionNotFound = errors.New("solution not found")
	ErrNoSolutions      = errors.New("solver produced no valid solutions")
)

type Service struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewService(queries *db.Queries, pool *pgxpool.Pool) *Service {
	return &Service{queries: queries, pool: pool}
}

func (svc *Service) TriggerRun(ctx context.Context, campID, sessionID string, cfg solver.SolverConfig) (RunDetailResponse, error) {
	snapshot, err := solver.BuildSnapshot(ctx, svc.queries, campID, sessionID)
	if err != nil {
		return RunDetailResponse{}, fmt.Errorf("error building snapshot: %w", err)
	}

	solutions := solver.Solve(snapshot, cfg)
	if len(solutions) == 0 {
		return RunDetailResponse{}, ErrNoSolutions
	}

	runID, err := solver.StoreSolutions(ctx, svc.pool, campID, sessionID, snapshot, solutions)
	if err != nil {
		return RunDetailResponse{}, fmt.Errorf("error storing solutions: %w", err)
	}

	return svc.GetRun(ctx, campID, runID)
}

func (svc *Service) ListRuns(ctx context.Context, campID, sessionID string) ([]RunResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return nil, err
	}

	rows, err := svc.queries.ListAssignmentRunsBySession(ctx, db.ListAssignmentRunsBySessionParams{
		SessionID: sessionUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing assignment runs: %w", err)
	}

	result := make([]RunResponse, len(rows))
	for i, r := range rows {
		result[i] = toRunResponse(r)
	}
	return result, nil
}

func (svc *Service) GetRun(ctx context.Context, campID, runID string) (RunDetailResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return RunDetailResponse{}, err
	}

	runUUID, err := api.ParseUUID(runID)
	if err != nil {
		return RunDetailResponse{}, err
	}

	run, err := svc.queries.GetAssignmentRun(ctx, db.GetAssignmentRunParams{
		ID:     runUUID,
		CampID: campUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RunDetailResponse{}, ErrRunNotFound
		}
		return RunDetailResponse{}, fmt.Errorf("error getting assignment run %s: %w", runID, err)
	}

	solRows, err := svc.queries.ListCounselorCabinSolutionsByRun(ctx, db.ListCounselorCabinSolutionsByRunParams{
		AssignmentRunID: runUUID,
		CampID:          campUUID,
	})
	if err != nil {
		return RunDetailResponse{}, fmt.Errorf("error listing solutions for run %s: %w", runID, err)
	}

	solutions := make([]SolutionSummaryResponse, len(solRows))
	for i, s := range solRows {
		solutions[i] = toSolutionSummaryResponse(s)
	}

	return RunDetailResponse{
		RunResponse: toRunResponseFromGet(run),
		Solutions:   solutions,
	}, nil
}

func (svc *Service) DeleteRun(ctx context.Context, campID, runID string) error {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return err
	}

	runUUID, err := api.ParseUUID(runID)
	if err != nil {
		return err
	}

	rows, err := svc.queries.DeleteAssignmentRun(ctx, db.DeleteAssignmentRunParams{
		ID:     runUUID,
		CampID: campUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting assignment run %s: %w", runID, err)
	}
	if rows == 0 {
		return ErrRunNotFound
	}

	return nil
}

func (svc *Service) GetSolution(ctx context.Context, campID, solutionID string) (SolutionDetailResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SolutionDetailResponse{}, err
	}

	solUUID, err := api.ParseUUID(solutionID)
	if err != nil {
		return SolutionDetailResponse{}, err
	}

	sol, err := svc.queries.GetCounselorCabinSolution(ctx, db.GetCounselorCabinSolutionParams{
		ID:     solUUID,
		CampID: campUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SolutionDetailResponse{}, ErrSolutionNotFound
		}
		return SolutionDetailResponse{}, fmt.Errorf("error getting solution %s: %w", solutionID, err)
	}

	assignments, err := svc.queries.ListCounselorCabinAssignmentsBySolution(ctx, db.ListCounselorCabinAssignmentsBySolutionParams{
		SolutionID: solUUID,
		CampID:     campUUID,
	})
	if err != nil {
		return SolutionDetailResponse{}, fmt.Errorf("error listing assignments for solution %s: %w", solutionID, err)
	}

	explanations, err := svc.queries.ListCounselorCabinExplanationsBySolution(ctx, db.ListCounselorCabinExplanationsBySolutionParams{
		SolutionID: solUUID,
		CampID:     campUUID,
	})
	if err != nil {
		return SolutionDetailResponse{}, fmt.Errorf("error listing explanations for solution %s: %w", solutionID, err)
	}

	assignmentResponses := make([]AssignmentResponse, len(assignments))
	for i, a := range assignments {
		assignmentResponses[i] = toAssignmentResponse(a)
	}

	explanationResponses := make([]ExplanationResponse, len(explanations))
	for i, e := range explanations {
		explanationResponses[i] = toExplanationResponse(e)
	}

	return SolutionDetailResponse{
		SolutionSummaryResponse: toSolutionSummaryResponse(sol),
		Assignments:             assignmentResponses,
		Explanations:            explanationResponses,
	}, nil
}

func (svc *Service) SelectSolution(ctx context.Context, campID, runID, solutionID string) (RunResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return RunResponse{}, err
	}

	runUUID, err := api.ParseUUID(runID)
	if err != nil {
		return RunResponse{}, err
	}

	solUUID, err := api.ParseUUID(solutionID)
	if err != nil {
		return RunResponse{}, err
	}

	// Verify the solution belongs to this run.
	sol, err := svc.queries.GetCounselorCabinSolution(ctx, db.GetCounselorCabinSolutionParams{
		ID:     solUUID,
		CampID: campUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RunResponse{}, ErrSolutionNotFound
		}
		return RunResponse{}, fmt.Errorf("error getting solution %s: %w", solutionID, err)
	}

	if api.UUIDToString(sol.AssignmentRunID) != runID {
		return RunResponse{}, ErrSolutionNotFound
	}

	run, err := svc.queries.SelectSolution(ctx, db.SelectSolutionParams{
		ID:                 runUUID,
		CampID:             campUUID,
		SelectedSolutionID: solUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RunResponse{}, ErrRunNotFound
		}
		return RunResponse{}, fmt.Errorf("error selecting solution %s for run %s: %w", solutionID, runID, err)
	}

	return toRunResponseFromSelect(run), nil
}

func toRunResponse(r db.ListAssignmentRunsBySessionRow) RunResponse {
	return RunResponse{
		ID:                 api.UUIDToString(r.ID),
		CampID:             api.UUIDToString(r.CampID),
		SessionID:          api.UUIDToString(r.SessionID),
		RunType:            r.RunType,
		Status:             r.Status,
		SelectedSolutionID: api.UUIDToStringPtr(r.SelectedSolutionID),
		CreatedAt:          r.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toRunResponseFromGet(r db.GetAssignmentRunRow) RunResponse {
	return RunResponse{
		ID:                 api.UUIDToString(r.ID),
		CampID:             api.UUIDToString(r.CampID),
		SessionID:          api.UUIDToString(r.SessionID),
		RunType:            r.RunType,
		Status:             r.Status,
		SelectedSolutionID: api.UUIDToStringPtr(r.SelectedSolutionID),
		CreatedAt:          r.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toRunResponseFromSelect(r db.SelectSolutionRow) RunResponse {
	return RunResponse{
		ID:                 api.UUIDToString(r.ID),
		CampID:             api.UUIDToString(r.CampID),
		SessionID:          api.UUIDToString(r.SessionID),
		RunType:            r.RunType,
		Status:             r.Status,
		SelectedSolutionID: api.UUIDToStringPtr(r.SelectedSolutionID),
		CreatedAt:          r.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toSolutionSummaryResponse(s db.CounselorCabinSolution) SolutionSummaryResponse {
	var breakdown []json.RawMessage
	_ = json.Unmarshal(s.ScoreBreakdown, &breakdown)

	return SolutionSummaryResponse{
		ID:              api.UUIDToString(s.ID),
		AssignmentRunID: api.UUIDToString(s.AssignmentRunID),
		SolutionIndex:   int(s.SolutionIndex),
		Score:           s.Score,
		ScoreBreakdown:  s.ScoreBreakdown,
	}
}

func toAssignmentResponse(a db.CounselorCabinAssignment) AssignmentResponse {
	return AssignmentResponse{
		ID:          api.UUIDToString(a.ID),
		CounselorID: api.UUIDToString(a.CounselorID),
		CabinID:     api.UUIDToString(a.CabinID),
	}
}

func toExplanationResponse(e db.CounselorCabinExplanation) ExplanationResponse {
	var constraintName *string
	if e.ConstraintName.Valid {
		constraintName = &e.ConstraintName.String
	}

	return ExplanationResponse{
		ID:              api.UUIDToString(e.ID),
		CounselorID:     api.UUIDToString(e.CounselorID),
		ExplanationType: e.ExplanationType,
		ConstraintName:  constraintName,
		Message:         e.Message,
	}
}
