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

func (s *Service) List(ctx context.Context) ([]CampResponse, error) {
	camps, err := s.queries.ListCamps(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing camps: %w", err)
	}

	result := make([]CampResponse, len(camps))
	for i, c := range camps {
		result[i] = toCampResponse(c)
	}
	return result, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (CampResponse, error) {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return CampResponse{}, err
	}

	camp, err := s.queries.GetCamp(ctx, uid)
	if err != nil {
		return CampResponse{}, fmt.Errorf("getting camp %s: %w", id, err)
	}

	return toCampResponse(camp), nil
}

func (s *Service) Create(ctx context.Context, req CreateCampRequest) (CampResponse, error) {
	camp, err := s.queries.CreateCamp(ctx, db.CreateCampParams{
		CampName:     req.Name,
		CampLocation: api.ToPgText(req.Location),
	})
	if err != nil {
		return CampResponse{}, fmt.Errorf("creating camp: %w", err)
	}

	return toCampResponse(camp), nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateCampRequest) (CampResponse, error) {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return CampResponse{}, err
	}

	camp, err := s.queries.UpdateCamp(ctx, db.UpdateCampParams{
		ID:           uid,
		CampName:     req.Name,
		CampLocation: api.ToPgText(req.Location),
		CampEnabled:  req.Enabled,
	})
	if err != nil {
		return CampResponse{}, fmt.Errorf("updating camp %s: %w", id, err)
	}

	return toCampResponse(camp), nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	rows, err := s.queries.DeleteCamp(ctx, uid)
	if err != nil {
		return fmt.Errorf("deleting camp %s: %w", id, err)
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
