package assignment

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
	"camp-scheduler/internal/solver"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrRunNotFound      = errors.New("assignment run not found")
	ErrSolutionNotFound = errors.New("solution not found")
	ErrNoSolutions      = errors.New("solver produced no valid solutions")
)

type PreconditionError struct {
	msg string
}

func NewPreconditionError(msg string) *PreconditionError {
	return &PreconditionError{msg: msg}
}

func (e *PreconditionError) Error() string {
	return e.msg
}

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

	if msg := validateCounselorCabin(snapshot); msg != "" {
		return RunDetailResponse{}, NewPreconditionError(msg)
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

func validateCounselorCabin(snapshot solver.SessionSnapshot) string {
	if len(snapshot.Cabins) == 0 {
		return "No cabins configured for this session"
	}

	totalRequired := 0
	for _, c := range snapshot.Cabins {
		totalRequired += c.RequiredCounselors
	}

	if totalRequired == 0 {
		return ""
	}

	if len(snapshot.Counselors) == 0 {
		return "No enabled counselors found"
	}

	hasSenior := false
	for _, c := range snapshot.Counselors {
		if !c.IsJunior {
			hasSenior = true
			break
		}
	}

	if !hasSenior {
		return "No senior counselors available; at least one senior is required to satisfy the senior-counselor constraint for staffed cabins"
	}

	if len(snapshot.Counselors) < totalRequired {
		return fmt.Sprintf("Not enough counselors (%d) to fill all cabin requirements (%d)", len(snapshot.Counselors), totalRequired)
	}

	// Per-gender feasibility: cabins are gender-segregated, so we need
	// enough counselors and at least one senior of each gender to staff
	// the cabins of that gender.
	requiredByGender := map[string]int{}
	cabinsByGender := map[string]int{}
	for _, c := range snapshot.Cabins {
		requiredByGender[c.Gender] += c.RequiredCounselors
		if c.RequiredCounselors > 0 {
			cabinsByGender[c.Gender]++
		}
	}
	counselorsByGender := map[string]int{}
	seniorsByGender := map[string]int{}
	for _, c := range snapshot.Counselors {
		counselorsByGender[c.Gender]++
		if !c.IsJunior {
			seniorsByGender[c.Gender]++
		}
	}
	for gender, need := range requiredByGender {
		if need == 0 {
			continue
		}
		if counselorsByGender[gender] < need {
			return fmt.Sprintf("Not enough %s counselors (%d) to fill %s cabin requirements (%d)", gender, counselorsByGender[gender], gender, need)
		}
		if seniorsByGender[gender] < cabinsByGender[gender] {
			return fmt.Sprintf("Not enough senior %s counselors (%d) to seat one in each of the %d staffed %s cabin(s)", gender, seniorsByGender[gender], cabinsByGender[gender], gender)
		}
	}

	return ""
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

func (svc *Service) TriggerActivityRun(ctx context.Context, campID, sessionID string, cfg solver.ActivitySolverConfig) (RunDetailResponse, error) {
	snapshot, err := solver.BuildActivitySnapshot(ctx, svc.queries, campID, sessionID)
	if err != nil {
		return RunDetailResponse{}, fmt.Errorf("error building activity snapshot: %w", err)
	}

	if msg := validateActivity(snapshot); msg != "" {
		return RunDetailResponse{}, NewPreconditionError(msg)
	}

	solutions := solver.SolveActivity(snapshot, cfg)
	if len(solutions) == 0 {
		return RunDetailResponse{}, ErrNoSolutions
	}

	runID, err := solver.StoreActivitySolutions(ctx, svc.pool, campID, sessionID, snapshot, solutions)
	if err != nil {
		return RunDetailResponse{}, fmt.Errorf("error storing activity solutions: %w", err)
	}

	return svc.GetRun(ctx, campID, runID)
}

func validateActivity(snapshot solver.ActivitySnapshot) string {
	if len(snapshot.Slots) == 0 {
		return "No activities configured for this session"
	}

	totalRequired := 0
	for _, s := range snapshot.Slots {
		totalRequired += s.RequiredCounselors
	}

	if totalRequired > 0 && len(snapshot.Counselors) == 0 {
		return "No enabled counselors found"
	}

	return ""
}

func (svc *Service) TriggerCamperRun(ctx context.Context, campID, sessionID string, cfg solver.CamperSolverConfig) (RunDetailResponse, error) {
	snapshot, err := solver.BuildCamperCabinSnapshot(ctx, svc.queries, campID, sessionID)
	if err != nil {
		return RunDetailResponse{}, fmt.Errorf("error building camper snapshot: %w", err)
	}

	if msg := validateCamperCabin(snapshot); msg != "" {
		return RunDetailResponse{}, NewPreconditionError(msg)
	}

	solutions := solver.SolveCamperCabin(snapshot, cfg)
	if len(solutions) == 0 {
		return RunDetailResponse{}, ErrNoSolutions
	}

	runID, err := solver.StoreCamperSolutions(ctx, svc.pool, campID, sessionID, snapshot, solutions)
	if err != nil {
		return RunDetailResponse{}, fmt.Errorf("error storing camper solutions: %w", err)
	}

	return svc.GetRun(ctx, campID, runID)
}

func validateCamperCabin(snapshot solver.CamperCabinSnapshot) string {
	if len(snapshot.Campers) == 0 {
		return "No campers enrolled in this session"
	}
	if len(snapshot.Cabins) == 0 {
		return "No cabins configured for this session"
	}

	// Per-(age group, gender) capacity feasibility check: every camper must
	// have at least one cabin of their age group and gender with capacity.
	type key struct{ ageGroup, gender string }
	capacityByKey := map[key]int{}
	ageGroupNames := map[string]string{}
	for _, c := range snapshot.Cabins {
		capacityByKey[key{c.AgeGroupID, c.Gender}] += c.Capacity
		ageGroupNames[c.AgeGroupID] = c.AgeGroupName
	}
	demandByKey := map[key]int{}
	for _, c := range snapshot.Campers {
		demandByKey[key{c.AgeGroupID, c.Gender}]++
	}
	keys := make([]key, 0, len(demandByKey))
	for k := range demandByKey {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].ageGroup != keys[j].ageGroup {
			return keys[i].ageGroup < keys[j].ageGroup
		}
		return keys[i].gender < keys[j].gender
	})
	for _, k := range keys {
		demand := demandByKey[k]
		if capacityByKey[k] < demand {
			ageGroup := ageGroupNames[k.ageGroup]
			if ageGroup == "" {
				ageGroup = k.ageGroup
			}
			return fmt.Sprintf("Not enough %s cabin capacity (%d) for %d %s camper(s) in age group %q", k.gender, capacityByKey[k], demand, k.gender, ageGroup)
		}
	}

	return ""
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

	// Fetch the selected solution from the join table.
	var selectedSolutionID *string
	sel, err := svc.queries.GetSelectedSolution(ctx, db.GetSelectedSolutionParams{
		RunID:  runUUID,
		CampID: campUUID,
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return RunDetailResponse{}, fmt.Errorf("error getting selected solution for run %s: %w", runID, err)
	}
	if err == nil && sel.SolutionType == run.RunType {
		s := api.UUIDToString(sel.SolutionID)
		selectedSolutionID = &s
	}

	var solutions []SolutionSummaryResponse
	switch run.RunType {
	case "camper_cabin":
		solRows, err := svc.queries.ListCamperCabinSolutionsByRun(ctx, db.ListCamperCabinSolutionsByRunParams{
			AssignmentRunID: runUUID,
			CampID:          campUUID,
		})
		if err != nil {
			return RunDetailResponse{}, fmt.Errorf("error listing camper solutions for run %s: %w", runID, err)
		}
		solutions = make([]SolutionSummaryResponse, len(solRows))
		for i, s := range solRows {
			solutions[i] = toCamperSolutionSummaryResponse(s)
		}
	case "activity_schedule":
		solRows, err := svc.queries.ListActivitySolutionsByRun(ctx, db.ListActivitySolutionsByRunParams{
			AssignmentRunID: runUUID,
			CampID:          campUUID,
		})
		if err != nil {
			return RunDetailResponse{}, fmt.Errorf("error listing activity solutions for run %s: %w", runID, err)
		}
		solutions = make([]SolutionSummaryResponse, len(solRows))
		for i, s := range solRows {
			solutions[i] = toActivitySolutionSummaryResponse(s)
		}
	default:
		solRows, err := svc.queries.ListCounselorCabinSolutionsByRun(ctx, db.ListCounselorCabinSolutionsByRunParams{
			AssignmentRunID: runUUID,
			CampID:          campUUID,
		})
		if err != nil {
			return RunDetailResponse{}, fmt.Errorf("error listing solutions for run %s: %w", runID, err)
		}
		solutions = make([]SolutionSummaryResponse, len(solRows))
		for i, s := range solRows {
			solutions[i] = toSolutionSummaryResponse(s)
		}
	}

	return RunDetailResponse{
		RunResponse: toRunResponseFromGet(run, selectedSolutionID),
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

func (svc *Service) GetSolution(ctx context.Context, campID, runID, solutionID string) (SolutionDetailResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SolutionDetailResponse{}, err
	}

	runUUID, err := api.ParseUUID(runID)
	if err != nil {
		return SolutionDetailResponse{}, err
	}

	run, err := svc.queries.GetAssignmentRun(ctx, db.GetAssignmentRunParams{
		ID:     runUUID,
		CampID: campUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SolutionDetailResponse{}, ErrRunNotFound
		}
		return SolutionDetailResponse{}, fmt.Errorf("error getting assignment run %s: %w", runID, err)
	}

	switch run.RunType {
	case "camper_cabin":
		return svc.getCamperSolution(ctx, campUUID, runUUID, solutionID)
	case "activity_schedule":
		return svc.getActivitySolution(ctx, campUUID, runUUID, solutionID)
	default:
		return svc.getCounselorSolution(ctx, campUUID, runUUID, solutionID)
	}
}

func (svc *Service) getCounselorSolution(ctx context.Context, campUUID, runUUID pgtype.UUID, solutionID string) (SolutionDetailResponse, error) {
	solUUID, err := api.ParseUUID(solutionID)
	if err != nil {
		return SolutionDetailResponse{}, err
	}

	sol, err := svc.queries.GetCounselorCabinSolution(ctx, db.GetCounselorCabinSolutionParams{
		ID:              solUUID,
		CampID:          campUUID,
		AssignmentRunID: runUUID,
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

func (svc *Service) getCamperSolution(ctx context.Context, campUUID, runUUID pgtype.UUID, solutionID string) (SolutionDetailResponse, error) {
	solUUID, err := api.ParseUUID(solutionID)
	if err != nil {
		return SolutionDetailResponse{}, err
	}

	sol, err := svc.queries.GetCamperCabinSolution(ctx, db.GetCamperCabinSolutionParams{
		ID:              solUUID,
		CampID:          campUUID,
		AssignmentRunID: runUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SolutionDetailResponse{}, ErrSolutionNotFound
		}
		return SolutionDetailResponse{}, fmt.Errorf("error getting camper solution %s: %w", solutionID, err)
	}

	assignments, err := svc.queries.ListCamperCabinAssignmentsBySolution(ctx, db.ListCamperCabinAssignmentsBySolutionParams{
		SolutionID: solUUID,
		CampID:     campUUID,
	})
	if err != nil {
		return SolutionDetailResponse{}, fmt.Errorf("error listing camper assignments for solution %s: %w", solutionID, err)
	}

	explanations, err := svc.queries.ListCamperCabinExplanationsBySolution(ctx, db.ListCamperCabinExplanationsBySolutionParams{
		SolutionID: solUUID,
		CampID:     campUUID,
	})
	if err != nil {
		return SolutionDetailResponse{}, fmt.Errorf("error listing camper explanations for solution %s: %w", solutionID, err)
	}

	assignmentResponses := make([]AssignmentResponse, len(assignments))
	for i, a := range assignments {
		assignmentResponses[i] = AssignmentResponse{
			ID:           api.UUIDToString(a.ID),
			CamperID:     api.UUIDToString(a.CamperID),
			CamperName:   a.CamperName,
			CabinID:      api.UUIDToString(a.CabinID),
			CabinName:    a.CabinName,
			AgeGroupName: a.AgeGroupName,
		}
	}

	explanationResponses := make([]ExplanationResponse, len(explanations))
	for i, e := range explanations {
		var constraintName *string
		if e.ConstraintName.Valid {
			constraintName = &e.ConstraintName.String
		}
		var rank *int32
		if e.Rank.Valid {
			rank = &e.Rank.Int32
		}
		explanationResponses[i] = ExplanationResponse{
			ID:              api.UUIDToString(e.ID),
			CamperID:        api.UUIDToString(e.CamperID),
			CamperName:      e.CamperName,
			ExplanationType: e.ExplanationType,
			ConstraintName:  constraintName,
			Rank:            rank,
			Message:         e.Message,
		}
	}

	return SolutionDetailResponse{
		SolutionSummaryResponse: toCamperSolutionSummaryResponse(sol),
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

	// Fetch the run to determine its type.
	run, err := svc.queries.GetAssignmentRun(ctx, db.GetAssignmentRunParams{
		ID:     runUUID,
		CampID: campUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RunResponse{}, ErrRunNotFound
		}
		return RunResponse{}, fmt.Errorf("error getting assignment run %s: %w", runID, err)
	}

	// Verify the solution exists and belongs to this run.
	switch run.RunType {
	case "camper_cabin":
		sol, err := svc.queries.GetCamperCabinSolution(ctx, db.GetCamperCabinSolutionParams{
			ID:              solUUID,
			CampID:          campUUID,
			AssignmentRunID: runUUID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return RunResponse{}, ErrSolutionNotFound
			}
			return RunResponse{}, fmt.Errorf("error getting camper solution %s: %w", solutionID, err)
		}
		if sol.AssignmentRunID != runUUID {
			return RunResponse{}, ErrSolutionNotFound
		}
	case "activity_schedule":
		sol, err := svc.queries.GetActivitySolution(ctx, db.GetActivitySolutionParams{
			ID:              solUUID,
			CampID:          campUUID,
			AssignmentRunID: runUUID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return RunResponse{}, ErrSolutionNotFound
			}
			return RunResponse{}, fmt.Errorf("error getting activity solution %s: %w", solutionID, err)
		}
		if sol.AssignmentRunID != runUUID {
			return RunResponse{}, ErrSolutionNotFound
		}
	default:
		sol, err := svc.queries.GetCounselorCabinSolution(ctx, db.GetCounselorCabinSolutionParams{
			ID:              solUUID,
			CampID:          campUUID,
			AssignmentRunID: runUUID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return RunResponse{}, ErrSolutionNotFound
			}
			return RunResponse{}, fmt.Errorf("error getting solution %s: %w", solutionID, err)
		}
		if sol.AssignmentRunID != runUUID {
			return RunResponse{}, ErrSolutionNotFound
		}
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return RunResponse{}, fmt.Errorf("error beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := db.New(tx)

	// Lock all assignment_runs rows for this (camp, session, run_type) so a
	// concurrent SelectSolution on a sibling run can't race past the
	// conflict check below and trip the unique constraint on insert.
	if _, err := qtx.LockAssignmentRunsBySessionAndType(ctx, db.LockAssignmentRunsBySessionAndTypeParams{
		CampID:    campUUID,
		SessionID: run.SessionID,
		RunType:   run.RunType,
	}); err != nil {
		return RunResponse{}, fmt.Errorf("error locking assignment runs for selection: %w", err)
	}

	// Clear any other selected run of the same type in this session, so that
	// each (session, solution_type) has at most one selected run.
	conflictingRunIDs, err := qtx.ListConflictingSelectedRuns(ctx, db.ListConflictingSelectedRunsParams{
		CampID:    campUUID,
		SessionID: run.SessionID,
		RunType:   run.RunType,
		ID:        runUUID,
	})
	if err != nil {
		return RunResponse{}, fmt.Errorf("error listing conflicting selected runs: %w", err)
	}
	for _, conflictRunID := range conflictingRunIDs {
		if _, err := qtx.DeselectSolution(ctx, db.DeselectSolutionParams{
			RunID:  conflictRunID,
			CampID: campUUID,
		}); err != nil {
			return RunResponse{}, fmt.Errorf("error clearing conflicting selection on run %s: %w",
				api.UUIDToString(conflictRunID), err)
		}
		if _, err := qtx.UpdateAssignmentRunStatus(ctx, db.UpdateAssignmentRunStatusParams{
			ID:     conflictRunID,
			CampID: campUUID,
			Status: "completed",
		}); err != nil {
			return RunResponse{}, fmt.Errorf("error reverting status on run %s: %w",
				api.UUIDToString(conflictRunID), err)
		}
	}

	_, err = qtx.SelectSolution(ctx, db.SelectSolutionParams{
		CampID:       campUUID,
		SessionID:    run.SessionID,
		RunID:        runUUID,
		SolutionID:   solUUID,
		SolutionType: run.RunType,
	})
	if err != nil {
		return RunResponse{}, fmt.Errorf("error selecting solution %s for run %s: %w", solutionID, runID, err)
	}

	updatedRun, err := qtx.UpdateAssignmentRunStatus(ctx, db.UpdateAssignmentRunStatusParams{
		ID:     runUUID,
		CampID: campUUID,
		Status: "selected",
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RunResponse{}, ErrRunNotFound
		}
		return RunResponse{}, fmt.Errorf("error updating run status for run %s: %w", runID, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return RunResponse{}, fmt.Errorf("error committing solution selection: %w", err)
	}

	selectedStr := api.UUIDToString(solUUID)
	return toRunResponseFromGet(updatedRun, &selectedStr), nil
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

func toRunResponseFromGet(r db.AssignmentRun, selectedSolutionID *string) RunResponse {
	return RunResponse{
		ID:                 api.UUIDToString(r.ID),
		CampID:             api.UUIDToString(r.CampID),
		SessionID:          api.UUIDToString(r.SessionID),
		RunType:            r.RunType,
		Status:             r.Status,
		SelectedSolutionID: selectedSolutionID,
		CreatedAt:          r.CreatedAt.Time.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toSolutionSummaryResponse(s db.CounselorCabinSolution) SolutionSummaryResponse {
	return SolutionSummaryResponse{
		ID:              api.UUIDToString(s.ID),
		AssignmentRunID: api.UUIDToString(s.AssignmentRunID),
		SolutionIndex:   int(s.SolutionIndex),
		Score:           s.Score,
		ScoreBreakdown:  s.ScoreBreakdown,
	}
}

func toAssignmentResponse(a db.ListCounselorCabinAssignmentsBySolutionRow) AssignmentResponse {
	return AssignmentResponse{
		ID:            api.UUIDToString(a.ID),
		CounselorID:   api.UUIDToString(a.CounselorID),
		CounselorName: a.CounselorName,
		CabinID:       api.UUIDToString(a.CabinID),
		CabinName:     a.CabinName,
		AgeGroupName:  a.AgeGroupName,
	}
}

func toExplanationResponse(e db.ListCounselorCabinExplanationsBySolutionRow) ExplanationResponse {
	var constraintName *string
	if e.ConstraintName.Valid {
		constraintName = &e.ConstraintName.String
	}
	var rank *int32
	if e.Rank.Valid {
		rank = &e.Rank.Int32
	}

	return ExplanationResponse{
		ID:              api.UUIDToString(e.ID),
		CounselorID:     api.UUIDToString(e.CounselorID),
		CounselorName:   e.CounselorName,
		ExplanationType: e.ExplanationType,
		ConstraintName:  constraintName,
		Rank:            rank,
		Message:         e.Message,
	}
}

func toCamperSolutionSummaryResponse(s db.CamperCabinSolution) SolutionSummaryResponse {
	return SolutionSummaryResponse{
		ID:              api.UUIDToString(s.ID),
		AssignmentRunID: api.UUIDToString(s.AssignmentRunID),
		SolutionIndex:   int(s.SolutionIndex),
		Score:           s.Score,
		ScoreBreakdown:  s.ScoreBreakdown,
	}
}

func toActivitySolutionSummaryResponse(s db.ActivitySolution) SolutionSummaryResponse {
	return SolutionSummaryResponse{
		ID:              api.UUIDToString(s.ID),
		AssignmentRunID: api.UUIDToString(s.AssignmentRunID),
		SolutionIndex:   int(s.SolutionIndex),
		Score:           s.Score,
		ScoreBreakdown:  s.ScoreBreakdown,
	}
}

func (svc *Service) getActivitySolution(ctx context.Context, campUUID, runUUID pgtype.UUID, solutionID string) (SolutionDetailResponse, error) {
	solUUID, err := api.ParseUUID(solutionID)
	if err != nil {
		return SolutionDetailResponse{}, err
	}

	sol, err := svc.queries.GetActivitySolution(ctx, db.GetActivitySolutionParams{
		ID:              solUUID,
		CampID:          campUUID,
		AssignmentRunID: runUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SolutionDetailResponse{}, ErrSolutionNotFound
		}
		return SolutionDetailResponse{}, fmt.Errorf("error getting activity solution %s: %w", solutionID, err)
	}

	assignments, err := svc.queries.ListActivityAssignmentsBySolution(ctx, db.ListActivityAssignmentsBySolutionParams{
		SolutionID: solUUID,
		CampID:     campUUID,
	})
	if err != nil {
		return SolutionDetailResponse{}, fmt.Errorf("error listing activity assignments for solution %s: %w", solutionID, err)
	}

	explanations, err := svc.queries.ListActivityExplanationsBySolution(ctx, db.ListActivityExplanationsBySolutionParams{
		SolutionID: solUUID,
		CampID:     campUUID,
	})
	if err != nil {
		return SolutionDetailResponse{}, fmt.Errorf("error listing activity explanations for solution %s: %w", solutionID, err)
	}

	assignmentResponses := make([]AssignmentResponse, len(assignments))
	for i, a := range assignments {
		assignmentResponses[i] = AssignmentResponse{
			ID:                api.UUIDToString(a.ID),
			CounselorID:       api.UUIDToString(a.CounselorID),
			CounselorName:     a.CounselorName,
			SessionActivityID: api.UUIDToString(a.SessionActivityID),
			ActivityName:      a.ActivityName,
			TimeSlotName:      a.TimeSlotName,
			SortOrder:         a.SortOrder,
		}
	}

	explanationResponses := make([]ExplanationResponse, len(explanations))
	for i, e := range explanations {
		var constraintName *string
		if e.ConstraintName.Valid {
			constraintName = &e.ConstraintName.String
		}
		var rank *int32
		if e.Rank.Valid {
			rank = &e.Rank.Int32
		}
		explanationResponses[i] = ExplanationResponse{
			ID:              api.UUIDToString(e.ID),
			CounselorID:     api.UUIDToString(e.CounselorID),
			CounselorName:   e.CounselorName,
			ExplanationType: e.ExplanationType,
			ConstraintName:  constraintName,
			Rank:            rank,
			Message:         e.Message,
		}
	}

	return SolutionDetailResponse{
		SolutionSummaryResponse: toActivitySolutionSummaryResponse(sol),
		Assignments:             assignmentResponses,
		Explanations:            explanationResponses,
	}, nil
}
