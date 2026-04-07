package activity

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
)

var ErrNotFound = errors.New("activity not found")

type Service struct {
	queries *db.Queries
}

func NewService(queries *db.Queries) *Service {
	return &Service{queries: queries}
}

func (svc *Service) List(ctx context.Context, campID string) ([]ActivityResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	activities, err := svc.queries.ListActivities(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("error listing activities: %w", err)
	}

	result := make([]ActivityResponse, len(activities))
	for i, a := range activities {
		result[i] = toActivityResponse(a)
	}
	return result, nil
}

func (svc *Service) GetByID(ctx context.Context, campID, id string) (ActivityResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return ActivityResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return ActivityResponse{}, err
	}

	a, err := svc.queries.GetActivity(ctx, db.GetActivityParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return ActivityResponse{}, fmt.Errorf("error getting activity %s: %w", id, err)
	}

	return toActivityResponse(a), nil
}

func (svc *Service) Create(ctx context.Context, campID string, req CreateActivityRequest) (ActivityResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return ActivityResponse{}, err
	}

	a, err := svc.queries.CreateActivity(ctx, db.CreateActivityParams{
		CampID:       uid,
		ActivityName: req.Name,
	})
	if err != nil {
		return ActivityResponse{}, fmt.Errorf("error creating activity: %w", err)
	}

	return toActivityResponse(a), nil
}

func (svc *Service) Update(ctx context.Context, campID, id string, req UpdateActivityRequest) (ActivityResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return ActivityResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return ActivityResponse{}, err
	}

	a, err := svc.queries.UpdateActivity(ctx, db.UpdateActivityParams{
		ID:           uid,
		CampID:       campUUID,
		ActivityName: req.Name,
	})
	if err != nil {
		return ActivityResponse{}, fmt.Errorf("error updating activity %s: %w", id, err)
	}

	return toActivityResponse(a), nil
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

	rows, err := svc.queries.DeleteActivity(ctx, db.DeleteActivityParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting activity %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func (svc *Service) ListCertifications(ctx context.Context, campID, activityID string) ([]ActivityCertificationResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	activityUUID, err := api.ParseUUID(activityID)
	if err != nil {
		return nil, err
	}

	rows, err := svc.queries.ListActivityCertifications(ctx, db.ListActivityCertificationsParams{
		ActivityID: activityUUID,
		CampID:     campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing activity certifications: %w", err)
	}

	result := make([]ActivityCertificationResponse, len(rows))
	for i, r := range rows {
		result[i] = ActivityCertificationResponse{
			ID:                api.UUIDToString(r.ID),
			CampID:            api.UUIDToString(r.CampID),
			ActivityID:        api.UUIDToString(r.ActivityID),
			CertificationID:   api.UUIDToString(r.CertificationID),
			CertificationName: r.CertificationName,
		}
	}
	return result, nil
}

func (svc *Service) AddCertification(ctx context.Context, campID, activityID string, req AddCertificationRequest) (ActivityCertificationResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return ActivityCertificationResponse{}, err
	}

	activityUUID, err := api.ParseUUID(activityID)
	if err != nil {
		return ActivityCertificationResponse{}, err
	}

	certUUID, err := api.ParseUUID(req.CertificationID)
	if err != nil {
		return ActivityCertificationResponse{}, err
	}

	ac, err := svc.queries.CreateActivityCertification(ctx, db.CreateActivityCertificationParams{
		CampID:          campUUID,
		ActivityID:      activityUUID,
		CertificationID: certUUID,
	})
	if err != nil {
		return ActivityCertificationResponse{}, fmt.Errorf("error adding certification to activity: %w", err)
	}

	return ActivityCertificationResponse{
		ID:              api.UUIDToString(ac.ID),
		CampID:          api.UUIDToString(ac.CampID),
		ActivityID:      api.UUIDToString(ac.ActivityID),
		CertificationID: api.UUIDToString(ac.CertificationID),
	}, nil
}

func (svc *Service) RemoveCertification(ctx context.Context, campID, activityID, id string) error {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return err
	}

	activityUUID, err := api.ParseUUID(activityID)
	if err != nil {
		return err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	rows, err := svc.queries.DeleteActivityCertification(ctx, db.DeleteActivityCertificationParams{
		ID:         uid,
		CampID:     campUUID,
		ActivityID: activityUUID,
	})
	if err != nil {
		return fmt.Errorf("error removing certification from activity: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func toActivityResponse(a db.Activity) ActivityResponse {
	return ActivityResponse{
		ID:     api.UUIDToString(a.ID),
		CampID: api.UUIDToString(a.CampID),
		Name:   a.ActivityName,
	}
}
