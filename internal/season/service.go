package season

import (
	"context"
	"errors"
	"fmt"
	"time"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
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

	startDate, err := parseDate(req.StartDate)
	if err != nil {
		return SeasonResponse{}, err
	}
	endDate, err := parseDate(req.EndDate)
	if err != nil {
		return SeasonResponse{}, err
	}

	if startDate.Time.After(endDate.Time) {
		return SeasonResponse{}, fmt.Errorf("error start_date must be on or before end_date: %w", api.ErrBadInput)
	}

	season, err := svc.queries.CreateSeason(ctx, db.CreateSeasonParams{
		CampID:     uid,
		SeasonName: req.Name,
		StartDate:  startDate,
		EndDate:    endDate,
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

	startDate, err := parseDate(req.StartDate)
	if err != nil {
		return SeasonResponse{}, err
	}
	endDate, err := parseDate(req.EndDate)
	if err != nil {
		return SeasonResponse{}, err
	}

	if startDate.Time.After(endDate.Time) {
		return SeasonResponse{}, fmt.Errorf("error start_date must be on or before end_date: %w", api.ErrBadInput)
	}

	season, err := svc.queries.UpdateSeason(ctx, db.UpdateSeasonParams{
		ID:         uid,
		CampID:     campUUID,
		SeasonName: req.Name,
		StartDate:  startDate,
		EndDate:    endDate,
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
		ID:        api.UUIDToString(s.ID),
		CampID:    api.UUIDToString(s.CampID),
		Name:      s.SeasonName,
		StartDate: s.StartDate.Time.Format("2006-01-02"),
		EndDate:   s.EndDate.Time.Format("2006-01-02"),
	}
}

func parseDate(s string) (pgtype.Date, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return pgtype.Date{}, fmt.Errorf("error invalid date %q (expected YYYY-MM-DD): %w", s, api.ErrBadInput)
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}
