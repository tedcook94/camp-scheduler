package history

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
	"camp-scheduler/internal/staleness"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSessionHistoryNotFound = errors.New("session history entry not found")

type Service struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	marker  *staleness.Marker
}

func NewService(queries *db.Queries, pool *pgxpool.Pool, marker *staleness.Marker) *Service {
	return &Service{queries: queries, pool: pool, marker: marker}
}

func (svc *Service) List(ctx context.Context, campID, counselorID string) ([]SessionHistoryResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	counselorUUID, err := api.ParseUUID(counselorID)
	if err != nil {
		return nil, err
	}

	entries, err := svc.queries.ListCounselorSessionHistory(ctx, db.ListCounselorSessionHistoryParams{
		CounselorID: counselorUUID,
		CampID:      campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing session history: %w", err)
	}

	result := make([]SessionHistoryResponse, len(entries))
	for i, e := range entries {
		result[i] = toSessionHistoryResponse(e)
	}
	return result, nil
}

func (svc *Service) GetByID(ctx context.Context, campID, counselorID, id string) (SessionHistoryResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionHistoryResponse{}, err
	}

	counselorUUID, err := api.ParseUUID(counselorID)
	if err != nil {
		return SessionHistoryResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return SessionHistoryResponse{}, err
	}

	entry, err := svc.queries.GetCounselorSessionHistoryEntry(ctx, db.GetCounselorSessionHistoryEntryParams{
		ID:          uid,
		CampID:      campUUID,
		CounselorID: counselorUUID,
	})
	if err != nil {
		return SessionHistoryResponse{}, fmt.Errorf("error getting session history entry %s: %w", id, err)
	}

	return toSessionHistoryResponse(entry), nil
}

func (svc *Service) Create(ctx context.Context, campID, counselorID string, req CreateSessionHistoryRequest) (SessionHistoryResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionHistoryResponse{}, err
	}

	counselorUUID, err := api.ParseUUID(counselorID)
	if err != nil {
		return SessionHistoryResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(req.SessionID)
	if err != nil {
		return SessionHistoryResponse{}, err
	}

	ageGroupUUID, err := api.ParseUUID(req.AgeGroupID)
	if err != nil {
		return SessionHistoryResponse{}, err
	}

	cabinUUID, err := api.ToPgUUID(req.CabinID)
	if err != nil {
		return SessionHistoryResponse{}, err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return SessionHistoryResponse{}, fmt.Errorf("error beginning create session history transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	entry, err := qtx.CreateCounselorSessionHistoryEntry(ctx, db.CreateCounselorSessionHistoryEntryParams{
		CampID:      campUUID,
		CounselorID: counselorUUID,
		SessionID:   sessionUUID,
		AgeGroupID:  ageGroupUUID,
		CabinID:     cabinUUID,
	})
	if err != nil {
		return SessionHistoryResponse{}, fmt.Errorf("error creating session history entry: %w", err)
	}

	// History on session A feeds the cabin solver of any session B with
	// previous_session=A. Mark via cascade.
	if err := svc.marker.MarkSessions(ctx, qtx, campUUID, []pgtype.UUID{sessionUUID}, []staleness.RunType{staleness.RunTypeCabin}); err != nil {
		return SessionHistoryResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return SessionHistoryResponse{}, fmt.Errorf("error committing create session history transaction: %w", err)
	}

	return toSessionHistoryResponse(entry), nil
}

