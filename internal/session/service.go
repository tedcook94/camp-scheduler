package session

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
)

var ErrNotFound = errors.New("session not found")

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

func (s *Service) List(ctx context.Context, campID string) ([]SessionResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	sessions, err := s.queries.ListSessions(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}

	result := make([]SessionResponse, len(sessions))
	for i, sn := range sessions {
		result[i] = toSessionResponse(sn)
	}
	return result, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (SessionResponse, error) {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return SessionResponse{}, err
	}

	session, err := s.queries.GetSession(ctx, uid)
	if err != nil {
		return SessionResponse{}, fmt.Errorf("getting session %s: %w", id, err)
	}

	return toSessionResponse(session), nil
}

func (s *Service) Create(ctx context.Context, campID string, req CreateSessionRequest) (SessionResponse, error) {
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

	session, err := s.queries.CreateSession(ctx, db.CreateSessionParams{
		CampID:          campUUID,
		SeasonID:        seasonUUID,
		SessionName:     req.Name,
		PreviousSession: prevUUID,
	})
	if err != nil {
		return SessionResponse{}, fmt.Errorf("creating session: %w", err)
	}

	return toSessionResponse(session), nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateSessionRequest) (SessionResponse, error) {
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

	session, err := s.queries.UpdateSession(ctx, db.UpdateSessionParams{
		ID:              uid,
		SeasonID:        seasonUUID,
		SessionName:     req.Name,
		PreviousSession: prevUUID,
	})
	if err != nil {
		return SessionResponse{}, fmt.Errorf("updating session %s: %w", id, err)
	}

	return toSessionResponse(session), nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	rows, err := s.queries.DeleteSession(ctx, uid)
	if err != nil {
		return fmt.Errorf("deleting session %s: %w", id, err)
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
