package certification

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
	"camp-scheduler/internal/staleness"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("certification not found")

type Service struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	marker  *staleness.Marker
}

func NewService(queries *db.Queries, pool *pgxpool.Pool, marker *staleness.Marker) *Service {
	return &Service{queries: queries, pool: pool, marker: marker}
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

func (svc *Service) ListArchived(ctx context.Context, campID string) ([]CertificationResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	certs, err := svc.queries.ListArchivedCertifications(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("error listing archived certifications: %w", err)
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

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error beginning delete certification transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	rows, err := qtx.DeleteCertification(ctx, db.DeleteCertificationParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting certification %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	if err := svc.marker.MarkCamp(ctx, qtx, campUUID, []staleness.RunType{staleness.RunTypeActivity}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing delete certification transaction: %w", err)
	}

	return nil
}

func toCertificationResponse(c db.Certification) CertificationResponse {
	return CertificationResponse{
		ID:       api.UUIDToString(c.ID),
		CampID:   api.UUIDToString(c.CampID),
		Name:     c.CertificationName,
		Archived: c.Archived,
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
		return fmt.Errorf("error beginning archive certification transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	var rows int64
	if archived {
		rows, err = qtx.ArchiveCertification(ctx, db.ArchiveCertificationParams{ID: uid, CampID: campUUID})
	} else {
		rows, err = qtx.UnarchiveCertification(ctx, db.UnarchiveCertificationParams{ID: uid, CampID: campUUID})
	}
	if err != nil {
		return fmt.Errorf("error setting certification %s archived=%t: %w", id, archived, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	if err := svc.marker.MarkCamp(ctx, qtx, campUUID, []staleness.RunType{staleness.RunTypeActivity}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing archive certification transaction: %w", err)
	}

	return nil
}
