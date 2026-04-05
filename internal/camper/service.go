package camper

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
)

var ErrNotFound = errors.New("camper not found")

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

func (svc *Service) List(ctx context.Context, campID string) ([]CamperResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	campers, err := svc.queries.ListCampers(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("error listing campers: %w", err)
	}

	result := make([]CamperResponse, len(campers))
	for i, c := range campers {
		result[i] = toCamperResponse(c)
	}
	return result, nil
}

func (svc *Service) GetByID(ctx context.Context, campID, id string) (CamperResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return CamperResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return CamperResponse{}, err
	}

	camper, err := svc.queries.GetCamper(ctx, db.GetCamperParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return CamperResponse{}, fmt.Errorf("error getting camper %s: %w", id, err)
	}

	return toCamperResponse(camper), nil
}

func (svc *Service) Create(ctx context.Context, campID string, req CreateCamperRequest) (CamperResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return CamperResponse{}, err
	}

	camper, err := svc.queries.CreateCamper(ctx, db.CreateCamperParams{
		CampID:     uid,
		CamperName: req.Name,
	})
	if err != nil {
		return CamperResponse{}, fmt.Errorf("error creating camper: %w", err)
	}

	return toCamperResponse(camper), nil
}

func (svc *Service) Update(ctx context.Context, campID, id string, req UpdateCamperRequest) (CamperResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return CamperResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return CamperResponse{}, err
	}

	camper, err := svc.queries.UpdateCamper(ctx, db.UpdateCamperParams{
		ID:         uid,
		CampID:     campUUID,
		CamperName: req.Name,
	})
	if err != nil {
		return CamperResponse{}, fmt.Errorf("error updating camper %s: %w", id, err)
	}

	return toCamperResponse(camper), nil
}

func (svc *Service) Delete(ctx context.Context, campID, id string) error {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	rows, err := svc.queries.DeleteCamper(ctx, db.DeleteCamperParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting camper %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func toCamperResponse(c db.Camper) CamperResponse {
	return CamperResponse{
		ID:     api.UUIDToString(c.ID),
		CampID: api.UUIDToString(c.CampID),
		Name:   c.CamperName,
	}
}
