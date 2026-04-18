package preferences

import (
	"context"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CocounselorService struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewCocounselorService(queries *db.Queries, pool *pgxpool.Pool) *CocounselorService {
	return &CocounselorService{queries: queries, pool: pool}
}

func (svc *CocounselorService) List(ctx context.Context, campID, sessionID, counselorID string) ([]CocounselorPreferenceResponse, error) {
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

	prefs, err := svc.queries.ListCounselorCocounselorPreferences(ctx, db.ListCounselorCocounselorPreferencesParams{
		CounselorID: counselorUUID,
		SessionID:   sessionUUID,
		CampID:      campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing co-counselor preferences: %w", err)
	}

	result := make([]CocounselorPreferenceResponse, len(prefs))
	for i, p := range prefs {
		result[i] = toCocounselorPreferenceResponse(p)
	}
	return result, nil
}

func (svc *CocounselorService) ReplaceAll(ctx context.Context, campID, sessionID, counselorID string, items []CocounselorPreferenceItem) ([]CocounselorPreferenceResponse, error) {
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
	for _, item := range items {
		if item.PreferredCounselorID == counselorID {
			return nil, api.BadInput("a counselor cannot prefer themselves")
		}
		if seen[item.PreferredCounselorID] {
			return nil, api.BadInput(fmt.Sprintf("duplicate preferred_counselor_id: %s", item.PreferredCounselorID))
		}
		seen[item.PreferredCounselorID] = true
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := svc.queries.WithTx(tx)

	err = qtx.DeleteAllCounselorCocounselorPreferences(ctx, db.DeleteAllCounselorCocounselorPreferencesParams{
		CounselorID: counselorUUID,
		SessionID:   sessionUUID,
		CampID:      campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error deleting co-counselor preferences: %w", err)
	}

	result := make([]CocounselorPreferenceResponse, len(items))
	for i, item := range items {
		preferredUUID, err := api.ParseUUID(item.PreferredCounselorID)
		if err != nil {
			return nil, err
		}

		pref, err := qtx.CreateCounselorCocounselorPreference(ctx, db.CreateCounselorCocounselorPreferenceParams{
			CampID:               campUUID,
			CounselorID:          counselorUUID,
			SessionID:            sessionUUID,
			PreferredCounselorID: preferredUUID,
			Rank:                 item.Rank,
		})
		if err != nil {
			return nil, fmt.Errorf("error creating co-counselor preference: %w", err)
		}

		result[i] = toCocounselorPreferenceResponse(pref)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	return result, nil
}

func toCocounselorPreferenceResponse(p db.CounselorCocounselorPreference) CocounselorPreferenceResponse {
	return CocounselorPreferenceResponse{
		ID:                   api.UUIDToString(p.ID),
		CampID:               api.UUIDToString(p.CampID),
		CounselorID:          api.UUIDToString(p.CounselorID),
		SessionID:            api.UUIDToString(p.SessionID),
		PreferredCounselorID: api.UUIDToString(p.PreferredCounselorID),
		Rank:                 p.Rank,
	}
}
