package history

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
)

var ErrSessionHistoryNotFound = errors.New("session history entry not found")

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
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
		return nil, fmt.Errorf("listing session history: %w", err)
	}

	result := make([]SessionHistoryResponse, len(entries))
	for i, e := range entries {
		result[i] = toSessionHistoryResponse(e)
	}
	return result, nil
}

func (svc *Service) GetByID(ctx context.Context, campID, id string) (SessionHistoryResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionHistoryResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return SessionHistoryResponse{}, err
	}

	entry, err := svc.queries.GetCounselorSessionHistoryEntry(ctx, db.GetCounselorSessionHistoryEntryParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return SessionHistoryResponse{}, fmt.Errorf("getting session history entry %s: %w", id, err)
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

	entry, err := svc.queries.CreateCounselorSessionHistoryEntry(ctx, db.CreateCounselorSessionHistoryEntryParams{
		CampID:      campUUID,
		CounselorID: counselorUUID,
		SessionID:   sessionUUID,
		AgeGroupID:  ageGroupUUID,
		CabinID:     cabinUUID,
	})
	if err != nil {
		return SessionHistoryResponse{}, fmt.Errorf("creating session history entry: %w", err)
	}

	return toSessionHistoryResponse(entry), nil
}

func (svc *Service) Update(ctx context.Context, campID, id string, req UpdateSessionHistoryRequest) (SessionHistoryResponse, error) {
	campUUID, err := api.ParseUUID(campID)
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

	entry, err := svc.queries.UpdateCounselorSessionHistoryEntry(ctx, db.UpdateCounselorSessionHistoryEntryParams{
		ID:         uid,
		CampID:     campUUID,
		SessionID:  sessionUUID,
		AgeGroupID: ageGroupUUID,
		CabinID:    cabinUUID,
	})
	if err != nil {
		return SessionHistoryResponse{}, fmt.Errorf("updating session history entry %s: %w", id, err)
	}

	return toSessionHistoryResponse(entry), nil
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

	rows, err := svc.queries.DeleteCounselorSessionHistoryEntry(ctx, db.DeleteCounselorSessionHistoryEntryParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return fmt.Errorf("deleting session history entry %s: %w", id, err)
	}
	if rows == 0 {
		return ErrSessionHistoryNotFound
	}

	return nil
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
