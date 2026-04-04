package solver

import "fmt"

func CheckHardConstraints(snapshot SessionSnapshot, assignment Assignment) []Violation {
	var violations []Violation
	violations = append(violations, checkCabinMinimumCounselors(snapshot, assignment)...)
	violations = append(violations, checkCabinWithoutSeniorCounselor(snapshot, assignment)...)
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
	counselorsByID := make(map[string]Counselor, len(snapshot.Counselors))
	for _, c := range snapshot.Counselors {
		counselorsByID[c.ID] = c
	}

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
