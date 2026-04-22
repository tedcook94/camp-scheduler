package sessionconfig

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
)

var ErrSessionCounselorNotFound = errors.New("session counselor not found")

type CounselorService struct {
	queries *db.Queries
}

func NewCounselorService(queries *db.Queries) *CounselorService {
	return &CounselorService{queries: queries}
}

type SessionCounselorResponse struct {
	ID               string `json:"id"`
	CampID           string `json:"camp_id"`
	SessionID        string `json:"session_id"`
	CounselorID      string `json:"counselor_id"`
	CounselorName    string `json:"counselor_name"`
	JuniorCounselor  bool   `json:"junior_counselor"`
	CounselorEnabled bool   `json:"counselor_enabled"`
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
			ID:               api.UUIDToString(r.ID),
			CampID:           api.UUIDToString(r.CampID),
			SessionID:        api.UUIDToString(r.SessionID),
			CounselorID:      api.UUIDToString(r.CounselorID),
			CounselorName:    r.CounselorName,
			JuniorCounselor:  r.JuniorCounselor,
			CounselorEnabled: r.CounselorEnabled,
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

	// Verify the session belongs to this camp before attempting to insert,
	// so we surface a clean error instead of an FK violation when the session
	// id is bogus.
	if _, err := svc.queries.GetSession(ctx, db.GetSessionParams{ID: sessionUUID, CampID: campUUID}); err != nil {
		return SessionCounselorResponse{}, fmt.Errorf("error validating session %s: %w", sessionID, err)
	}

	// Load the counselor first so we can validate eligibility (must belong to
	// the camp and be enabled) and so we have the details needed to build the
	// response without a second round-trip after the insert.
	counselor, err := svc.queries.GetCounselor(ctx, db.GetCounselorParams{ID: counselorUUID, CampID: campUUID})
	if err != nil {
		return SessionCounselorResponse{}, fmt.Errorf("error loading counselor %s: %w", req.CounselorID, err)
	}
	if !counselor.CounselorEnabled {
		return SessionCounselorResponse{}, api.BadInput("counselor is disabled and cannot be added to a session")
	}

	row, err := svc.queries.AddSessionCounselor(ctx, db.AddSessionCounselorParams{
		CampID:      campUUID,
		SessionID:   sessionUUID,
		CounselorID: counselorUUID,
	})
	if err != nil {
		return SessionCounselorResponse{}, fmt.Errorf("error adding session counselor: %w", err)
	}

	return SessionCounselorResponse{
		ID:               api.UUIDToString(row.ID),
		CampID:           api.UUIDToString(row.CampID),
		SessionID:        api.UUIDToString(row.SessionID),
		CounselorID:      api.UUIDToString(row.CounselorID),
		CounselorName:    counselor.CounselorName,
		JuniorCounselor:  counselor.JuniorCounselor,
		CounselorEnabled: counselor.CounselorEnabled,
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

	rows, err := svc.queries.RemoveSessionCounselor(ctx, db.RemoveSessionCounselorParams{
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
	return nil
}
