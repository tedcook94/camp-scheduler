package override

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

var ErrNotFound = errors.New("override not found")

type Service struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	marker  *staleness.Marker
}

func NewService(queries *db.Queries, pool *pgxpool.Pool, marker *staleness.Marker) *Service {
	return &Service{queries: queries, pool: pool, marker: marker}
}

// ListBySession returns all overrides for a session, grouped by subtype.
func (svc *Service) ListBySession(ctx context.Context, campID, sessionID string) (SessionOverridesResponse, error) {
	resp := SessionOverridesResponse{
		CounselorCabin:    []CounselorCabinOverrideResponse{},
		CamperCabin:       []CamperCabinOverrideResponse{},
		CounselorActivity: []CounselorActivityOverrideResponse{},
	}

	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return resp, err
	}
	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return resp, err
	}

	ccRows, err := svc.queries.ListCounselorCabinOverridesBySession(ctx, db.ListCounselorCabinOverridesBySessionParams{
		SessionID: sessionUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return resp, fmt.Errorf("error listing counselor-cabin overrides: %w", err)
	}
	for _, r := range ccRows {
		resp.CounselorCabin = append(resp.CounselorCabin, CounselorCabinOverrideResponse{
			ID:                     api.UUIDToString(r.ID),
			CampID:                 api.UUIDToString(r.CampID),
			SessionID:              api.UUIDToString(r.SessionID),
			CounselorID:            api.UUIDToString(r.CounselorID),
			SessionAgeGroupCabinID: api.UUIDToString(r.SessionAgeGroupCabinID),
			CabinID:                api.UUIDToString(r.CabinID),
			CounselorFirstName:     r.CounselorFirstName,
			CounselorLastName:      r.CounselorLastName,
			CounselorName:          r.CounselorName,
			CabinName:              r.CabinName,
			AgeGroupName:           r.AgeGroupName,
		})
	}

	cmRows, err := svc.queries.ListCamperCabinOverridesBySession(ctx, db.ListCamperCabinOverridesBySessionParams{
		SessionID: sessionUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return resp, fmt.Errorf("error listing camper-cabin overrides: %w", err)
	}
	for _, r := range cmRows {
		resp.CamperCabin = append(resp.CamperCabin, CamperCabinOverrideResponse{
			ID:                     api.UUIDToString(r.ID),
			CampID:                 api.UUIDToString(r.CampID),
			SessionID:              api.UUIDToString(r.SessionID),
			CamperID:               api.UUIDToString(r.CamperID),
			SessionAgeGroupCabinID: api.UUIDToString(r.SessionAgeGroupCabinID),
			CabinID:                api.UUIDToString(r.CabinID),
			CamperFirstName:        r.CamperFirstName,
			CamperLastName:         r.CamperLastName,
			CamperName:             r.CamperName,
			CabinName:              r.CabinName,
			AgeGroupName:           r.AgeGroupName,
		})
	}

	caRows, err := svc.queries.ListCounselorActivityOverridesBySession(ctx, db.ListCounselorActivityOverridesBySessionParams{
		SessionID: sessionUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return resp, fmt.Errorf("error listing counselor-activity overrides: %w", err)
	}
	for _, r := range caRows {
		resp.CounselorActivity = append(resp.CounselorActivity, CounselorActivityOverrideResponse{
			ID:                 api.UUIDToString(r.ID),
			CampID:             api.UUIDToString(r.CampID),
			SessionID:          api.UUIDToString(r.SessionID),
			CounselorID:        api.UUIDToString(r.CounselorID),
			SessionActivityID:  api.UUIDToString(r.SessionActivityID),
			SessionTimeSlotID:  api.UUIDToString(r.SessionTimeSlotID),
			CounselorFirstName: r.CounselorFirstName,
			CounselorLastName:  r.CounselorLastName,
			CounselorName:      r.CounselorName,
			ActivityName:       r.ActivityName,
			TimeSlotName:       r.TimeSlotName,
		})
	}

	return resp, nil
}

func (svc *Service) CreateCounselorCabin(ctx context.Context, campID, sessionID string, req CreateCounselorCabinRequest) (CounselorCabinOverrideResponse, error) {
	var resp CounselorCabinOverrideResponse

	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return resp, err
	}
	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return resp, err
	}
	counselorUUID, err := api.ParseUUID(req.CounselorID)
	if err != nil {
		return resp, err
	}
	cabinUUID, err := api.ParseUUID(req.SessionAgeGroupCabinID)
	if err != nil {
		return resp, err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return resp, fmt.Errorf("error beginning create counselor-cabin override transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	if err := validateCounselorCabin(ctx, qtx, campUUID, sessionUUID, counselorUUID, cabinUUID); err != nil {
		return resp, err
	}

	row, err := qtx.CreateCounselorCabinOverride(ctx, db.CreateCounselorCabinOverrideParams{
		CampID:                 campUUID,
		SessionID:              sessionUUID,
		CounselorID:            counselorUUID,
		SessionAgeGroupCabinID: cabinUUID,
	})
	if err != nil {
		return resp, fmt.Errorf("error creating counselor-cabin override: %w", err)
	}

	if err := svc.marker.MarkSessions(ctx, qtx, campUUID, []pgtype.UUID{sessionUUID}, []staleness.RunType{staleness.RunTypeCabin}); err != nil {
		return resp, err
	}

	if err := tx.Commit(ctx); err != nil {
		return resp, fmt.Errorf("error committing create counselor-cabin override transaction: %w", err)
	}

	return svc.getCounselorCabin(ctx, campUUID, sessionUUID, row.ID)
}

func (svc *Service) DeleteCounselorCabin(ctx context.Context, campID, sessionID, id string) error {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return err
	}
	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return err
	}
	idUUID, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error beginning delete counselor-cabin override transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	// Verify the override belongs to the session before deleting.
	existing, err := qtx.GetCounselorCabinOverride(ctx, db.GetCounselorCabinOverrideParams{ID: idUUID, CampID: campUUID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("error getting counselor-cabin override %s: %w", id, err)
	}
	if existing.SessionID != sessionUUID {
		return ErrNotFound
	}

	rows, err := qtx.DeleteCounselorCabinOverride(ctx, db.DeleteCounselorCabinOverrideParams{ID: idUUID, CampID: campUUID})
	if err != nil {
		return fmt.Errorf("error deleting counselor-cabin override %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	if err := svc.marker.MarkSessions(ctx, qtx, campUUID, []pgtype.UUID{sessionUUID}, []staleness.RunType{staleness.RunTypeCabin}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing delete counselor-cabin override transaction: %w", err)
	}
	return nil
}

