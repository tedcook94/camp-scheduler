package solver

import (
	"context"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
)

func BuildCamperCabinSnapshot(ctx context.Context, queries *db.Queries, campID, sessionID string) (CamperCabinSnapshot, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return CamperCabinSnapshot{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return CamperCabinSnapshot{}, err
	}

	_, err = queries.GetSession(ctx, db.GetSessionParams{
		ID:     sessionUUID,
		CampID: campUUID,
	})
	if err != nil {
		return CamperCabinSnapshot{}, fmt.Errorf("error getting session: %w", err)
	}

	cabins, err := loadCamperCabins(ctx, queries, sessionUUID, campUUID)
	if err != nil {
		return CamperCabinSnapshot{}, err
	}

	campers, err := loadCampers(ctx, queries, sessionUUID, campUUID)
	if err != nil {
		return CamperCabinSnapshot{}, err
	}

	friendPrefs, err := loadFriendPreferences(ctx, queries, sessionUUID, campUUID)
	if err != nil {
		return CamperCabinSnapshot{}, err
	}

	return CamperCabinSnapshot{
		SessionID:         sessionID,
		Cabins:            cabins,
		Campers:           campers,
		FriendPreferences: friendPrefs,
	}, nil
}

func loadCamperCabins(ctx context.Context, queries *db.Queries, sessionID, campID pgtype.UUID) ([]CamperCabin, error) {
	rows, err := queries.ListSessionCabinsWithCapacity(ctx, db.ListSessionCabinsWithCapacityParams{
		SessionID: sessionID,
		CampID:    campID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing session cabins for camper solver: %w", err)
	}

	cabins := make([]CamperCabin, len(rows))
	for i, r := range rows {
		cabins[i] = CamperCabin{
			ID:                     api.UUIDToString(r.ID),
			Name:                   r.CabinName,
			AgeGroupID:             api.UUIDToString(r.AgeGroupID),
			AgeGroupName:           r.AgeGroupName,
			SessionAgeGroupCabinID: api.UUIDToString(r.SessionAgeGroupCabinID),
			Capacity:               int(r.GroupSize),
			Gender:                 r.Gender,
		}
	}
	return cabins, nil
}

// loadCampers loads all campers enrolled in the session via their
// session_age_group enrollments, resolving each camper's age group.
func loadCampers(ctx context.Context, queries *db.Queries, sessionID, campID pgtype.UUID) ([]Camper, error) {
	rows, err := queries.ListSessionEnrollments(ctx, db.ListSessionEnrollmentsParams{
		SessionID: sessionID,
		CampID:    campID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing session enrollments for camper solver: %w", err)
	}

	campers := make([]Camper, len(rows))
	for i, r := range rows {
		campers[i] = Camper{
			ID:         api.UUIDToString(r.CamperID),
			Name:       r.CamperName,
			AgeGroupID: api.UUIDToString(r.AgeGroupID),
			Gender:     r.Gender,
		}
	}
	return campers, nil
}

func loadFriendPreferences(ctx context.Context, queries *db.Queries, sessionID, campID pgtype.UUID) (map[string][]RankedPreference, error) {
	rows, err := queries.ListSessionCamperFriendPreferences(ctx, db.ListSessionCamperFriendPreferencesParams{
		SessionID: sessionID,
		CampID:    campID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing session camper friend preferences: %w", err)
	}

	prefs := make(map[string][]RankedPreference)
	for _, r := range rows {
		cID := api.UUIDToString(r.CamperID)
		prefs[cID] = append(prefs[cID], RankedPreference{
			TargetID: api.UUIDToString(r.PreferredCamperID),
			Rank:     int(r.Rank),
		})
	}
	return prefs, nil
}
