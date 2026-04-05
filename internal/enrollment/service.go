package enrollment

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
)

var ErrNotFound = errors.New("enrollment not found")

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

func (svc *Service) ListBySession(ctx context.Context, campID, sessionID string) ([]EnrollmentResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return nil, err
	}

	rows, err := svc.queries.ListSessionEnrollments(ctx, db.ListSessionEnrollmentsParams{
		SessionID: sessionUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing session enrollments: %w", err)
	}

	result := make([]EnrollmentResponse, len(rows))
	for i, r := range rows {
		result[i] = toEnrollmentResponseFromList(r)
	}
	return result, nil
}

func (svc *Service) GetByID(ctx context.Context, campID, sessionID, id string) (EnrollmentResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return EnrollmentResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return EnrollmentResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return EnrollmentResponse{}, err
	}

	row, err := svc.queries.GetSessionEnrollment(ctx, db.GetSessionEnrollmentParams{
		ID:        uid,
		CampID:    campUUID,
		SessionID: sessionUUID,
	})
	if err != nil {
		return EnrollmentResponse{}, fmt.Errorf("error getting enrollment %s: %w", id, err)
	}

	return toEnrollmentResponseFromGet(row), nil
}

func (svc *Service) Create(ctx context.Context, campID, sessionID string, req CreateEnrollmentRequest) (EnrollmentResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return EnrollmentResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return EnrollmentResponse{}, err
	}

	camperUUID, err := api.ParseUUID(req.CamperID)
	if err != nil {
		return EnrollmentResponse{}, err
	}

	sessionAgeGroupUUID, err := api.ParseUUID(req.SessionAgeGroupID)
	if err != nil {
		return EnrollmentResponse{}, err
	}

	// Verify the session age group belongs to this session.
	_, err = svc.queries.GetSessionAgeGroup(ctx, db.GetSessionAgeGroupParams{
		ID:        sessionAgeGroupUUID,
		CampID:    campUUID,
		SessionID: sessionUUID,
	})
	if err != nil {
		return EnrollmentResponse{}, fmt.Errorf("error validating session age group %s: %w", req.SessionAgeGroupID, err)
	}

	row, err := svc.queries.CreateSessionEnrollment(ctx, db.CreateSessionEnrollmentParams{
		CampID:            campUUID,
		CamperID:          camperUUID,
		SessionID:         sessionUUID,
		SessionAgeGroupID: sessionAgeGroupUUID,
	})
	if err != nil {
		return EnrollmentResponse{}, fmt.Errorf("error creating enrollment: %w", err)
	}

	return svc.GetByID(ctx, campID, sessionID, api.UUIDToString(row.ID))
}

func (svc *Service) Delete(ctx context.Context, campID, sessionID, id string) error {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	rows, err := svc.queries.DeleteSessionEnrollment(ctx, db.DeleteSessionEnrollmentParams{
		ID:        uid,
		CampID:    campUUID,
		SessionID: sessionUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting enrollment %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func toEnrollmentFields(id, campID, camperID, sessionAgeGroupID pgtype.UUID, camperName string, sessionID, ageGroupID pgtype.UUID) EnrollmentResponse {
	return EnrollmentResponse{
		ID:                api.UUIDToString(id),
		CampID:            api.UUIDToString(campID),
		CamperID:          api.UUIDToString(camperID),
		SessionAgeGroupID: api.UUIDToString(sessionAgeGroupID),
		CamperName:        camperName,
		SessionID:         api.UUIDToString(sessionID),
		AgeGroupID:        api.UUIDToString(ageGroupID),
	}
}

func toEnrollmentResponseFromList(r db.ListSessionEnrollmentsRow) EnrollmentResponse {
	return toEnrollmentFields(r.ID, r.CampID, r.CamperID, r.SessionAgeGroupID, r.CamperName, r.SessionID, r.AgeGroupID)
}

func toEnrollmentResponseFromGet(r db.GetSessionEnrollmentRow) EnrollmentResponse {
	return toEnrollmentFields(r.ID, r.CampID, r.CamperID, r.SessionAgeGroupID, r.CamperName, r.SessionID, r.AgeGroupID)
}
