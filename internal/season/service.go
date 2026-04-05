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

func (svc *Service) List(ctx context.Context, campID string) ([]SeasonResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	seasons, err := svc.queries.ListSeasons(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("error listing seasons: %w", err)
	}

	result := make([]SeasonResponse, len(seasons))
	for i, sn := range seasons {
		result[i] = toSeasonResponse(sn)
	}
	return result, nil
}

func (svc *Service) GetByID(ctx context.Context, campID, id string) (SeasonResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SeasonResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return SeasonResponse{}, err
	}

	season, err := svc.queries.GetSeason(ctx, db.GetSeasonParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return SeasonResponse{}, fmt.Errorf("error getting season %s: %w", id, err)
	}

	return toSeasonResponse(season), nil
}

func (svc *Service) Create(ctx context.Context, campID string, req CreateSeasonRequest) (SeasonResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return SeasonResponse{}, err
	}

	season, err := svc.queries.CreateSeason(ctx, db.CreateSeasonParams{
		CampID:     uid,
		SeasonName: req.Name,
	})
	if err != nil {
		return SeasonResponse{}, fmt.Errorf("error creating season: %w", err)
	}

	return toSeasonResponse(season), nil
}

func (svc *Service) Update(ctx context.Context, campID, id string, req UpdateSeasonRequest) (SeasonResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SeasonResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return SeasonResponse{}, err
	}

	season, err := svc.queries.UpdateSeason(ctx, db.UpdateSeasonParams{
		ID:         uid,
		CampID:     campUUID,
		SeasonName: req.Name,
	})
	if err != nil {
		return SeasonResponse{}, fmt.Errorf("error updating season %s: %w", id, err)
	}

	return toSeasonResponse(season), nil
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

	rows, err := svc.queries.DeleteSeason(ctx, db.DeleteSeasonParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting season %s: %w", id, err)
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
