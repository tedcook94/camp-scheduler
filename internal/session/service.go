package session

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrNotFound = errors.New("session not found")

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

func (svc *Service) List(ctx context.Context, campID string) ([]SessionResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	sessions, err := svc.queries.ListSessions(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("error listing sessions: %w", err)
	}

	result := make([]SessionResponse, len(sessions))
	for i, sn := range sessions {
		result[i] = toSessionResponse(sn)
	}
	return result, nil
}

func (svc *Service) GetByID(ctx context.Context, campID, id string) (SessionResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return SessionResponse{}, err
	}

	session, err := svc.queries.GetSession(ctx, db.GetSessionParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return SessionResponse{}, fmt.Errorf("error getting session %s: %w", id, err)
	}

	return toSessionResponse(session), nil
}

func (svc *Service) Create(ctx context.Context, campID string, req CreateSessionRequest) (SessionResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionResponse{}, err
	}

	seasonUUID, err := api.ParseUUID(req.SeasonID)
	if err != nil {
		return SessionResponse{}, err
	}

	prevUUID, err := api.ToPgUUID(req.PreviousSessionID)
	if err != nil {
		return SessionResponse{}, err
	}

	if err := svc.validatePreviousSession(ctx, campUUID, seasonUUID, pgtype.UUID{}, req.PreviousSessionID); err != nil {
		return SessionResponse{}, err
	}

	session, err := svc.queries.CreateSession(ctx, db.CreateSessionParams{
		CampID:          campUUID,
		SeasonID:        seasonUUID,
		SessionName:     req.Name,
		PreviousSession: prevUUID,
	})
	if err != nil {
		return SessionResponse{}, fmt.Errorf("error creating session: %w", err)
	}

	return toSessionResponse(session), nil
}

func (svc *Service) Update(ctx context.Context, campID, id string, req UpdateSessionRequest) (SessionResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return SessionResponse{}, err
	}

	seasonUUID, err := api.ParseUUID(req.SeasonID)
	if err != nil {
		return SessionResponse{}, err
	}

	prevUUID, err := api.ToPgUUID(req.PreviousSessionID)
	if err != nil {
		return SessionResponse{}, err
	}

	current, err := svc.queries.GetSession(ctx, db.GetSessionParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SessionResponse{}, ErrNotFound
		}
		return SessionResponse{}, fmt.Errorf("error getting session %s: %w", id, err)
	}

	if err := svc.validatePreviousSession(ctx, campUUID, seasonUUID, uid, req.PreviousSessionID); err != nil {
		return SessionResponse{}, err
	}

	if current.SeasonID != seasonUUID {
		hasDeps, err := svc.queries.HasDependentSessions(ctx, db.HasDependentSessionsParams{
			PreviousSession: uid,
			CampID:          campUUID,
		})
		if err != nil {
			return SessionResponse{}, fmt.Errorf("error checking session dependents: %w", err)
		}
		if hasDeps {
			return SessionResponse{}, api.BadInput("cannot change season: other sessions reference this session as previous")
		}
	}

	session, err := svc.queries.UpdateSession(ctx, db.UpdateSessionParams{
		ID:              uid,
		CampID:          campUUID,
		SeasonID:        seasonUUID,
		SessionName:     req.Name,
		PreviousSession: prevUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SessionResponse{}, ErrNotFound
		}
		return SessionResponse{}, fmt.Errorf("error updating session %s: %w", id, err)
	}

	return toSessionResponse(session), nil
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

	rows, err := svc.queries.DeleteSession(ctx, db.DeleteSessionParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting session %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func toSessionResponse(s db.Session) SessionResponse {
	return SessionResponse{
		ID:                api.UUIDToString(s.ID),
		CampID:            api.UUIDToString(s.CampID),
		SeasonID:          api.UUIDToString(s.SeasonID),
		Name:              s.SessionName,
		PreviousSessionID: api.UUIDToStringPtr(s.PreviousSession),
	}
}

// validatePreviousSession checks that the referenced previous session belongs
// to the same season and is not a self-reference. No-op when previousSessionID is nil.
func (svc *Service) validatePreviousSession(ctx context.Context, campID, seasonID, sessionID pgtype.UUID, previousSessionID *string) error {
	if previousSessionID == nil {
		return nil
	}

	prevUUID, err := api.ParseUUID(*previousSessionID)
	if err != nil {
		return err
	}

	if sessionID.Valid && prevUUID == sessionID {
		return api.BadInput("previous session cannot be the same session")
	}

	prev, err := svc.queries.GetSession(ctx, db.GetSessionParams{
		ID:     prevUUID,
		CampID: campID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return api.BadInput("previous session not found")
		}
		return fmt.Errorf("error looking up previous session: %w", err)
	}

	if prev.SeasonID != seasonID {
		return api.BadInput("previous session must belong to the same season")
	}

	return nil
}
