package timeslot

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
	"camp-scheduler/internal/staleness"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("time slot not found")

type Service struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	marker  *staleness.Marker
}

func NewService(queries *db.Queries, pool *pgxpool.Pool, marker *staleness.Marker) *Service {
	return &Service{queries: queries, pool: pool, marker: marker}
}

func (svc *Service) List(ctx context.Context, campID string) ([]TimeSlotResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	slots, err := svc.queries.ListTimeSlots(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("error listing time slots: %w", err)
	}

	result := make([]TimeSlotResponse, len(slots))
	for i, s := range slots {
		result[i] = toTimeSlotResponse(s)
	}
	return result, nil
}

func (svc *Service) GetByID(ctx context.Context, campID, id string) (TimeSlotResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return TimeSlotResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return TimeSlotResponse{}, err
	}

	slot, err := svc.queries.GetTimeSlot(ctx, db.GetTimeSlotParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return TimeSlotResponse{}, fmt.Errorf("error getting time slot %s: %w", id, err)
	}

	return toTimeSlotResponse(slot), nil
}

func (svc *Service) Create(ctx context.Context, campID string, req CreateTimeSlotRequest) (TimeSlotResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return TimeSlotResponse{}, err
	}

	slot, err := svc.queries.CreateTimeSlot(ctx, db.CreateTimeSlotParams{
		CampID:       uid,
		TimeSlotName: req.Name,
	})
	if err != nil {
		return TimeSlotResponse{}, fmt.Errorf("error creating time slot: %w", err)
	}

	return toTimeSlotResponse(slot), nil
}

func (svc *Service) Update(ctx context.Context, campID, id string, req UpdateTimeSlotRequest) (TimeSlotResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return TimeSlotResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return TimeSlotResponse{}, err
	}

	slot, err := svc.queries.UpdateTimeSlot(ctx, db.UpdateTimeSlotParams{
		ID:           uid,
		CampID:       campUUID,
		TimeSlotName: req.Name,
	})
	if err != nil {
		return TimeSlotResponse{}, fmt.Errorf("error updating time slot %s: %w", id, err)
	}

	return toTimeSlotResponse(slot), nil
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
		return fmt.Errorf("error beginning delete time slot transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	rows, err := qtx.DeleteTimeSlot(ctx, db.DeleteTimeSlotParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting time slot %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	if err := svc.marker.MarkCamp(ctx, qtx, campUUID, []staleness.RunType{staleness.RunTypeActivity}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing delete time slot transaction: %w", err)
	}

	return nil
}

func toTimeSlotResponse(s db.TimeSlot) TimeSlotResponse {
	return TimeSlotResponse{
		ID:     api.UUIDToString(s.ID),
		CampID: api.UUIDToString(s.CampID),
		Name:   s.TimeSlotName,
	}
}