func (svc *Service) CreateCamperCabin(ctx context.Context, campID, sessionID string, req CreateCamperCabinRequest) (CamperCabinOverrideResponse, error) {
	var resp CamperCabinOverrideResponse

	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return resp, err
	}
	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return resp, err
	}
	camperUUID, err := api.ParseUUID(req.CamperID)
	if err != nil {
		return resp, err
	}
	cabinUUID, err := api.ParseUUID(req.SessionAgeGroupCabinID)
	if err != nil {
		return resp, err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return resp, fmt.Errorf("error beginning create camper-cabin override transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	if err := validateCamperCabin(ctx, qtx, campUUID, sessionUUID, camperUUID, cabinUUID); err != nil {
		return resp, err
	}

	row, err := qtx.CreateCamperCabinOverride(ctx, db.CreateCamperCabinOverrideParams{
		CampID:                 campUUID,
		SessionID:              sessionUUID,
		CamperID:               camperUUID,
		SessionAgeGroupCabinID: cabinUUID,
	})
	if err != nil {
		return resp, fmt.Errorf("error creating camper-cabin override: %w", err)
	}

	if err := svc.marker.MarkSessions(ctx, qtx, campUUID, []pgtype.UUID{sessionUUID}, []staleness.RunType{staleness.RunTypeCabin}); err != nil {
		return resp, err
	}

	if err := tx.Commit(ctx); err != nil {
		return resp, fmt.Errorf("error committing create camper-cabin override transaction: %w", err)
	}

	return svc.getCamperCabin(ctx, campUUID, sessionUUID, row.ID)
}

