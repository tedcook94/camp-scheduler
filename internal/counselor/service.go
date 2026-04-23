package counselor

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("counselor not found")

type Service struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewService(queries *db.Queries, pool *pgxpool.Pool) *Service {
	return &Service{queries: queries, pool: pool}
}

func (svc *Service) List(ctx context.Context, campID string) ([]CounselorResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	counselors, err := svc.queries.ListCounselors(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("error listing counselors: %w", err)
	}

	result := make([]CounselorResponse, len(counselors))
	for i, c := range counselors {
		result[i] = toCounselorResponse(c)
	}
	return result, nil
}

func (svc *Service) GetByID(ctx context.Context, campID, id string) (CounselorResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return CounselorResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return CounselorResponse{}, err
	}

	counselor, err := svc.queries.GetCounselor(ctx, db.GetCounselorParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return CounselorResponse{}, fmt.Errorf("error getting counselor %s: %w", id, err)
	}

	return toCounselorResponse(counselor), nil
}

func (svc *Service) Create(ctx context.Context, campID string, req CreateCounselorRequest) (CounselorResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return CounselorResponse{}, err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return CounselorResponse{}, fmt.Errorf("error beginning create counselor transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	counselor, err := qtx.CreateCounselor(ctx, db.CreateCounselorParams{
		CampID:          uid,
		CounselorName:   req.Name,
		JuniorCounselor: req.JuniorCounselor,
		Gender:          req.Gender,
	})
	if err != nil {
		return CounselorResponse{}, fmt.Errorf("error creating counselor: %w", err)
	}

	// Roster the new counselor onto every existing session in the camp so
	// they're available to the solver and to preference filtering. Admins can
	// trim the roster per-session as needed.
	sessions, err := qtx.ListSessions(ctx, uid)
	if err != nil {
		return CounselorResponse{}, fmt.Errorf("error listing sessions for counselor roster: %w", err)
	}
	for _, s := range sessions {
		if _, err := qtx.AddSessionCounselor(ctx, db.AddSessionCounselorParams{
			CampID:      uid,
			SessionID:   s.ID,
			CounselorID: counselor.ID,
		}); err != nil {
			return CounselorResponse{}, fmt.Errorf("error rostering new counselor onto session: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return CounselorResponse{}, fmt.Errorf("error committing create counselor transaction: %w", err)
	}

	return toCounselorResponse(counselor), nil
}

func (svc *Service) Update(ctx context.Context, campID, id string, req UpdateCounselorRequest) (CounselorResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return CounselorResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return CounselorResponse{}, err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return CounselorResponse{}, fmt.Errorf("error beginning update counselor transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	existing, err := qtx.GetCounselor(ctx, db.GetCounselorParams{ID: uid, CampID: campUUID})
	if err != nil {
		return CounselorResponse{}, fmt.Errorf("error loading counselor %s: %w", id, err)
	}

	counselor, err := qtx.UpdateCounselor(ctx, db.UpdateCounselorParams{
		ID:               uid,
		CampID:           campUUID,
		CounselorName:    req.Name,
		JuniorCounselor:  req.JuniorCounselor,
		CounselorEnabled: req.Enabled,
		Gender:           req.Gender,
	})
	if err != nil {
		return CounselorResponse{}, fmt.Errorf("error updating counselor %s: %w", id, err)
	}

	// When a counselor transitions from enabled to disabled, remove them from
	// every session roster. This keeps the roster aligned with assignability;
	// re-enabling does not auto-restore prior memberships.
	if existing.CounselorEnabled && !req.Enabled {
		if err := qtx.RemoveCounselorFromAllSessions(ctx, db.RemoveCounselorFromAllSessionsParams{
			CampID:      campUUID,
			CounselorID: uid,
		}); err != nil {
			return CounselorResponse{}, fmt.Errorf("error removing disabled counselor from session rosters: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return CounselorResponse{}, fmt.Errorf("error committing update counselor transaction: %w", err)
	}

	return toCounselorResponse(counselor), nil
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

	rows, err := svc.queries.DeleteCounselor(ctx, db.DeleteCounselorParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting counselor %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func toCounselorResponse(c db.Counselor) CounselorResponse {
	return CounselorResponse{
		ID:              api.UUIDToString(c.ID),
		CampID:          api.UUIDToString(c.CampID),
		Name:            c.CounselorName,
		JuniorCounselor: c.JuniorCounselor,
		Enabled:         c.CounselorEnabled,
		Gender:          c.Gender,
	}
}
