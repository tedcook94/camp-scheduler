package sessionconfig

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
	"camp-scheduler/internal/staleness"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrTimeSlotNotFound        = errors.New("session time slot not found")
	ErrSessionActivityNotFound = errors.New("session activity not found")
	ErrInvalidCounselorCount   = fmt.Errorf("required_counselors must not exceed capacity: %w", api.ErrBadInput)
)

type ActivityService struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	marker  *staleness.Marker
}

func NewActivityService(queries *db.Queries, pool *pgxpool.Pool, marker *staleness.Marker) *ActivityService {
	return &ActivityService{queries: queries, pool: pool, marker: marker}
}

func (svc *ActivityService) markActivityStale(ctx context.Context, q *db.Queries, campID, sessionID pgtype.UUID) error {
	return svc.marker.MarkSessions(ctx, q, campID, []pgtype.UUID{sessionID}, []staleness.RunType{staleness.RunTypeActivity})
}

// ensureTimeSlotActive rejects attaching an archived time slot to a session.
// The solver ignores archived rows, so silently allowing them would create
// dead session config.
func (svc *ActivityService) ensureTimeSlotActive(ctx context.Context, q *db.Queries, campID, timeSlotID pgtype.UUID) error {
	ts, err := q.GetTimeSlot(ctx, db.GetTimeSlotParams{
		ID:     timeSlotID,
		CampID: campID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return api.BadInput("time slot not found")
		}
		return fmt.Errorf("error looking up time slot: %w", err)
	}
	if ts.Archived {
		return api.BadInput("cannot use an archived time slot")
	}
	return nil
}

