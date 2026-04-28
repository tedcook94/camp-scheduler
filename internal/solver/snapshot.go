package solver

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func BuildSnapshot(ctx context.Context, queries *db.Queries, campID, sessionID string) (SessionSnapshot, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionSnapshot{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return SessionSnapshot{}, err
	}

	session, err := queries.GetSession(ctx, db.GetSessionParams{
		ID:     sessionUUID,
		CampID: campUUID,
	})
	if err != nil {
		return SessionSnapshot{}, fmt.Errorf("error getting session: %w", err)
	}

	cabins, err := loadCabins(ctx, queries, sessionUUID, campUUID)
	if err != nil {
		return SessionSnapshot{}, err
	}

	counselors, err := loadCounselors(ctx, queries, sessionUUID, campUUID)
	if err != nil {
		return SessionSnapshot{}, err
	}
	rosterSet := buildRosterSet(counselors)

	ageGroupPrefs, err := loadAgeGroupPreferences(ctx, queries, sessionUUID, campUUID)
	if err != nil {
		return SessionSnapshot{}, err
	}

	cocounselorPrefs, err := loadCocounselorPreferences(ctx, queries, sessionUUID, campUUID)
	if err != nil {
		return SessionSnapshot{}, err
	}

	placements, err := loadPreviousPlacements(ctx, queries, session, campUUID)
	if err != nil {
		return SessionSnapshot{}, err
	}

	unmetAG, unmetCo, err := loadPreviouslyUnmetCounselorPreferences(ctx, queries, session, campUUID)
	if err != nil {
		return SessionSnapshot{}, err
	}

	overrides, err := loadCounselorCabinOverrides(ctx, queries, sessionUUID, campUUID, rosterSet)
	if err != nil {
		return SessionSnapshot{}, err
	}

	// Resolve override values from session_cabin_id (the persisted
	// FK target) to the cabins.id keys the solver uses internally.
	sagcToCabin := make(map[string]string, len(cabins))
	for _, c := range cabins {
		sagcToCabin[c.SessionCabinID] = c.ID
	}
	overrides = remapOverrideValues(overrides, sagcToCabin)

	// Prune all counselor-keyed maps to only counselors on the session roster.
	// This means a counselor removed from the roster (or with stale prefs from
	// a prior roster membership) is invisible to the solver and explainer.
	ageGroupPrefs = filterMapByRoster(ageGroupPrefs, rosterSet)
	cocounselorPrefs = filterCocounselorPrefsBySource(cocounselorPrefs, rosterSet)
	placements = filterMapByRoster(placements, rosterSet)
	unmetAG = filterMapByRoster(unmetAG, rosterSet)
	unmetCo = filterCocounselorUnmetByRoster(unmetCo, rosterSet)

	return SessionSnapshot{
		SessionID:                   sessionID,
		Cabins:                      cabins,
		Counselors:                  counselors,
		AgeGroupPreferences:         ageGroupPrefs,
		CocounselorPreferences:      cocounselorPrefs,
		CounselorPreviousPlacements: placements,
		UnmetAgeGroupPreferences:    unmetAG,
		UnmetCocounselorPreferences: unmetCo,
		Overrides:                   overrides,
	}, nil
}

// remapOverrideValues replaces each override target value via the provided
// map, dropping entries whose target is not present in the map. Used to
// translate persisted session_cabin IDs to the cabins.id keys
// used by the cabin solvers, and to drop overrides whose target cabin has
// been archived.
func remapOverrideValues(overrides, valueMap map[string]string) map[string]string {
	if len(overrides) == 0 {
		return overrides
	}
	out := make(map[string]string, len(overrides))
	for k, v := range overrides {
		if mapped, ok := valueMap[v]; ok {
			out[k] = mapped
		}
	}
	return out
}

// filterOverridesByCabin drops overrides whose target ID is no longer
// present in the snapshot. Used by solvers that key their cabins by the
// persisted FK directly (e.g. activity solver keying on session_activity_id).
func filterOverridesByCabin(overrides map[string]string, cabins map[string]bool) map[string]string {
	if len(overrides) == 0 {
		return overrides
	}
	out := make(map[string]string, len(overrides))
	for k, v := range overrides {
		if cabins[v] {
			out[k] = v
		}
	}
	return out
}

