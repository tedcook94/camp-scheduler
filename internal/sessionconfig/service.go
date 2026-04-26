package sessionconfig

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

var (
	ErrAgeGroupNotFound = errors.New("session age group not found")
	ErrCabinNotFound    = errors.New("session cabin not found")
)

type Service struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	marker  *staleness.Marker
}

func NewService(queries *db.Queries, pool *pgxpool.Pool, marker *staleness.Marker) *Service {
	return &Service{queries: queries, pool: pool, marker: marker}
}

func (svc *Service) markCabinStale(ctx context.Context, q *db.Queries, campID, sessionID pgtype.UUID) error {
	return svc.marker.MarkSessions(ctx, q, campID, []pgtype.UUID{sessionID}, []staleness.RunType{staleness.RunTypeCabin})
}

// ensureAgeGroupActive rejects attaching an archived age group to a session.
// Archived resources are filtered from default lists and ignored by the
// solver, so accepting them here would create config that exists in DB but
// silently goes unused.
func (svc *Service) ensureAgeGroupActive(ctx context.Context, q *db.Queries, campID, ageGroupID pgtype.UUID) error {
	ag, err := q.GetAgeGroup(ctx, db.GetAgeGroupParams{
		ID:     ageGroupID,
		CampID: campID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return api.BadInput("age group not found")
		}
		return fmt.Errorf("error looking up age group: %w", err)
	}
	if ag.Archived {
		return api.BadInput("cannot use an archived age group")
	}
	return nil
}

func (svc *Service) ensureCabinActive(ctx context.Context, q *db.Queries, campID, cabinID pgtype.UUID) error {
	cabin, err := q.GetCabin(ctx, db.GetCabinParams{
		ID:     cabinID,
		CampID: campID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return api.BadInput("cabin not found")
		}
		return fmt.Errorf("error looking up cabin: %w", err)
	}
	if cabin.Archived {
		return api.BadInput("cannot use an archived cabin")
	}
	return nil
}

// Session age group operations

func (svc *Service) ListAgeGroups(ctx context.Context, campID, sessionID string) ([]SessionAgeGroupResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return nil, err
	}

	rows, err := svc.queries.ListSessionAgeGroups(ctx, db.ListSessionAgeGroupsParams{
		SessionID: sessionUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing session age groups: %w", err)
	}

	result := make([]SessionAgeGroupResponse, len(rows))
	for i, r := range rows {
		result[i] = toSessionAgeGroupResponse(r)
	}
	return result, nil
}

func (svc *Service) GetAgeGroup(ctx context.Context, campID, sessionID, id string) (SessionAgeGroupResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionAgeGroupResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return SessionAgeGroupResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return SessionAgeGroupResponse{}, err
	}

	row, err := svc.queries.GetSessionAgeGroup(ctx, db.GetSessionAgeGroupParams{
		ID:        uid,
		CampID:    campUUID,
		SessionID: sessionUUID,
	})
	if err != nil {
		return SessionAgeGroupResponse{}, fmt.Errorf("error getting session age group %s: %w", id, err)
	}

	return toSessionAgeGroupResponse(row), nil
}

func (svc *Service) CreateAgeGroup(ctx context.Context, campID, sessionID string, req CreateSessionAgeGroupRequest) (SessionAgeGroupResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionAgeGroupResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return SessionAgeGroupResponse{}, err
	}

	ageGroupUUID, err := api.ParseUUID(req.AgeGroupID)
	if err != nil {
		return SessionAgeGroupResponse{}, err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return SessionAgeGroupResponse{}, fmt.Errorf("error beginning create session age group transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	if err := svc.ensureAgeGroupActive(ctx, qtx, campUUID, ageGroupUUID); err != nil {
		return SessionAgeGroupResponse{}, err
	}

	row, err := qtx.CreateSessionAgeGroup(ctx, db.CreateSessionAgeGroupParams{
		CampID:     campUUID,
		SessionID:  sessionUUID,
		AgeGroupID: ageGroupUUID,
	})
	if err != nil {
		return SessionAgeGroupResponse{}, fmt.Errorf("error creating session age group: %w", err)
	}

	if err := svc.markCabinStale(ctx, qtx, campUUID, sessionUUID); err != nil {
		return SessionAgeGroupResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return SessionAgeGroupResponse{}, fmt.Errorf("error committing create session age group transaction: %w", err)
	}

	return toSessionAgeGroupResponse(row), nil
}