func (svc *Service) Update(ctx context.Context, campID, counselorID, id string, req UpdateSessionHistoryRequest) (SessionHistoryResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionHistoryResponse{}, err
	}

	counselorUUID, err := api.ParseUUID(counselorID)
	if err != nil {
		return SessionHistoryResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return SessionHistoryResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(req.SessionID)
	if err != nil {
		return SessionHistoryResponse{}, err
	}

	ageGroupUUID, err := api.ParseUUID(req.AgeGroupID)
	if err != nil {
		return SessionHistoryResponse{}, err
	}

	cabinUUID, err := api.ToPgUUID(req.CabinID)
	if err != nil {
		return SessionHistoryResponse{}, err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return SessionHistoryResponse{}, fmt.Errorf("error beginning update session history transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	// Fetch the current entry first so we can detect a session move and
	// dirty both the old and new session's dependent runs.
	existing, err := qtx.GetCounselorSessionHistoryEntry(ctx, db.GetCounselorSessionHistoryEntryParams{
		ID:          uid,
		CampID:      campUUID,
		CounselorID: counselorUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SessionHistoryResponse{}, ErrSessionHistoryNotFound
		}
		return SessionHistoryResponse{}, fmt.Errorf("error getting session history entry %s: %w", id, err)
	}

	entry, err := qtx.UpdateCounselorSessionHistoryEntry(ctx, db.UpdateCounselorSessionHistoryEntryParams{
		ID:          uid,
		CampID:      campUUID,
		CounselorID: counselorUUID,
		SessionID:   sessionUUID,
		AgeGroupID:  ageGroupUUID,
		CabinID:     cabinUUID,
	})
	if err != nil {
		return SessionHistoryResponse{}, fmt.Errorf("error updating session history entry %s: %w", id, err)
	}

	// When a history entry moves between sessions, both the old and new
	// session's dependents need to be invalidated — the old loses an input,
	// the new gains one.
	sessionIDs := []pgtype.UUID{sessionUUID}
	if existing.SessionID != sessionUUID {
		sessionIDs = append(sessionIDs, existing.SessionID)
	}
	if err := svc.marker.MarkSessions(ctx, qtx, campUUID, sessionIDs, []staleness.RunType{staleness.RunTypeCabin}); err != nil {
		return SessionHistoryResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return SessionHistoryResponse{}, fmt.Errorf("error committing update session history transaction: %w", err)
	}

	return toSessionHistoryResponse(entry), nil
}

func (svc *Service) Delete(ctx context.Context, campID, counselorID, id string) error {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return err
	}

	counselorUUID, err := api.ParseUUID(counselorID)
	if err != nil {
		return err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error beginning delete session history transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	// Fetch the entry first so we know which session is affected by its
	// removal — the row is gone after the delete.
	entry, err := qtx.GetCounselorSessionHistoryEntry(ctx, db.GetCounselorSessionHistoryEntryParams{
		ID:          uid,
		CampID:      campUUID,
		CounselorID: counselorUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrSessionHistoryNotFound
		}
		return fmt.Errorf("error getting session history entry %s: %w", id, err)
	}

	rows, err := qtx.DeleteCounselorSessionHistoryEntry(ctx, db.DeleteCounselorSessionHistoryEntryParams{
		ID:          uid,
		CampID:      campUUID,
		CounselorID: counselorUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting session history entry %s: %w", id, err)
	}
	if rows == 0 {
		return ErrSessionHistoryNotFound
	}

	if err := svc.marker.MarkSessions(ctx, qtx, campUUID, []pgtype.UUID{entry.SessionID}, []staleness.RunType{staleness.RunTypeCabin}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing delete session history transaction: %w", err)
	}

	return nil
}

func (svc *Service) GetHistory(ctx context.Context, campID, counselorID string, seasonID *string) ([]HistorySummaryEntry, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	counselorUUID, err := api.ParseUUID(counselorID)
	if err != nil {
		return nil, err
	}

	seasonUUID, err := svc.resolveSeasonID(ctx, campUUID, seasonID)
	if err != nil {
		return nil, err
	}

	rows, err := svc.queries.GetCounselorHistorySummary(ctx, db.GetCounselorHistorySummaryParams{
		CounselorID: counselorUUID,
		CampID:      campUUID,
		SeasonID:    seasonUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error getting counselor history summary: %w", err)
	}

	result := make([]HistorySummaryEntry, len(rows))
	for i, r := range rows {
		result[i] = toHistorySummaryEntry(r)
	}
	return result, nil
}

// resolveSeasonID returns the pgtype.UUID for the given season ID string.
// If seasonID is nil, it looks up the most recent season for the camp.
// If seasonID is the literal "all", it returns an invalid UUID to skip filtering.
func (svc *Service) resolveSeasonID(ctx context.Context, campUUID pgtype.UUID, seasonID *string) (pgtype.UUID, error) {
	if seasonID != nil {
		if *seasonID == "all" {
			return pgtype.UUID{}, nil
		}
		return api.ParseUUID(*seasonID)
	}

	season, err := svc.queries.GetMostRecentSeason(ctx, campUUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgtype.UUID{}, nil
		}
		return pgtype.UUID{}, fmt.Errorf("error resolving most recent season: %w", err)
	}

	return season.ID, nil
}

func toHistorySummaryEntry(r db.GetCounselorHistorySummaryRow) HistorySummaryEntry {
	var cabinName *string
	if r.CabinName.Valid {
		cabinName = &r.CabinName.String
	}

	return HistorySummaryEntry{
		ID:           api.UUIDToString(r.ID),
		SessionName:  r.SessionName,
		SeasonID:     api.UUIDToString(r.SeasonID),
		SeasonName:   r.SeasonName,
		AgeGroupName: r.AgeGroupName,
		CabinName:    cabinName,
	}
}

func toSessionHistoryResponse(e db.CounselorSessionHistory) SessionHistoryResponse {
	return SessionHistoryResponse{
		ID:          api.UUIDToString(e.ID),
		CampID:      api.UUIDToString(e.CampID),
		CounselorID: api.UUIDToString(e.CounselorID),
		SessionID:   api.UUIDToString(e.SessionID),
		AgeGroupID:  api.UUIDToString(e.AgeGroupID),
		CabinID:     api.UUIDToStringPtr(e.CabinID),
	}
}
