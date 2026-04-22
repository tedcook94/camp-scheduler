package solver

import (
	"fmt"
	"sort"
	"strings"
)

func ExplainActivity(snapshot ActivitySnapshot, solution ActivitySolution) ActivityExplanation {
	slotsByID := indexActivitySlots(snapshot)

	reasonsByAssignment := buildActivityReasonMap(solution.Score.Breakdown)

	// Build counselor -> list of slot IDs they're assigned to.
	counselorSlots := make(map[string][]string)
	for slotID, counselorIDs := range solution.Assignment.SlotCounselors {
		for _, cID := range counselorIDs {
			counselorSlots[cID] = append(counselorSlots[cID], slotID)
		}
	}

	var assignments []ActivityAssignmentExplanation
	for slotID, counselorIDs := range solution.Assignment.SlotCounselors {
		for _, cID := range counselorIDs {
			key := cID + "|" + slotID
			reasons := reasonsByAssignment[key]
			if len(reasons) == 0 {
				continue
			}

			assignments = append(assignments, ActivityAssignmentExplanation{
				CounselorID: cID,
				SlotID:      slotID,
				Reasons:     reasons,
			})
		}
	}

	sort.Slice(assignments, func(i, j int) bool {
		if assignments[i].SlotID != assignments[j].SlotID {
			return assignments[i].SlotID < assignments[j].SlotID
		}
		return assignments[i].CounselorID < assignments[j].CounselorID
	})

	var unmet []ActivityUnmetPreference
	unmet = append(unmet, findUnmetActivityPreferences(snapshot, counselorSlots, slotsByID)...)
	unmet = append(unmet, findUnassignedCounselors(snapshot, counselorSlots)...)

	sort.Slice(unmet, func(i, j int) bool {
		if unmet[i].CounselorID != unmet[j].CounselorID {
			return unmet[i].CounselorID < unmet[j].CounselorID
		}
		if unmet[i].Constraint != unmet[j].Constraint {
			return unmet[i].Constraint < unmet[j].Constraint
		}
		return unmet[i].Message < unmet[j].Message
	})

	return ActivityExplanation{
		Assignments:      assignments,
		UnmetPreferences: unmet,
	}
}

// buildActivityReasonMap keys reasons by "counselorID|slotID" so that
// explanations are accurate per assignment, not just per counselor.
func buildActivityReasonMap(breakdown []ScoreComponent) map[string][]string {
	reasons := make(map[string][]string)
	for _, c := range breakdown {
		if c.CounselorID == "" {
			continue
		}
		key := c.CounselorID + "|" + c.SlotID
		reason := fmt.Sprintf("%s (+%.1f)", c.Message, c.Score)
		reasons[key] = append(reasons[key], reason)
	}
	return reasons
}

func findUnmetActivityPreferences(snapshot ActivitySnapshot, counselorSlots map[string][]string, slotsByID map[string]ActivitySlot) []ActivityUnmetPreference {
	// Build counselor -> set of activity IDs they're assigned to.
	counselorActivities := make(map[string]map[string]bool)
	for cID, slotIDs := range counselorSlots {
		for _, slotID := range slotIDs {
			slot := slotsByID[slotID]
			if counselorActivities[cID] == nil {
				counselorActivities[cID] = make(map[string]bool)
			}
			counselorActivities[cID][slot.ActivityID] = true
		}
	}

	// Build activityID -> activityName lookup from slots.
	activityNames := make(map[string]string)
	for _, slot := range snapshot.Slots {
		activityNames[slot.ActivityID] = slot.ActivityName
	}
	activityLabel := func(id string) string {
		if name, ok := activityNames[id]; ok && name != "" {
			return name
		}
		return id
	}

	var unmet []ActivityUnmetPreference
	for counselorID, prefs := range snapshot.ActivityPreferences {
		assigned := counselorActivities[counselorID]
		var unmetPrefs []RankedPreference
		for _, pref := range prefs {
			if !assigned[pref.TargetID] {
				unmetPrefs = append(unmetPrefs, pref)
			}
		}
		if len(unmetPrefs) == 0 {
			continue
		}

		sort.Slice(unmetPrefs, func(i, j int) bool {
			return unmetPrefs[i].Rank < unmetPrefs[j].Rank
		})

		// Hybrid aggregation: single row when ALL prefs unmet, individual
		// rows otherwise.
		if len(unmetPrefs) == len(prefs) {
			parts := make([]string, len(unmetPrefs))
			for i, p := range unmetPrefs {
				parts[i] = fmt.Sprintf("%s (rank %d)", activityLabel(p.TargetID), p.Rank)
			}
			unmet = append(unmet, ActivityUnmetPreference{
				CounselorID: counselorID,
				Constraint:  "activity_preference",
				Message: fmt.Sprintf(
					"not assigned to any preferred activity (%s)",
					strings.Join(parts, ", "),
				),
			})
		} else {
			for _, pref := range unmetPrefs {
				unmet = append(unmet, ActivityUnmetPreference{
					CounselorID: counselorID,
					Constraint:  "activity_preference",
					Message: fmt.Sprintf(
						"not assigned to preferred activity %s (rank %d)",
						activityLabel(pref.TargetID), pref.Rank,
					),
				})
			}
		}
	}
	return unmet
}

func findUnassignedCounselors(snapshot ActivitySnapshot, counselorSlots map[string][]string) []ActivityUnmetPreference {
	var unmet []ActivityUnmetPreference
	for _, c := range snapshot.Counselors {
		if len(counselorSlots[c.ID]) == 0 {
			unmet = append(unmet, ActivityUnmetPreference{
				CounselorID: c.ID,
				Constraint:  "unassigned",
				Message: fmt.Sprintf(
					"counselor %q not assigned to any activity",
					c.Name,
				),
			})
		}
	}
	return unmet
}