func (svc *ActivityService) ensureActivityActive(ctx context.Context, q *db.Queries, campID, activityID pgtype.UUID) error {
	a, err := q.GetActivity(ctx, db.GetActivityParams{
		ID:     activityID,
		CampID: campID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return api.BadInput("activity not found")
		}
		return fmt.Errorf("error looking up activity: %w", err)
	}
	if a.Archived {
		return api.BadInput("cannot use an archived activity")
	}
	return nil
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

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return SessionTimeSlotResponse{}, fmt.Errorf("error beginning create session time slot transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	if err := svc.ensureTimeSlotActive(ctx, qtx, campUUID, timeSlotUUID); err != nil {
		return SessionTimeSlotResponse{}, err
	}

	row, err := qtx.CreateSessionTimeSlot(ctx, db.CreateSessionTimeSlotParams{
		CampID:     campUUID,
		SessionID:  sessionUUID,
		TimeSlotID: timeSlotUUID,
		SortOrder:  req.SortOrder,
	})
	if err != nil {
		return SessionTimeSlotResponse{}, fmt.Errorf("error creating session time slot: %w", err)
	}

	if err := svc.markActivityStale(ctx, qtx, campUUID, sessionUUID); err != nil {
		return SessionTimeSlotResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return SessionTimeSlotResponse{}, fmt.Errorf("error committing create session time slot transaction: %w", err)
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

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return SessionTimeSlotResponse{}, fmt.Errorf("error beginning update session time slot transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	if err := svc.ensureTimeSlotActive(ctx, qtx, campUUID, timeSlotUUID); err != nil {
		return SessionTimeSlotResponse{}, err
	}

	row, err := qtx.UpdateSessionTimeSlot(ctx, db.UpdateSessionTimeSlotParams{
		ID:         uid,
		CampID:     campUUID,
		SessionID:  sessionUUID,
		TimeSlotID: timeSlotUUID,
		SortOrder:  req.SortOrder,
	})
	if err != nil {
		return SessionTimeSlotResponse{}, fmt.Errorf("error updating session time slot %s: %w", id, err)
	}

	if err := svc.markActivityStale(ctx, qtx, campUUID, sessionUUID); err != nil {
		return SessionTimeSlotResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return SessionTimeSlotResponse{}, fmt.Errorf("error committing update session time slot transaction: %w", err)
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

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error beginning delete session time slot transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	rows, err := qtx.DeleteSessionTimeSlot(ctx, db.DeleteSessionTimeSlotParams{
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

	if err := svc.markActivityStale(ctx, qtx, campUUID, sessionUUID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing delete session time slot transaction: %w", err)
	}

	return nil
}

func (svc *ActivityService) ReorderTimeSlots(ctx context.Context, campID, sessionID string, orderedIDs []string) error {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return err
	}

	if len(orderedIDs) == 0 {
		return api.BadInput("ordered_ids must not be empty")
	}

	existing, err := svc.queries.ListSessionTimeSlots(ctx, db.ListSessionTimeSlotsParams{
		CampID:    campUUID,
		SessionID: sessionUUID,
	})
	if err != nil {
		return fmt.Errorf("error listing session time slots: %w", err)
	}

	existingIDs := make(map[string]struct{}, len(existing))
	for _, st := range existing {
		existingIDs[st.ID.String()] = struct{}{}
	}

	if len(orderedIDs) != len(existing) {
		return api.BadInput("ordered_ids must contain exactly all time slots for the session")
	}

	seen := make(map[string]struct{}, len(orderedIDs))
	for _, id := range orderedIDs {
		if _, dup := seen[id]; dup {
			return api.BadInput("ordered_ids contains duplicate: " + id)
		}
		seen[id] = struct{}{}
		if _, ok := existingIDs[id]; !ok {
			return api.BadInput("ordered_ids contains invalid time slot: " + id)
		}
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := svc.queries.WithTx(tx)

	for i, id := range orderedIDs {
		uid, err := api.ParseUUID(id)
		if err != nil {
			return err
		}
		err = qtx.UpdateSessionTimeSlotSortOrder(ctx, db.UpdateSessionTimeSlotSortOrderParams{
			ID:        uid,
			CampID:    campUUID,
			SessionID: sessionUUID,
			SortOrder: int32(i + 1),
		})
		if err != nil {
			return fmt.Errorf("error updating sort order for %s: %w", id, err)
		}
	}

	if err := svc.markActivityStale(ctx, qtx, campUUID, sessionUUID); err != nil {
		return err
	}

	return tx.Commit(ctx)
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

// ListAllActivities returns all session activities across all time slots for a session.
func (svc *ActivityService) ListAllActivities(ctx context.Context, campID, sessionID string) ([]SessionActivityResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return nil, err
	}

	rows, err := svc.queries.ListSessionActivities(ctx, db.ListSessionActivitiesParams{
		SessionID: sessionUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing all session activities: %w", err)
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

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return SessionActivityResponse{}, fmt.Errorf("error beginning create session activity transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	// Verify the session time slot belongs to this session.
	_, err = qtx.GetSessionTimeSlot(ctx, db.GetSessionTimeSlotParams{
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

	if err := svc.ensureActivityActive(ctx, qtx, campUUID, activityUUID); err != nil {
		return SessionActivityResponse{}, err
	}

	if req.RequiredCounselors > req.Capacity {
		return SessionActivityResponse{}, ErrInvalidCounselorCount
	}

	row, err := qtx.CreateSessionActivity(ctx, db.CreateSessionActivityParams{
		CampID:             campUUID,
		SessionTimeSlotID:  timeSlotUUID,
		ActivityID:         activityUUID,
		Capacity:           req.Capacity,
		RequiredCounselors: req.RequiredCounselors,
	})
	if err != nil {
		return SessionActivityResponse{}, fmt.Errorf("error creating session activity: %w", err)
	}

	if err := svc.markActivityStale(ctx, qtx, campUUID, sessionUUID); err != nil {
		return SessionActivityResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return SessionActivityResponse{}, fmt.Errorf("error committing create session activity transaction: %w", err)
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

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return SessionActivityResponse{}, fmt.Errorf("error beginning update session activity transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	if err := svc.ensureActivityActive(ctx, qtx, campUUID, activityUUID); err != nil {
		return SessionActivityResponse{}, err
	}

	row, err := qtx.UpdateSessionActivity(ctx, db.UpdateSessionActivityParams{
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

	if err := svc.markActivityStale(ctx, qtx, campUUID, sessionUUID); err != nil {
		return SessionActivityResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return SessionActivityResponse{}, fmt.Errorf("error committing update session activity transaction: %w", err)
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

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error beginning delete session activity transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	rows, err := qtx.DeleteSessionActivity(ctx, db.DeleteSessionActivityParams{
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

	if err := svc.markActivityStale(ctx, qtx, campUUID, sessionUUID); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing delete session activity transaction: %w", err)
	}

	return nil
}

func (svc *ActivityService) CopyActivities(ctx context.Context, campID, sessionID, targetTimeSlotID, sourceTimeSlotID string) ([]SessionActivityResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return nil, err
	}

	targetUUID, err := api.ParseUUID(targetTimeSlotID)
	if err != nil {
		return nil, err
	}

	sourceUUID, err := api.ParseUUID(sourceTimeSlotID)
	if err != nil {
		return nil, err
	}

	if targetUUID == sourceUUID {
		return nil, api.BadInput("source and target time slots must be different")
	}

	// Verify both time slots belong to this session.
	_, err = svc.queries.GetSessionTimeSlot(ctx, db.GetSessionTimeSlotParams{
		ID:        targetUUID,
		CampID:    campUUID,
		SessionID: sessionUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error validating target session time slot %s: %w", targetTimeSlotID, err)
	}

	_, err = svc.queries.GetSessionTimeSlot(ctx, db.GetSessionTimeSlotParams{
		ID:        sourceUUID,
		CampID:    campUUID,
		SessionID: sessionUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error validating source session time slot %s: %w", sourceTimeSlotID, err)
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := svc.queries.WithTx(tx)

	_, err = qtx.DeleteSessionActivitiesByTimeSlot(ctx, db.DeleteSessionActivitiesByTimeSlotParams{
		SessionTimeSlotID: targetUUID,
		CampID:            campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error clearing target time slot activities: %w", err)
	}

	sourceActivities, err := qtx.ListSessionActivitiesByTimeSlot(ctx, db.ListSessionActivitiesByTimeSlotParams{
		SessionTimeSlotID: sourceUUID,
		CampID:            campUUID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing source activities: %w", err)
	}

	result := make([]SessionActivityResponse, 0, len(sourceActivities))
	for _, sa := range sourceActivities {
		created, err := qtx.CreateSessionActivity(ctx, db.CreateSessionActivityParams{
			CampID:             campUUID,
			SessionTimeSlotID:  targetUUID,
			ActivityID:         sa.ActivityID,
			Capacity:           sa.Capacity,
			RequiredCounselors: sa.RequiredCounselors,
		})
		if err != nil {
			return nil, fmt.Errorf("error copying activity: %w", err)
		}
		result = append(result, toSessionActivityResponse(created))
	}

	if err := svc.markActivityStale(ctx, qtx, campUUID, sessionUUID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("error committing transaction: %w", err)
	}

	return result, nil
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
