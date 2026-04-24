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

// CamperUnassignedError is returned when a cabin run fails because one or
// more enrolled campers cannot be placed in any cabin. The diagnostic is
// computed via greedy fill against the camper snapshot, so it surfaces
// the campers that genuinely have no eligible cabin (capacity, age group,
// or gender mismatch) rather than a transient solver dead-end.
type CamperUnassignedError struct {
	Campers []UnassignedCamper
}

type UnassignedCamper struct {
	ID   string
	Name string
}

func (e *CamperUnassignedError) Error() string {
	if len(e.Campers) == 0 {
		return "one or more campers could not be assigned to any cabin"
	}
	names := make([]string, len(e.Campers))
	for i, c := range e.Campers {
		names[i] = c.Name
	}
	return fmt.Sprintf("%d camper(s) could not be assigned to any cabin: %s",
		len(e.Campers), joinNames(names))
}

func joinNames(names []string) string {
	if len(names) == 0 {
		return ""
	}
	out := names[0]
	for i := 1; i < len(names); i++ {
		out += ", " + names[i]
	}
	return out
}

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

// TriggerCabinRun replaces the existing cabin run for the session (if any)
// with a new combined counselor + camper cabin assignment. Counselors and
// campers share each cabin's total occupancy capacity.
func (svc *Service) TriggerCabinRun(ctx context.Context, campID, sessionID string, cfg solver.CabinSolverConfig) (RunDetailResponse, error) {
	snapshot, err := solver.BuildCabinSnapshot(ctx, svc.queries, campID, sessionID)
	if err != nil {
		return RunDetailResponse{}, fmt.Errorf("error building cabin snapshot: %w", err)
	}

	if msg := validateCabin(snapshot); msg != "" {
		return RunDetailResponse{}, NewPreconditionError(msg)
	}

	solutions := solver.SolveCabin(snapshot, cfg)
	if len(solutions) == 0 {
		if unassignable := solver.UnassignableCampers(snapshot.Camper); len(unassignable) > 0 {
			out := make([]UnassignedCamper, len(unassignable))
			for i, c := range unassignable {
				out[i] = UnassignedCamper{ID: c.ID, Name: c.Name}
			}
			return RunDetailResponse{}, &CamperUnassignedError{Campers: out}
		}
		return RunDetailResponse{}, ErrNoSolutions
	}

	runID, err := solver.StoreCabinSolutions(ctx, svc.pool, campID, sessionID, snapshot, solutions)
	if err != nil {
		return RunDetailResponse{}, fmt.Errorf("error storing cabin solutions: %w", err)
	}

	return svc.GetRun(ctx, campID, runID)
}

// validateCabin checks preconditions for a combined cabin run: the session
// must have cabins, the counselor side must satisfy minimum and gender-feasibility
// requirements, and per-(age group, gender) cabin capacity (after subtracting
// required counselors) must accommodate enrolled campers.
func validateCabin(snapshot solver.CabinSnapshot) string {
	if len(snapshot.Counselor.Cabins) == 0 {
		return "No cabins configured for this session"
	}

	if msg := validateCabinCapacities(snapshot.Counselor); msg != "" {
		return msg
	}
	if msg := validateCounselorSide(snapshot.Counselor); msg != "" {
		return msg
	}
	if msg := validateCamperSideWithCounselors(snapshot); msg != "" {
		return msg
	}
	return ""
}

// validateCabinCapacities catches misconfigured cabins where the cabin's
// required counselor count exceeds its total occupancy capacity, which
// would make the cabin impossible to staff under the combined-occupancy
// model.
func validateCabinCapacities(snapshot solver.SessionSnapshot) string {
	for _, c := range snapshot.Cabins {
		if c.RequiredCounselors > c.Capacity {
			return fmt.Sprintf("Cabin %q in age group %q requires %d counselors but has total capacity %d", c.Name, c.AgeGroupName, c.RequiredCounselors, c.Capacity)
		}
	}
	return ""
}

