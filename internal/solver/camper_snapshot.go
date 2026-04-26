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

	camperSet := make(map[string]bool, len(campers))
	for _, c := range campers {
		camperSet[c.ID] = true
	}
	overrides, err := loadCamperCabinOverrides(ctx, queries, sessionUUID, campUUID, camperSet)
	if err != nil {
		return CamperCabinSnapshot{}, err
	}

	return CamperCabinSnapshot{
		SessionID:         sessionID,
		Cabins:            cabins,
		Campers:           campers,
		FriendPreferences: friendPrefs,
		Overrides:         overrides,
	}, nil
}

// loadCamperCabinOverrides loads admin-pinned camper->cabin assignments for
// the session and filters out any whose camper is no longer enrolled.
func loadCamperCabinOverrides(ctx context.Context, queries *db.Queries, sessionID, campID pgtype.UUID, enrolled map[string]bool) (map[string]string, error) {
	rows, err := queries.ListCamperCabinOverridesForSolver(ctx, db.ListCamperCabinOverridesForSolverParams{
		SessionID: sessionID,
		CampID:    campID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing camper cabin overrides: %w", err)
	}
	if len(rows) == 0 {
		return nil, nil
	}
	overrides := make(map[string]string, len(rows))
	for _, r := range rows {
		camperID := api.UUIDToString(r.CamperID)
		if !enrolled[camperID] {
			continue
		}
		overrides[camperID] = api.UUIDToString(r.SessionAgeGroupCabinID)
	}
	return overrides, nil
}

func loadCamperCabins(ctx context.Context, queries *db.Queries, sessionID, campID pgtype.UUID) ([]CamperCabin, error) {
	rows, err := queries.ListSessionCabinsWithCapacity(ctx, db.ListSessionCabinsWithCapacityParams{
		SessionID: sessionID,
		CampID:    campID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing session cabins for camper solver: %w", err)
	}

	cabins := make([]CamperCabin, 0, len(rows))
	for _, r := range rows {
		if r.CabinArchived || r.AgeGroupArchived {
			continue
		}
		cabins = append(cabins, CamperCabin{
			ID:                     api.UUIDToString(r.ID),
			Name:                   r.CabinName,
			AgeGroupID:             api.UUIDToString(r.AgeGroupID),
			AgeGroupName:           r.AgeGroupName,
			SessionAgeGroupCabinID: api.UUIDToString(r.SessionAgeGroupCabinID),
			Capacity:               int(r.GroupSize),
			Gender:                 r.Gender,
		})
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

	campers := make([]Camper, 0, len(rows))
	for _, r := range rows {
		// Skip enrollments whose underlying camper has been archived.
		if r.CamperArchived {
			continue
		}
		campers = append(campers, Camper{
			ID:         api.UUIDToString(r.CamperID),
			FirstName:  r.CamperFirstName,
			LastName:   r.CamperLastName,
			Name:       r.CamperName,
			AgeGroupID: api.UUIDToString(r.AgeGroupID),
			Gender:     r.Gender,
		})
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
