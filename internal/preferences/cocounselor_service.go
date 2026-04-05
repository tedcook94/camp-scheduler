package preferences

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
)

var ErrCocounselorPreferenceNotFound = errors.New("co-counselor preference not found")

type CocounselorService struct {
	queries *db.Queries
}

func NewCocounselorService(queries *db.Queries) *CocounselorService {
	return &CocounselorService{queries: queries}
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

func (svc *CocounselorService) GetByID(ctx context.Context, campID, sessionID, counselorID, id string) (CocounselorPreferenceResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return CocounselorPreferenceResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return CocounselorPreferenceResponse{}, err
	}

	counselorUUID, err := api.ParseUUID(counselorID)
	if err != nil {
		return CocounselorPreferenceResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return CocounselorPreferenceResponse{}, err
	}

	pref, err := svc.queries.GetCounselorCocounselorPreference(ctx, db.GetCounselorCocounselorPreferenceParams{
		ID:          uid,
		CampID:      campUUID,
		SessionID:   sessionUUID,
		CounselorID: counselorUUID,
	})
	if err != nil {
		return CocounselorPreferenceResponse{}, fmt.Errorf("error getting co-counselor preference %s: %w", id, err)
	}

	return toCocounselorPreferenceResponse(pref), nil
}

func (svc *CocounselorService) Create(ctx context.Context, campID, sessionID, counselorID string, req CreateCocounselorPreferenceRequest) (CocounselorPreferenceResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return CocounselorPreferenceResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return CocounselorPreferenceResponse{}, err
	}

	counselorUUID, err := api.ParseUUID(counselorID)
	if err != nil {
		return CocounselorPreferenceResponse{}, err
	}

	preferredUUID, err := api.ParseUUID(req.PreferredCounselorID)
	if err != nil {
		return CocounselorPreferenceResponse{}, err
	}

	pref, err := svc.queries.CreateCounselorCocounselorPreference(ctx, db.CreateCounselorCocounselorPreferenceParams{
		CampID:               campUUID,
		CounselorID:          counselorUUID,
		SessionID:            sessionUUID,
		PreferredCounselorID: preferredUUID,
		Rank:                 req.Rank,
	})
	if err != nil {
		return CocounselorPreferenceResponse{}, fmt.Errorf("error creating co-counselor preference: %w", err)
	}

	return toCocounselorPreferenceResponse(pref), nil
}

func (svc *CocounselorService) Update(ctx context.Context, campID, sessionID, counselorID, id string, req UpdateCocounselorPreferenceRequest) (CocounselorPreferenceResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return CocounselorPreferenceResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return CocounselorPreferenceResponse{}, err
	}

	counselorUUID, err := api.ParseUUID(counselorID)
	if err != nil {
		return CocounselorPreferenceResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return CocounselorPreferenceResponse{}, err
	}

	preferredUUID, err := api.ParseUUID(req.PreferredCounselorID)
	if err != nil {
		return CocounselorPreferenceResponse{}, err
	}

	pref, err := svc.queries.UpdateCounselorCocounselorPreference(ctx, db.UpdateCounselorCocounselorPreferenceParams{
		ID:                   uid,
		CampID:               campUUID,
		SessionID:            sessionUUID,
		CounselorID:          counselorUUID,
		PreferredCounselorID: preferredUUID,
		Rank:                 req.Rank,
	})
	if err != nil {
		return CocounselorPreferenceResponse{}, fmt.Errorf("error updating co-counselor preference %s: %w", id, err)
	}

	return toCocounselorPreferenceResponse(pref), nil
}

func (svc *CocounselorService) Delete(ctx context.Context, campID, sessionID, counselorID, id string) error {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return err
	}

	counselorUUID, err := api.ParseUUID(counselorID)
	if err != nil {
		return err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	rows, err := svc.queries.DeleteCounselorCocounselorPreference(ctx, db.DeleteCounselorCocounselorPreferenceParams{
		ID:          uid,
		CampID:      campUUID,
		SessionID:   sessionUUID,
		CounselorID: counselorUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting co-counselor preference %s: %w", id, err)
	}
	if rows == 0 {
		return ErrCocounselorPreferenceNotFound
	}

	return nil
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
