package cabin

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
	"camp-scheduler/internal/staleness"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("cabin not found")

type Service struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	marker  *staleness.Marker
}

func NewService(queries *db.Queries, pool *pgxpool.Pool, marker *staleness.Marker) *Service {
	return &Service{queries: queries, pool: pool, marker: marker}
}

func (svc *Service) List(ctx context.Context, campID string) ([]CabinResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	cabins, err := svc.queries.ListCabins(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("error listing cabins: %w", err)
	}

	result := make([]CabinResponse, len(cabins))
	for i, c := range cabins {
		result[i] = CabinResponse{
			ID:                        api.UUIDToString(c.ID),
			CampID:                    api.UUIDToString(c.CampID),
			DefaultAgeGroupID:         api.UUIDToString(c.DefaultAgeGroupID),
			DefaultAgeGroupName:       c.DefaultAgeGroupName,
			Name:                      c.CabinName,
			DefaultGroupSize:          c.DefaultGroupSize,
			DefaultRequiredCounselors: c.DefaultRequiredCounselors,
			Gender:                    c.Gender,
			Archived:                  c.Archived,
		}
	}
	return result, nil
}

func (svc *Service) ListArchived(ctx context.Context, campID string) ([]CabinResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	cabins, err := svc.queries.ListArchivedCabins(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("error listing archived cabins: %w", err)
	}

	result := make([]CabinResponse, len(cabins))
	for i, c := range cabins {
		result[i] = CabinResponse{
			ID:                        api.UUIDToString(c.ID),
			CampID:                    api.UUIDToString(c.CampID),
			DefaultAgeGroupID:         api.UUIDToString(c.DefaultAgeGroupID),
			DefaultAgeGroupName:       c.DefaultAgeGroupName,
			Name:                      c.CabinName,
			DefaultGroupSize:          c.DefaultGroupSize,
			DefaultRequiredCounselors: c.DefaultRequiredCounselors,
			Gender:                    c.Gender,
			Archived:                  c.Archived,
		}
	}
	return result, nil
}

func (svc *Service) GetByID(ctx context.Context, campID, id string) (CabinResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return CabinResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return CabinResponse{}, err
	}

	cabin, err := svc.queries.GetCabin(ctx, db.GetCabinParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return CabinResponse{}, fmt.Errorf("error getting cabin %s: %w", id, err)
	}

	return CabinResponse{
		ID:                        api.UUIDToString(cabin.ID),
		CampID:                    api.UUIDToString(cabin.CampID),
		DefaultAgeGroupID:         api.UUIDToString(cabin.DefaultAgeGroupID),
		DefaultAgeGroupName:       cabin.DefaultAgeGroupName,
		Name:                      cabin.CabinName,
		DefaultGroupSize:          cabin.DefaultGroupSize,
		DefaultRequiredCounselors: cabin.DefaultRequiredCounselors,
		Gender:                    cabin.Gender,
		Archived:                  cabin.Archived,
	}, nil
}

func (svc *Service) Create(ctx context.Context, campID string, req CreateCabinRequest) (CabinResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return CabinResponse{}, err
	}

	ageGroupUUID, err := api.ParseUUID(req.DefaultAgeGroupID)
	if err != nil {
		return CabinResponse{}, err
	}

	cabin, err := svc.queries.CreateCabin(ctx, db.CreateCabinParams{
		CampID:                    campUUID,
		DefaultAgeGroupID:         ageGroupUUID,
		CabinName:                 req.Name,
		DefaultGroupSize:          req.DefaultGroupSize,
		DefaultRequiredCounselors: req.DefaultRequiredCounselors,
		Gender:                    req.Gender,
	})
	if err != nil {
		return CabinResponse{}, fmt.Errorf("error creating cabin: %w", err)
	}

	return CabinResponse{
		ID:                        api.UUIDToString(cabin.ID),
		CampID:                    api.UUIDToString(cabin.CampID),
		DefaultAgeGroupID:         api.UUIDToString(cabin.DefaultAgeGroupID),
		DefaultAgeGroupName:       cabin.DefaultAgeGroupName,
		Name:                      cabin.CabinName,
		DefaultGroupSize:          cabin.DefaultGroupSize,
		DefaultRequiredCounselors: cabin.DefaultRequiredCounselors,
		Gender:                    cabin.Gender,
		Archived:                  cabin.Archived,
	}, nil
}

func (svc *Service) Update(ctx context.Context, campID, id string, req UpdateCabinRequest) (CabinResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return CabinResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return CabinResponse{}, err
	}

	ageGroupUUID, err := api.ParseUUID(req.DefaultAgeGroupID)
	if err != nil {
		return CabinResponse{}, err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return CabinResponse{}, fmt.Errorf("error beginning update cabin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	cabin, err := qtx.UpdateCabin(ctx, db.UpdateCabinParams{
		ID:                        uid,
		CampID:                    campUUID,
		DefaultAgeGroupID:         ageGroupUUID,
		CabinName:                 req.Name,
		DefaultGroupSize:          req.DefaultGroupSize,
		DefaultRequiredCounselors: req.DefaultRequiredCounselors,
		Gender:                    req.Gender,
	})
	if err != nil {
		return CabinResponse{}, fmt.Errorf("error updating cabin %s: %w", id, err)
	}

	if err := svc.marker.MarkCamp(ctx, qtx, campUUID, []staleness.RunType{staleness.RunTypeCabin}); err != nil {
		return CabinResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return CabinResponse{}, fmt.Errorf("error committing update cabin transaction: %w", err)
	}

	return CabinResponse{
		ID:                        api.UUIDToString(cabin.ID),
		CampID:                    api.UUIDToString(cabin.CampID),
		DefaultAgeGroupID:         api.UUIDToString(cabin.DefaultAgeGroupID),
		DefaultAgeGroupName:       cabin.DefaultAgeGroupName,
		Name:                      cabin.CabinName,
		DefaultGroupSize:          cabin.DefaultGroupSize,
		DefaultRequiredCounselors: cabin.DefaultRequiredCounselors,
		Gender:                    cabin.Gender,
		Archived:                  cabin.Archived,
	}, nil
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
		return fmt.Errorf("error beginning delete cabin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	rows, err := qtx.DeleteCabin(ctx, db.DeleteCabinParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting cabin %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	if err := svc.marker.MarkCamp(ctx, qtx, campUUID, []staleness.RunType{staleness.RunTypeCabin}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing delete cabin transaction: %w", err)
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
		return fmt.Errorf("error beginning archive cabin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	var rows int64
	if archived {
		rows, err = qtx.ArchiveCabin(ctx, db.ArchiveCabinParams{ID: uid, CampID: campUUID})
	} else {
		rows, err = qtx.UnarchiveCabin(ctx, db.UnarchiveCabinParams{ID: uid, CampID: campUUID})
	}
	if err != nil {
		return fmt.Errorf("error setting cabin %s archived=%t: %w", id, archived, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	if err := svc.marker.MarkCamp(ctx, qtx, campUUID, []staleness.RunType{staleness.RunTypeCabin}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing archive cabin transaction: %w", err)
	}

	return nil
}
