package preferences

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
)

var ErrAgeGroupPreferenceNotFound = errors.New("age group preference not found")

type AgeGroupService struct {
	queries *db.Queries
}

func NewAgeGroupService(queries *db.Queries) *AgeGroupService {
	return &AgeGroupService{queries: queries}
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

func (svc *AgeGroupService) GetByID(ctx context.Context, campID, id string) (AgeGroupPreferenceResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return AgeGroupPreferenceResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return AgeGroupPreferenceResponse{}, err
	}

	pref, err := svc.queries.GetCounselorAgeGroupPreference(ctx, db.GetCounselorAgeGroupPreferenceParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return AgeGroupPreferenceResponse{}, fmt.Errorf("error getting age group preference %s: %w", id, err)
	}

	return toAgeGroupPreferenceResponse(pref), nil
}

func (svc *AgeGroupService) Create(ctx context.Context, campID, sessionID, counselorID string, req CreateAgeGroupPreferenceRequest) (AgeGroupPreferenceResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return AgeGroupPreferenceResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return AgeGroupPreferenceResponse{}, err
	}

	counselorUUID, err := api.ParseUUID(counselorID)
	if err != nil {
		return AgeGroupPreferenceResponse{}, err
	}

	ageGroupUUID, err := api.ParseUUID(req.AgeGroupID)
	if err != nil {
		return AgeGroupPreferenceResponse{}, err
	}

	pref, err := svc.queries.CreateCounselorAgeGroupPreference(ctx, db.CreateCounselorAgeGroupPreferenceParams{
		CampID:      campUUID,
		CounselorID: counselorUUID,
		SessionID:   sessionUUID,
		AgeGroupID:  ageGroupUUID,
		Rank:        req.Rank,
	})
	if err != nil {
		return AgeGroupPreferenceResponse{}, fmt.Errorf("error creating age group preference: %w", err)
	}

	return toAgeGroupPreferenceResponse(pref), nil
}

func (svc *AgeGroupService) Update(ctx context.Context, campID, id string, req UpdateAgeGroupPreferenceRequest) (AgeGroupPreferenceResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return AgeGroupPreferenceResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return AgeGroupPreferenceResponse{}, err
	}

	ageGroupUUID, err := api.ParseUUID(req.AgeGroupID)
	if err != nil {
		return AgeGroupPreferenceResponse{}, err
	}

	pref, err := svc.queries.UpdateCounselorAgeGroupPreference(ctx, db.UpdateCounselorAgeGroupPreferenceParams{
		ID:         uid,
		CampID:     campUUID,
		AgeGroupID: ageGroupUUID,
		Rank:       req.Rank,
	})
	if err != nil {
		return AgeGroupPreferenceResponse{}, fmt.Errorf("error updating age group preference %s: %w", id, err)
	}

	return toAgeGroupPreferenceResponse(pref), nil
}

func (svc *AgeGroupService) Delete(ctx context.Context, campID, id string) error {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	rows, err := svc.queries.DeleteCounselorAgeGroupPreference(ctx, db.DeleteCounselorAgeGroupPreferenceParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting age group preference %s: %w", id, err)
	}
	if rows == 0 {
		return ErrAgeGroupPreferenceNotFound
	}

	return nil
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
