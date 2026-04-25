package preferences

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
	"camp-scheduler/internal/staleness"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrCounselorCertificationNotFound = errors.New("counselor certification not found")

type CounselorCertificationService struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	marker  *staleness.Marker
}

func NewCounselorCertificationService(queries *db.Queries, pool *pgxpool.Pool, marker *staleness.Marker) *CounselorCertificationService {
	return &CounselorCertificationService{queries: queries, pool: pool, marker: marker}
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

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return CounselorCertificationResponse{}, fmt.Errorf("error beginning add counselor certification transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	cc, err := qtx.CreateCounselorCertification(ctx, db.CreateCounselorCertificationParams{
		CampID:          campUUID,
		CounselorID:     counselorUUID,
		CertificationID: certUUID,
	})
	if err != nil {
		return CounselorCertificationResponse{}, fmt.Errorf("error adding counselor certification: %w", err)
	}

	if err := svc.marker.MarkSessionsForCounselor(ctx, qtx, campUUID, counselorUUID, []staleness.RunType{staleness.RunTypeActivity}); err != nil {
		return CounselorCertificationResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return CounselorCertificationResponse{}, fmt.Errorf("error committing add counselor certification transaction: %w", err)
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

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error beginning remove counselor certification transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	rows, err := qtx.DeleteCounselorCertification(ctx, db.DeleteCounselorCertificationParams{
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

	if err := svc.marker.MarkSessionsForCounselor(ctx, qtx, campUUID, counselorUUID, []staleness.RunType{staleness.RunTypeActivity}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing remove counselor certification transaction: %w", err)
	}

	return nil
}
