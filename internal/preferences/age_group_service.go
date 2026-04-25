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

type AgeGroupService struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	marker  *staleness.Marker
}

func NewAgeGroupService(queries *db.Queries, pool *pgxpool.Pool, marker *staleness.Marker) *AgeGroupService {
	return &AgeGroupService{queries: queries, pool: pool, marker: marker}
}

func (svc *AgeGroupService) List(ctx context.Context, campID, sessionID, counselorID string) ([]AgeGroupPreferenceResponse, error) {
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

	prefs, err := svc.queries.ListCounselorAgeGroupPreferences(ctx, db.ListCounselorAgeGroupPreferencesParams{
		CounselorID: counselorUUID,
		SessionID:   sessionUUID,
		CampID:      campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing age group preferences: %w", err)
	}

	result := make([]AgeGroupPreferenceResponse, len(prefs))
	for i, p := range prefs {
		result[i] = toAgeGroupPreferenceResponse(p)
	}
	return result, nil
}

func (svc *AgeGroupService) ReplaceAll(ctx context.Context, campID, sessionID, counselorID string, items []AgeGroupPreferenceItem) ([]AgeGroupPreferenceResponse, error) {
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
		parsed, err := api.ParseUUID(item.AgeGroupID)
		if err != nil {
			return nil, err
		}
		canonical := api.UUIDToString(parsed)
		if seen[canonical] {
			return nil, api.BadInput(fmt.Sprintf("duplicate age_group_id: %s", item.AgeGroupID))
		}
		seen[canonical] = true
		canonicalIDs[i] = canonical
	}

	if len(items) > 0 {
		sessionAgeGroups, err := svc.queries.ListSessionAgeGroups(ctx, db.ListSessionAgeGroupsParams{
			SessionID: sessionUUID,
			CampID:    campUUID,
		})
		if err != nil {
			return nil, fmt.Errorf("error loading session age groups for validation: %w", err)
		}
		allowed := make(map[string]bool, len(sessionAgeGroups))
		for _, sag := range sessionAgeGroups {
			allowed[api.UUIDToString(sag.AgeGroupID)] = true
		}
		for i, canonical := range canonicalIDs {
			if !allowed[canonical] {
				return nil, api.BadInput(fmt.Sprintf("age_group_id %s is not configured for this session", items[i].AgeGroupID))
			}
		}
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := svc.queries.WithTx(tx)

	err = qtx.DeleteAllCounselorAgeGroupPreferences(ctx, db.DeleteAllCounselorAgeGroupPreferencesParams{
		CounselorID: counselorUUID,
		SessionID:   sessionUUID,
		CampID:      campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error deleting age group preferences: %w", err)
	}

	result := make([]AgeGroupPreferenceResponse, len(items))
	for i, item := range items {
		ageGroupUUID, err := api.ParseUUID(item.AgeGroupID)
		if err != nil {
			return nil, err
		}

		pref, err := qtx.CreateCounselorAgeGroupPreference(ctx, db.CreateCounselorAgeGroupPreferenceParams{
			CampID:      campUUID,
			CounselorID: counselorUUID,
			SessionID:   sessionUUID,
			AgeGroupID:  ageGroupUUID,
			Rank:        item.Rank,
		})
		if err != nil {
			return nil, fmt.Errorf("error creating age group preference: %w", err)
		}

		result[i] = toAgeGroupPreferenceResponse(pref)
	}

	if err := svc.marker.MarkSessions(ctx, qtx, campUUID, []pgtype.UUID{sessionUUID}, []staleness.RunType{staleness.RunTypeCabin}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	return result, nil
}

func toAgeGroupPreferenceResponse(p db.CounselorAgeGroupPreference) AgeGroupPreferenceResponse {
	return AgeGroupPreferenceResponse{
		ID:          api.UUIDToString(p.ID),
		CampID:      api.UUIDToString(p.CampID),
		CounselorID: api.UUIDToString(p.CounselorID),
		SessionID:   api.UUIDToString(p.SessionID),
		AgeGroupID:  api.UUIDToString(p.AgeGroupID),
		Rank:        p.Rank,
	}
}