func validateCounselorSide(snapshot solver.SessionSnapshot) string {
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

// validateCamperSideWithCounselors checks per-(age group, gender) capacity
// for campers, after subtracting the cabin's required counselor count from
// each cabin's total capacity. This catches the case where group_size leaves
// no room for the required counselors plus the enrolled campers.
func validateCamperSideWithCounselors(snapshot solver.CabinSnapshot) string {
	if len(snapshot.Camper.Campers) == 0 {
		return ""
	}

	type key struct{ ageGroup, gender string }
	camperCapByKey := map[key]int{}
	ageGroupNames := map[string]string{}
	requiredByCabin := map[string]int{}
	for _, c := range snapshot.Counselor.Cabins {
		requiredByCabin[c.ID] = c.RequiredCounselors
	}
	for _, c := range snapshot.Camper.Cabins {
		// Subtract required counselors (which the counselor solver will
		// place) from the cabin's total capacity to get the worst-case
		// remaining capacity available for campers.
		req := requiredByCabin[c.ID]
		remaining := c.Capacity - req
		if remaining < 0 {
			remaining = 0
		}
		camperCapByKey[key{c.AgeGroupID, c.Gender}] += remaining
		ageGroupNames[c.AgeGroupID] = c.AgeGroupName
	}

	demandByKey := map[key]int{}
	for _, c := range snapshot.Camper.Campers {
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
		if camperCapByKey[k] < demand {
			ageGroup := ageGroupNames[k.ageGroup]
			if ageGroup == "" {
				ageGroup = k.ageGroup
			}
			return fmt.Sprintf("Not enough %s cabin capacity (%d available for campers after required counselors) for %d %s camper(s) in age group %q", k.gender, camperCapByKey[k], demand, k.gender, ageGroup)
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

// TriggerActivityRun replaces the existing activity run for the session
// (if any) with a fresh activity-schedule assignment.
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
	case solver.RunTypeCabin:
		solutions, err = svc.listCabinRunSolutions(ctx, campUUID, runUUID, runID)
		if err != nil {
			return RunDetailResponse{}, err
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
		return RunDetailResponse{}, fmt.Errorf("unknown run_type %q for run %s", run.RunType, runID)
	}

	return RunDetailResponse{
		RunResponse: toRunResponseFromGet(run, selectedSolutionID),
		Solutions:   solutions,
	}, nil
}

// listCabinRunSolutions fetches the counselor and camper halves of a cabin
// run and merges them into one summary per solution_index. The summary's ID
// is the counselor solution's ID (canonical for selection); the camper
// half's ID is exposed via CamperSolutionID. Scores are summed.
func (svc *Service) listCabinRunSolutions(ctx context.Context, campUUID, runUUID pgtype.UUID, runID string) ([]SolutionSummaryResponse, error) {
	counselorRows, err := svc.queries.ListCounselorCabinSolutionsByRun(ctx, db.ListCounselorCabinSolutionsByRunParams{
		AssignmentRunID: runUUID,
		CampID:          campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing counselor solutions for run %s: %w", runID, err)
	}
	camperRows, err := svc.queries.ListCamperCabinSolutionsByRun(ctx, db.ListCamperCabinSolutionsByRunParams{
		AssignmentRunID: runUUID,
		CampID:          campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing camper solutions for run %s: %w", runID, err)
	}

	camperByIndex := make(map[int32]db.CamperCabinSolution, len(camperRows))
	for _, c := range camperRows {
		camperByIndex[c.SolutionIndex] = c
	}

	out := make([]SolutionSummaryResponse, 0, len(counselorRows))
	for _, cs := range counselorRows {
		camper, ok := camperByIndex[cs.SolutionIndex]
		if !ok {
			return nil, fmt.Errorf("error pairing solution_index %d for cabin run %s: missing camper half", cs.SolutionIndex, runID)
		}
		out = append(out, SolutionSummaryResponse{
			ID:               api.UUIDToString(cs.ID),
			CamperSolutionID: api.UUIDToString(camper.ID),
			AssignmentRunID:  api.UUIDToString(cs.AssignmentRunID),
			SolutionIndex:    int(cs.SolutionIndex),
			Score:            cs.Score + camper.Score,
			ScoreBreakdown:   cs.ScoreBreakdown,
			CamperScoreBreakdown: camper.ScoreBreakdown,
		})
	}
	return out, nil
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
	case solver.RunTypeCabin:
		return svc.getCabinSolution(ctx, campUUID, runUUID, solutionID)
	case "activity_schedule":
		return svc.getActivitySolution(ctx, campUUID, runUUID, solutionID)
	default:
		return SolutionDetailResponse{}, fmt.Errorf("unknown run_type %q for run %s", run.RunType, runID)
	}
}

// getCabinSolution returns both the counselor and camper halves of a paired
// cabin solution, looked up by the counselor solution's ID. The camper half
// is found by matching solution_index within the same run.
func (svc *Service) getCabinSolution(ctx context.Context, campUUID, runUUID pgtype.UUID, solutionID string) (SolutionDetailResponse, error) {
	solUUID, err := api.ParseUUID(solutionID)
	if err != nil {
		return SolutionDetailResponse{}, err
	}

	counselorSol, err := svc.queries.GetCounselorCabinSolution(ctx, db.GetCounselorCabinSolutionParams{
		ID:              solUUID,
		CampID:          campUUID,
		AssignmentRunID: runUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SolutionDetailResponse{}, ErrSolutionNotFound
		}
		return SolutionDetailResponse{}, fmt.Errorf("error getting counselor solution %s: %w", solutionID, err)
	}

	// Find the camper half with the same solution_index.
	camperRows, err := svc.queries.ListCamperCabinSolutionsByRun(ctx, db.ListCamperCabinSolutionsByRunParams{
		AssignmentRunID: runUUID,
		CampID:          campUUID,
	})
	if err != nil {
		return SolutionDetailResponse{}, fmt.Errorf("error listing camper solutions for run: %w", err)
	}
	var camperSol db.CamperCabinSolution
	found := false
	for _, c := range camperRows {
		if c.SolutionIndex == counselorSol.SolutionIndex {
			camperSol = c
			found = true
			break
		}
	}
	if !found {
		return SolutionDetailResponse{}, fmt.Errorf("error finding camper half for solution_index %d", counselorSol.SolutionIndex)
	}

	counselorAssigns, err := svc.queries.ListCounselorCabinAssignmentsBySolution(ctx, db.ListCounselorCabinAssignmentsBySolutionParams{
		SolutionID: counselorSol.ID,
		CampID:     campUUID,
	})
	if err != nil {
		return SolutionDetailResponse{}, fmt.Errorf("error listing counselor assignments: %w", err)
	}
	counselorExpls, err := svc.queries.ListCounselorCabinExplanationsBySolution(ctx, db.ListCounselorCabinExplanationsBySolutionParams{
		SolutionID: counselorSol.ID,
		CampID:     campUUID,
	})
	if err != nil {
		return SolutionDetailResponse{}, fmt.Errorf("error listing counselor explanations: %w", err)
	}
	camperAssigns, err := svc.queries.ListCamperCabinAssignmentsBySolution(ctx, db.ListCamperCabinAssignmentsBySolutionParams{
		SolutionID: camperSol.ID,
		CampID:     campUUID,
	})
	if err != nil {
		return SolutionDetailResponse{}, fmt.Errorf("error listing camper assignments: %w", err)
	}
	camperExpls, err := svc.queries.ListCamperCabinExplanationsBySolution(ctx, db.ListCamperCabinExplanationsBySolutionParams{
		SolutionID: camperSol.ID,
		CampID:     campUUID,
	})
	if err != nil {
		return SolutionDetailResponse{}, fmt.Errorf("error listing camper explanations: %w", err)
	}

	unassignedRows, err := svc.queries.ListCounselorCabinUnassignedBySolution(ctx, db.ListCounselorCabinUnassignedBySolutionParams{
		SolutionID: counselorSol.ID,
		CampID:     campUUID,
	})
	if err != nil {
		return SolutionDetailResponse{}, fmt.Errorf("error listing unassigned counselors: %w", err)
	}

	assignments := make([]AssignmentResponse, 0, len(counselorAssigns)+len(camperAssigns))
	for _, a := range counselorAssigns {
		assignments = append(assignments, toAssignmentResponse(a))
	}
	for _, a := range camperAssigns {
		assignments = append(assignments, AssignmentResponse{
			ID:           api.UUIDToString(a.ID),
			CamperID:     api.UUIDToString(a.CamperID),
			CamperName:   a.CamperName,
			CabinID:      api.UUIDToString(a.CabinID),
			CabinName:    a.CabinName,
			AgeGroupName: a.AgeGroupName,
		})
	}

	explanations := make([]ExplanationResponse, 0, len(counselorExpls)+len(camperExpls))
	for _, e := range counselorExpls {
		explanations = append(explanations, toExplanationResponse(e))
	}
	for _, e := range camperExpls {
		var constraintName *string
		if e.ConstraintName.Valid {
			constraintName = &e.ConstraintName.String
		}
		var rank *int32
		if e.Rank.Valid {
			rank = &e.Rank.Int32
		}
		explanations = append(explanations, ExplanationResponse{
			ID:              api.UUIDToString(e.ID),
			CamperID:        api.UUIDToString(e.CamperID),
			CamperName:      e.CamperName,
			ExplanationType: e.ExplanationType,
			ConstraintName:  constraintName,
			Rank:            rank,
			Message:         e.Message,
		})
	}

	return SolutionDetailResponse{
		SolutionSummaryResponse: SolutionSummaryResponse{
			ID:                   api.UUIDToString(counselorSol.ID),
			CamperSolutionID:     api.UUIDToString(camperSol.ID),
			AssignmentRunID:      api.UUIDToString(counselorSol.AssignmentRunID),
			SolutionIndex:        int(counselorSol.SolutionIndex),
			Score:                counselorSol.Score + camperSol.Score,
			ScoreBreakdown:       counselorSol.ScoreBreakdown,
			CamperScoreBreakdown: camperSol.ScoreBreakdown,
		},
		Assignments:          assignments,
		Explanations:         explanations,
		UnassignedCounselors: toCabinUnassignedCounselorResponses(unassignedRows),
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

	switch run.RunType {
	case solver.RunTypeCabin:
		// For cabin runs, the canonical solution_id is the counselor half.
		sol, err := svc.queries.GetCounselorCabinSolution(ctx, db.GetCounselorCabinSolutionParams{
			ID:              solUUID,
			CampID:          campUUID,
			AssignmentRunID: runUUID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return RunResponse{}, ErrSolutionNotFound
			}
			return RunResponse{}, fmt.Errorf("error getting cabin solution %s: %w", solutionID, err)
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
		return RunResponse{}, fmt.Errorf("unknown run_type %q for run %s", run.RunType, runID)
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return RunResponse{}, fmt.Errorf("error beginning transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := db.New(tx)

	if _, err := qtx.LockAssignmentRunsBySessionAndType(ctx, db.LockAssignmentRunsBySessionAndTypeParams{
		CampID:    campUUID,
		SessionID: run.SessionID,
		RunType:   run.RunType,
	}); err != nil {
		return RunResponse{}, fmt.Errorf("error locking assignment runs for selection: %w", err)
	}

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

	unassignedRows, err := svc.queries.ListCounselorActivityUnassignedBySolution(ctx, db.ListCounselorActivityUnassignedBySolutionParams{
		SolutionID: solUUID,
		CampID:     campUUID,
	})
	if err != nil {
		return SolutionDetailResponse{}, fmt.Errorf("error listing unassigned counselors for solution %s: %w", solutionID, err)
	}
	unassignedSlotRows, err := svc.queries.ListCounselorActivityUnassignedSlotsBySolution(ctx, db.ListCounselorActivityUnassignedSlotsBySolutionParams{
		SolutionID: solUUID,
		CampID:     campUUID,
	})
	if err != nil {
		return SolutionDetailResponse{}, fmt.Errorf("error listing unassigned counselor slots for solution %s: %w", solutionID, err)
	}
	unassignedResponses := toActivityUnassignedCounselorResponses(unassignedRows, unassignedSlotRows)

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
		UnassignedCounselors:    unassignedResponses,
	}, nil
}

func toCabinUnassignedCounselorResponses(rows []db.ListCounselorCabinUnassignedBySolutionRow) []UnassignedCounselorResponse {
	out := make([]UnassignedCounselorResponse, len(rows))
	for i, r := range rows {
		out[i] = UnassignedCounselorResponse{
			CounselorID:   api.UUIDToString(r.CounselorID),
			CounselorName: r.CounselorName,
		}
	}
	return out
}

func toActivityUnassignedCounselorResponses(
	rows []db.ListCounselorActivityUnassignedBySolutionRow,
	slotRows []db.ListCounselorActivityUnassignedSlotsBySolutionRow,
) []UnassignedCounselorResponse {
	slotsByCounselor := make(map[string][]UnassignedTimeSlotRef)
	for _, sr := range slotRows {
		cID := api.UUIDToString(sr.CounselorID)
		slotsByCounselor[cID] = append(slotsByCounselor[cID], UnassignedTimeSlotRef{
			SessionTimeSlotID: api.UUIDToString(sr.SessionTimeSlotID),
			TimeSlotName:      sr.TimeSlotName,
		})
	}
	out := make([]UnassignedCounselorResponse, len(rows))
	for i, r := range rows {
		cID := api.UUIDToString(r.CounselorID)
		out[i] = UnassignedCounselorResponse{
			CounselorID:      cID,
			CounselorName:    r.CounselorName,
			MissingTimeSlots: slotsByCounselor[cID],
		}
	}
	return out
}
