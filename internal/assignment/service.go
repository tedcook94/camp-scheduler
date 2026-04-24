package assignment

import (
	"context"
	"errors"
	"fmt"

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

// CamperUnassignedError is returned when a cabin run fails because cabin
// capacity is insufficient for the enrolled campers, given the counselor
// occupancy each cabin must reserve. Diagnostics are bucketed by
// (age group, gender) rather than naming individual campers, since a
// pure capacity shortage doesn't single out specific people.
type CamperUnassignedError struct {
	Shortages []solver.CamperShortage
}

func (e *CamperUnassignedError) Error() string {
	if len(e.Shortages) == 0 {
		return "one or more campers could not be assigned to any cabin"
	}
	total := 0
	parts := make([]string, len(e.Shortages))
	for i, s := range e.Shortages {
		total += s.Count
		switch s.Reason {
		case solver.ShortageNoMatchingCabin:
			parts[i] = fmt.Sprintf("%s (%s) has no matching cabin (%d camper(s))", s.AgeGroupName, s.Gender, s.Count)
		case solver.ShortageOverCapacity:
			parts[i] = fmt.Sprintf("%s (%s) is over capacity by %d", s.AgeGroupName, s.Gender, s.Count)
		default:
			parts[i] = fmt.Sprintf("%s (%s): %d camper(s)", s.AgeGroupName, s.Gender, s.Count)
		}
	}
	return fmt.Sprintf("%d camper(s) cannot be placed in any cabin: %s",
		total, joinShortageParts(parts))
}

func joinShortageParts(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += "; " + parts[i]
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

	if shortages := solver.CamperShortages(reduceCamperCapacityForCounselors(snapshot)); len(shortages) > 0 {
		return RunDetailResponse{}, &CamperUnassignedError{Shortages: shortages}
	}

	solutions := solver.SolveCabin(snapshot, cfg)
	if len(solutions) == 0 {
		return RunDetailResponse{}, ErrNoSolutions
	}

	runID, err := solver.StoreCabinSolutions(ctx, svc.pool, campID, sessionID, snapshot, solutions)
	if err != nil {
		return RunDetailResponse{}, fmt.Errorf("error storing cabin solutions: %w", err)
	}

	return svc.GetRun(ctx, campID, runID)
}

// validateCabin checks preconditions for a combined cabin run: the session
// must have cabins and the counselor side must satisfy minimum and
// gender-feasibility requirements. Per-(age group, gender) camper capacity
// shortages are surfaced separately via CamperShortages so the error
// payload carries structured bucket information.
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

// reduceCamperCapacityForCounselors returns a copy of the cabin snapshot's
// camper side with each cabin's capacity reduced by the minimum required
// counselor reservation. This is a best-case / necessary-condition precheck
// for camper capacity only: the combined cabin solver reduces camper
// capacity by the actual number of counselors placed in each cabin, which
// may be greater than the required minimum.
func reduceCamperCapacityForCounselors(snapshot solver.CabinSnapshot) solver.CamperCabinSnapshot {
	requiredByCabin := map[string]int{}
	for _, c := range snapshot.Counselor.Cabins {
		requiredByCabin[c.ID] = c.RequiredCounselors
	}
	reduced := snapshot.Camper
	cabins := make([]solver.CamperCabin, len(snapshot.Camper.Cabins))
	for i, c := range snapshot.Camper.Cabins {
		c.Capacity -= requiredByCabin[c.ID]
		if c.Capacity < 0 {
			c.Capacity = 0
		}
		cabins[i] = c
	}
	reduced.Cabins = cabins
	return reduced
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

// getCabinSolution is a thin wrapper around loadCabinSolution that uses
// the service's queries handle.
func (svc *Service) getCabinSolution(ctx context.Context, campUUID, runUUID pgtype.UUID, solutionID string) (SolutionDetailResponse, error) {
	return loadCabinSolution(ctx, svc.queries, campUUID, runUUID, solutionID)
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
	return loadActivitySolution(ctx, svc.queries, campUUID, runUUID, solutionID)
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
