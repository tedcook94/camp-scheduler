package sessionconfig

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
)

var (
	ErrAgeGroupNotFound = errors.New("session age group not found")
	ErrCabinNotFound    = errors.New("session cabin not found")
)

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
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

func (svc *Service) GetAgeGroup(ctx context.Context, campID, id string) (SessionAgeGroupResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionAgeGroupResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return SessionAgeGroupResponse{}, err
	}

	row, err := svc.queries.GetSessionAgeGroup(ctx, db.GetSessionAgeGroupParams{
		ID:     uid,
		CampID: campUUID,
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

	row, err := svc.queries.CreateSessionAgeGroup(ctx, db.CreateSessionAgeGroupParams{
		CampID:     campUUID,
		SessionID:  sessionUUID,
		AgeGroupID: ageGroupUUID,
		GroupSize:  api.ToPgInt4(req.GroupSize),
	})
	if err != nil {
		return SessionAgeGroupResponse{}, fmt.Errorf("error creating session age group: %w", err)
	}

	return toSessionAgeGroupResponse(row), nil
}

func (svc *Service) UpdateAgeGroup(ctx context.Context, campID, id string, req UpdateSessionAgeGroupRequest) (SessionAgeGroupResponse, error) {
	campUUID, err := api.ParseUUID(campID)
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

	row, err := svc.queries.UpdateSessionAgeGroup(ctx, db.UpdateSessionAgeGroupParams{
		ID:         uid,
		CampID:     campUUID,
		AgeGroupID: ageGroupUUID,
		GroupSize:  api.ToPgInt4(req.GroupSize),
	})
	if err != nil {
		return SessionAgeGroupResponse{}, fmt.Errorf("error updating session age group %s: %w", id, err)
	}

	return toSessionAgeGroupResponse(row), nil
}

func (svc *Service) DeleteAgeGroup(ctx context.Context, campID, id string) error {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	rows, err := svc.queries.DeleteSessionAgeGroup(ctx, db.DeleteSessionAgeGroupParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting session age group %s: %w", id, err)
	}
	if rows == 0 {
		return ErrAgeGroupNotFound
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

	ageGroups, err := svc.queries.ListSessionAgeGroups(ctx, db.ListSessionAgeGroupsParams{
		SessionID: sessionUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing session age groups: %w", err)
	}

	var result []SessionCabinResponse
	for _, ag := range ageGroups {
		rows, err := svc.queries.ListSessionAgeGroupCabins(ctx, db.ListSessionAgeGroupCabinsParams{
			SessionAgeGroupID: ag.ID,
			CampID:            campUUID,
		})
		if err != nil {
			return nil, fmt.Errorf("error listing session cabins: %w", err)
		}

		for _, r := range rows {
			result = append(result, toSessionCabinResponse(r))
		}
	}
	return result, nil
}

func (svc *Service) GetCabin(ctx context.Context, campID, id string) (SessionCabinResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionCabinResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return SessionCabinResponse{}, err
	}

	row, err := svc.queries.GetSessionAgeGroupCabin(ctx, db.GetSessionAgeGroupCabinParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return SessionCabinResponse{}, fmt.Errorf("error getting session cabin %s: %w", id, err)
	}

	return toSessionCabinResponse(row), nil
}

func (svc *Service) CreateCabin(ctx context.Context, campID string, req CreateSessionCabinRequest) (SessionCabinResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionCabinResponse{}, err
	}

	sessionAgeGroupUUID, err := api.ParseUUID(req.SessionAgeGroupID)
	if err != nil {
		return SessionCabinResponse{}, err
	}

	cabinUUID, err := api.ParseUUID(req.CabinID)
	if err != nil {
		return SessionCabinResponse{}, err
	}

	row, err := svc.queries.CreateSessionAgeGroupCabin(ctx, db.CreateSessionAgeGroupCabinParams{
		CampID:             campUUID,
		SessionAgeGroupID:  sessionAgeGroupUUID,
		CabinID:            cabinUUID,
		GroupSize:          api.ToPgInt4(req.GroupSize),
		RequiredCounselors: api.ToPgInt4(req.RequiredCounselors),
	})
	if err != nil {
		return SessionCabinResponse{}, fmt.Errorf("error creating session cabin: %w", err)
	}

	return toSessionCabinResponse(row), nil
}

func (svc *Service) UpdateCabin(ctx context.Context, campID, id string, req UpdateSessionCabinRequest) (SessionCabinResponse, error) {
	campUUID, err := api.ParseUUID(campID)
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

	row, err := svc.queries.UpdateSessionAgeGroupCabin(ctx, db.UpdateSessionAgeGroupCabinParams{
		ID:                 uid,
		CampID:             campUUID,
		CabinID:            cabinUUID,
		GroupSize:          api.ToPgInt4(req.GroupSize),
		RequiredCounselors: api.ToPgInt4(req.RequiredCounselors),
	})
	if err != nil {
		return SessionCabinResponse{}, fmt.Errorf("error updating session cabin %s: %w", id, err)
	}

	return toSessionCabinResponse(row), nil
}

func (svc *Service) DeleteCabin(ctx context.Context, campID, id string) error {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	rows, err := svc.queries.DeleteSessionAgeGroupCabin(ctx, db.DeleteSessionAgeGroupCabinParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting session cabin %s: %w", id, err)
	}
	if rows == 0 {
		return ErrCabinNotFound
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
		GroupSize:  api.FromPgInt4(r.GroupSize),
	}
}

func toSessionCabinResponse(r db.SessionAgeGroupCabin) SessionCabinResponse {
	return SessionCabinResponse{
		ID:                 api.UUIDToString(r.ID),
		CampID:             api.UUIDToString(r.CampID),
		SessionAgeGroupID:  api.UUIDToString(r.SessionAgeGroupID),
		CabinID:            api.UUIDToString(r.CabinID),
		GroupSize:          api.FromPgInt4(r.GroupSize),
		RequiredCounselors: api.FromPgInt4(r.RequiredCounselors),
	}
}
