package agegroup

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
	"camp-scheduler/internal/staleness"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("age group not found")

type Service struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	marker  *staleness.Marker
}

func NewService(queries *db.Queries, pool *pgxpool.Pool, marker *staleness.Marker) *Service {
	return &Service{queries: queries, pool: pool, marker: marker}
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

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error beginning delete age group transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	rows, err := qtx.DeleteAgeGroup(ctx, db.DeleteAgeGroupParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting age group %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	if err := svc.marker.MarkCamp(ctx, qtx, campUUID, []staleness.RunType{staleness.RunTypeCabin}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing delete age group transaction: %w", err)
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
