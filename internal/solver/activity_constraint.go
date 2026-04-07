package solver

import (
	"fmt"
	"strings"
)

func CheckActivityHardConstraints(snapshot ActivitySnapshot, assignment ActivityAssignment) []Violation {
	var violations []Violation
	violations = append(violations, checkMinimumCounselors(snapshot, assignment)...)
	violations = append(violations, checkCounselorCertification(snapshot, assignment)...)
	violations = append(violations, checkTimeConflict(snapshot, assignment)...)
	violations = append(violations, checkActivityCapacity(snapshot, assignment)...)
	return violations
}

func checkMinimumCounselors(snapshot ActivitySnapshot, assignment ActivityAssignment) []Violation {
	var violations []Violation
	for _, slot := range snapshot.Slots {
		assigned := len(assignment.SlotCounselors[slot.ID])
		if assigned < slot.RequiredCounselors {
			violations = append(violations, Violation{
				Constraint: "minimum_counselors",
				Message: fmt.Sprintf(
					"activity %q (%s) has %d counselor(s) assigned but requires at least %d",
					slot.ActivityName, slot.TimeSlotName, assigned, slot.RequiredCounselors,
				),
			})
		}
	}
	return violations
}

func checkCounselorCertification(snapshot ActivitySnapshot, assignment ActivityAssignment) []Violation {
	counselorsByID := indexActivityCounselors(snapshot)

	var violations []Violation
	for _, slot := range snapshot.Slots {
		counselorIDs := assignment.SlotCounselors[slot.ID]
		for _, cID := range counselorIDs {
			counselor := counselorsByID[cID]
			for _, certID := range slot.RequiredCertifications {
				if !counselor.Certifications[certID] {
					violations = append(violations, Violation{
						Constraint:  "counselor_certification",
						CounselorID: cID,
						Message: fmt.Sprintf(
							"counselor %q lacks required certification for activity %q",
							counselor.Name, slot.ActivityName,
						),
					})
				}
			}
		}
	}
	return violations
}

func checkTimeConflict(snapshot ActivitySnapshot, assignment ActivityAssignment) []Violation {
	slotsByID := indexActivitySlots(snapshot)
	counselorsByID := indexActivityCounselors(snapshot)

	// Build counselor -> list of assigned slots grouped by time slot.
	type assignedSlot struct {
		slotID     string
		timeSlotID string
	}
	counselorAssigned := make(map[string][]assignedSlot)
	for slotID, counselorIDs := range assignment.SlotCounselors {
		slot := slotsByID[slotID]
		for _, cID := range counselorIDs {
			counselorAssigned[cID] = append(counselorAssigned[cID], assignedSlot{
				slotID:     slotID,
				timeSlotID: slot.TimeSlotID,
			})
		}
	}

	var violations []Violation
	for cID, assigned := range counselorAssigned {
		// Group by time slot to detect conflicts.
		byTimeSlot := make(map[string][]string) // timeSlotID -> []slotID
		for _, a := range assigned {
			byTimeSlot[a.timeSlotID] = append(byTimeSlot[a.timeSlotID], a.slotID)
		}
		for _, slotIDs := range byTimeSlot {
			if len(slotIDs) <= 1 {
				continue
			}
			names := make([]string, len(slotIDs))
			var timeSlotName string
			for i, sID := range slotIDs {
				s := slotsByID[sID]
				names[i] = s.ActivityName
				timeSlotName = s.TimeSlotName
			}
			counselor := counselorsByID[cID]
			violations = append(violations, Violation{
				Constraint:  "time_conflict",
				CounselorID: cID,
				Message: fmt.Sprintf(
					"counselor %q assigned to multiple activities (%s) in time slot %q",
					counselor.Name, strings.Join(names, ", "), timeSlotName,
				),
			})
		}
	}
	return violations
}

func checkActivityCapacity(snapshot ActivitySnapshot, assignment ActivityAssignment) []Violation {
	var violations []Violation
	for _, slot := range snapshot.Slots {
		assigned := len(assignment.SlotCounselors[slot.ID])
		if assigned > slot.Capacity {
			violations = append(violations, Violation{
				Constraint: "activity_capacity",
				Message: fmt.Sprintf(
					"activity %q (%s) has %d counselors assigned but capacity is %d",
					slot.ActivityName, slot.TimeSlotName, assigned, slot.Capacity,
				),
			})
		}
	}
	return violations
}

func indexActivityCounselors(snapshot ActivitySnapshot) map[string]ActivityCounselor {
	m := make(map[string]ActivityCounselor, len(snapshot.Counselors))
	for _, c := range snapshot.Counselors {
		m[c.ID] = c
	}
	return m
}

func indexActivitySlots(snapshot ActivitySnapshot) map[string]ActivitySlot {
	m := make(map[string]ActivitySlot, len(snapshot.Slots))
	for _, s := range snapshot.Slots {
		m[s.ID] = s
	}
	return m
}
