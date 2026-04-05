package preferences

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
)

var ErrCamperFriendPreferenceNotFound = errors.New("camper friend preference not found")

type CamperFriendService struct {
	queries *db.Queries
}

func NewCamperFriendService(queries *db.Queries) *CamperFriendService {
	return &CamperFriendService{queries: queries}
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

func (svc *CamperFriendService) GetByID(ctx context.Context, campID, sessionID, camperID, id string) (CamperFriendPreferenceResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return CamperFriendPreferenceResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return CamperFriendPreferenceResponse{}, err
	}

	camperUUID, err := api.ParseUUID(camperID)
	if err != nil {
		return CamperFriendPreferenceResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return CamperFriendPreferenceResponse{}, err
	}

	pref, err := svc.queries.GetCamperFriendPreference(ctx, db.GetCamperFriendPreferenceParams{
		ID:        uid,
		CampID:    campUUID,
		SessionID: sessionUUID,
		CamperID:  camperUUID,
	})
	if err != nil {
		return CamperFriendPreferenceResponse{}, fmt.Errorf("error getting camper friend preference %s: %w", id, err)
	}

	return toCamperFriendPreferenceResponse(pref), nil
}

func (svc *CamperFriendService) Create(ctx context.Context, campID, sessionID, camperID string, req CreateCamperFriendPreferenceRequest) (CamperFriendPreferenceResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return CamperFriendPreferenceResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return CamperFriendPreferenceResponse{}, err
	}

	camperUUID, err := api.ParseUUID(camperID)
	if err != nil {
		return CamperFriendPreferenceResponse{}, err
	}

	preferredUUID, err := api.ParseUUID(req.PreferredCamperID)
	if err != nil {
		return CamperFriendPreferenceResponse{}, err
	}

	pref, err := svc.queries.CreateCamperFriendPreference(ctx, db.CreateCamperFriendPreferenceParams{
		CampID:            campUUID,
		CamperID:          camperUUID,
		SessionID:         sessionUUID,
		PreferredCamperID: preferredUUID,
		Rank:              req.Rank,
	})
	if err != nil {
		return CamperFriendPreferenceResponse{}, fmt.Errorf("error creating camper friend preference: %w", err)
	}

	return toCamperFriendPreferenceResponse(pref), nil
}

func (svc *CamperFriendService) Update(ctx context.Context, campID, sessionID, camperID, id string, req UpdateCamperFriendPreferenceRequest) (CamperFriendPreferenceResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return CamperFriendPreferenceResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return CamperFriendPreferenceResponse{}, err
	}

	camperUUID, err := api.ParseUUID(camperID)
	if err != nil {
		return CamperFriendPreferenceResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return CamperFriendPreferenceResponse{}, err
	}

	preferredUUID, err := api.ParseUUID(req.PreferredCamperID)
	if err != nil {
		return CamperFriendPreferenceResponse{}, err
	}

	pref, err := svc.queries.UpdateCamperFriendPreference(ctx, db.UpdateCamperFriendPreferenceParams{
		ID:                uid,
		CampID:            campUUID,
		SessionID:         sessionUUID,
		CamperID:          camperUUID,
		PreferredCamperID: preferredUUID,
		Rank:              req.Rank,
	})
	if err != nil {
		return CamperFriendPreferenceResponse{}, fmt.Errorf("error updating camper friend preference %s: %w", id, err)
	}

	return toCamperFriendPreferenceResponse(pref), nil
}

func (svc *CamperFriendService) Delete(ctx context.Context, campID, sessionID, camperID, id string) error {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return err
	}

	camperUUID, err := api.ParseUUID(camperID)
	if err != nil {
		return err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	rows, err := svc.queries.DeleteCamperFriendPreference(ctx, db.DeleteCamperFriendPreferenceParams{
		ID:        uid,
		CampID:    campUUID,
		SessionID: sessionUUID,
		CamperID:  camperUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting camper friend preference %s: %w", id, err)
	}
	if rows == 0 {
		return ErrCamperFriendPreferenceNotFound
	}

	return nil
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
