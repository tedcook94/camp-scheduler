package solver

import (
	"fmt"
	"sort"
)

func ScoreActivitySoftConstraints(snapshot ActivitySnapshot, assignment ActivityAssignment, weights ActivityWeights) ScoreResult {
	var result ScoreResult

	scorers := []func(ActivitySnapshot, ActivityAssignment, ActivityWeights) []ScoreComponent{
		scoreActivityPreference,
		scoreMissingTimeSlots,
	}

	for _, scorer := range scorers {
		for _, c := range scorer(snapshot, assignment, weights) {
			result.Total += c.Score
			result.Breakdown = append(result.Breakdown, c)
		}
	}

	sort.Slice(result.Breakdown, func(i, j int) bool {
		a, b := result.Breakdown[i], result.Breakdown[j]
		if a.CounselorID != b.CounselorID {
			return a.CounselorID < b.CounselorID
		}
		if a.SlotID != b.SlotID {
			return a.SlotID < b.SlotID
		}
		if a.Constraint != b.Constraint {
			return a.Constraint < b.Constraint
		}
		return a.Message < b.Message
	})

	return result
}

func scoreActivityPreference(snapshot ActivitySnapshot, assignment ActivityAssignment, weights ActivityWeights) []ScoreComponent {
	if weights.ActivityPreference == 0 {
		return nil
	}

	slotsByID := indexActivitySlots(snapshot)
	boost := effectiveBoost(weights.RepeatedUnmetBoost)

	// Build counselor -> list of (slotID, activityID) from assignment.
	type slotActivity struct {
		SlotID     string
		ActivityID string
	}
	counselorAssignments := make(map[string][]slotActivity)
	for slotID, counselorIDs := range assignment.SlotCounselors {
		slot := slotsByID[slotID]
		for _, cID := range counselorIDs {
			counselorAssignments[cID] = append(counselorAssignments[cID], slotActivity{
				SlotID:     slotID,
				ActivityID: slot.ActivityID,
			})
		}
	}

	var components []ScoreComponent
	for counselorID, prefs := range snapshot.ActivityPreferences {
		assignments := counselorAssignments[counselorID]
		if len(assignments) == 0 {
			continue
		}
		for _, pref := range prefs {
			if pref.Rank <= 0 {
				continue
			}
			// Find all slots where this counselor is assigned to the preferred activity.
			for _, sa := range assignments {
				if sa.ActivityID == pref.TargetID {
					slot := slotsByID[sa.SlotID]
					score := weights.ActivityPreference / float64(pref.Rank)
					msg := fmt.Sprintf(
						"assigned to preferred activity %s (rank %d) in %s",
						slot.ActivityName, pref.Rank, slot.TimeSlotName,
					)
					if snapshot.UnmetActivityPreferences[counselorID][pref.TargetID] {
						score *= boost
						msg += " (previously unmet)"
					}
					components = append(components, ScoreComponent{
						Constraint:  "activity_preference",
						Score:       score,
						Rank:        pref.Rank,
						CounselorID: counselorID,
						SlotID:      sa.SlotID,
						Message:     msg,
					})
				}
			}
		}
	}
	return components
}

// scoreMissingTimeSlots applies a negative score for every (counselor, time
// slot) pair where the counselor was eligible to serve at least one
// activity slot in that time slot but received no assignment. Time slots
// where the counselor has no eligible activity (structurally impossible)
// do not produce a penalty — those are reported separately as ineligible
// rather than as actionable gaps.
func scoreMissingTimeSlots(snapshot ActivitySnapshot, assignment ActivityAssignment, weights ActivityWeights) []ScoreComponent {
	if weights.MissingTimeSlotPenalty == 0 {
		return nil
	}

	slotsByID := indexActivitySlots(snapshot)

	// Build the deterministic list of time slot IDs from the snapshot.
	timeSlotOrder := orderedTimeSlots(snapshot)
	timeSlotName := make(map[string]string, len(timeSlotOrder))
	for _, slot := range snapshot.Slots {
		if _, ok := timeSlotName[slot.TimeSlotID]; !ok {
			timeSlotName[slot.TimeSlotID] = slot.TimeSlotName
		}
	}

	// For each counselor, build the set of time slots they currently serve.
	served := make(map[string]map[string]bool, len(snapshot.Counselors))
	for slotID, counselorIDs := range assignment.SlotCounselors {
		tsID := slotsByID[slotID].TimeSlotID
		for _, cID := range counselorIDs {
			if served[cID] == nil {
				served[cID] = make(map[string]bool)
			}
			served[cID][tsID] = true
		}
	}

	// For each counselor, build the set of time slots where at least one
	// eligible activity slot exists.
	eligibleByTS := make(map[string]map[string]bool, len(snapshot.Counselors))
	for _, c := range snapshot.Counselors {
		for _, slot := range snapshot.Slots {
			if !counselorHasCerts(c, slot) {
				continue
			}
			if eligibleByTS[c.ID] == nil {
				eligibleByTS[c.ID] = make(map[string]bool)
			}
			eligibleByTS[c.ID][slot.TimeSlotID] = true
		}
	}

	var components []ScoreComponent
	for _, c := range snapshot.Counselors {
		for _, tsID := range timeSlotOrder {
			if served[c.ID][tsID] {
				continue
			}
			if !eligibleByTS[c.ID][tsID] {
				continue
			}
			components = append(components, ScoreComponent{
				Constraint:  "missing_time_slot",
				Score:       -weights.MissingTimeSlotPenalty,
				CounselorID: c.ID,
				Message: fmt.Sprintf(
					"counselor %q has no assignment in %s (eligible activities full)",
					c.Name, timeSlotName[tsID],
				),
			})
		}
	}
	return components
}
