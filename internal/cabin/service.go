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

func (s *Service) List(ctx context.Context, campID string) ([]CabinResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	cabins, err := s.queries.ListCabins(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("listing cabins: %w", err)
	}

	result := make([]CabinResponse, len(cabins))
	for i, c := range cabins {
		result[i] = toCabinResponse(c)
	}
	return result, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (CabinResponse, error) {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return CabinResponse{}, err
	}

	cabin, err := s.queries.GetCabin(ctx, uid)
	if err != nil {
		return CabinResponse{}, fmt.Errorf("getting cabin %s: %w", id, err)
	}

	return toCabinResponse(cabin), nil
}

func (s *Service) Create(ctx context.Context, campID string, req CreateCabinRequest) (CabinResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return CabinResponse{}, err
	}

	ageGroupUUID, err := api.ParseUUID(req.AgeGroupID)
	if err != nil {
		return CabinResponse{}, err
	}

	cabin, err := s.queries.CreateCabin(ctx, db.CreateCabinParams{
		CampID:     campUUID,
		AgeGroupID: ageGroupUUID,
		CabinName:  req.Name,
	})
	if err != nil {
		return CabinResponse{}, fmt.Errorf("creating cabin: %w", err)
	}

	return toCabinResponse(cabin), nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateCabinRequest) (CabinResponse, error) {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return CabinResponse{}, err
	}

	ageGroupUUID, err := api.ParseUUID(req.AgeGroupID)
	if err != nil {
		return CabinResponse{}, err
	}

	cabin, err := s.queries.UpdateCabin(ctx, db.UpdateCabinParams{
		ID:         uid,
		AgeGroupID: ageGroupUUID,
		CabinName:  req.Name,
	})
	if err != nil {
		return CabinResponse{}, fmt.Errorf("updating cabin %s: %w", id, err)
	}

	return toCabinResponse(cabin), nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	rows, err := s.queries.DeleteCabin(ctx, uid)
	if err != nil {
		return fmt.Errorf("deleting cabin %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func toCabinResponse(c db.Cabin) CabinResponse {
	return CabinResponse{
		ID:         api.UUIDToString(c.ID),
		CampID:     api.UUIDToString(c.CampID),
		AgeGroupID: api.UUIDToString(c.AgeGroupID),
		Name:       c.CabinName,
	}
}