func (svc *Service) UpdateAgeGroup(ctx context.Context, campID, sessionID, id string, req UpdateSessionAgeGroupRequest) (SessionAgeGroupResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionAgeGroupResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return SessionAgeGroupResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return SessionAgeGroupResponse{}, err
	}

	ageGroupUUID, err := api.ParseUUID(req.AgeGroupID)
	if err != nil {
		return SessionAgeGroupResponse{}, err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return SessionAgeGroupResponse{}, fmt.Errorf("error beginning update session age group transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	if err := svc.ensureAgeGroupActive(ctx, qtx, campUUID, ageGroupUUID); err != nil {
		return SessionAgeGroupResponse{}, err
	}

	row, err := qtx.UpdateSessionAgeGroup(ctx, db.UpdateSessionAgeGroupParams{
		ID:         uid,
		CampID:     campUUID,
		SessionID:  sessionUUID,
		AgeGroupID: ageGroupUUID,
	})
	if err != nil {
		return SessionAgeGroupResponse{}, fmt.Errorf("error updating session age group %s: %w", id, err)
	}

	if err := svc.markCabinStale(ctx, qtx, campUUID, sessionUUID); err != nil {
		return SessionAgeGroupResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return SessionAgeGroupResponse{}, fmt.Errorf("error committing update session age group transaction: %w", err)
	}

	return toSessionAgeGroupResponse(row), nil
}

func (svc *Service) DeleteAgeGroup(ctx context.Context, campID, sessionID, id string) error {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error beginning delete session age group transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	rows, err := qtx.DeleteSessionAgeGroup(ctx, db.DeleteSessionAgeGroupParams{
		ID:        uid,
		CampID:    campUUID,
		SessionID: sessionUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting session age group %s: %w", id, err)
	}
	if rows == 0 {
		return ErrAgeGroupNotFound
	}

	if err := svc.markCabinStale(ctx, qtx, campUUID, sessionUUID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing delete session age group transaction: %w", err)
	}

	return nil
}

// Session cabin operations

func (svc *Service) ListCabins(ctx context.Context, campID, sessionID string) ([]SessionCabinResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return nil, err
	}

	rows, err := svc.queries.ListSessionAgeGroupCabinsBySession(ctx, db.ListSessionAgeGroupCabinsBySessionParams{
		SessionID: sessionUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing session cabins: %w", err)
	}

	result := make([]SessionCabinResponse, len(rows))
	for i, r := range rows {
		result[i] = toSessionCabinResponse(r, sessionID)
	}
	return result, nil
}

func (svc *Service) GetCabin(ctx context.Context, campID, sessionID, id string) (SessionCabinResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionCabinResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return SessionCabinResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return SessionCabinResponse{}, err
	}

	row, err := svc.queries.GetSessionAgeGroupCabin(ctx, db.GetSessionAgeGroupCabinParams{
		ID:        uid,
		CampID:    campUUID,
		SessionID: sessionUUID,
	})
	if err != nil {
		return SessionCabinResponse{}, fmt.Errorf("error getting session cabin %s: %w", id, err)
	}

	return toSessionCabinResponse(row, sessionID), nil
}

