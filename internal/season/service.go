package season

import (
	"context"
	"errors"
	"fmt"
	"time"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
	"camp-scheduler/internal/staleness"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("season not found")

type Service struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	marker  *staleness.Marker
}

func NewService(queries *db.Queries, pool *pgxpool.Pool, marker *staleness.Marker) *Service {
	return &Service{queries: queries, pool: pool, marker: marker}
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

func (svc *Service) ListArchived(ctx context.Context, campID string) ([]SeasonResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	seasons, err := svc.queries.ListArchivedSeasons(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("error listing archived seasons: %w", err)
	}

	result := make([]SeasonResponse, len(seasons))
	for i, sn := range seasons {
		result[i] = toSeasonResponse(sn)
	}
	return result, nil
}

func (svc *Service) Archive(ctx context.Context, campID, id string) error {
	return svc.setArchived(ctx, campID, id, true)
}

func (svc *Service) Unarchive(ctx context.Context, campID, id string) error {
	return svc.setArchived(ctx, campID, id, false)
}

func (svc *Service) setArchived(ctx context.Context, campID, id string, archived bool) error {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error beginning archive season transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	// Look up child sessions before flipping flags so we can mark them stale.
	sessionIDs, err := qtx.ListSessionIDsBySeason(ctx, db.ListSessionIDsBySeasonParams{
		SeasonID: uid,
		CampID:   campUUID,
	})
	if err != nil {
		return fmt.Errorf("error listing sessions for season %s: %w", id, err)
	}

	if err := svc.marker.MarkSessions(ctx, qtx, campUUID, sessionIDs, staleness.AllRunTypes); err != nil {
		return err
	}

	var rows int64
	if archived {
		rows, err = qtx.ArchiveSeason(ctx, db.ArchiveSeasonParams{ID: uid, CampID: campUUID})
	} else {
		rows, err = qtx.UnarchiveSeason(ctx, db.UnarchiveSeasonParams{ID: uid, CampID: campUUID})
	}
	if err != nil {
		return fmt.Errorf("error setting season %s archived=%t: %w", id, archived, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	// Cascade archive (but not unarchive) to child sessions so the user
	// doesn't have to archive each session individually. On unarchive we
	// leave child sessions in whatever state they were last in.
	if archived {
		if _, err := qtx.ArchiveSessionsBySeason(ctx, db.ArchiveSessionsBySeasonParams{
			SeasonID: uid,
			CampID:   campUUID,
		}); err != nil {
			return fmt.Errorf("error archiving sessions for season %s: %w", id, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing archive season transaction: %w", err)
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
		Archived:  s.Archived,
	}
}

func parseDate(s string) (pgtype.Date, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return pgtype.Date{}, fmt.Errorf("error invalid date %q (expected YYYY-MM-DD): %w", s, api.ErrBadInput)
	}
	return pgtype.Date{Time: t, Valid: true}, nil
}
