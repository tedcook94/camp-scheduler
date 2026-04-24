package assignment

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// LoadCabinSolution returns both the counselor and camper halves of a paired
// cabin solution, looked up by the counselor solution's ID. The camper half
// is found by matching solution_index within the same run. Exposed at
// package scope so other packages (e.g. report) can reuse the assembly
// without depending on Service state.
func LoadCabinSolution(ctx context.Context, q *db.Queries, campID, runID, solutionID string) (SolutionDetailResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SolutionDetailResponse{}, err
	}
	runUUID, err := api.ParseUUID(runID)
	if err != nil {
		return SolutionDetailResponse{}, err
	}
	return loadCabinSolution(ctx, q, campUUID, runUUID, solutionID)
}

// LoadActivitySolution returns an activity solution detail (assignments,
// explanations, unassigned counselors) by its ID.
func LoadActivitySolution(ctx context.Context, q *db.Queries, campID, runID, solutionID string) (SolutionDetailResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SolutionDetailResponse{}, err
	}
	runUUID, err := api.ParseUUID(runID)
	if err != nil {
		return SolutionDetailResponse{}, err
	}
	return loadActivitySolution(ctx, q, campUUID, runUUID, solutionID)
}

func loadCabinSolution(ctx context.Context, q *db.Queries, campUUID, runUUID pgtype.UUID, solutionID string) (SolutionDetailResponse, error) {
	solUUID, err := api.ParseUUID(solutionID)
	if err != nil {
		return SolutionDetailResponse{}, err
	}

	counselorSol, err := q.GetCounselorCabinSolution(ctx, db.GetCounselorCabinSolutionParams{
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

	camperRows, err := q.ListCamperCabinSolutionsByRun(ctx, db.ListCamperCabinSolutionsByRunParams{
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

	counselorAssigns, err := q.ListCounselorCabinAssignmentsBySolution(ctx, db.ListCounselorCabinAssignmentsBySolutionParams{
		SolutionID: counselorSol.ID,
		CampID:     campUUID,
	})
	if err != nil {
		return SolutionDetailResponse{}, fmt.Errorf("error listing counselor assignments: %w", err)
	}
	counselorExpls, err := q.ListCounselorCabinExplanationsBySolution(ctx, db.ListCounselorCabinExplanationsBySolutionParams{
		SolutionID: counselorSol.ID,
		CampID:     campUUID,
	})
	if err != nil {
		return SolutionDetailResponse{}, fmt.Errorf("error listing counselor explanations: %w", err)
	}
	camperAssigns, err := q.ListCamperCabinAssignmentsBySolution(ctx, db.ListCamperCabinAssignmentsBySolutionParams{
		SolutionID: camperSol.ID,
		CampID:     campUUID,
	})
	if err != nil {
		return SolutionDetailResponse{}, fmt.Errorf("error listing camper assignments: %w", err)
	}
	camperExpls, err := q.ListCamperCabinExplanationsBySolution(ctx, db.ListCamperCabinExplanationsBySolutionParams{
		SolutionID: camperSol.ID,
		CampID:     campUUID,
	})
	if err != nil {
		return SolutionDetailResponse{}, fmt.Errorf("error listing camper explanations: %w", err)
	}

	unassignedRows, err := q.ListCounselorCabinUnassignedBySolution(ctx, db.ListCounselorCabinUnassignedBySolutionParams{
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

func loadActivitySolution(ctx context.Context, q *db.Queries, campUUID, runUUID pgtype.UUID, solutionID string) (SolutionDetailResponse, error) {
	solUUID, err := api.ParseUUID(solutionID)
	if err != nil {
		return SolutionDetailResponse{}, err
	}

	sol, err := q.GetActivitySolution(ctx, db.GetActivitySolutionParams{
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

	assignments, err := q.ListActivityAssignmentsBySolution(ctx, db.ListActivityAssignmentsBySolutionParams{
		SolutionID: solUUID,
		CampID:     campUUID,
	})
	if err != nil {
		return SolutionDetailResponse{}, fmt.Errorf("error listing activity assignments for solution %s: %w", solutionID, err)
	}

	explanations, err := q.ListActivityExplanationsBySolution(ctx, db.ListActivityExplanationsBySolutionParams{
		SolutionID: solUUID,
		CampID:     campUUID,
	})
	if err != nil {
		return SolutionDetailResponse{}, fmt.Errorf("error listing activity explanations for solution %s: %w", solutionID, err)
	}

	unassignedRows, err := q.ListCounselorActivityUnassignedBySolution(ctx, db.ListCounselorActivityUnassignedBySolutionParams{
		SolutionID: solUUID,
		CampID:     campUUID,
	})
	if err != nil {
		return SolutionDetailResponse{}, fmt.Errorf("error listing unassigned counselors for solution %s: %w", solutionID, err)
	}
	unassignedSlotRows, err := q.ListCounselorActivityUnassignedSlotsBySolution(ctx, db.ListCounselorActivityUnassignedSlotsBySolutionParams{
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
