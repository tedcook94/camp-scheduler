package override

import (
	"context"
	"fmt"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
)

// validateCounselorCabin enforces hard solver constraints at save time.
// Rules:
//   - counselor must be on the session roster
//   - cabin must belong to the session
//   - counselor.gender must match cabin.gender
//   - existing counselor pins for this cabin + 1 must not exceed group_size
//   - existing counselor pins + existing camper pins + 1 must not exceed group_size
//     (group_size is the combined capacity)
//   - DB unique constraint already prevents pinning the same counselor twice
//     in the same session, so we don't precheck that.
func validateCounselorCabin(ctx context.Context, q *db.Queries, campID, sessionID, counselorID, sagcID pgtype.UUID) error {
	counselor, err := q.GetCounselor(ctx, db.GetCounselorParams{ID: counselorID, CampID: campID})
	if err != nil {
		return fmt.Errorf("error loading counselor for override: %w", err)
	}
	if counselor.Archived {
		return api.BadInput("counselor is archived")
	}

	onRoster, err := q.IsCounselorInSession(ctx, db.IsCounselorInSessionParams{
		SessionID:   sessionID,
		CampID:      campID,
		CounselorID: counselorID,
	})
	if err != nil {
		return fmt.Errorf("error checking counselor session roster: %w", err)
	}
	if !onRoster {
		return api.BadInput("counselor is not on the session roster")
	}

	cabin, err := findSessionCabin(ctx, q, campID, sessionID, sagcID)
	if err != nil {
		return err
	}
	if cabin.CabinArchived {
		return api.BadInput("cabin is archived")
	}
	if cabin.AgeGroupArchived {
		return api.BadInput("age group is archived")
	}
	if counselor.Gender != cabin.Gender {
		return api.BadInput(fmt.Sprintf("counselor gender %q does not match cabin gender %q", counselor.Gender, cabin.Gender))
	}

	counselorOverrides, camperOverrides, err := loadCabinOverrideCounts(ctx, q, campID, sessionID)
	if err != nil {
		return err
	}
	counselorCount := counselorOverrides[sagcID]
	camperCount := camperOverrides[sagcID]

	if counselorCount+1 > int(cabin.GroupSize) {
		return api.BadInput("counselor overrides for this cabin would exceed cabin capacity")
	}
	if counselorCount+camperCount+1 > int(cabin.GroupSize) {
		return api.BadInput("counselor and camper overrides for this cabin would exceed cabin capacity")
	}
	return nil
}

// validateCamperCabin enforces hard solver constraints at save time.
// Rules:
//   - camper must be enrolled in the session
//   - the camper's session_age_group must match the cabin's session_age_group
//   - cabin must belong to the session
//   - camper.gender must match cabin.gender
//   - existing counselor pins + existing camper pins + 1 must not exceed group_size
func validateCamperCabin(ctx context.Context, q *db.Queries, campID, sessionID, camperID, sagcID pgtype.UUID) error {
	camper, err := q.GetCamper(ctx, db.GetCamperParams{ID: camperID, CampID: campID})
	if err != nil {
		return fmt.Errorf("error loading camper for override: %w", err)
	}
	if camper.Archived {
		return api.BadInput("camper is archived")
	}

	enrollment, err := q.GetCamperEnrollmentInSession(ctx, db.GetCamperEnrollmentInSessionParams{
		CampID:    campID,
		SessionID: sessionID,
		CamperID:  camperID,
	})
	if err != nil {
		return fmt.Errorf("error checking camper enrollment: %w", err)
	}

	cabin, err := findSessionCabin(ctx, q, campID, sessionID, sagcID)
	if err != nil {
		return err
	}
	if cabin.CabinArchived {
		return api.BadInput("cabin is archived")
	}
	if cabin.AgeGroupArchived {
		return api.BadInput("age group is archived")
	}
	if camper.Gender != cabin.Gender {
		return api.BadInput(fmt.Sprintf("camper gender %q does not match cabin gender %q", camper.Gender, cabin.Gender))
	}
	// The cabin row carries age_group_id; the enrollment carries session_age_group_id
	// whose age_group_id we joined in. They must match.
	if enrollment.AgeGroupID != cabin.AgeGroupID {
		return api.BadInput("camper's enrolled age group does not match cabin's age group")
	}

	counselorOverrides, camperOverrides, err := loadCabinOverrideCounts(ctx, q, campID, sessionID)
	if err != nil {
		return err
	}
	counselorCount := counselorOverrides[sagcID]
	camperCount := camperOverrides[sagcID]

	if counselorCount+camperCount+1 > int(cabin.GroupSize) {
		return api.BadInput("counselor and camper overrides for this cabin would exceed cabin capacity")
	}
	return nil
}

