package preferences

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
)

var ErrActivityPreferenceNotFound = errors.New("activity preference not found")

type ActivityPreferenceService struct {
	queries *db.Queries
}

func NewActivityPreferenceService(queries *db.Queries) *ActivityPreferenceService {
	return &ActivityPreferenceService{queries: queries}
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

func (svc *ActivityPreferenceService) GetByID(ctx context.Context, campID, sessionID, counselorID, id string) (ActivityPreferenceResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return ActivityPreferenceResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return ActivityPreferenceResponse{}, err
	}

	counselorUUID, err := api.ParseUUID(counselorID)
	if err != nil {
		return ActivityPreferenceResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return ActivityPreferenceResponse{}, err
	}

	pref, err := svc.queries.GetCounselorActivityPreference(ctx, db.GetCounselorActivityPreferenceParams{
		ID:          uid,
		CampID:      campUUID,
		SessionID:   sessionUUID,
		CounselorID: counselorUUID,
	})
	if err != nil {
		return ActivityPreferenceResponse{}, fmt.Errorf("error getting activity preference %s: %w", id, err)
	}

	return toActivityPreferenceResponse(pref), nil
}

func (svc *ActivityPreferenceService) Create(ctx context.Context, campID, sessionID, counselorID string, req CreateActivityPreferenceRequest) (ActivityPreferenceResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return ActivityPreferenceResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return ActivityPreferenceResponse{}, err
	}

	counselorUUID, err := api.ParseUUID(counselorID)
	if err != nil {
		return ActivityPreferenceResponse{}, err
	}

	activityUUID, err := api.ParseUUID(req.ActivityID)
	if err != nil {
		return ActivityPreferenceResponse{}, err
	}

	pref, err := svc.queries.CreateCounselorActivityPreference(ctx, db.CreateCounselorActivityPreferenceParams{
		CampID:      campUUID,
		CounselorID: counselorUUID,
		SessionID:   sessionUUID,
		ActivityID:  activityUUID,
		Rank:        req.Rank,
	})
	if err != nil {
		return ActivityPreferenceResponse{}, fmt.Errorf("error creating activity preference: %w", err)
	}

	return toActivityPreferenceResponse(pref), nil
}

func (svc *ActivityPreferenceService) Update(ctx context.Context, campID, sessionID, counselorID, id string, req UpdateActivityPreferenceRequest) (ActivityPreferenceResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return ActivityPreferenceResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return ActivityPreferenceResponse{}, err
	}

	counselorUUID, err := api.ParseUUID(counselorID)
	if err != nil {
		return ActivityPreferenceResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return ActivityPreferenceResponse{}, err
	}

	activityUUID, err := api.ParseUUID(req.ActivityID)
	if err != nil {
		return ActivityPreferenceResponse{}, err
	}

	pref, err := svc.queries.UpdateCounselorActivityPreference(ctx, db.UpdateCounselorActivityPreferenceParams{
		ID:          uid,
		CampID:      campUUID,
		SessionID:   sessionUUID,
		CounselorID: counselorUUID,
		ActivityID:  activityUUID,
		Rank:        req.Rank,
	})
	if err != nil {
		return ActivityPreferenceResponse{}, fmt.Errorf("error updating activity preference %s: %w", id, err)
	}

	return toActivityPreferenceResponse(pref), nil
}

func (svc *ActivityPreferenceService) Delete(ctx context.Context, campID, sessionID, counselorID, id string) error {
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

	rows, err := svc.queries.DeleteCounselorActivityPreference(ctx, db.DeleteCounselorActivityPreferenceParams{
		ID:          uid,
		CampID:      campUUID,
		SessionID:   sessionUUID,
		CounselorID: counselorUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting activity preference %s: %w", id, err)
	}
	if rows == 0 {
		return ErrActivityPreferenceNotFound
	}

	return nil
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
