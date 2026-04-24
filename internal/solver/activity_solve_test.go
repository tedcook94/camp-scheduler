package solver

import (
	"slices"
	"testing"
)

func TestSolveActivityUnassignedCounselors(t *testing.T) {
	t.Run("counselor missing time slots is reported", func(t *testing.T) {
		// Two time slots, two activity slots. Each only needs 1 counselor and
		// has capacity 1. With three counselors at least one will end up
		// unassigned for each time slot.
		morningArts := ActivitySlot{
			ID: "slot-arts-am", ActivityID: "act-arts", ActivityName: "Arts",
			TimeSlotID: "ts-am", TimeSlotName: "Morning",
			RequiredCounselors: 1, Capacity: 1,
		}
		afternoonArts := ActivitySlot{
			ID: "slot-arts-pm", ActivityID: "act-arts", ActivityName: "Arts",
			TimeSlotID: "ts-pm", TimeSlotName: "Afternoon",
			RequiredCounselors: 1, Capacity: 1,
		}
		snapshot := ActivitySnapshot{
			Slots: []ActivitySlot{morningArts, afternoonArts},
			Counselors: []ActivityCounselor{
				{ID: "c1", Name: "Alice", Certifications: map[string]bool{}},
				{ID: "c2", Name: "Bob", Certifications: map[string]bool{}},
				{ID: "c3", Name: "Carol", Certifications: map[string]bool{}},
			},
		}

		solutions := SolveActivity(snapshot, DefaultActivitySolverConfig())
		if len(solutions) == 0 {
			t.Fatal("expected at least one solution")
		}

		for _, sol := range solutions {
			// At least one counselor must be reported as unassigned for at
			// least one time slot.
			totalMissing := 0
			for _, uc := range sol.UnassignedCounselors {
				totalMissing += len(uc.MissingTimeSlotIDs)
			}
			if totalMissing == 0 {
				t.Errorf("expected unassigned counselor entries, got none for solution %+v", sol.Assignment)
			}

			// Verify entries are consistent with the assignment.
			placedByCounselor := map[string]map[string]bool{}
			slotsByID := map[string]ActivitySlot{}
			for _, s := range snapshot.Slots {
				slotsByID[s.ID] = s
			}
			for slotID, counselorIDs := range sol.Assignment.SlotCounselors {
				ts := slotsByID[slotID].TimeSlotID
				for _, cID := range counselorIDs {
					if placedByCounselor[cID] == nil {
						placedByCounselor[cID] = map[string]bool{}
					}
					placedByCounselor[cID][ts] = true
				}
			}
			for _, uc := range sol.UnassignedCounselors {
				for _, tsID := range uc.MissingTimeSlotIDs {
					if placedByCounselor[uc.CounselorID][tsID] {
						t.Errorf("counselor %s reported missing time slot %s but is actually assigned there",
							uc.CounselorID, tsID)
					}
				}
			}
		}
	})

	t.Run("counselor with no eligible slots is omitted", func(t *testing.T) {
		// One slot requires Lifeguard. Counselor lacks it — they have no
		// eligible time slots so the solver could not have placed them and
		// they should NOT appear in UnassignedCounselors.
		snapshot := ActivitySnapshot{
			Slots: []ActivitySlot{
				{
					ID: "slot-canoe", ActivityID: "act-canoe", ActivityName: "Canoeing",
					TimeSlotID: "ts-am", TimeSlotName: "Morning",
					RequiredCounselors: 1, Capacity: 1,
					RequiredCertifications: []string{"cert-lifeguard"},
				},
			},
			Counselors: []ActivityCounselor{
				{ID: "c-lg", Name: "Lifeguard", Certifications: map[string]bool{"cert-lifeguard": true}},
				{ID: "c-no", Name: "Nobody", Certifications: map[string]bool{}},
			},
			CertificationNames: map[string]string{"cert-lifeguard": "Lifeguard"},
		}

		solutions := SolveActivity(snapshot, DefaultActivitySolverConfig())
		if len(solutions) == 0 {
			t.Fatal("expected at least one solution")
		}

		for _, sol := range solutions {
			ids := []string{}
			for _, uc := range sol.UnassignedCounselors {
				ids = append(ids, uc.CounselorID)
			}
			if slices.Contains(ids, "c-no") {
				t.Errorf("counselor with no eligible slots should not be reported, got %v", ids)
			}
		}
	})
}
