package counselor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrNotFound = errors.New("counselor not found")

type Service struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewService(queries *db.Queries, pool *pgxpool.Pool) *Service {
	return &Service{queries: queries, pool: pool}
}

// FullName joins first and last name into a display string. Empty parts are
// trimmed so single-token names render cleanly.
func FullName(first, last string) string {
	return strings.TrimSpace(first + " " + last)
}

func toResponse(id, campID pgtype.UUID, first, last string, junior, enabled bool, gender string) CounselorResponse {
	return CounselorResponse{
		ID:              api.UUIDToString(id),
		CampID:          api.UUIDToString(campID),
		FirstName:       first,
		LastName:        last,
		Name:            FullName(first, last),
		JuniorCounselor: junior,
		Enabled:         enabled,
		Gender:          gender,
	}
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
		result[i] = toResponse(c.ID, c.CampID, c.FirstName, c.LastName, c.JuniorCounselor, c.CounselorEnabled, c.Gender)
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

	c, err := svc.queries.GetCounselor(ctx, db.GetCounselorParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return CounselorResponse{}, fmt.Errorf("error getting counselor %s: %w", id, err)
	}

	return toResponse(c.ID, c.CampID, c.FirstName, c.LastName, c.JuniorCounselor, c.CounselorEnabled, c.Gender), nil
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

	c, err := qtx.CreateCounselor(ctx, db.CreateCounselorParams{
		CampID:          uid,
		FirstName:       req.FirstName,
		LastName:        req.LastName,
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
			CounselorID: c.ID,
		}); err != nil {
			return CounselorResponse{}, fmt.Errorf("error rostering new counselor onto session: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return CounselorResponse{}, fmt.Errorf("error committing create counselor transaction: %w", err)
	}

	return toResponse(c.ID, c.CampID, c.FirstName, c.LastName, c.JuniorCounselor, c.CounselorEnabled, c.Gender), nil
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

	c, err := qtx.UpdateCounselor(ctx, db.UpdateCounselorParams{
		ID:               uid,
		CampID:           campUUID,
		FirstName:        req.FirstName,
		LastName:         req.LastName,
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

	return toResponse(c.ID, c.CampID, c.FirstName, c.LastName, c.JuniorCounselor, c.CounselorEnabled, c.Gender), nil
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
