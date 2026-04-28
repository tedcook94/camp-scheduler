package solver

import (
	"fmt"
	"sort"
)

func ScoreActivitySoftConstraints(snapshot ActivitySnapshot, assignment CounselorActivityAssignment, weights ActivityWeights) ScoreResult {
	var result ScoreResult

	components := scoreActivityPreference(snapshot, assignment, weights)
	for _, c := range components {
		result.Total += c.Score
		result.Breakdown = append(result.Breakdown, c)
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

func scoreActivityPreference(snapshot ActivitySnapshot, assignment CounselorActivityAssignment, weights ActivityWeights) []ScoreComponent {
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
