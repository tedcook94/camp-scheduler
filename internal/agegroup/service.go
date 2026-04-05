package agegroup

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
)

var ErrNotFound = errors.New("age group not found")

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

func (svc *Service) List(ctx context.Context, campID string) ([]AgeGroupResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	groups, err := svc.queries.ListAgeGroups(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("error listing age groups: %w", err)
	}

	result := make([]AgeGroupResponse, len(groups))
	for i, g := range groups {
		result[i] = toAgeGroupResponse(g)
	}
	return result, nil
}

func (svc *Service) GetByID(ctx context.Context, campID, id string) (AgeGroupResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return AgeGroupResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return AgeGroupResponse{}, err
	}

	group, err := svc.queries.GetAgeGroup(ctx, db.GetAgeGroupParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return AgeGroupResponse{}, fmt.Errorf("error getting age group %s: %w", id, err)
	}

	return toAgeGroupResponse(group), nil
}

func (svc *Service) Create(ctx context.Context, campID string, req CreateAgeGroupRequest) (AgeGroupResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return AgeGroupResponse{}, err
	}

	group, err := svc.queries.CreateAgeGroup(ctx, db.CreateAgeGroupParams{
		CampID:       uid,
		AgeGroupName: req.Name,
	})
	if err != nil {
		return AgeGroupResponse{}, fmt.Errorf("error creating age group: %w", err)
	}

	return toAgeGroupResponse(group), nil
}

func (svc *Service) Update(ctx context.Context, campID, id string, req UpdateAgeGroupRequest) (AgeGroupResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return AgeGroupResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return AgeGroupResponse{}, err
	}

	group, err := svc.queries.UpdateAgeGroup(ctx, db.UpdateAgeGroupParams{
		ID:           uid,
		CampID:       campUUID,
		AgeGroupName: req.Name,
	})
	if err != nil {
		return AgeGroupResponse{}, fmt.Errorf("error updating age group %s: %w", id, err)
	}

	return toAgeGroupResponse(group), nil
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

	rows, err := svc.queries.DeleteAgeGroup(ctx, db.DeleteAgeGroupParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting age group %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func toAgeGroupResponse(g db.AgeGroup) AgeGroupResponse {
	return AgeGroupResponse{
		ID:     api.UUIDToString(g.ID),
		CampID: api.UUIDToString(g.CampID),
		Name:   g.AgeGroupName,
	}
}