func (svc *Service) DeleteCamperCabin(ctx context.Context, campID, sessionID, id string) error {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return err
	}
	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return err
	}
	idUUID, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error beginning delete camper-cabin override transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	existing, err := qtx.GetCamperCabinOverride(ctx, db.GetCamperCabinOverrideParams{ID: idUUID, CampID: campUUID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("error getting camper-cabin override %s: %w", id, err)
	}
	if existing.SessionID != sessionUUID {
		return ErrNotFound
	}

	rows, err := qtx.DeleteCamperCabinOverride(ctx, db.DeleteCamperCabinOverrideParams{ID: idUUID, CampID: campUUID})
	if err != nil {
		return fmt.Errorf("error deleting camper-cabin override %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	if err := svc.marker.MarkSessions(ctx, qtx, campUUID, []pgtype.UUID{sessionUUID}, []staleness.RunType{staleness.RunTypeCabin}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing delete camper-cabin override transaction: %w", err)
	}
	return nil
}

func (svc *Service) CreateCounselorActivity(ctx context.Context, campID, sessionID string, req CreateCounselorActivityRequest) (CounselorActivityOverrideResponse, error) {
	var resp CounselorActivityOverrideResponse

	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return resp, err
	}
	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return resp, err
	}
	counselorUUID, err := api.ParseUUID(req.CounselorID)
	if err != nil {
		return resp, err
	}
	saUUID, err := api.ParseUUID(req.SessionActivityID)
	if err != nil {
		return resp, err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return resp, fmt.Errorf("error beginning create counselor-activity override transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	if err := validateCounselorActivity(ctx, qtx, campUUID, sessionUUID, counselorUUID, saUUID); err != nil {
		return resp, err
	}

	row, err := qtx.CreateCounselorActivityOverride(ctx, db.CreateCounselorActivityOverrideParams{
		CampID:            campUUID,
		SessionID:         sessionUUID,
		CounselorID:       counselorUUID,
		SessionActivityID: saUUID,
	})
	if err != nil {
		return resp, fmt.Errorf("error creating counselor-activity override: %w", err)
	}

	if err := svc.marker.MarkSessions(ctx, qtx, campUUID, []pgtype.UUID{sessionUUID}, []staleness.RunType{staleness.RunTypeActivity}); err != nil {
		return resp, err
	}

	if err := tx.Commit(ctx); err != nil {
		return resp, fmt.Errorf("error committing create counselor-activity override transaction: %w", err)
	}

	return svc.getCounselorActivity(ctx, campUUID, sessionUUID, row.ID)
}

func (svc *Service) DeleteCounselorActivity(ctx context.Context, campID, sessionID, id string) error {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return err
	}
	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return err
	}
	idUUID, err := api.ParseUUID(id)
	if err != nil {
		return err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error beginning delete counselor-activity override transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	existing, err := qtx.GetCounselorActivityOverride(ctx, db.GetCounselorActivityOverrideParams{ID: idUUID, CampID: campUUID})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("error getting counselor-activity override %s: %w", id, err)
	}
	if existing.SessionID != sessionUUID {
		return ErrNotFound
	}

	rows, err := qtx.DeleteCounselorActivityOverride(ctx, db.DeleteCounselorActivityOverrideParams{ID: idUUID, CampID: campUUID})
	if err != nil {
		return fmt.Errorf("error deleting counselor-activity override %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	if err := svc.marker.MarkSessions(ctx, qtx, campUUID, []pgtype.UUID{sessionUUID}, []staleness.RunType{staleness.RunTypeActivity}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing delete counselor-activity override transaction: %w", err)
	}
	return nil
}

