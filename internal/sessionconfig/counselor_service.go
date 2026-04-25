package sessionconfig

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/counselor"
	"camp-scheduler/internal/db"
	"camp-scheduler/internal/staleness"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSessionCounselorNotFound = errors.New("session counselor not found")

type CounselorService struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	marker  *staleness.Marker
}

func NewCounselorService(queries *db.Queries, pool *pgxpool.Pool, marker *staleness.Marker) *CounselorService {
	return &CounselorService{queries: queries, pool: pool, marker: marker}
}

type SessionCounselorResponse struct {
	ID                 string `json:"id"`
	CampID             string `json:"camp_id"`
	SessionID          string `json:"session_id"`
	CounselorID        string `json:"counselor_id"`
	CounselorFirstName string `json:"counselor_first_name"`
	CounselorLastName  string `json:"counselor_last_name"`
	CounselorName      string `json:"counselor_name"`
	JuniorCounselor    bool   `json:"junior_counselor"`
	Archived           bool   `json:"archived"`
	Gender             string `json:"gender"`
}

type AddSessionCounselorRequest struct {
	CounselorID string `json:"counselor_id" binding:"required"`
}

func (svc *CounselorService) List(ctx context.Context, campID, sessionID string) ([]SessionCounselorResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return nil, err
	}

	rows, err := svc.queries.ListSessionCounselors(ctx, db.ListSessionCounselorsParams{
		SessionID: sessionUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing session counselors: %w", err)
	}

	result := make([]SessionCounselorResponse, len(rows))
	for i, r := range rows {
		result[i] = SessionCounselorResponse{
			ID:                 api.UUIDToString(r.ID),
			CampID:             api.UUIDToString(r.CampID),
			SessionID:          api.UUIDToString(r.SessionID),
			CounselorID:        api.UUIDToString(r.CounselorID),
			CounselorFirstName: r.CounselorFirstName,
			CounselorLastName:  r.CounselorLastName,
			CounselorName:      r.CounselorName,
			JuniorCounselor:    r.JuniorCounselor,
			Archived:           r.Archived,
			Gender:             r.Gender,
		}
	}
	return result, nil
}

func (svc *CounselorService) Add(ctx context.Context, campID, sessionID string, req AddSessionCounselorRequest) (SessionCounselorResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionCounselorResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return SessionCounselorResponse{}, err
	}

	counselorUUID, err := api.ParseUUID(req.CounselorID)
	if err != nil {
		return SessionCounselorResponse{}, err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return SessionCounselorResponse{}, fmt.Errorf("error beginning add session counselor transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	// Verify the session belongs to this camp before attempting to insert,
	// so we surface a clean error instead of an FK violation when the session
	// id is bogus.
	if _, err := qtx.GetSession(ctx, db.GetSessionParams{ID: sessionUUID, CampID: campUUID}); err != nil {
		return SessionCounselorResponse{}, fmt.Errorf("error validating session %s: %w", sessionID, err)
	}

	// Load the counselor first so we can validate eligibility (must belong to
	// the camp and not be archived) and so we have the details needed to build
	// the response without a second round-trip after the insert.
	c, err := qtx.GetCounselor(ctx, db.GetCounselorParams{ID: counselorUUID, CampID: campUUID})
	if err != nil {
		return SessionCounselorResponse{}, fmt.Errorf("error loading counselor %s: %w", req.CounselorID, err)
	}
	if c.Archived {
		return SessionCounselorResponse{}, api.BadInput("counselor is archived and cannot be added to a session")
	}

	row, err := qtx.AddSessionCounselor(ctx, db.AddSessionCounselorParams{
		CampID:      campUUID,
		SessionID:   sessionUUID,
		CounselorID: counselorUUID,
	})
	if err != nil {
		return SessionCounselorResponse{}, fmt.Errorf("error adding session counselor: %w", err)
	}

	if err := svc.marker.MarkSessions(ctx, qtx, campUUID, []pgtype.UUID{sessionUUID}, staleness.AllRunTypes); err != nil {
		return SessionCounselorResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return SessionCounselorResponse{}, fmt.Errorf("error committing add session counselor transaction: %w", err)
	}

	return SessionCounselorResponse{
		ID:                 api.UUIDToString(row.ID),
		CampID:             api.UUIDToString(row.CampID),
		SessionID:          api.UUIDToString(row.SessionID),
		CounselorID:        api.UUIDToString(row.CounselorID),
		CounselorFirstName: c.FirstName,
		CounselorLastName:  c.LastName,
		CounselorName:      counselor.FullName(c.FirstName, c.LastName),
		JuniorCounselor:    c.JuniorCounselor,
		Archived:           c.Archived,
		Gender:             c.Gender,
	}, nil
}

func (svc *CounselorService) Remove(ctx context.Context, campID, sessionID, counselorID string) error {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return err
	}

	counselorUUID, err := api.ParseUUID(counselorID)
	if err != nil {
		return err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error beginning remove session counselor transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	rows, err := qtx.RemoveSessionCounselor(ctx, db.RemoveSessionCounselorParams{
		SessionID:   sessionUUID,
		CampID:      campUUID,
		CounselorID: counselorUUID,
	})
	if err != nil {
		return fmt.Errorf("error removing session counselor: %w", err)
	}
	if rows == 0 {
		return ErrSessionCounselorNotFound
	}

	if err := svc.marker.MarkSessions(ctx, qtx, campUUID, []pgtype.UUID{sessionUUID}, staleness.AllRunTypes); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing remove session counselor transaction: %w", err)
	}
	return nil
}
