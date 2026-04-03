package season

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
)

var ErrNotFound = errors.New("season not found")

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

func (s *Service) List(ctx context.Context, campID string) ([]SeasonResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	seasons, err := s.queries.ListSeasons(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("listing seasons: %w", err)
	}

	result := make([]SeasonResponse, len(seasons))
	for i, sn := range seasons {
		result[i] = toSeasonResponse(sn)
	}
	return result, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (SeasonResponse, error) {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return SeasonResponse{}, err
	}

	season, err := s.queries.GetSeason(ctx, uid)
	if err != nil {
		return SeasonResponse{}, fmt.Errorf("getting season %s: %w", id, err)
	}

	return toSeasonResponse(season), nil
}

func (s *Service) Create(ctx context.Context, campID string, req CreateSeasonRequest) (SeasonResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return SeasonResponse{}, err
	}

	season, err := s.queries.CreateSeason(ctx, db.CreateSeasonParams{
		CampID:     uid,
		SeasonName: req.Name,
	})
	if err != nil {
		return SeasonResponse{}, fmt.Errorf("creating season: %w", err)
	}

	return toSeasonResponse(season), nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateSeasonRequest) (SeasonResponse, error) {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return SeasonResponse{}, err
	}

	season, err := s.queries.UpdateSeason(ctx, db.UpdateSeasonParams{
		ID:         uid,
		SeasonName: req.Name,
	})
	if err != nil {
		return SeasonResponse{}, fmt.Errorf("updating season %s: %w", id, err)
	}

	return toSeasonResponse(season), nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	rows, err := s.queries.DeleteSeason(ctx, uid)
	if err != nil {
		return fmt.Errorf("deleting season %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func toSeasonResponse(s db.Season) SeasonResponse {
	return SeasonResponse{
		ID:     api.UUIDToString(s.ID),
		CampID: api.UUIDToString(s.CampID),
		Name:   s.SeasonName,
	}
}
