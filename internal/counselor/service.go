package counselor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
	"camp-scheduler/internal/staleness"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("counselor not found")

type Service struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	marker  *staleness.Marker
}

func NewService(queries *db.Queries, pool *pgxpool.Pool, marker *staleness.Marker) *Service {
	return &Service{queries: queries, pool: pool, marker: marker}
}

// FullName joins first and last name into a display string. Empty parts are
// trimmed so single-token names render cleanly.
func FullName(first, last string) string {
	return strings.TrimSpace(first + " " + last)
}

func toResponse(id, campID pgtype.UUID, first, last string, junior, archived bool, gender string) CounselorResponse {
	return CounselorResponse{
		ID:              api.UUIDToString(id),
		CampID:          api.UUIDToString(campID),
		FirstName:       first,
		LastName:        last,
		Name:            FullName(first, last),
		JuniorCounselor: junior,
		Archived:        archived,
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
		result[i] = toResponse(c.ID, c.CampID, c.FirstName, c.LastName, c.JuniorCounselor, c.Archived, c.Gender)
	}
	return result, nil
}

func (svc *Service) ListArchived(ctx context.Context, campID string) ([]CounselorResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	counselors, err := svc.queries.ListArchivedCounselors(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("error listing archived counselors: %w", err)
	}

	result := make([]CounselorResponse, len(counselors))
	for i, c := range counselors {
		result[i] = toResponse(c.ID, c.CampID, c.FirstName, c.LastName, c.JuniorCounselor, c.Archived, c.Gender)
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

	return toResponse(c.ID, c.CampID, c.FirstName, c.LastName, c.JuniorCounselor, c.Archived, c.Gender), nil
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
	sessionIDs := make([]pgtype.UUID, 0, len(sessions))
	for _, s := range sessions {
		if _, err := qtx.AddSessionCounselor(ctx, db.AddSessionCounselorParams{
			CampID:      uid,
			SessionID:   s.ID,
			CounselorID: c.ID,
		}); err != nil {
			return CounselorResponse{}, fmt.Errorf("error rostering new counselor onto session: %w", err)
		}
		sessionIDs = append(sessionIDs, s.ID)
	}

	if err := svc.marker.MarkSessions(ctx, qtx, uid, sessionIDs, staleness.AllRunTypes); err != nil {
		return CounselorResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return CounselorResponse{}, fmt.Errorf("error committing create counselor transaction: %w", err)
	}

	return toResponse(c.ID, c.CampID, c.FirstName, c.LastName, c.JuniorCounselor, c.Archived, c.Gender), nil
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

	c, err := qtx.UpdateCounselor(ctx, db.UpdateCounselorParams{
		ID:              uid,
		CampID:          campUUID,
		FirstName:       req.FirstName,
		LastName:        req.LastName,
		JuniorCounselor: req.JuniorCounselor,
		Gender:          req.Gender,
	})
	if err != nil {
		return CounselorResponse{}, fmt.Errorf("error updating counselor %s: %w", id, err)
	}

	if err := svc.marker.MarkSessionsForCounselor(ctx, qtx, campUUID, uid, staleness.AllRunTypes); err != nil {
		return CounselorResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return CounselorResponse{}, fmt.Errorf("error committing update counselor transaction: %w", err)
	}

	return toResponse(c.ID, c.CampID, c.FirstName, c.LastName, c.JuniorCounselor, c.Archived, c.Gender), nil
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

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error beginning delete counselor transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	// Mark stale before the FK cascade removes session_counselors rows so
	// we still see which sessions were affected.
	if err := svc.marker.MarkSessionsForCounselor(ctx, qtx, campUUID, uid, staleness.AllRunTypes); err != nil {
		return err
	}

	rows, err := qtx.DeleteCounselor(ctx, db.DeleteCounselorParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting counselor %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing delete counselor transaction: %w", err)
	}
	return nil
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
		return fmt.Errorf("error beginning archive counselor transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	// Mark stale before flipping the flag so the roster lookup still sees
	// this counselor's session memberships.
	if err := svc.marker.MarkSessionsForCounselor(ctx, qtx, campUUID, uid, staleness.AllRunTypes); err != nil {
		return err
	}

	var rows int64
	if archived {
		rows, err = qtx.ArchiveCounselor(ctx, db.ArchiveCounselorParams{ID: uid, CampID: campUUID})
	} else {
		rows, err = qtx.UnarchiveCounselor(ctx, db.UnarchiveCounselorParams{ID: uid, CampID: campUUID})
	}
	if err != nil {
		return fmt.Errorf("error setting counselor %s archived=%t: %w", id, archived, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	// On archive, remove from all session counselor rosters so the solver
	// and preference filtering stop considering them. Unarchiving does not
	// auto-restore membership; admins must re-add manually.
	if archived {
		if err := qtx.RemoveCounselorFromAllSessions(ctx, db.RemoveCounselorFromAllSessionsParams{
			CounselorID: uid,
			CampID:      campUUID,
		}); err != nil {
			return fmt.Errorf("error removing archived counselor %s from session rosters: %w", id, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing archive counselor transaction: %w", err)
	}

	return nil
}
