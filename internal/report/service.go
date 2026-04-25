package report

import (
	"context"
	"errors"
	"fmt"
	"time"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/assignment"
	"camp-scheduler/internal/db"
	"camp-scheduler/internal/solver"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ErrNoSelectedSolution indicates that the requested session has no
// selected assignment for the requested run type. Callers should map this
// to a 404 with an explanatory message pointing at the assignments page.
var ErrNoSelectedSolution = errors.New("no selected solution for session")

// ErrSessionNotFound and ErrCampNotFound surface from header loading when
// the requested camp/session does not exist or is not visible to the
// caller. Mapped to 404 by the controller.
var (
	ErrSessionNotFound = errors.New("session not found")
	ErrCampNotFound    = errors.New("camp not found")
)

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

// Header carries the camp identity on every report so directors can hand a
// printout to staff without ambiguity.
type Header struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SessionHeader carries minimal session identity for report headers.
type SessionHeader struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type CabinReport struct {
	Camp        Header          `json:"camp"`
	Session     SessionHeader   `json:"session"`
	GeneratedAt time.Time       `json:"generated_at"`
	Cabins      []CabinGroup    `json:"cabins"`
	Unassigned  CabinUnassigned `json:"unassigned"`
}

type CabinGroup struct {
	AgeGroupName       string         `json:"age_group_name"`
	CabinName          string         `json:"cabin_name"`
	Gender             string         `json:"gender"`
	Capacity           int32          `json:"capacity"`
	RequiredCounselors int32          `json:"required_counselors"`
	Counselors         []CounselorRow `json:"counselors"`
	Campers            []CamperRow    `json:"campers"`
}

type CabinUnassigned struct {
	Counselors []CounselorRow `json:"counselors"`
	Campers    []CamperRow    `json:"campers"`
}

type CounselorRow struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Junior bool   `json:"junior"`
	Gender string `json:"gender"`
}

type CamperRow struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Gender string `json:"gender"`
}

type ActivityReport struct {
	Camp        Header                  `json:"camp"`
	Session     SessionHeader           `json:"session"`
	GeneratedAt time.Time               `json:"generated_at"`
	TimeSlots   []TimeSlotGroup         `json:"time_slots"`
	Unassigned  []ActivityUnassignedRow `json:"unassigned"`
}

type TimeSlotGroup struct {
	Name       string             `json:"name"`
	SortOrder  int32              `json:"sort_order"`
	Activities []ActivityRowGroup `json:"activities"`
}

type ActivityRowGroup struct {
	Name       string         `json:"name"`
	Counselors []CounselorRow `json:"counselors"`
}

// ActivityUnassignedRow describes a counselor that was either left fully
// unassigned for a session or missed specific time slots despite having
// eligible activities in them.
type ActivityUnassignedRow struct {
	CounselorRow
	MissingTimeSlots []string `json:"missing_time_slots,omitempty"`
}

