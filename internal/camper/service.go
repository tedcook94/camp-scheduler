package camper

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
)

var ErrNotFound = errors.New("camper not found")

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

// FullName joins first and last name into a display string. Empty parts are
// trimmed so single-token names render cleanly.
func FullName(first, last string) string {
	return strings.TrimSpace(first + " " + last)
}

func (svc *Service) List(ctx context.Context, campID string) ([]CamperResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	campers, err := svc.queries.ListCampers(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("error listing campers: %w", err)
	}

	result := make([]CamperResponse, len(campers))
	for i, c := range campers {
		result[i] = CamperResponse{
			ID:        api.UUIDToString(c.ID),
			CampID:    api.UUIDToString(c.CampID),
			FirstName: c.FirstName,
			LastName:  c.LastName,
			Name:      FullName(c.FirstName, c.LastName),
			Gender:    c.Gender,
		}
	}
	return result, nil
}

func (svc *Service) GetByID(ctx context.Context, campID, id string) (CamperResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return CamperResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return CamperResponse{}, err
	}

	c, err := svc.queries.GetCamper(ctx, db.GetCamperParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return CamperResponse{}, fmt.Errorf("error getting camper %s: %w", id, err)
	}

	return CamperResponse{
		ID:        api.UUIDToString(c.ID),
		CampID:    api.UUIDToString(c.CampID),
		FirstName: c.FirstName,
		LastName:  c.LastName,
		Name:      FullName(c.FirstName, c.LastName),
		Gender:    c.Gender,
	}, nil
}

func (svc *Service) Create(ctx context.Context, campID string, req CreateCamperRequest) (CamperResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return CamperResponse{}, err
	}

	c, err := svc.queries.CreateCamper(ctx, db.CreateCamperParams{
		CampID:    uid,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Gender:    req.Gender,
	})
	if err != nil {
		return CamperResponse{}, fmt.Errorf("error creating camper: %w", err)
	}

	return CamperResponse{
		ID:        api.UUIDToString(c.ID),
		CampID:    api.UUIDToString(c.CampID),
		FirstName: c.FirstName,
		LastName:  c.LastName,
		Name:      FullName(c.FirstName, c.LastName),
		Gender:    c.Gender,
	}, nil
}

func (svc *Service) Update(ctx context.Context, campID, id string, req UpdateCamperRequest) (CamperResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return CamperResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return CamperResponse{}, err
	}

	c, err := svc.queries.UpdateCamper(ctx, db.UpdateCamperParams{
		ID:        uid,
		CampID:    campUUID,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Gender:    req.Gender,
	})
	if err != nil {
		return CamperResponse{}, fmt.Errorf("error updating camper %s: %w", id, err)
	}

	return CamperResponse{
		ID:        api.UUIDToString(c.ID),
		CampID:    api.UUIDToString(c.CampID),
		FirstName: c.FirstName,
		LastName:  c.LastName,
		Name:      FullName(c.FirstName, c.LastName),
		Gender:    c.Gender,
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

	rows, err := svc.queries.DeleteCamper(ctx, db.DeleteCamperParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting camper %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}
