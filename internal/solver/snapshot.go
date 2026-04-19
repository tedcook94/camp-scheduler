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

	counselors, err := loadCounselors(ctx, queries, campUUID)
	if err != nil {
		return SessionSnapshot{}, err
	}

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

	return SessionSnapshot{
		SessionID:                   sessionID,
		Cabins:                      cabins,
		Counselors:                  counselors,
		AgeGroupPreferences:         ageGroupPrefs,
		CocounselorPreferences:      cocounselorPrefs,
		CounselorPreviousPlacements: placements,
		UnmetAgeGroupPreferences:    unmetAG,
		UnmetCocounselorPreferences: unmetCo,
	}, nil
}

func loadCabins(ctx context.Context, queries *db.Queries, sessionID, campID pgtype.UUID) ([]Cabin, error) {
	rows, err := queries.ListSessionCabins(ctx, db.ListSessionCabinsParams{
		SessionID: sessionID,
		CampID:    campID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing session cabins: %w", err)
	}

	cabins := make([]Cabin, len(rows))
	for i, r := range rows {
		cabins[i] = Cabin{
			ID:                 api.UUIDToString(r.ID),
			Name:               r.CabinName,
			AgeGroupID:         api.UUIDToString(r.AgeGroupID),
			RequiredCounselors: int(r.RequiredCounselors),
		}
	}
	return cabins, nil
}

func loadCounselors(ctx context.Context, queries *db.Queries, campID pgtype.UUID) ([]Counselor, error) {
	rows, err := queries.ListEnabledCounselors(ctx, campID)
	if err != nil {
		return nil, fmt.Errorf("error listing enabled counselors: %w", err)
	}

	counselors := make([]Counselor, len(rows))
	for i, r := range rows {
		counselors[i] = Counselor{
			ID:       api.UUIDToString(r.ID),
			Name:     r.CounselorName,
			IsJunior: r.JuniorCounselor,
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