// loadCounselorCabinOverrides loads admin-pinned counselor->cabin assignments
// for the session and filters out any whose counselor is no longer on the
// active roster. Defensive only: the override domain validates roster
// membership at save time and the solver trigger revalidates.
func loadCounselorCabinOverrides(ctx context.Context, queries *db.Queries, sessionID, campID pgtype.UUID, roster map[string]bool) (map[string]string, error) {
	rows, err := queries.ListCounselorCabinOverridesForSolver(ctx, db.ListCounselorCabinOverridesForSolverParams{
		SessionID: sessionID,
		CampID:    campID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing counselor cabin overrides: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	overrides := make(map[string]string, len(rows))
	for _, r := range rows {
		counselorID := api.UUIDToString(r.CounselorID)
		if !roster[counselorID] {
			continue
		}
		overrides[counselorID] = api.UUIDToString(r.SessionCabinID)
	}
	return overrides, nil
}

func loadCabins(ctx context.Context, queries *db.Queries, sessionID, campID pgtype.UUID) ([]Cabin, error) {
	rows, err := queries.ListSessionCabinsWithCapacity(ctx, db.ListSessionCabinsWithCapacityParams{
		SessionID: sessionID,
		CampID:    campID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing session cabins: %w", err)
	}

	cabins := make([]Cabin, 0, len(rows))
	for _, r := range rows {
		// Skip cabins (or their age groups) that have been archived since the
		// session config was set up. Admins clean these up via the session config UI.
		if r.CabinArchived || r.AgeGroupArchived {
			continue
		}
		cabins = append(cabins, Cabin{
			ID:                     api.UUIDToString(r.ID),
			Name:                   r.CabinName,
			AgeGroupID:             api.UUIDToString(r.AgeGroupID),
			AgeGroupName:           r.AgeGroupName,
			SessionCabinID: api.UUIDToString(r.SessionCabinID),
			RequiredCounselors:     int(r.RequiredCounselors),
			Capacity:               int(r.GroupSize),
			Gender:                 r.Gender,
		})
	}
	return cabins, nil
}

// loadSessionRoster returns the rows from the session counselor roster,
// filtered to only currently active (non-archived) counselors. Archived
// counselors left on the roster are skipped as a safety net so they never
// reach a solver.
func loadSessionRoster(ctx context.Context, queries *db.Queries, sessionID, campID pgtype.UUID) ([]db.ListSessionCounselorsRow, error) {
	rows, err := queries.ListSessionCounselors(ctx, db.ListSessionCounselorsParams{
		SessionID: sessionID,
		CampID:    campID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing session counselors: %w", err)
	}
	active := rows[:0]
	for _, r := range rows {
		if !r.Archived {
			active = append(active, r)
		}
	}
	return active, nil
}

// buildRosterSet turns a roster slice into a lookup set keyed by counselor ID.
func buildRosterSet(counselors []Counselor) map[string]bool {
	set := make(map[string]bool, len(counselors))
	for _, c := range counselors {
		set[c.ID] = true
	}
	return set
}

// filterMapByRoster drops keys not present in the roster set. Used for any
// counselor-keyed map whose values do not themselves reference counselor IDs.
func filterMapByRoster[V any](m map[string]V, roster map[string]bool) map[string]V {
	if len(m) == 0 {
		return m
	}
	out := make(map[string]V, len(m))
	for k, v := range m {
		if roster[k] {
			out[k] = v
		}
	}
	return out
}

// filterCocounselorPrefsBySource drops outer keys (source counselors) that are
// not on the roster. Target IDs are kept regardless so the explainer can flag
// off-roster targets as ineligible preferences instead of silently dropping
// them.
func filterCocounselorPrefsBySource(m map[string][]RankedPreference, roster map[string]bool) map[string][]RankedPreference {
	if len(m) == 0 {
		return m
	}
	out := make(map[string][]RankedPreference, len(m))
	for k, prefs := range m {
		if !roster[k] {
			continue
		}
		out[k] = prefs
	}
	return out
}

// filterCocounselorUnmetByRoster drops outer and inner keys that are not on
// the roster. The unmet-cocounselor map is keyed by counselor ID with values
// that are sets of preferred-counselor IDs.
func filterCocounselorUnmetByRoster(m map[string]map[string]bool, roster map[string]bool) map[string]map[string]bool {
	if len(m) == 0 {
		return m
	}
	out := make(map[string]map[string]bool, len(m))
	for k, inner := range m {
		if !roster[k] {
			continue
		}
		keptInner := make(map[string]bool, len(inner))
		for tID := range inner {
			if roster[tID] {
				keptInner[tID] = true
			}
		}
		out[k] = keptInner
	}
	return out
}

func loadCounselors(ctx context.Context, queries *db.Queries, sessionID, campID pgtype.UUID) ([]Counselor, error) {
	rows, err := loadSessionRoster(ctx, queries, sessionID, campID)
	if err != nil {
		return nil, err
	}

	counselors := make([]Counselor, len(rows))
	for i, r := range rows {
		counselors[i] = Counselor{
			ID:        api.UUIDToString(r.CounselorID),
			FirstName: r.CounselorFirstName,
			LastName:  r.CounselorLastName,
			Name:      r.CounselorName,
			IsJunior:  r.JuniorCounselor,
			Gender:    r.Gender,
		}
	}
	return counselors, nil
}

func loadAgeGroupPreferences(ctx context.Context, queries *db.Queries, sessionID, campID pgtype.UUID) (map[string][]RankedPreference, error) {
	rows, err := queries.ListSessionAgeGroupPreferences(ctx, db.ListSessionAgeGroupPreferencesParams{
		SessionID: sessionID,
		CampID:    campID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing session age group preferences: %w", err)
	}

	prefs := make(map[string][]RankedPreference)
	for _, r := range rows {
		cID := api.UUIDToString(r.CounselorID)
		prefs[cID] = append(prefs[cID], RankedPreference{
			TargetID: api.UUIDToString(r.AgeGroupID),
			Rank:     int(r.Rank),
		})
	}
	return prefs, nil
}

func loadCocounselorPreferences(ctx context.Context, queries *db.Queries, sessionID, campID pgtype.UUID) (map[string][]RankedPreference, error) {
	rows, err := queries.ListSessionCocounselorPreferences(ctx, db.ListSessionCocounselorPreferencesParams{
		SessionID: sessionID,
		CampID:    campID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing session cocounselor preferences: %w", err)
	}

	prefs := make(map[string][]RankedPreference)
	for _, r := range rows {
		cID := api.UUIDToString(r.CounselorID)
		prefs[cID] = append(prefs[cID], RankedPreference{
			TargetID: api.UUIDToString(r.PreferredCounselorID),
			Rank:     int(r.Rank),
		})
	}
	return prefs, nil
}

func loadPreviousPlacements(ctx context.Context, queries *db.Queries, session db.Session, campID pgtype.UUID) (map[string]CounselorPreviousPlacement, error) {
	if !session.PreviousSession.Valid {
		return nil, nil
	}

	rows, err := queries.ListSessionHistory(ctx, db.ListSessionHistoryParams{
		SessionID: session.PreviousSession,
		CampID:    campID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing previous session history: %w", err)
	}

	if len(rows) == 0 {
		return nil, nil
	}

	// Group counselors by cabin to derive co-counselor relationships.
	cabinCounselors := make(map[string][]string)
	for _, r := range rows {
		if !r.CabinID.Valid {
			continue
		}
		cabinID := api.UUIDToString(r.CabinID)
		counselorID := api.UUIDToString(r.CounselorID)
		cabinCounselors[cabinID] = append(cabinCounselors[cabinID], counselorID)
	}

	placements := make(map[string]CounselorPreviousPlacement, len(rows))
	for _, r := range rows {
		counselorID := api.UUIDToString(r.CounselorID)
		cabinID := ""
		if r.CabinID.Valid {
			cabinID = api.UUIDToString(r.CabinID)
		}

		var cocounselorIDs []string
		if cabinID != "" {
			for _, id := range cabinCounselors[cabinID] {
				if id != counselorID {
					cocounselorIDs = append(cocounselorIDs, id)
				}
			}
		}

		placements[counselorID] = CounselorPreviousPlacement{
			AgeGroupID:     api.UUIDToString(r.AgeGroupID),
			CabinID:        cabinID,
			CocounselorIDs: cocounselorIDs,
		}
	}
	return placements, nil
}

// loadPreviouslyUnmetCounselorPreferences finds preferences from the previous
// session's selected solution that were submitted but not satisfied. It returns
// two maps: unmet age group preferences and unmet cocounselor preferences,
// each keyed by counselor ID -> set of target IDs.
func loadPreviouslyUnmetCounselorPreferences(ctx context.Context, queries *db.Queries, session db.Session, campID pgtype.UUID) (map[string]map[string]bool, map[string]map[string]bool, error) {
	if !session.PreviousSession.Valid {
		return nil, nil, nil
	}

	solutionID, err := queries.GetSelectedCounselorCabinSolutionBySession(ctx, db.GetSelectedCounselorCabinSolutionBySessionParams{
		SessionID: session.PreviousSession,
		CampID:    campID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("error getting previous session selected solution: %w", err)
	}

	prevAGPrefs, err := loadAgeGroupPreferences(ctx, queries, session.PreviousSession, campID)
	if err != nil {
		return nil, nil, err
	}

	prevCoPrefs, err := loadCocounselorPreferences(ctx, queries, session.PreviousSession, campID)
	if err != nil {
		return nil, nil, err
	}

	if len(prevAGPrefs) == 0 && len(prevCoPrefs) == 0 {
		return nil, nil, nil
	}

	assignments, err := queries.ListCounselorCabinAssignmentsBySolution(ctx, db.ListCounselorCabinAssignmentsBySolutionParams{
		SolutionID: solutionID,
		CampID:     campID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error listing previous solution assignments: %w", err)
	}

	prevCabins, err := loadCabins(ctx, queries, session.PreviousSession, campID)
	if err != nil {
		return nil, nil, err
	}

	cabinAgeGroup := make(map[string]string, len(prevCabins))
	for _, c := range prevCabins {
		cabinAgeGroup[c.ID] = c.AgeGroupID
	}

	counselorCabin := make(map[string]string)
	for _, a := range assignments {
		cID := api.UUIDToString(a.CounselorID)
		cabID := api.UUIDToString(a.CabinID)
		counselorCabin[cID] = cabID
	}

	unmetAG := diffAgeGroupPreferences(prevAGPrefs, counselorCabin, cabinAgeGroup)
	unmetCo := diffCocounselorPreferences(prevCoPrefs, counselorCabin)

	return unmetAG, unmetCo, nil
}

func diffAgeGroupPreferences(prefs map[string][]RankedPreference, counselorCabin map[string]string, cabinAgeGroup map[string]string) map[string]map[string]bool {
	result := make(map[string]map[string]bool)
	for counselorID, prefList := range prefs {
		if len(prefList) == 0 {
			continue
		}

		cabinID, assigned := counselorCabin[counselorID]
		assignedAG := ""
		if assigned {
			assignedAG = cabinAgeGroup[cabinID]
		}

		// Only the top-ranked preference is considered unmet, consistent
		// with findUnmetAgeGroupPreferences in explain.go.
		topPref := prefList[0]
		if topPref.TargetID != assignedAG {
			if result[counselorID] == nil {
				result[counselorID] = make(map[string]bool)
			}
			result[counselorID][topPref.TargetID] = true
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func diffCocounselorPreferences(prefs map[string][]RankedPreference, counselorCabin map[string]string) map[string]map[string]bool {
	result := make(map[string]map[string]bool)
	for counselorID, prefList := range prefs {
		cabinID := counselorCabin[counselorID]
		for _, pref := range prefList {
			prefCabinID := counselorCabin[pref.TargetID]
			if cabinID == "" || prefCabinID == "" || prefCabinID != cabinID {
				if result[counselorID] == nil {
					result[counselorID] = make(map[string]bool)
				}
				result[counselorID][pref.TargetID] = true
			}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
