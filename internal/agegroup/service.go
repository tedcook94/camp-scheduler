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

func (svc *Service) ListArchived(ctx context.Context, campID string) ([]AgeGroupResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	groups, err := svc.queries.ListArchivedAgeGroups(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("error listing archived age groups: %w", err)
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
		ID:       api.UUIDToString(g.ID),
		CampID:   api.UUIDToString(g.CampID),
		Name:     g.AgeGroupName,
		Archived: g.Archived,
	}
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
		return fmt.Errorf("error beginning archive age group transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	var rows int64
	if archived {
		rows, err = qtx.ArchiveAgeGroup(ctx, db.ArchiveAgeGroupParams{ID: uid, CampID: campUUID})
	} else {
		rows, err = qtx.UnarchiveAgeGroup(ctx, db.UnarchiveAgeGroupParams{ID: uid, CampID: campUUID})
	}
	if err != nil {
		return fmt.Errorf("error setting age group %s archived=%t: %w", id, archived, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	if err := svc.marker.MarkCamp(ctx, qtx, campUUID, []staleness.RunType{staleness.RunTypeCabin}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing archive age group transaction: %w", err)
	}

	return nil
}
