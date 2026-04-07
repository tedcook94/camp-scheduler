package certification

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
)

var ErrNotFound = errors.New("certification not found")

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

func (svc *Service) List(ctx context.Context, campID string) ([]CertificationResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	certs, err := svc.queries.ListCertifications(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("error listing certifications: %w", err)
	}

	result := make([]CertificationResponse, len(certs))
	for i, c := range certs {
		result[i] = toCertificationResponse(c)
	}
	return result, nil
}

func (svc *Service) GetByID(ctx context.Context, campID, id string) (CertificationResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return CertificationResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return CertificationResponse{}, err
	}

	cert, err := svc.queries.GetCertification(ctx, db.GetCertificationParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return CertificationResponse{}, fmt.Errorf("error getting certification %s: %w", id, err)
	}

	return toCertificationResponse(cert), nil
}

func (svc *Service) Create(ctx context.Context, campID string, req CreateCertificationRequest) (CertificationResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return CertificationResponse{}, err
	}

	cert, err := svc.queries.CreateCertification(ctx, db.CreateCertificationParams{
		CampID:            uid,
		CertificationName: req.Name,
	})
	if err != nil {
		return CertificationResponse{}, fmt.Errorf("error creating certification: %w", err)
	}

	return toCertificationResponse(cert), nil
}

func (svc *Service) Update(ctx context.Context, campID, id string, req UpdateCertificationRequest) (CertificationResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return CertificationResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return CertificationResponse{}, err
	}

	cert, err := svc.queries.UpdateCertification(ctx, db.UpdateCertificationParams{
		ID:                uid,
		CampID:            campUUID,
		CertificationName: req.Name,
	})
	if err != nil {
		return CertificationResponse{}, fmt.Errorf("error updating certification %s: %w", id, err)
	}

	return toCertificationResponse(cert), nil
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

	rows, err := svc.queries.DeleteCertification(ctx, db.DeleteCertificationParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting certification %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func toCertificationResponse(c db.Certification) CertificationResponse {
	return CertificationResponse{
		ID:     api.UUIDToString(c.ID),
		CampID: api.UUIDToString(c.CampID),
		Name:   c.CertificationName,
	}
}