func (svc *Service) CreateCabin(ctx context.Context, campID, sessionID string, req CreateSessionCabinRequest) (SessionCabinResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionCabinResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return SessionCabinResponse{}, err
	}

	sessionAgeGroupUUID, err := api.ParseUUID(req.SessionAgeGroupID)
	if err != nil {
		return SessionCabinResponse{}, err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return SessionCabinResponse{}, fmt.Errorf("error beginning create session cabin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	// Verify the session age group belongs to this session.
	// The composite FK enforces camp-scoping, but not session-scoping.
	_, err = qtx.GetSessionAgeGroup(ctx, db.GetSessionAgeGroupParams{
		ID:        sessionAgeGroupUUID,
		CampID:    campUUID,
		SessionID: sessionUUID,
	})
	if err != nil {
		return SessionCabinResponse{}, fmt.Errorf("error validating session age group %s: %w", req.SessionAgeGroupID, err)
	}

	cabinUUID, err := api.ParseUUID(req.CabinID)
	if err != nil {
		return SessionCabinResponse{}, err
	}

	if err := svc.ensureCabinActive(ctx, qtx, campUUID, cabinUUID); err != nil {
		return SessionCabinResponse{}, err
	}

	row, err := qtx.CreateSessionAgeGroupCabin(ctx, db.CreateSessionAgeGroupCabinParams{
		CampID:             campUUID,
		SessionAgeGroupID:  sessionAgeGroupUUID,
		CabinID:            cabinUUID,
		GroupSize:          req.GroupSize,
		RequiredCounselors: req.RequiredCounselors,
		Gender:             req.Gender,
	})
	if err != nil {
		return SessionCabinResponse{}, fmt.Errorf("error creating session cabin: %w", err)
	}

	if err := svc.markCabinStale(ctx, qtx, campUUID, sessionUUID); err != nil {
		return SessionCabinResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return SessionCabinResponse{}, fmt.Errorf("error committing create session cabin transaction: %w", err)
	}

	return toSessionCabinResponse(row, sessionID), nil
}

func (svc *Service) UpdateCabin(ctx context.Context, campID, sessionID, id string, req UpdateSessionCabinRequest) (SessionCabinResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionCabinResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return SessionCabinResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return SessionCabinResponse{}, err
	}

	cabinUUID, err := api.ParseUUID(req.CabinID)
	if err != nil {
		return SessionCabinResponse{}, err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return SessionCabinResponse{}, fmt.Errorf("error beginning update session cabin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	if err := svc.ensureCabinActive(ctx, qtx, campUUID, cabinUUID); err != nil {
		return SessionCabinResponse{}, err
	}

	row, err := qtx.UpdateSessionAgeGroupCabin(ctx, db.UpdateSessionAgeGroupCabinParams{
		ID:                 uid,
		CampID:             campUUID,
		SessionID:          sessionUUID,
		CabinID:            cabinUUID,
		GroupSize:          req.GroupSize,
		RequiredCounselors: req.RequiredCounselors,
		Gender:             req.Gender,
	})
	if err != nil {
		return SessionCabinResponse{}, fmt.Errorf("error updating session cabin %s: %w", id, err)
	}

	if err := svc.markCabinStale(ctx, qtx, campUUID, sessionUUID); err != nil {
		return SessionCabinResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return SessionCabinResponse{}, fmt.Errorf("error committing update session cabin transaction: %w", err)
	}

	return toSessionCabinResponse(row, sessionID), nil
}

func (svc *Service) DeleteCabin(ctx context.Context, campID, sessionID, id string) error {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error beginning delete session cabin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	rows, err := qtx.DeleteSessionAgeGroupCabin(ctx, db.DeleteSessionAgeGroupCabinParams{
		ID:        uid,
		CampID:    campUUID,
		SessionID: sessionUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting session cabin %s: %w", id, err)
	}
	if rows == 0 {
		return ErrCabinNotFound
	}

	if err := svc.markCabinStale(ctx, qtx, campUUID, sessionUUID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing delete session cabin transaction: %w", err)
	}

	return nil
}

// Response constructors

func toSessionAgeGroupResponse(r db.SessionAgeGroup) SessionAgeGroupResponse {
	return SessionAgeGroupResponse{
		ID:         api.UUIDToString(r.ID),
		CampID:     api.UUIDToString(r.CampID),
		SessionID:  api.UUIDToString(r.SessionID),
		AgeGroupID: api.UUIDToString(r.AgeGroupID),
	}
}

func toSessionCabinResponse(r db.SessionAgeGroupCabin, sessionID string) SessionCabinResponse {
	return SessionCabinResponse{
		ID:                 api.UUIDToString(r.ID),
		CampID:             api.UUIDToString(r.CampID),
		SessionID:          sessionID,
		SessionAgeGroupID:  api.UUIDToString(r.SessionAgeGroupID),
		CabinID:            api.UUIDToString(r.CabinID),
		GroupSize:          r.GroupSize,
		RequiredCounselors: r.RequiredCounselors,
		Gender:             r.Gender,
	}
}