// getCounselorCabin reloads the override with display fields after create.
func (svc *Service) getCounselorCabin(ctx context.Context, campUUID, sessionUUID, id pgtype.UUID) (CounselorCabinOverrideResponse, error) {
	rows, err := svc.queries.ListCounselorCabinOverridesBySession(ctx, db.ListCounselorCabinOverridesBySessionParams{
		SessionID: sessionUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return CounselorCabinOverrideResponse{}, fmt.Errorf("error reloading counselor-cabin override: %w", err)
	}
	for _, r := range rows {
		if r.ID == id {
			return CounselorCabinOverrideResponse{
				ID:                     api.UUIDToString(r.ID),
				CampID:                 api.UUIDToString(r.CampID),
				SessionID:              api.UUIDToString(r.SessionID),
				CounselorID:            api.UUIDToString(r.CounselorID),
				SessionAgeGroupCabinID: api.UUIDToString(r.SessionAgeGroupCabinID),
				CabinID:                api.UUIDToString(r.CabinID),
				CounselorFirstName:     r.CounselorFirstName,
				CounselorLastName:      r.CounselorLastName,
				CounselorName:          r.CounselorName,
				CabinName:              r.CabinName,
				AgeGroupName:           r.AgeGroupName,
			}, nil
		}
	}
	return CounselorCabinOverrideResponse{}, ErrNotFound
}

func (svc *Service) getCamperCabin(ctx context.Context, campUUID, sessionUUID, id pgtype.UUID) (CamperCabinOverrideResponse, error) {
	rows, err := svc.queries.ListCamperCabinOverridesBySession(ctx, db.ListCamperCabinOverridesBySessionParams{
		SessionID: sessionUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return CamperCabinOverrideResponse{}, fmt.Errorf("error reloading camper-cabin override: %w", err)
	}
	for _, r := range rows {
		if r.ID == id {
			return CamperCabinOverrideResponse{
				ID:                     api.UUIDToString(r.ID),
				CampID:                 api.UUIDToString(r.CampID),
				SessionID:              api.UUIDToString(r.SessionID),
				CamperID:               api.UUIDToString(r.CamperID),
				SessionAgeGroupCabinID: api.UUIDToString(r.SessionAgeGroupCabinID),
				CabinID:                api.UUIDToString(r.CabinID),
				CamperFirstName:        r.CamperFirstName,
				CamperLastName:         r.CamperLastName,
				CamperName:             r.CamperName,
				CabinName:              r.CabinName,
				AgeGroupName:           r.AgeGroupName,
			}, nil
		}
	}
	return CamperCabinOverrideResponse{}, ErrNotFound
}

func (svc *Service) getCounselorActivity(ctx context.Context, campUUID, sessionUUID, id pgtype.UUID) (CounselorActivityOverrideResponse, error) {
	rows, err := svc.queries.ListCounselorActivityOverridesBySession(ctx, db.ListCounselorActivityOverridesBySessionParams{
		SessionID: sessionUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return CounselorActivityOverrideResponse{}, fmt.Errorf("error reloading counselor-activity override: %w", err)
	}
	for _, r := range rows {
		if r.ID == id {
			return CounselorActivityOverrideResponse{
				ID:                 api.UUIDToString(r.ID),
				CampID:             api.UUIDToString(r.CampID),
				SessionID:          api.UUIDToString(r.SessionID),
				CounselorID:        api.UUIDToString(r.CounselorID),
				SessionActivityID:  api.UUIDToString(r.SessionActivityID),
				SessionTimeSlotID:  api.UUIDToString(r.SessionTimeSlotID),
				CounselorFirstName: r.CounselorFirstName,
				CounselorLastName:  r.CounselorLastName,
				CounselorName:      r.CounselorName,
				ActivityName:       r.ActivityName,
				TimeSlotName:       r.TimeSlotName,
			}, nil
		}
	}
	return CounselorActivityOverrideResponse{}, ErrNotFound
}
