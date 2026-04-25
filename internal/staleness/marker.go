// Package staleness marks assignment runs as stale when their inputs change.
//
// A run becomes stale when any input the solver consumed could now produce a
// different result. The Marker is invoked from each mutating service right
// after the mutation succeeds, ideally inside the same transaction so the
// staleness flip is atomic with the change that caused it.
//
// Cross-session cascade: a session's solver also reads its `previous_session`
// (counselor history and the prior run's selected solution feed the
// previously-unmet-preferences boost). Marker.MarkSessions therefore also
// marks the immediately-following session(s) stale for the same run types.
package staleness

import (
	"context"
	"fmt"

	"camp-scheduler/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
)

// RunType matches the assignment_runs.run_type CHECK constraint values.
type RunType string

const (
	RunTypeCabin    RunType = "cabin"
	RunTypeActivity RunType = "activity_schedule"
)

// AllRunTypes is a convenience for mutations that affect both run types
// (e.g., adding/removing a counselor from a session roster).
var AllRunTypes = []RunType{RunTypeCabin, RunTypeActivity}

// Marker marks assignment runs stale. It is safe to share across services.
// Pass a transactional *db.Queries (qtx) when the mutation runs inside a tx
// so the staleness flip lands or rolls back with the mutation.
type Marker struct{}

// New returns a Marker. Methods take the *db.Queries to use; a service can
// pass either svc.queries or qtx.
func New() *Marker {
	return &Marker{}
}

// MarkSessions marks runs for the given (campID, sessionIDs, runTypes) stale,
// then cascades to any sessions whose previous_session is one of sessionIDs.
// Pass queries=qtx to participate in an in-flight transaction.
//
// Empty sessionIDs or runTypes is a no-op.
func (m *Marker) MarkSessions(
	ctx context.Context,
	queries *db.Queries,
	campID pgtype.UUID,
	sessionIDs []pgtype.UUID,
	runTypes []RunType,
) error {
	if len(sessionIDs) == 0 || len(runTypes) == 0 {
		return nil
	}

	dependents, err := queries.ListSessionsByPreviousSession(ctx, db.ListSessionsByPreviousSessionParams{
		CampID:             campID,
		PreviousSessionIds: sessionIDs,
	})
	if err != nil {
		return fmt.Errorf("error finding dependent sessions: %w", err)
	}

	all := append(make([]pgtype.UUID, 0, len(sessionIDs)+len(dependents)), sessionIDs...)
	all = append(all, dependents...)

	if _, err := queries.MarkAssignmentRunsStale(ctx, db.MarkAssignmentRunsStaleParams{
		CampID:     campID,
		SessionIds: all,
		RunTypes:   runTypeStrings(runTypes),
	}); err != nil {
		return fmt.Errorf("error marking assignment runs stale: %w", err)
	}
	return nil
}

// MarkCamp marks every session in the camp stale for the given run types.
// Used for camp-wide catalog mutations (cabin, age group, activity, time slot,
// certification) where computing exact session membership isn't worthwhile.
func (m *Marker) MarkCamp(
	ctx context.Context,
	queries *db.Queries,
	campID pgtype.UUID,
	runTypes []RunType,
) error {
	if len(runTypes) == 0 {
		return nil
	}
	sessionIDs, err := queries.ListSessionsByCamp(ctx, campID)
	if err != nil {
		return fmt.Errorf("error listing sessions for camp: %w", err)
	}
	if len(sessionIDs) == 0 {
		return nil
	}
	// previous_session cascade is implicit here: every session in the camp is
	// already in the set, so dependents are too.
	if _, err := queries.MarkAssignmentRunsStale(ctx, db.MarkAssignmentRunsStaleParams{
		CampID:     campID,
		SessionIds: sessionIDs,
		RunTypes:   runTypeStrings(runTypes),
	}); err != nil {
		return fmt.Errorf("error marking assignment runs stale: %w", err)
	}
	return nil
}

// MarkDependentSessions marks runs stale only for sessions whose
// previous_session is in sessionIDs (the cascade targets, not the sources
// themselves). Used when the source session's *selected solution* changes:
// the source run is the one being acted on and shouldn't be marked stale,
// but any next session that consumes its selected solution must be.
func (m *Marker) MarkDependentSessions(
	ctx context.Context,
	queries *db.Queries,
	campID pgtype.UUID,
	sessionIDs []pgtype.UUID,
	runTypes []RunType,
) error {
	if len(sessionIDs) == 0 || len(runTypes) == 0 {
		return nil
	}
	dependents, err := queries.ListSessionsByPreviousSession(ctx, db.ListSessionsByPreviousSessionParams{
		CampID:             campID,
		PreviousSessionIds: sessionIDs,
	})
	if err != nil {
		return fmt.Errorf("error finding dependent sessions: %w", err)
	}
	if len(dependents) == 0 {
		return nil
	}
	if _, err := queries.MarkAssignmentRunsStale(ctx, db.MarkAssignmentRunsStaleParams{
		CampID:     campID,
		SessionIds: dependents,
		RunTypes:   runTypeStrings(runTypes),
	}); err != nil {
		return fmt.Errorf("error marking dependent assignment runs stale: %w", err)
	}
	return nil
}

// MarkSessionsForCounselor marks runs stale for every session where the
// counselor is on the roster. Used when a counselor's camp-wide attributes
// change (name, gender, junior flag, enabled flag).
func (m *Marker) MarkSessionsForCounselor(
	ctx context.Context,
	queries *db.Queries,
	campID, counselorID pgtype.UUID,
	runTypes []RunType,
) error {
	sessionIDs, err := queries.ListSessionsByCounselorRoster(ctx, db.ListSessionsByCounselorRosterParams{
		CampID:      campID,
		CounselorID: counselorID,
	})
	if err != nil {
		return fmt.Errorf("error listing sessions for counselor: %w", err)
	}
	return m.MarkSessions(ctx, queries, campID, sessionIDs, runTypes)
}

// MarkSessionsForCamper marks runs stale for every session where the camper is
// enrolled. Used when a camper's camp-wide attributes change (name, gender,
// age) — only cabin runs are affected since the activity solver doesn't read
// camper data.
func (m *Marker) MarkSessionsForCamper(
	ctx context.Context,
	queries *db.Queries,
	campID, camperID pgtype.UUID,
	runTypes []RunType,
) error {
	sessionIDs, err := queries.ListSessionsByCamperEnrollment(ctx, db.ListSessionsByCamperEnrollmentParams{
		CampID:    campID,
		CamperID:  camperID,
	})
	if err != nil {
		return fmt.Errorf("error listing sessions for camper: %w", err)
	}
	return m.MarkSessions(ctx, queries, campID, sessionIDs, runTypes)
}

func runTypeStrings(rs []RunType) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = string(r)
	}
	return out
}
