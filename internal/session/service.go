package session

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"
	"camp-scheduler/internal/staleness"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("session not found")

type Service struct {
	queries *db.Queries
	pool    *pgxpool.Pool
	marker  *staleness.Marker
}

func NewService(queries *db.Queries, pool *pgxpool.Pool, marker *staleness.Marker) *Service {
	return &Service{queries: queries, pool: pool, marker: marker}
}

// ensureSeasonActive rejects sessions being created or updated under an
// archived season. Archived seasons are filtered out of the default sessions
// listing, so allowing new sessions under them would orphan the data.
func (svc *Service) ensureSeasonActive(ctx context.Context, q *db.Queries, campID, seasonID pgtype.UUID) error {
	season, err := q.GetSeason(ctx, db.GetSeasonParams{
		ID:     seasonID,
		CampID: campID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return api.BadInput("season not found")
		}
		return fmt.Errorf("error looking up season: %w", err)
	}
	if season.Archived {
		return api.BadInput("cannot use an archived season")
	}
	return nil
}

func (svc *Service) List(ctx context.Context, campID string) ([]SessionResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	sessions, err := svc.queries.ListSessions(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("error listing sessions: %w", err)
	}

	result := make([]SessionResponse, len(sessions))
	for i, sn := range sessions {
		result[i] = toSessionResponse(sn)
	}
	return result, nil
}

func (svc *Service) GetByID(ctx context.Context, campID, id string) (SessionResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return SessionResponse{}, err
	}

	session, err := svc.queries.GetSession(ctx, db.GetSessionParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return SessionResponse{}, fmt.Errorf("error getting session %s: %w", id, err)
	}

	return toSessionResponse(session), nil
}

