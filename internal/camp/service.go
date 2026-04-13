package camp

import (
	"context"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
)

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
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
