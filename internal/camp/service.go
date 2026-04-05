package camp

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
)

var ErrNotFound = errors.New("camp not found")

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

func (svc *Service) List(ctx context.Context) ([]CampResponse, error) {
	camps, err := svc.queries.ListCamps(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing camps: %w", err)
	}

	result := make([]CampResponse, len(camps))
	for i, c := range camps {
		result[i] = toCampResponse(c)
	}
	return result, nil
}

func (svc *Service) GetByID(ctx context.Context, id string) (CampResponse, error) {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return CampResponse{}, err
	}

	camp, err := svc.queries.GetCamp(ctx, uid)
	if err != nil {
		return CampResponse{}, fmt.Errorf("error getting camp %s: %w", id, err)
	}

	return toCampResponse(camp), nil
}

func (svc *Service) Create(ctx context.Context, req CreateCampRequest) (CampResponse, error) {
	camp, err := svc.queries.CreateCamp(ctx, db.CreateCampParams{
		CampName:     req.Name,
		CampLocation: api.ToPgText(req.Location),
	})
	if err != nil {
		return CampResponse{}, fmt.Errorf("error creating camp: %w", err)
	}

	return toCampResponse(camp), nil
}

func (svc *Service) Update(ctx context.Context, id string, req UpdateCampRequest) (CampResponse, error) {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return CampResponse{}, err
	}

	camp, err := svc.queries.UpdateCamp(ctx, db.UpdateCampParams{
		ID:           uid,
		CampName:     req.Name,
		CampLocation: api.ToPgText(req.Location),
		CampEnabled:  req.Enabled,
	})
	if err != nil {
		return CampResponse{}, fmt.Errorf("error updating camp %s: %w", id, err)
	}

	return toCampResponse(camp), nil
}

func (svc *Service) Delete(ctx context.Context, id string) error {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	rows, err := svc.queries.DeleteCamp(ctx, uid)
	if err != nil {
		return fmt.Errorf("error deleting camp %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func toCampResponse(c db.Camp) CampResponse {
	resp := CampResponse{
		ID:      api.UUIDToString(c.ID),
		Name:    c.CampName,
		Enabled: c.CampEnabled,
	}
	if c.CampLocation.Valid {
		resp.Location = &c.CampLocation.String
	}
	return resp
}
