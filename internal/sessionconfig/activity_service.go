package sessionconfig

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
)

var (
	ErrTimeSlotNotFound        = errors.New("session time slot not found")
	ErrSessionActivityNotFound = errors.New("session activity not found")
	ErrInvalidCounselorCount   = fmt.Errorf("required_counselors must not exceed capacity: %w", api.ErrBadInput)
)

type ActivityService struct {
	queries *db.Queries
}

func NewActivityService(queries *db.Queries) *ActivityService {
	return &ActivityService{queries: queries}
}

func (svc *ActivityService) ListTimeSlots(ctx context.Context, campID, sessionID string) ([]SessionTimeSlotResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return nil, err
	}

	rows, err := svc.queries.ListSessionTimeSlots(ctx, db.ListSessionTimeSlotsParams{
		SessionID: sessionUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing session time slots: %w", err)
	}

	result := make([]SessionTimeSlotResponse, len(rows))
	for i, r := range rows {
		result[i] = toSessionTimeSlotResponse(r)
	}
	return result, nil
}

func (svc *ActivityService) GetTimeSlot(ctx context.Context, campID, sessionID, id string) (SessionTimeSlotResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionTimeSlotResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return SessionTimeSlotResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return SessionTimeSlotResponse{}, err
	}

	row, err := svc.queries.GetSessionTimeSlot(ctx, db.GetSessionTimeSlotParams{
		ID:        uid,
		CampID:    campUUID,
		SessionID: sessionUUID,
	})
	if err != nil {
		return SessionTimeSlotResponse{}, fmt.Errorf("error getting session time slot %s: %w", id, err)
	}

	return toSessionTimeSlotResponse(row), nil
}

func (svc *ActivityService) CreateTimeSlot(ctx context.Context, campID, sessionID string, req CreateSessionTimeSlotRequest) (SessionTimeSlotResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionTimeSlotResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return SessionTimeSlotResponse{}, err
	}

	timeSlotUUID, err := api.ParseUUID(req.TimeSlotID)
	if err != nil {
		return SessionTimeSlotResponse{}, err
	}

	row, err := svc.queries.CreateSessionTimeSlot(ctx, db.CreateSessionTimeSlotParams{
		CampID:     campUUID,
		SessionID:  sessionUUID,
		TimeSlotID: timeSlotUUID,
		SortOrder:  req.SortOrder,
	})
	if err != nil {
		return SessionTimeSlotResponse{}, fmt.Errorf("error creating session time slot: %w", err)
	}

	return toSessionTimeSlotResponse(row), nil
}

func (svc *ActivityService) UpdateTimeSlot(ctx context.Context, campID, sessionID, id string, req UpdateSessionTimeSlotRequest) (SessionTimeSlotResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionTimeSlotResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return SessionTimeSlotResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return SessionTimeSlotResponse{}, err
	}

	timeSlotUUID, err := api.ParseUUID(req.TimeSlotID)
	if err != nil {
		return SessionTimeSlotResponse{}, err
	}

	row, err := svc.queries.UpdateSessionTimeSlot(ctx, db.UpdateSessionTimeSlotParams{
		ID:         uid,
		CampID:     campUUID,
		SessionID:  sessionUUID,
		TimeSlotID: timeSlotUUID,
		SortOrder:  req.SortOrder,
	})
	if err != nil {
		return SessionTimeSlotResponse{}, fmt.Errorf("error updating session time slot %s: %w", id, err)
	}

	return toSessionTimeSlotResponse(row), nil
}

