package counselor

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
)

var ErrNotFound = errors.New("counselor not found")

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

func (svc *Service) List(ctx context.Context, campID string) ([]CounselorResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	counselors, err := svc.queries.ListCounselors(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("listing counselors: %w", err)
	}

	result := make([]CounselorResponse, len(counselors))
	for i, c := range counselors {
		result[i] = toCounselorResponse(c)
	}
	return result, nil
}

func (svc *Service) GetByID(ctx context.Context, id string) (CounselorResponse, error) {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return CounselorResponse{}, err
	}

	counselor, err := svc.queries.GetCounselor(ctx, uid)
	if err != nil {
		return CounselorResponse{}, fmt.Errorf("getting counselor %s: %w", id, err)
	}

	return toCounselorResponse(counselor), nil
}

func (svc *Service) Create(ctx context.Context, campID string, req CreateCounselorRequest) (CounselorResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return CounselorResponse{}, err
	}

	counselor, err := svc.queries.CreateCounselor(ctx, db.CreateCounselorParams{
		CampID:          uid,
		CounselorName:   req.Name,
		JuniorCounselor: req.JuniorCounselor,
	})
	if err != nil {
		return CounselorResponse{}, fmt.Errorf("creating counselor: %w", err)
	}

	return toCounselorResponse(counselor), nil
}

func (svc *Service) Update(ctx context.Context, id string, req UpdateCounselorRequest) (CounselorResponse, error) {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return CounselorResponse{}, err
	}

	counselor, err := svc.queries.UpdateCounselor(ctx, db.UpdateCounselorParams{
		ID:               uid,
		CounselorName:    req.Name,
		JuniorCounselor:  req.JuniorCounselor,
		CounselorEnabled: req.Enabled,
	})
	if err != nil {
		return CounselorResponse{}, fmt.Errorf("updating counselor %s: %w", id, err)
	}

	return toCounselorResponse(counselor), nil
}

func (svc *Service) Delete(ctx context.Context, id string) error {
	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	rows, err := svc.queries.DeleteCounselor(ctx, uid)
	if err != nil {
		return fmt.Errorf("deleting counselor %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func toCounselorResponse(c db.Counselor) CounselorResponse {
	return CounselorResponse{
		ID:              api.UUIDToString(c.ID),
		CampID:          api.UUIDToString(c.CampID),
		Name:            c.CounselorName,
		JuniorCounselor: c.JuniorCounselor,
		Enabled:         c.CounselorEnabled,
	}
}