func (svc *Service) GetCabinReport(ctx context.Context, campID, sessionID string) (*CabinReport, error) {
	campUUID, sessionUUID, err := parseIDs(campID, sessionID)
	if err != nil {
		return nil, err
	}

	// Validate camp + session up front so invalid IDs surface as a clear
	// "not found" instead of the misleading "no selected assignment".
	camp, session, err := svc.loadHeaders(ctx, campUUID, sessionUUID)
	if err != nil {
		return nil, err
	}

	runID, solutionID, err := svc.findSelectedRun(ctx, campUUID, sessionUUID, solver.RunTypeCabin)
	if err != nil {
		return nil, err
	}

	detail, err := assignment.LoadCabinSolution(ctx, svc.queries, campID, runID, solutionID)
	if err != nil {
		return nil, fmt.Errorf("error loading cabin solution: %w", err)
	}

	counselors, campers, err := svc.loadPeopleByID(ctx, campUUID)
	if err != nil {
		return nil, err
	}

	cabins, err := svc.queries.ListSessionCabinsWithCapacity(ctx, db.ListSessionCabinsWithCapacityParams{
		SessionID: sessionUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing session cabins: %w", err)
	}

	report := buildCabinReport(detail, cabins, counselors, campers)
	report.Camp = camp
	report.Session = session
	report.GeneratedAt = time.Now().UTC()
	return report, nil
}

func (svc *Service) GetActivityReport(ctx context.Context, campID, sessionID string) (*ActivityReport, error) {
	campUUID, sessionUUID, err := parseIDs(campID, sessionID)
	if err != nil {
		return nil, err
	}

	camp, session, err := svc.loadHeaders(ctx, campUUID, sessionUUID)
	if err != nil {
		return nil, err
	}

	runID, solutionID, err := svc.findSelectedRun(ctx, campUUID, sessionUUID, solver.RunTypeActivitySchedule)
	if err != nil {
		return nil, err
	}

	detail, err := assignment.LoadActivitySolution(ctx, svc.queries, campID, runID, solutionID)
	if err != nil {
		return nil, fmt.Errorf("error loading activity solution: %w", err)
	}

	counselors, _, err := svc.loadPeopleByID(ctx, campUUID)
	if err != nil {
		return nil, err
	}

	// Pull the full schedule so empty (slot, activity) cells appear in the
	// report. Without this, configured-but-unstaffed activities would be
	// invisible in CSV/PDF exports.
	schedule, err := svc.queries.ListSessionActivitiesWithDetails(ctx, db.ListSessionActivitiesWithDetailsParams{
		SessionID: sessionUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing session activities: %w", err)
	}

	report := buildActivityReport(detail, schedule, counselors)
	report.Camp = camp
	report.Session = session
	report.GeneratedAt = time.Now().UTC()
	return report, nil
}

func parseIDs(campID, sessionID string) (pgtype.UUID, pgtype.UUID, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, err
	}
	return campUUID, sessionUUID, nil
}

func (svc *Service) findSelectedRun(ctx context.Context, campUUID, sessionUUID pgtype.UUID, runType string) (string, string, error) {
	rows, err := svc.queries.ListAssignmentRunsBySession(ctx, db.ListAssignmentRunsBySessionParams{
		SessionID: sessionUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return "", "", fmt.Errorf("error listing assignment runs: %w", err)
	}
	for _, r := range rows {
		if r.RunType != runType || r.Status != "selected" || !r.SelectedSolutionID.Valid {
			continue
		}
		return api.UUIDToString(r.ID), api.UUIDToString(r.SelectedSolutionID), nil
	}
	return "", "", ErrNoSelectedSolution
}

func (svc *Service) loadHeaders(ctx context.Context, campUUID, sessionUUID pgtype.UUID) (Header, SessionHeader, error) {
	camp, err := svc.queries.GetCamp(ctx, campUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Header{}, SessionHeader{}, ErrCampNotFound
		}
		return Header{}, SessionHeader{}, fmt.Errorf("error getting camp: %w", err)
	}
	session, err := svc.queries.GetSession(ctx, db.GetSessionParams{
		ID:     sessionUUID,
		CampID: campUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Header{}, SessionHeader{}, ErrSessionNotFound
		}
		return Header{}, SessionHeader{}, fmt.Errorf("error getting session: %w", err)
	}
	return Header{
			ID:   api.UUIDToString(camp.ID),
			Name: camp.CampName,
		}, SessionHeader{
			ID:   api.UUIDToString(session.ID),
			Name: session.SessionName,
		}, nil
}

// loadPeopleByID fetches all counselors and campers for the camp once and
// indexes them by ID so per-assignment lookups don't fan out.
func (svc *Service) loadPeopleByID(ctx context.Context, campUUID pgtype.UUID) (map[string]db.Counselor, map[string]db.Camper, error) {
	counselors, err := svc.queries.ListCounselors(ctx, campUUID)
	if err != nil {
		return nil, nil, fmt.Errorf("error listing counselors: %w", err)
	}
	campers, err := svc.queries.ListCampers(ctx, campUUID)
	if err != nil {
		return nil, nil, fmt.Errorf("error listing campers: %w", err)
	}
	cMap := make(map[string]db.Counselor, len(counselors))
	for _, c := range counselors {
		cMap[api.UUIDToString(c.ID)] = c
	}
	mMap := make(map[string]db.Camper, len(campers))
	for _, c := range campers {
		mMap[api.UUIDToString(c.ID)] = db.Camper{
			ID:        c.ID,
			CampID:    c.CampID,
			FirstName: c.FirstName,
			LastName:  c.LastName,
			Gender:    c.Gender,
		}
	}
	return cMap, mMap, nil
}
