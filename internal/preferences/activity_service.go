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

type ActivityPreferenceService struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	marker  *staleness.Marker
}

func NewActivityPreferenceService(queries *db.Queries, pool *pgxpool.Pool, marker *staleness.Marker) *ActivityPreferenceService {
	return &ActivityPreferenceService{queries: queries, pool: pool, marker: marker}
}

func (svc *ActivityPreferenceService) List(ctx context.Context, campID, sessionID, counselorID string) ([]ActivityPreferenceResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return nil, err
	}

	counselorUUID, err := api.ParseUUID(counselorID)
	if err != nil {
		return nil, err
	}

	prefs, err := svc.queries.ListCounselorActivityPreferences(ctx, db.ListCounselorActivityPreferencesParams{
		CounselorID: counselorUUID,
		SessionID:   sessionUUID,
		CampID:      campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing activity preferences: %w", err)
	}

	result := make([]ActivityPreferenceResponse, len(prefs))
	for i, p := range prefs {
		result[i] = toActivityPreferenceResponse(p)
	}
	return result, nil
}

func (svc *ActivityPreferenceService) ReplaceAll(ctx context.Context, campID, sessionID, counselorID string, items []ActivityPreferenceItem) ([]ActivityPreferenceResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return nil, err
	}

	counselorUUID, err := api.ParseUUID(counselorID)
	if err != nil {
		return nil, err
	}

	if err := validateRanks(len(items), func(i int) int32 { return items[i].Rank }); err != nil {
		return nil, err
	}

	seen := make(map[string]bool, len(items))
	canonicalIDs := make([]string, len(items))
	for i, item := range items {
		parsed, err := api.ParseUUID(item.ActivityID)
		if err != nil {
			return nil, err
		}
		canonical := api.UUIDToString(parsed)
		if seen[canonical] {
			return nil, api.BadInput(fmt.Sprintf("duplicate activity_id: %s", item.ActivityID))
		}
		seen[canonical] = true
		canonicalIDs[i] = canonical
	}

	if len(items) > 0 {
		sessionActivities, err := svc.queries.ListSessionActivities(ctx, db.ListSessionActivitiesParams{
			SessionID: sessionUUID,
			CampID:    campUUID,
		})
		if err != nil {
			return nil, fmt.Errorf("error loading session activities for validation: %w", err)
		}
		allowed := make(map[string]bool, len(sessionActivities))
		for _, sa := range sessionActivities {
			allowed[api.UUIDToString(sa.ActivityID)] = true
		}
		for i, canonical := range canonicalIDs {
			if !allowed[canonical] {
				return nil, api.BadInput(fmt.Sprintf("activity_id %s is not configured for this session", items[i].ActivityID))
			}
		}
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := svc.queries.WithTx(tx)

	err = qtx.DeleteAllCounselorActivityPreferences(ctx, db.DeleteAllCounselorActivityPreferencesParams{
		CounselorID: counselorUUID,
		SessionID:   sessionUUID,
		CampID:      campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error deleting activity preferences: %w", err)
	}

	result := make([]ActivityPreferenceResponse, len(items))
	for i, item := range items {
		activityUUID, err := api.ParseUUID(item.ActivityID)
		if err != nil {
			return nil, err
		}

		pref, err := qtx.CreateCounselorActivityPreference(ctx, db.CreateCounselorActivityPreferenceParams{
			CampID:      campUUID,
			CounselorID: counselorUUID,
			SessionID:   sessionUUID,
			ActivityID:  activityUUID,
			Rank:        item.Rank,
		})
		if err != nil {
			return nil, fmt.Errorf("error creating activity preference: %w", err)
		}

		result[i] = toActivityPreferenceResponse(pref)
	}

	if err := svc.marker.MarkSessions(ctx, qtx, campUUID, []pgtype.UUID{sessionUUID}, []staleness.RunType{staleness.RunTypeActivity}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	return result, nil
}

func toActivityPreferenceResponse(p db.CounselorActivityPreference) ActivityPreferenceResponse {
	return ActivityPreferenceResponse{
		ID:          api.UUIDToString(p.ID),
		CampID:      api.UUIDToString(p.CampID),
		CounselorID: api.UUIDToString(p.CounselorID),
		SessionID:   api.UUIDToString(p.SessionID),
		ActivityID:  api.UUIDToString(p.ActivityID),
		Rank:        p.Rank,
	}
}
