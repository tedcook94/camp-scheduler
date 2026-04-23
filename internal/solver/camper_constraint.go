package solver

import "fmt"

func CheckCamperHardConstraints(snapshot CamperCabinSnapshot, assignment CamperAssignment) []Violation {
	var violations []Violation
	violations = append(violations, checkCabinCapacity(snapshot, assignment)...)
	violations = append(violations, checkCamperAgeGroupMatch(snapshot, assignment)...)
	violations = append(violations, checkCabinGenderMatchCampers(snapshot, assignment)...)
	violations = append(violations, checkAllCampersAssigned(snapshot, assignment)...)
	return violations
}

func checkCabinCapacity(snapshot CamperCabinSnapshot, assignment CamperAssignment) []Violation {
	var violations []Violation
	for _, cabin := range snapshot.Cabins {
		assigned := len(assignment.CabinCampers[cabin.ID])
		if assigned > cabin.Capacity {
			violations = append(violations, Violation{
				Constraint: "cabin_capacity",
				CabinID:    cabin.ID,
				Message: fmt.Sprintf(
					"cabin %q has %d campers assigned but capacity is %d",
					cabin.Name, assigned, cabin.Capacity,
				),
			})
		}
	}
	return violations
}

func checkCamperAgeGroupMatch(snapshot CamperCabinSnapshot, assignment CamperAssignment) []Violation {
	cabinsByID := indexCamperCabins(snapshot)
	campersByID := indexCampersByID(snapshot)

	var violations []Violation
	for cabinID, camperIDs := range assignment.CabinCampers {
		cabin := cabinsByID[cabinID]
		for _, camperID := range camperIDs {
			camper := campersByID[camperID]
			if camper.AgeGroupID != cabin.AgeGroupID {
				violations = append(violations, Violation{
					Constraint: "camper_age_group_mismatch",
					CabinID:    cabinID,
					CamperID:   camperID,
					Message: fmt.Sprintf(
						"camper %q is in age group %q but assigned to cabin %q in age group %q",
						camper.Name, camper.AgeGroupID, cabin.Name, cabin.AgeGroupID,
					),
				})
			}
		}
	}
	return violations
}

// checkCabinGenderMatchCampers verifies every assigned camper's gender
// matches the cabin's gender.
func checkCabinGenderMatchCampers(snapshot CamperCabinSnapshot, assignment CamperAssignment) []Violation {
	cabinsByID := indexCamperCabins(snapshot)
	campersByID := indexCampersByID(snapshot)

	var violations []Violation
	for cabinID, camperIDs := range assignment.CabinCampers {
		cabin := cabinsByID[cabinID]
		for _, camperID := range camperIDs {
			camper := campersByID[camperID]
			if camper.Gender != cabin.Gender {
				violations = append(violations, Violation{
					Constraint: "cabin_gender_mismatch",
					CabinID:    cabinID,
					CamperID:   camperID,
					Message: fmt.Sprintf(
						"camper %q (%s) cannot be assigned to cabin %q (%s)",
						camper.Name, camper.Gender, cabin.Name, cabin.Gender,
					),
				})
			}
		}
	}
	return violations
}

func checkAllCampersAssigned(snapshot CamperCabinSnapshot, assignment CamperAssignment) []Violation {
	assigned := make(map[string]bool)
	for _, camperIDs := range assignment.CabinCampers {
		for _, id := range camperIDs {
			assigned[id] = true
		}
	}

	var violations []Violation
	for _, camper := range snapshot.Campers {
		if !assigned[camper.ID] {
			violations = append(violations, Violation{
				Constraint: "camper_unassigned",
				Message: fmt.Sprintf(
					"camper %q is enrolled but not assigned to any cabin",
					camper.Name,
				),
			})
		}
	}
	return violations
}

func indexCamperCabins(snapshot CamperCabinSnapshot) map[string]CamperCabin {
	m := make(map[string]CamperCabin, len(snapshot.Cabins))
	for _, c := range snapshot.Cabins {
		m[c.ID] = c
	}
	return m
}

func indexCampersByID(snapshot CamperCabinSnapshot) map[string]Camper {
	m := make(map[string]Camper, len(snapshot.Campers))
	for _, c := range snapshot.Campers {
		m[c.ID] = c
	}
	return m
}