func (svc *ActivityService) DeleteTimeSlot(ctx context.Context, campID, sessionID, id string) error {
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

	rows, err := svc.queries.DeleteSessionTimeSlot(ctx, db.DeleteSessionTimeSlotParams{
		ID:        uid,
		CampID:    campUUID,
		SessionID: sessionUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting session time slot %s: %w", id, err)
	}
	if rows == 0 {
		return ErrTimeSlotNotFound
	}

	return nil
}

func (svc *ActivityService) ListActivities(ctx context.Context, campID, sessionID, timeSlotID string) ([]SessionActivityResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return nil, err
	}

	timeSlotUUID, err := api.ParseUUID(timeSlotID)
	if err != nil {
		return nil, err
	}

	// Verify the session time slot belongs to this session.
	_, err = svc.queries.GetSessionTimeSlot(ctx, db.GetSessionTimeSlotParams{
		ID:        timeSlotUUID,
		CampID:    campUUID,
		SessionID: sessionUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error validating session time slot %s: %w", timeSlotID, err)
	}

	rows, err := svc.queries.ListSessionActivitiesByTimeSlot(ctx, db.ListSessionActivitiesByTimeSlotParams{
		SessionTimeSlotID: timeSlotUUID,
		CampID:            campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing session activities: %w", err)
	}

	result := make([]SessionActivityResponse, len(rows))
	for i, r := range rows {
		result[i] = toSessionActivityResponse(r)
	}
	return result, nil
}

func (svc *ActivityService) GetActivity(ctx context.Context, campID, sessionID, id string) (SessionActivityResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionActivityResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return SessionActivityResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return SessionActivityResponse{}, err
	}

	row, err := svc.queries.GetSessionActivity(ctx, db.GetSessionActivityParams{
		ID:        uid,
		CampID:    campUUID,
		SessionID: sessionUUID,
	})
	if err != nil {
		return SessionActivityResponse{}, fmt.Errorf("error getting session activity %s: %w", id, err)
	}

	return toSessionActivityResponse(row), nil
}

func (svc *ActivityService) CreateActivity(ctx context.Context, campID, sessionID, timeSlotID string, req CreateSessionActivityRequest) (SessionActivityResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionActivityResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return SessionActivityResponse{}, err
	}

	timeSlotUUID, err := api.ParseUUID(timeSlotID)
	if err != nil {
		return SessionActivityResponse{}, err
	}

	// Verify the session time slot belongs to this session.
	_, err = svc.queries.GetSessionTimeSlot(ctx, db.GetSessionTimeSlotParams{
		ID:        timeSlotUUID,
		CampID:    campUUID,
		SessionID: sessionUUID,
	})
	if err != nil {
		return SessionActivityResponse{}, fmt.Errorf("error validating session time slot %s: %w", timeSlotID, err)
	}

	activityUUID, err := api.ParseUUID(req.ActivityID)
	if err != nil {
		return SessionActivityResponse{}, err
	}

	if req.RequiredCounselors > req.Capacity {
		return SessionActivityResponse{}, ErrInvalidCounselorCount
	}

	row, err := svc.queries.CreateSessionActivity(ctx, db.CreateSessionActivityParams{
		CampID:             campUUID,
		SessionTimeSlotID:  timeSlotUUID,
		ActivityID:         activityUUID,
		Capacity:           req.Capacity,
		RequiredCounselors: req.RequiredCounselors,
	})
	if err != nil {
		return SessionActivityResponse{}, fmt.Errorf("error creating session activity: %w", err)
	}

	return toSessionActivityResponse(row), nil
}

func (svc *ActivityService) UpdateActivity(ctx context.Context, campID, sessionID, id string, req UpdateSessionActivityRequest) (SessionActivityResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionActivityResponse{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return SessionActivityResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return SessionActivityResponse{}, err
	}

	activityUUID, err := api.ParseUUID(req.ActivityID)
	if err != nil {
		return SessionActivityResponse{}, err
	}

	if req.RequiredCounselors > req.Capacity {
		return SessionActivityResponse{}, ErrInvalidCounselorCount
	}

	row, err := svc.queries.UpdateSessionActivity(ctx, db.UpdateSessionActivityParams{
		ID:                 uid,
		CampID:             campUUID,
		SessionID:          sessionUUID,
		ActivityID:         activityUUID,
		Capacity:           req.Capacity,
		RequiredCounselors: req.RequiredCounselors,
	})
	if err != nil {
		return SessionActivityResponse{}, fmt.Errorf("error updating session activity %s: %w", id, err)
	}

	return toSessionActivityResponse(row), nil
}

func (svc *ActivityService) DeleteActivity(ctx context.Context, campID, sessionID, id string) error {
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

	rows, err := svc.queries.DeleteSessionActivity(ctx, db.DeleteSessionActivityParams{
		ID:        uid,
		CampID:    campUUID,
		SessionID: sessionUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting session activity %s: %w", id, err)
	}
	if rows == 0 {
		return ErrSessionActivityNotFound
	}

	return nil
}

func toSessionTimeSlotResponse(r db.SessionTimeSlot) SessionTimeSlotResponse {
	return SessionTimeSlotResponse{
		ID:         api.UUIDToString(r.ID),
		CampID:     api.UUIDToString(r.CampID),
		SessionID:  api.UUIDToString(r.SessionID),
		TimeSlotID: api.UUIDToString(r.TimeSlotID),
		SortOrder:  r.SortOrder,
	}
}

func toSessionActivityResponse(r db.SessionActivity) SessionActivityResponse {
	return SessionActivityResponse{
		ID:                 api.UUIDToString(r.ID),
		CampID:             api.UUIDToString(r.CampID),
		SessionTimeSlotID:  api.UUIDToString(r.SessionTimeSlotID),
		ActivityID:         api.UUIDToString(r.ActivityID),
		Capacity:           r.Capacity,
		RequiredCounselors: r.RequiredCounselors,
	}
}
