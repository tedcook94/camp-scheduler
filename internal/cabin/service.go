package cabin

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
)

var ErrNotFound = errors.New("cabin not found")

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
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
			ID:                  api.UUIDToString(c.ID),
			CampID:              api.UUIDToString(c.CampID),
			DefaultAgeGroupID:   api.UUIDToString(c.DefaultAgeGroupID),
			DefaultAgeGroupName: c.DefaultAgeGroupName,
			Name:                c.CabinName,
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
		ID:                  api.UUIDToString(cabin.ID),
		CampID:              api.UUIDToString(cabin.CampID),
		DefaultAgeGroupID:   api.UUIDToString(cabin.DefaultAgeGroupID),
		DefaultAgeGroupName: cabin.DefaultAgeGroupName,
		Name:                cabin.CabinName,
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
		CampID:            campUUID,
		DefaultAgeGroupID: ageGroupUUID,
		CabinName:         req.Name,
	})
	if err != nil {
		return CabinResponse{}, fmt.Errorf("error creating cabin: %w", err)
	}

	return CabinResponse{
		ID:                  api.UUIDToString(cabin.ID),
		CampID:              api.UUIDToString(cabin.CampID),
		DefaultAgeGroupID:   api.UUIDToString(cabin.DefaultAgeGroupID),
		DefaultAgeGroupName: cabin.DefaultAgeGroupName,
		Name:                cabin.CabinName,
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

	cabin, err := svc.queries.UpdateCabin(ctx, db.UpdateCabinParams{
		ID:                uid,
		CampID:            campUUID,
		DefaultAgeGroupID: ageGroupUUID,
		CabinName:         req.Name,
	})
	if err != nil {
		return CabinResponse{}, fmt.Errorf("error updating cabin %s: %w", id, err)
	}

	return CabinResponse{
		ID:                  api.UUIDToString(cabin.ID),
		CampID:              api.UUIDToString(cabin.CampID),
		DefaultAgeGroupID:   api.UUIDToString(cabin.DefaultAgeGroupID),
		DefaultAgeGroupName: cabin.DefaultAgeGroupName,
		Name:                cabin.CabinName,
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

	rows, err := svc.queries.DeleteCabin(ctx, db.DeleteCabinParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting cabin %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}
