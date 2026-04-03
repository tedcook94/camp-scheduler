package camp

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
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
	uid, err := parseUUID(id)
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
		CampLocation: toPgText(req.Location),
	})
	if err != nil {
		return CampResponse{}, fmt.Errorf("creating camp: %w", err)
	}

	return toCampResponse(camp), nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateCampRequest) (CampResponse, error) {
	uid, err := parseUUID(id)
	if err != nil {
		return CampResponse{}, err
	}

	camp, err := s.queries.UpdateCamp(ctx, db.UpdateCampParams{
		ID:           uid,
		CampName:     req.Name,
		CampLocation: toPgText(req.Location),
		CampEnabled:  req.Enabled,
	})
	if err != nil {
		return CampResponse{}, fmt.Errorf("updating camp %s: %w", id, err)
	}

	return toCampResponse(camp), nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	uid, err := parseUUID(id)
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
		ID:      uuidToString(c.ID),
		Name:    c.CampName,
		Enabled: c.CampEnabled,
	}
	if c.CampLocation.Valid {
		resp.Location = &c.CampLocation.String
	}
	return resp
}

func parseUUID(s string) (pgtype.UUID, error) {
	var uid pgtype.UUID
	if err := uid.Scan(s); err != nil {
		return uid, fmt.Errorf("invalid uuid %q: %w", s, err)
	}
	return uid, nil
}

func uuidToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", u.Bytes[0:4], u.Bytes[4:6], u.Bytes[6:8], u.Bytes[8:10], u.Bytes[10:16])
}

func toPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}