func (svc *Service) Create(ctx context.Context, campID string, req CreateSessionRequest) (SessionResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionResponse{}, err
	}

	seasonUUID, err := api.ParseUUID(req.SeasonID)
	if err != nil {
		return SessionResponse{}, err
	}

	prevUUID, err := api.ToPgUUID(req.PreviousSessionID)
	if err != nil {
		return SessionResponse{}, err
	}

	if err := svc.validatePreviousSession(ctx, campUUID, seasonUUID, pgtype.UUID{}, req.PreviousSessionID); err != nil {
		return SessionResponse{}, err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return SessionResponse{}, fmt.Errorf("error beginning create session transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	if err := svc.ensureSeasonActive(ctx, qtx, campUUID, seasonUUID); err != nil {
		return SessionResponse{}, err
	}

	session, err := qtx.CreateSession(ctx, db.CreateSessionParams{
		CampID:          campUUID,
		SeasonID:        seasonUUID,
		SessionName:     req.Name,
		PreviousSession: prevUUID,
	})
	if err != nil {
		return SessionResponse{}, fmt.Errorf("error creating session: %w", err)
	}

	// Bootstrap the session counselor roster with all currently active
	// (non-archived) counselors. Admins curate it from there; the solver and
	// preference filtering both read this roster.
	activeCounselors, err := qtx.ListCounselors(ctx, campUUID)
	if err != nil {
		return SessionResponse{}, fmt.Errorf("error loading counselors for new session roster: %w", err)
	}
	for _, c := range activeCounselors {
		if _, err := qtx.AddSessionCounselor(ctx, db.AddSessionCounselorParams{
			CampID:      campUUID,
			SessionID:   session.ID,
			CounselorID: c.ID,
		}); err != nil {
			return SessionResponse{}, fmt.Errorf("error seeding session counselor roster: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return SessionResponse{}, fmt.Errorf("error committing create session transaction: %w", err)
	}

	return toSessionResponse(session), nil
}

func (svc *Service) Update(ctx context.Context, campID, id string, req UpdateSessionRequest) (SessionResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionResponse{}, err
	}

	uid, err := api.ParseUUID(id)
	if err != nil {
		return SessionResponse{}, err
	}

	seasonUUID, err := api.ParseUUID(req.SeasonID)
	if err != nil {
		return SessionResponse{}, err
	}

	prevUUID, err := api.ToPgUUID(req.PreviousSessionID)
	if err != nil {
		return SessionResponse{}, err
	}

	current, err := svc.queries.GetSession(ctx, db.GetSessionParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SessionResponse{}, ErrNotFound
		}
		return SessionResponse{}, fmt.Errorf("error getting session %s: %w", id, err)
	}

	if err := svc.validatePreviousSession(ctx, campUUID, seasonUUID, uid, req.PreviousSessionID); err != nil {
		return SessionResponse{}, err
	}

	if current.SeasonID != seasonUUID {
		hasDeps, err := svc.queries.HasDependentSessions(ctx, db.HasDependentSessionsParams{
			PreviousSession: uid,
			CampID:          campUUID,
		})
		if err != nil {
			return SessionResponse{}, fmt.Errorf("error checking session dependents: %w", err)
		}
		if hasDeps {
			return SessionResponse{}, api.BadInput("cannot change season: other sessions reference this session as previous")
		}
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return SessionResponse{}, fmt.Errorf("error beginning update session transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	if current.SeasonID != seasonUUID {
		if err := svc.ensureSeasonActive(ctx, qtx, campUUID, seasonUUID); err != nil {
			return SessionResponse{}, err
		}
	}

	session, err := qtx.UpdateSession(ctx, db.UpdateSessionParams{
		ID:              uid,
		CampID:          campUUID,
		SeasonID:        seasonUUID,
		SessionName:     req.Name,
		PreviousSession: prevUUID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SessionResponse{}, ErrNotFound
		}
		return SessionResponse{}, fmt.Errorf("error updating session %s: %w", id, err)
	}

	if err := svc.marker.MarkSessions(ctx, qtx, campUUID, []pgtype.UUID{uid}, staleness.AllRunTypes); err != nil {
		return SessionResponse{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return SessionResponse{}, fmt.Errorf("error committing update session transaction: %w", err)
	}

	return toSessionResponse(session), nil
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
		return fmt.Errorf("error beginning delete session transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	// Mark stale inside the tx before delete so the cascade lookup still
	// sees this session as a previous_session for any dependents.
	if err := svc.marker.MarkSessions(ctx, qtx, campUUID, []pgtype.UUID{uid}, staleness.AllRunTypes); err != nil {
		return err
	}

	rows, err := qtx.DeleteSession(ctx, db.DeleteSessionParams{
		ID:     uid,
		CampID: campUUID,
	})
	if err != nil {
		return fmt.Errorf("error deleting session %s: %w", id, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing delete session transaction: %w", err)
	}

	return nil
}

func toSessionResponse(s db.Session) SessionResponse {
	return SessionResponse{
		ID:                api.UUIDToString(s.ID),
		CampID:            api.UUIDToString(s.CampID),
		SeasonID:          api.UUIDToString(s.SeasonID),
		Name:              s.SessionName,
		PreviousSessionID: api.UUIDToStringPtr(s.PreviousSession),
		Archived:          s.Archived,
	}
}

func (svc *Service) ListArchived(ctx context.Context, campID string) ([]SessionResponse, error) {
	uid, err := api.ParseUUID(campID)
	if err != nil {
		return nil, err
	}

	sessions, err := svc.queries.ListArchivedSessions(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("error listing archived sessions: %w", err)
	}

	result := make([]SessionResponse, len(sessions))
	for i, sn := range sessions {
		result[i] = toSessionResponse(sn)
	}
	return result, nil
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
		return fmt.Errorf("error beginning archive session transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	if err := svc.marker.MarkSessions(ctx, qtx, campUUID, []pgtype.UUID{uid}, staleness.AllRunTypes); err != nil {
		return err
	}

	var rows int64
	if archived {
		rows, err = qtx.ArchiveSession(ctx, db.ArchiveSessionParams{ID: uid, CampID: campUUID})
	} else {
		rows, err = qtx.UnarchiveSession(ctx, db.UnarchiveSessionParams{ID: uid, CampID: campUUID})
	}
	if err != nil {
		return fmt.Errorf("error setting session %s archived=%t: %w", id, archived, err)
	}
	if rows == 0 {
		return ErrNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("error committing archive session transaction: %w", err)
	}

	return nil
}

// validatePreviousSession checks that the referenced previous session belongs
// to the same season and is not a self-reference. No-op when previousSessionID is nil.
func (svc *Service) validatePreviousSession(ctx context.Context, campID, seasonID, sessionID pgtype.UUID, previousSessionID *string) error {
	if previousSessionID == nil {
		return nil
	}

	prevUUID, err := api.ParseUUID(*previousSessionID)
	if err != nil {
		return err
	}

	if sessionID.Valid && prevUUID == sessionID {
		return api.BadInput("previous session cannot be the same session")
	}

	prev, err := svc.queries.GetSession(ctx, db.GetSessionParams{
		ID:     prevUUID,
		CampID: campID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return api.BadInput("previous session not found")
		}
		return fmt.Errorf("error looking up previous session: %w", err)
	}

	if prev.SeasonID != seasonID {
		return api.BadInput("previous session must belong to the same season")
	}

	return nil
}

// Copy structurally clones an existing session into a new one. The clone
// includes session age groups, session age group cabins, session time slots,
// and session activities — all in a single transaction. Camper enrollments,
// counselor preferences, and assignment runs are deliberately not copied.
func (svc *Service) Copy(ctx context.Context, campID, sourceID string, req CopySessionRequest) (SessionResponse, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return SessionResponse{}, err
	}

	sourceUUID, err := api.ParseUUID(sourceID)
	if err != nil {
		return SessionResponse{}, err
	}

	seasonUUID, err := api.ParseUUID(req.SeasonID)
	if err != nil {
		return SessionResponse{}, err
	}

	prevUUID, err := api.ToPgUUID(req.PreviousSessionID)
	if err != nil {
		return SessionResponse{}, err
	}

	// Source must exist in this camp.
	if _, err := svc.queries.GetSession(ctx, db.GetSessionParams{
		ID:     sourceUUID,
		CampID: campUUID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return SessionResponse{}, ErrNotFound
		}
		return SessionResponse{}, fmt.Errorf("error getting source session %s: %w", sourceID, err)
	}

	if err := svc.validatePreviousSession(ctx, campUUID, seasonUUID, pgtype.UUID{}, req.PreviousSessionID); err != nil {
		return SessionResponse{}, err
	}

	tx, err := svc.pool.Begin(ctx)
	if err != nil {
		return SessionResponse{}, fmt.Errorf("error beginning copy session transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := svc.queries.WithTx(tx)

	if err := svc.ensureSeasonActive(ctx, qtx, campUUID, seasonUUID); err != nil {
		return SessionResponse{}, err
	}

	newSession, err := qtx.CreateSession(ctx, db.CreateSessionParams{
		CampID:          campUUID,
		SeasonID:        seasonUUID,
		SessionName:     req.Name,
		PreviousSession: prevUUID,
	})
	if err != nil {
		return SessionResponse{}, fmt.Errorf("error creating copied session: %w", err)
	}

	// Track skipped references so we can surface a single summary log at the
	// end. Archived parents are intentionally omitted from the clone so the
	// new active session does not inherit references the admin already
	// retired.
	var skippedAgeGroups, skippedCabins, skippedTimeSlots, skippedActivities, skippedCounselors int

	// Copy session_age_groups, building a map from old SAG id to new SAG id
	// so we can remap session_age_group_cabins below.
	sourceSAGs, err := qtx.ListSessionAgeGroups(ctx, db.ListSessionAgeGroupsParams{
		SessionID: sourceUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return SessionResponse{}, fmt.Errorf("error listing source session age groups: %w", err)
	}
	sagIDMap := make(map[pgtype.UUID]pgtype.UUID, len(sourceSAGs))
	for _, sag := range sourceSAGs {
		ag, err := qtx.GetAgeGroup(ctx, db.GetAgeGroupParams{
			ID:     sag.AgeGroupID,
			CampID: campUUID,
		})
		if err != nil {
			return SessionResponse{}, fmt.Errorf("error loading source age group %v: %w", sag.AgeGroupID, err)
		}
		if ag.Archived {
			skippedAgeGroups++
			continue
		}
		newSAG, err := qtx.CreateSessionAgeGroup(ctx, db.CreateSessionAgeGroupParams{
			CampID:     campUUID,
			SessionID:  newSession.ID,
			AgeGroupID: sag.AgeGroupID,
		})
		if err != nil {
			return SessionResponse{}, fmt.Errorf("error copying session age group: %w", err)
		}
		sagIDMap[sag.ID] = newSAG.ID
	}

	sourceCabins, err := qtx.ListSessionAgeGroupCabinsBySession(ctx, db.ListSessionAgeGroupCabinsBySessionParams{
		SessionID: sourceUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return SessionResponse{}, fmt.Errorf("error listing source session cabins: %w", err)
	}
	for _, sc := range sourceCabins {
		newSAGID, ok := sagIDMap[sc.SessionAgeGroupID]
		if !ok {
			// Parent SAG was skipped (archived age group); skip dependent cabin.
			skippedCabins++
			continue
		}
		cabin, err := qtx.GetCabin(ctx, db.GetCabinParams{
			ID:     sc.CabinID,
			CampID: campUUID,
		})
		if err != nil {
			return SessionResponse{}, fmt.Errorf("error loading source cabin %v: %w", sc.CabinID, err)
		}
		if cabin.Archived {
			skippedCabins++
			continue
		}
		if _, err := qtx.CreateSessionAgeGroupCabin(ctx, db.CreateSessionAgeGroupCabinParams{
			CampID:             campUUID,
			SessionAgeGroupID:  newSAGID,
			CabinID:            sc.CabinID,
			GroupSize:          sc.GroupSize,
			RequiredCounselors: sc.RequiredCounselors,
			Gender:             sc.Gender,
		}); err != nil {
			return SessionResponse{}, fmt.Errorf("error copying session cabin: %w", err)
		}
	}

	// Copy session_time_slots, building a map from old STS id to new STS id
	// so we can remap session_activities below.
	sourceSTSs, err := qtx.ListSessionTimeSlots(ctx, db.ListSessionTimeSlotsParams{
		SessionID: sourceUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return SessionResponse{}, fmt.Errorf("error listing source session time slots: %w", err)
	}
	stsIDMap := make(map[pgtype.UUID]pgtype.UUID, len(sourceSTSs))
	for _, sts := range sourceSTSs {
		ts, err := qtx.GetTimeSlot(ctx, db.GetTimeSlotParams{
			ID:     sts.TimeSlotID,
			CampID: campUUID,
		})
		if err != nil {
			return SessionResponse{}, fmt.Errorf("error loading source time slot %v: %w", sts.TimeSlotID, err)
		}
		if ts.Archived {
			skippedTimeSlots++
			continue
		}
		newSTS, err := qtx.CreateSessionTimeSlot(ctx, db.CreateSessionTimeSlotParams{
			CampID:     campUUID,
			SessionID:  newSession.ID,
			TimeSlotID: sts.TimeSlotID,
			SortOrder:  sts.SortOrder,
		})
		if err != nil {
			return SessionResponse{}, fmt.Errorf("error copying session time slot: %w", err)
		}
		stsIDMap[sts.ID] = newSTS.ID
	}

	sourceActivities, err := qtx.ListSessionActivities(ctx, db.ListSessionActivitiesParams{
		SessionID: sourceUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return SessionResponse{}, fmt.Errorf("error listing source session activities: %w", err)
	}
	for _, sa := range sourceActivities {
		newSTSID, ok := stsIDMap[sa.SessionTimeSlotID]
		if !ok {
			// Parent STS was skipped (archived time slot); skip dependent activity.
			skippedActivities++
			continue
		}
		act, err := qtx.GetActivity(ctx, db.GetActivityParams{
			ID:     sa.ActivityID,
			CampID: campUUID,
		})
		if err != nil {
			return SessionResponse{}, fmt.Errorf("error loading source activity %v: %w", sa.ActivityID, err)
		}
		if act.Archived {
			skippedActivities++
			continue
		}
		if _, err := qtx.CreateSessionActivity(ctx, db.CreateSessionActivityParams{
			CampID:             campUUID,
			SessionTimeSlotID:  newSTSID,
			ActivityID:         sa.ActivityID,
			Capacity:           sa.Capacity,
			RequiredCounselors: sa.RequiredCounselors,
		}); err != nil {
			return SessionResponse{}, fmt.Errorf("error copying session activity: %w", err)
		}
	}

	// Copy the session counselor roster from the source session so the new
	// session inherits the same staffing pool.
	sourceCounselors, err := qtx.ListSessionCounselorIDs(ctx, db.ListSessionCounselorIDsParams{
		SessionID: sourceUUID,
		CampID:    campUUID,
	})
	if err != nil {
		return SessionResponse{}, fmt.Errorf("error listing source session counselors: %w", err)
	}
	for _, counselorID := range sourceCounselors {
		c, err := qtx.GetCounselor(ctx, db.GetCounselorParams{
			ID:     counselorID,
			CampID: campUUID,
		})
		if err != nil {
			return SessionResponse{}, fmt.Errorf("error loading source counselor %v: %w", counselorID, err)
		}
		if c.Archived {
			skippedCounselors++
			continue
		}
		if _, err := qtx.AddSessionCounselor(ctx, db.AddSessionCounselorParams{
			CampID:      campUUID,
			SessionID:   newSession.ID,
			CounselorID: counselorID,
		}); err != nil {
			return SessionResponse{}, fmt.Errorf("error copying session counselor: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return SessionResponse{}, fmt.Errorf("error committing copy session transaction: %w", err)
	}

	if skippedAgeGroups+skippedCabins+skippedTimeSlots+skippedActivities+skippedCounselors > 0 {
		slog.
			With("source_session_id", sourceID).
			With("new_session_id", newSession.ID.String()).
			With("skipped_age_groups", skippedAgeGroups).
			With("skipped_cabins", skippedCabins).
			With("skipped_time_slots", skippedTimeSlots).
			With("skipped_activities", skippedActivities).
			With("skipped_counselors", skippedCounselors).
			Info("skipped archived references during session copy")
	}

	return toSessionResponse(newSession), nil
}