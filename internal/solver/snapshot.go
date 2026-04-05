package solver

import (
	"context"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"

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

	return SessionSnapshot{
		SessionID:                   sessionID,
		Cabins:                      cabins,
		Counselors:                  counselors,
		AgeGroupPreferences:         ageGroupPrefs,
		CocounselorPreferences:      cocounselorPrefs,
		CounselorPreviousPlacements: placements,
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
		required := 1
		if r.RequiredCounselors.Valid {
			required = int(r.RequiredCounselors.Int32)
		}
		cabins[i] = Cabin{
			ID:                 api.UUIDToString(r.ID),
			Name:               r.CabinName,
			AgeGroupID:         api.UUIDToString(r.AgeGroupID),
			RequiredCounselors: required,
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
