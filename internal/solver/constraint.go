package solver

import "fmt"

func CheckHardConstraints(snapshot SessionSnapshot, assignment Assignment) []Violation {
	var violations []Violation
	violations = append(violations, checkCabinMinimumCounselors(snapshot, assignment)...)
	violations = append(violations, checkCabinWithoutSeniorCounselor(snapshot, assignment)...)
	violations = append(violations, checkCabinGenderMatchCounselors(snapshot, assignment)...)
	violations = append(violations, checkCabinCounselorCapacity(snapshot, assignment)...)
	return violations
}

// checkCabinCounselorCapacity verifies that counselors alone do not exceed
// the cabin's total occupancy capacity. The combined cabin solver further
// enforces that counselors + campers stay within capacity per cabin.
func checkCabinCounselorCapacity(snapshot SessionSnapshot, assignment Assignment) []Violation {
	var violations []Violation
	for _, cabin := range snapshot.Cabins {
		assigned := len(assignment.CabinCounselors[cabin.ID])
		if assigned > cabin.Capacity {
			violations = append(violations, Violation{
				Constraint: "cabin_capacity",
				CabinID:    cabin.ID,
				Message: fmt.Sprintf(
					"cabin %q has %d counselor(s) assigned but capacity is %d",
					cabin.Name, assigned, cabin.Capacity,
				),
			})
		}
	}
	return violations
}

func checkCabinMinimumCounselors(snapshot SessionSnapshot, assignment Assignment) []Violation {
	var violations []Violation
	for _, cabin := range snapshot.Cabins {
		assigned := len(assignment.CabinCounselors[cabin.ID])
		if assigned < cabin.RequiredCounselors {
			violations = append(violations, Violation{
				Constraint: "cabin_minimum_counselors",
				CabinID:    cabin.ID,
				Message: fmt.Sprintf(
					"cabin %q has %d counselor(s) assigned but requires at least %d",
					cabin.Name, assigned, cabin.RequiredCounselors,
				),
			})
		}
	}
	return violations
}

// checkCabinWithoutSeniorCounselor verifies that every staffed cabin has at
// least one non-junior counselor. A cabin with zero counselors is not flagged
// here — that's caught by the minimum counselors check.
func checkCabinWithoutSeniorCounselor(snapshot SessionSnapshot, assignment Assignment) []Violation {
	counselorsByID := indexCounselors(snapshot)

	var violations []Violation
	for _, cabin := range snapshot.Cabins {
		counselorIDs := assignment.CabinCounselors[cabin.ID]
		if len(counselorIDs) == 0 {
			continue
		}

		hasSenior := false
		for _, cID := range counselorIDs {
			if !counselorsByID[cID].IsJunior {
				hasSenior = true
				break
			}
		}

		if !hasSenior {
			violations = append(violations, Violation{
				Constraint: "cabin_without_senior_counselor",
				CabinID:    cabin.ID,
				Message: fmt.Sprintf(
					"cabin %q has no senior counselor assigned",
					cabin.Name,
				),
			})
		}
	}
	return violations
}

// checkCabinGenderMatchCounselors verifies that every assigned counselor's
// gender matches the cabin's gender. Cabins are gender-segregated so a
// counselor can only staff a cabin of their own gender.
func checkCabinGenderMatchCounselors(snapshot SessionSnapshot, assignment Assignment) []Violation {
	counselorsByID := indexCounselors(snapshot)

	var violations []Violation
	for _, cabin := range snapshot.Cabins {
		for _, cID := range assignment.CabinCounselors[cabin.ID] {
			counselor := counselorsByID[cID]
			if counselor.Gender != cabin.Gender {
				violations = append(violations, Violation{
					Constraint:  "cabin_gender_mismatch",
					CabinID:     cabin.ID,
					CounselorID: cID,
					Message: fmt.Sprintf(
						"counselor %q (%s) cannot be assigned to cabin %q (%s)",
						counselor.Name, counselor.Gender, cabin.Name, cabin.Gender,
					),
				})
			}
		}
	}
	return violations
}
