package preferences

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
)

var ErrCounselorCertificationNotFound = errors.New("counselor certification not found")

type CounselorCertificationService struct {
	queries *db.Queries
}

func NewCounselorCertificationService(queries *db.Queries) *CounselorCertificationService {
	return &CounselorCertificationService{queries: queries}
}

func (svc *CounselorCertificationService) List(ctx context.Context, campID, counselorID string) ([]CounselorCertificationResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	counselorUUID, err := api.ParseUUID(counselorID)
	if err != nil {
		return nil, err
	}

	rows, err := svc.queries.ListCounselorCertifications(ctx, db.ListCounselorCertificationsParams{
		CounselorID: counselorUUID,
		CampID:      campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing counselor certifications: %w", err)
	}

	result := make([]CounselorCertificationResponse, len(rows))
	for i, r := range rows {
		result[i] = CounselorCertificationResponse{
			ID:                api.UUIDToString(r.ID),
			CampID:            api.UUIDToString(r.CampID),
			CounselorID:       api.UUIDToString(r.CounselorID),
			CertificationID:   api.UUIDToString(r.CertificationID),
			CertificationName: r.CertificationName,
		}
	}
	return result, nil
}

func (svc *CounselorCertificationService) Add(ctx context.Context, campID, counselorID string, req AddCounselorCertificationRequest) (CounselorCertificationResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return CounselorCertificationResponse{}, err
	}

	counselorUUID, err := api.ParseUUID(counselorID)
	if err != nil {
		return CounselorCertificationResponse{}, err
	}

	certUUID, err := api.ParseUUID(req.CertificationID)
	if err != nil {
		return CounselorCertificationResponse{}, err
	}

	cc, err := svc.queries.CreateCounselorCertification(ctx, db.CreateCounselorCertificationParams{
		CampID:          campUUID,
		CounselorID:     counselorUUID,
		CertificationID: certUUID,
	})
	if err != nil {
		return CounselorCertificationResponse{}, fmt.Errorf("error adding counselor certification: %w", err)
	}

	return CounselorCertificationResponse{
		ID:              api.UUIDToString(cc.ID),
		CampID:          api.UUIDToString(cc.CampID),
		CounselorID:     api.UUIDToString(cc.CounselorID),
		CertificationID: api.UUIDToString(cc.CertificationID),
	}, nil
}

func (svc *CounselorCertificationService) Remove(ctx context.Context, campID, counselorID, id string) error {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return err
	}

	counselorUUID, err := api.ParseUUID(counselorID)
	if err != nil {
		return err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	rows, err := svc.queries.DeleteCounselorCertification(ctx, db.DeleteCounselorCertificationParams{
		ID:          uid,
		CampID:      campUUID,
		CounselorID: counselorUUID,
	})
	if err != nil {
		return fmt.Errorf("error removing counselor certification: %w", err)
	}
	if rows == 0 {
		return ErrCounselorCertificationNotFound
	}

	return nil
}