// validateCounselorActivity enforces hard solver constraints at save time.
// Rules:
//   - counselor must be on the session roster
//   - session_activity must belong to the session
//   - counselor must hold every certification required by the activity
//   - counselor must not already be pinned to a different activity in the
//     same session_time_slot (DB unique handles same session_activity)
//   - existing pins for this session_activity + 1 must not exceed capacity
func validateCounselorActivity(ctx context.Context, q *db.Queries, campID, sessionID, counselorID, sessionActivityID pgtype.UUID) error {
	counselor, err := q.GetCounselor(ctx, db.GetCounselorParams{ID: counselorID, CampID: campID})
	if err != nil {
		return fmt.Errorf("error loading counselor for override: %w", err)
	}
	if counselor.Archived {
		return api.BadInput("counselor is archived")
	}

	onRoster, err := q.IsCounselorInSession(ctx, db.IsCounselorInSessionParams{
		SessionID:   sessionID,
		CampID:      campID,
		CounselorID: counselorID,
	})
	if err != nil {
		return fmt.Errorf("error checking counselor session roster: %w", err)
	}
	if !onRoster {
		return api.BadInput("counselor is not on the session roster")
	}

	sa, err := q.GetSessionActivity(ctx, db.GetSessionActivityParams{
		ID:        sessionActivityID,
		CampID:    campID,
		SessionID: sessionID,
	})
	if err != nil {
		return fmt.Errorf("error loading session activity for override: %w", err)
	}

	// Required certifications for this session_activity.
	allCerts, err := q.ListActivityCertificationsBySession(ctx, db.ListActivityCertificationsBySessionParams{
		SessionID: sessionID,
		CampID:    campID,
	})
	if err != nil {
		return fmt.Errorf("error loading activity certifications: %w", err)
	}
	required := map[pgtype.UUID]struct{}{}
	for _, ac := range allCerts {
		if ac.SessionActivityID == sessionActivityID {
			required[ac.CertificationID] = struct{}{}
		}
	}
	if len(required) > 0 {
		held, err := q.ListCounselorCertifications(ctx, db.ListCounselorCertificationsParams{
			CounselorID: counselorID,
			CampID:      campID,
		})
		if err != nil {
			return fmt.Errorf("error loading counselor certifications: %w", err)
		}
		heldSet := map[pgtype.UUID]struct{}{}
		for _, h := range held {
			heldSet[h.CertificationID] = struct{}{}
		}
		for certID := range required {
			if _, ok := heldSet[certID]; !ok {
				return api.BadInput("counselor does not hold all certifications required for this activity")
			}
		}
	}

	// Existing activity overrides for the session.
	existing, err := q.ListCounselorActivityOverridesForSolver(ctx, db.ListCounselorActivityOverridesForSolverParams{
		SessionID: sessionID,
		CampID:    campID,
	})
	if err != nil {
		return fmt.Errorf("error loading existing activity overrides: %w", err)
	}
	pinsBySA := map[pgtype.UUID]int{}
	for _, e := range existing {
		// Same counselor in same time slot but a different session_activity: conflict.
		if e.CounselorID == counselorID && e.SessionTimeSlotID == sa.SessionTimeSlotID && e.SessionActivityID != sessionActivityID {
			return api.BadInput("counselor is already pinned to a different activity in this time slot")
		}
		pinsBySA[e.SessionActivityID]++
	}
	if pinsBySA[sessionActivityID]+1 > int(sa.Capacity) {
		return api.BadInput("activity overrides would exceed activity capacity")
	}
	return nil
}

// findSessionCabin returns the row from ListSessionCabinsWithCapacity matching
// the given session_age_group_cabin id. Returns BadInput if not in this session.
func findSessionCabin(ctx context.Context, q *db.Queries, campID, sessionID, sagcID pgtype.UUID) (db.ListSessionCabinsWithCapacityRow, error) {
	rows, err := q.ListSessionCabinsWithCapacity(ctx, db.ListSessionCabinsWithCapacityParams{
		SessionID: sessionID,
		CampID:    campID,
	})
	if err != nil {
		return db.ListSessionCabinsWithCapacityRow{}, fmt.Errorf("error loading session cabins: %w", err)
	}
	for _, r := range rows {
		if r.SessionAgeGroupCabinID == sagcID {
			return r, nil
		}
	}
	return db.ListSessionCabinsWithCapacityRow{}, api.BadInput("cabin does not belong to this session")
}

// loadCabinOverrideCounts returns per-(session_age_group_cabin) counts of
// existing counselor and camper overrides for the session.
func loadCabinOverrideCounts(ctx context.Context, q *db.Queries, campID, sessionID pgtype.UUID) (map[pgtype.UUID]int, map[pgtype.UUID]int, error) {
	counselorRows, err := q.ListCounselorCabinOverridesForSolver(ctx, db.ListCounselorCabinOverridesForSolverParams{
		SessionID: sessionID,
		CampID:    campID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error loading counselor-cabin overrides: %w", err)
	}
	camperRows, err := q.ListCamperCabinOverridesForSolver(ctx, db.ListCamperCabinOverridesForSolverParams{
		SessionID: sessionID,
		CampID:    campID,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error loading camper-cabin overrides: %w", err)
	}
	counselorCounts := map[pgtype.UUID]int{}
	camperCounts := map[pgtype.UUID]int{}
	for _, r := range counselorRows {
		counselorCounts[r.SessionAgeGroupCabinID]++
	}
	for _, r := range camperRows {
		camperCounts[r.SessionAgeGroupCabinID]++
	}
	return counselorCounts, camperCounts, nil
}
