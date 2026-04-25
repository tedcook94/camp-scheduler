package preferences

import (
	"context"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
	"camp-scheduler/internal/staleness"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CamperFriendService struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	marker  *staleness.Marker
}

func NewCamperFriendService(queries *db.Queries, pool *pgxpool.Pool, marker *staleness.Marker) *CamperFriendService {
	return &CamperFriendService{queries: queries, pool: pool, marker: marker}
}

func (svc *CamperFriendService) List(ctx context.Context, campID, sessionID, camperID string) ([]CamperFriendPreferenceResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return nil, err
	}

	camperUUID, err := api.ParseUUID(camperID)
	if err != nil {
		return nil, err
	}

	prefs, err := svc.queries.ListCamperFriendPreferences(ctx, db.ListCamperFriendPreferencesParams{
		CamperID:  camperUUID,
		SessionID: sessionUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing camper friend preferences: %w", err)
	}

	result := make([]CamperFriendPreferenceResponse, len(prefs))
	for i, p := range prefs {
		result[i] = toCamperFriendPreferenceResponse(p)
	}
	return result, nil
}

func (svc *CamperFriendService) ReplaceAll(ctx context.Context, campID, sessionID, camperID string, items []CamperFriendPreferenceItem) ([]CamperFriendPreferenceResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return nil, err
	}

	camperUUID, err := api.ParseUUID(camperID)
	if err != nil {
		return nil, err
	}

	if err := validateRanks(len(items), func(i int) int32 { return items[i].Rank }); err != nil {
		return nil, err
	}

	seen := make(map[string]bool, len(items))
	canonicalIDs := make([]string, len(items))
	canonicalCamperID := api.UUIDToString(camperUUID)
	for i, item := range items {
		parsed, err := api.ParseUUID(item.PreferredCamperID)
		if err != nil {
			return nil, err
		}
		canonical := api.UUIDToString(parsed)
		if canonical == canonicalCamperID {
			return nil, api.BadInput("a camper cannot prefer themselves")
		}
		if seen[canonical] {
			return nil, api.BadInput(fmt.Sprintf("duplicate preferred_camper_id: %s", item.PreferredCamperID))
		}
		seen[canonical] = true
		canonicalIDs[i] = canonical
	}

	if len(items) > 0 {
		enrollments, err := svc.queries.ListSessionEnrollments(ctx, db.ListSessionEnrollmentsParams{
			SessionID: sessionUUID,
			CampID:    campUUID,
		})
		if err != nil {
			return nil, fmt.Errorf("error loading session enrollments for validation: %w", err)
		}
		allowed := make(map[string]bool, len(enrollments))
		for _, e := range enrollments {
			allowed[api.UUIDToString(e.CamperID)] = true
		}
		for i, canonical := range canonicalIDs {
			if !allowed[canonical] {
				return nil, api.BadInput(fmt.Sprintf("preferred_camper_id %s is not enrolled in this session", items[i].PreferredCamperID))
			}
		}
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := svc.queries.WithTx(tx)

	err = qtx.DeleteAllCamperFriendPreferences(ctx, db.DeleteAllCamperFriendPreferencesParams{
		CamperID:  camperUUID,
		SessionID: sessionUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error deleting camper friend preferences: %w", err)
	}

	result := make([]CamperFriendPreferenceResponse, len(items))
	for i, item := range items {
		preferredUUID, err := api.ParseUUID(item.PreferredCamperID)
		if err != nil {
			return nil, err
		}

		pref, err := qtx.CreateCamperFriendPreference(ctx, db.CreateCamperFriendPreferenceParams{
			CampID:            campUUID,
			CamperID:          camperUUID,
			SessionID:         sessionUUID,
			PreferredCamperID: preferredUUID,
			Rank:              item.Rank,
		})
		if err != nil {
			return nil, fmt.Errorf("error creating camper friend preference: %w", err)
		}

		result[i] = toCamperFriendPreferenceResponse(pref)
	}

	if err := svc.marker.MarkSessions(ctx, qtx, campUUID, []pgtype.UUID{sessionUUID}, []staleness.RunType{staleness.RunTypeCabin}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	return result, nil
}

func toCamperFriendPreferenceResponse(p db.CamperFriendPreference) CamperFriendPreferenceResponse {
	return CamperFriendPreferenceResponse{
		ID:                api.UUIDToString(p.ID),
		CampID:            api.UUIDToString(p.CampID),
		CamperID:          api.UUIDToString(p.CamperID),
		SessionID:         api.UUIDToString(p.SessionID),
		PreferredCamperID: api.UUIDToString(p.PreferredCamperID),
		Rank:              p.Rank,
	}
}
