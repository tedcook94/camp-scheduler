package solver

import (
	"context"
	"errors"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func BuildActivitySnapshot(ctx context.Context, queries *db.Queries, campID, sessionID string) (ActivitySnapshot, error) {
	campUUID, err := api.ParseUUID(campID)
	if err != nil {
		return ActivitySnapshot{}, err
	}

	sessionUUID, err := api.ParseUUID(sessionID)
	if err != nil {
		return ActivitySnapshot{}, err
	}

	session, err := queries.GetSession(ctx, db.GetSessionParams{
		ID:     sessionUUID,
		CampID: campUUID,
	})
	if err != nil {
		return ActivitySnapshot{}, fmt.Errorf("error getting session: %w", err)
	}

	slots, err := loadActivitySlots(ctx, queries, sessionUUID, campUUID)
	if err != nil {
		return ActivitySnapshot{}, err
	}

	counselors, err := loadActivityCounselors(ctx, queries, campUUID)
	if err != nil {
		return ActivitySnapshot{}, err
	}

	prefs, err := loadActivityPreferences(ctx, queries, sessionUUID, campUUID)
	if err != nil {
		return ActivitySnapshot{}, err
	}

	unmetPrefs, err := loadPreviouslyUnmetActivityPreferences(ctx, queries, session, campUUID)
	if err != nil {
		return ActivitySnapshot{}, err
	}

	return ActivitySnapshot{
		SessionID:                sessionID,
		Slots:                    slots,
		Counselors:               counselors,
		ActivityPreferences:      prefs,
		UnmetActivityPreferences: unmetPrefs,
	}, nil
}

func loadActivitySlots(ctx context.Context, queries *db.Queries, sessionID, campID pgtype.UUID) ([]ActivitySlot, error) {
	rows, err := queries.ListSessionActivitiesWithDetails(ctx, db.ListSessionActivitiesWithDetailsParams{
		SessionID: sessionID,
		CampID:    campID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing session activities with details: %w", err)
	}

	certRows, err := queries.ListActivityCertificationsBySession(ctx, db.ListActivityCertificationsBySessionParams{
		SessionID: sessionID,
		CampID:    campID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing activity certifications by session: %w", err)
	}

	certsBySlot := make(map[string][]string)
	for _, r := range certRows {
		slotID := api.UUIDToString(r.SessionActivityID)
		certID := api.UUIDToString(r.CertificationID)
		certsBySlot[slotID] = append(certsBySlot[slotID], certID)
	}

	slots := make([]ActivitySlot, len(rows))
	for i, r := range rows {
		slotID := api.UUIDToString(r.SessionActivityID)
		slots[i] = ActivitySlot{
			ID:                     slotID,
			ActivityID:             api.UUIDToString(r.ActivityID),
			ActivityName:           r.ActivityName,
			TimeSlotID:             api.UUIDToString(r.SessionTimeSlotID),
			TimeSlotName:           r.TimeSlotName,
			RequiredCounselors:     int(r.RequiredCounselors),
			Capacity:               int(r.Capacity),
			RequiredCertifications: certsBySlot[slotID],
		}
	}
	return slots, nil
}

func loadActivityCounselors(ctx context.Context, queries *db.Queries, campID pgtype.UUID) ([]ActivityCounselor, error) {
	rows, err := queries.ListEnabledCounselors(ctx, campID)
	if err != nil {
		return nil, fmt.Errorf("error listing enabled counselors: %w", err)
	}

	certRows, err := queries.ListSessionCounselorCertifications(ctx, campID)
	if err != nil {
		return nil, fmt.Errorf("error listing counselor certifications: %w", err)
	}

	certsByCounselor := make(map[string]map[string]bool)
	for _, r := range certRows {
		cID := api.UUIDToString(r.CounselorID)
		if certsByCounselor[cID] == nil {
			certsByCounselor[cID] = make(map[string]bool)
		}
		certsByCounselor[cID][api.UUIDToString(r.CertificationID)] = true
	}

	counselors := make([]ActivityCounselor, len(rows))
	for i, r := range rows {
		cID := api.UUIDToString(r.ID)
		certs := certsByCounselor[cID]
		if certs == nil {
			certs = make(map[string]bool)
		}
		counselors[i] = ActivityCounselor{
			ID:             cID,
			Name:           r.CounselorName,
			IsJunior:       r.JuniorCounselor,
			Certifications: certs,
		}
	}
	return counselors, nil
}

func loadActivityPreferences(ctx context.Context, queries *db.Queries, sessionID, campID pgtype.UUID) (map[string][]RankedPreference, error) {
	rows, err := queries.ListSessionActivityPreferences(ctx, db.ListSessionActivityPreferencesParams{
		SessionID: sessionID,
		CampID:    campID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing session activity preferences: %w", err)
	}

	prefs := make(map[string][]RankedPreference)
	for _, r := range rows {
		cID := api.UUIDToString(r.CounselorID)
		prefs[cID] = append(prefs[cID], RankedPreference{
			TargetID: api.UUIDToString(r.ActivityID),
			Rank:     int(r.Rank),
		})
	}
	return prefs, nil
}

func loadPreviouslyUnmetActivityPreferences(ctx context.Context, queries *db.Queries, session db.Session, campID pgtype.UUID) (map[string]map[string]bool, error) {
	if !session.PreviousSession.Valid {
		return nil, nil
	}

	solutionID, err := queries.GetSelectedActivitySolutionBySession(ctx, db.GetSelectedActivitySolutionBySessionParams{
		SessionID: session.PreviousSession,
		CampID:    campID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting previous session selected activity solution: %w", err)
	}

	prevPrefs, err := loadActivityPreferences(ctx, queries, session.PreviousSession, campID)
	if err != nil {
		return nil, err
	}

	if len(prevPrefs) == 0 {
		return nil, nil
	}

	assignments, err := queries.ListActivityAssignmentsBySolution(ctx, db.ListActivityAssignmentsBySolutionParams{
		SolutionID: solutionID,
		CampID:     campID,
	})
	if err != nil {
		return nil, fmt.Errorf("error listing previous activity solution assignments: %w", err)
	}

	prevSlots, err := loadActivitySlots(ctx, queries, session.PreviousSession, campID)
	if err != nil {
		return nil, err
	}

	slotActivity := make(map[string]string, len(prevSlots))
	for _, s := range prevSlots {
		slotActivity[s.ID] = s.ActivityID
	}

	// Build counselor -> set of activity IDs they were assigned to.
	counselorActivities := make(map[string]map[string]bool)
	for _, a := range assignments {
		cID := api.UUIDToString(a.CounselorID)
		actID := slotActivity[api.UUIDToString(a.SessionActivityID)]
		if counselorActivities[cID] == nil {
			counselorActivities[cID] = make(map[string]bool)
		}
		counselorActivities[cID][actID] = true
	}

	return diffActivityPreferences(prevPrefs, counselorActivities), nil
}

func diffActivityPreferences(prefs map[string][]RankedPreference, counselorActivities map[string]map[string]bool) map[string]map[string]bool {
	result := make(map[string]map[string]bool)
	for counselorID, prefList := range prefs {
		activities := counselorActivities[counselorID]
		for _, pref := range prefList {
			if !activities[pref.TargetID] {
				if result[counselorID] == nil {
					result[counselorID] = make(map[string]bool)
				}
				result[counselorID][pref.TargetID] = true
			}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
