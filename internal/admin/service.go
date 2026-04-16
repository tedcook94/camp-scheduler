package admin

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/camp"
	"camp-scheduler/internal/db"
)

var ErrNotFound = errors.New("camp not found")

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

func (svc *Service) List(ctx context.Context) ([]camp.CampResponse, error) {
	camps, err := svc.queries.ListCamps(ctx)
	if err != nil {
		return nil, fmt.Errorf("error listing camps: %w", err)
	}

	result := make([]camp.CampResponse, len(camps))
	for i, c := range camps {
		result[i] = camp.ToCampResponse(c)
	}
	return result, nil
}

func (svc *Service) GetByID(ctx context.Context, id string) (camp.CampResponse, error) {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return camp.CampResponse{}, err
	}

	c, err := svc.queries.GetCamp(ctx, uid)
	if err != nil {
		return camp.CampResponse{}, fmt.Errorf("error getting camp %s: %w", id, err)
	}

	return camp.ToCampResponse(c), nil
}

func (svc *Service) Create(ctx context.Context, req CreateCampRequest) (camp.CampResponse, error) {
	c, err := svc.queries.CreateCamp(ctx, db.CreateCampParams{
		CampName:     req.Name,
		CampLocation: api.ToPgText(req.Location),
	})
	if err != nil {
		return camp.CampResponse{}, fmt.Errorf("error creating camp: %w", err)
	}

	return camp.ToCampResponse(c), nil
}

func (svc *Service) Update(ctx context.Context, id string, req UpdateCampRequest) (camp.CampResponse, error) {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return camp.CampResponse{}, err
	}

	c, err := svc.queries.UpdateCamp(ctx, db.UpdateCampParams{
		ID:           uid,
		CampName:     req.Name,
		CampLocation: api.ToPgText(req.Location),
		CampEnabled:  api.ToPgBool(req.Enabled),
	})
	if err != nil {
		return camp.CampResponse{}, fmt.Errorf("error updating camp %s: %w", id, err)
	}

	return camp.ToCampResponse(c), nil
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
