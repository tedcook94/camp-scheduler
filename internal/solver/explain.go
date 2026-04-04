package solver

import "fmt"

func Explain(snapshot SessionSnapshot, solution Solution) Explanation {
	cabinsByID := indexCabins(snapshot)
	counselorCabin := invertAssignment(solution.Assignment)
	counselorsByID := indexCounselors(snapshot)

	reasonsByCounselor := buildReasonMap(solution.Score.Breakdown)

	var assignments []AssignmentExplanation
	for _, counselor := range snapshot.Counselors {
		cabinID, assigned := counselorCabin[counselor.ID]
		if !assigned {
			continue
		}

		reasons := reasonsByCounselor[counselor.ID]
		if len(reasons) == 0 {
			reasons = []string{"assigned to fill cabin requirement"}
		}

		assignments = append(assignments, AssignmentExplanation{
			CounselorID: counselor.ID,
			CabinID:     cabinID,
			Reasons:     reasons,
		})
	}

	var unmet []UnmetPreference
	unmet = append(unmet, findUnmetAgeGroupPreferences(snapshot, counselorCabin, cabinsByID)...)
	unmet = append(unmet, findUnmetReturningAgeGroup(snapshot, counselorCabin, cabinsByID)...)
	unmet = append(unmet, findUnmetReturningCabin(snapshot, counselorCabin, cabinsByID)...)
	unmet = append(unmet, findUnmetCocounselorPreferences(snapshot, counselorCabin, counselorsByID)...)

	return Explanation{
		Assignments:      assignments,
		UnmetPreferences: unmet,
	}
}

func buildReasonMap(breakdown []ScoreComponent) map[string][]string {
	reasons := make(map[string][]string)
	for _, c := range breakdown {
		if c.CounselorID == "" {
			continue
		}
		reason := fmt.Sprintf("%s (+%.1f)", c.Message, c.Score)
		reasons[c.CounselorID] = append(reasons[c.CounselorID], reason)
	}
	return reasons
}

func findUnmetAgeGroupPreferences(snapshot SessionSnapshot, counselorCabin map[string]string, cabinsByID map[string]Cabin) []UnmetPreference {
	var unmet []UnmetPreference
	for counselorID, prefs := range snapshot.AgeGroupPreferences {
		cabinID, assigned := counselorCabin[counselorID]
		if !assigned {
			continue
		}

		cabin := cabinsByID[cabinID]
		assignedAgeGroup := cabin.AgeGroupID

		var topPrefAgeGroup string
		for _, p := range prefs {
			if p.Rank == 1 {
				topPrefAgeGroup = p.TargetID
				break
			}
		}

		if assignedAgeGroup == topPrefAgeGroup {
			continue
		}

		var rankMsg string
		for _, p := range prefs {
			if p.TargetID == assignedAgeGroup {
				rankMsg = fmt.Sprintf("rank %d of %d preferences", p.Rank, len(prefs))
				break
			}
		}
		if rankMsg == "" {
			rankMsg = "not ranked"
		}

		unmet = append(unmet, UnmetPreference{
			CounselorID: counselorID,
			Constraint:  "age_group_preference",
			Message: fmt.Sprintf(
				"preferred age group %q but assigned to age group %q (%s)",
				topPrefAgeGroup, assignedAgeGroup, rankMsg,
			),
		})
	}
	return unmet
}

func findUnmetReturningAgeGroup(snapshot SessionSnapshot, counselorCabin map[string]string, cabinsByID map[string]Cabin) []UnmetPreference {
	var unmet []UnmetPreference
	for counselorID, placement := range snapshot.CounselorPreviousPlacements {
		if !counselorWantsToReturn(counselorID, snapshot) {
			continue
		}

		cabinID, assigned := counselorCabin[counselorID]
		if !assigned {
			continue
		}

		cabin := cabinsByID[cabinID]
		if cabin.AgeGroupID != placement.AgeGroupID {
			unmet = append(unmet, UnmetPreference{
				CounselorID: counselorID,
				Constraint:  "returning_age_group",
				Message: fmt.Sprintf(
					"previously in age group %q but assigned to age group %q",
					placement.AgeGroupID, cabin.AgeGroupID,
				),
			})
		}
	}
	return unmet
}

func findUnmetReturningCabin(snapshot SessionSnapshot, counselorCabin map[string]string, cabinsByID map[string]Cabin) []UnmetPreference {
	var unmet []UnmetPreference
	for counselorID, placement := range snapshot.CounselorPreviousPlacements {
		if placement.CabinID == "" {
			continue
		}
		if !counselorWantsToReturn(counselorID, snapshot) {
			continue
		}

		cabinID, assigned := counselorCabin[counselorID]
		if !assigned {
			continue
		}

		if cabinID != placement.CabinID {
			prevCabin := cabinsByID[placement.CabinID]
			assignedCabin := cabinsByID[cabinID]
			unmet = append(unmet, UnmetPreference{
				CounselorID: counselorID,
				Constraint:  "returning_cabin",
				Message: fmt.Sprintf(
					"previously in cabin %q but assigned to cabin %q",
					prevCabin.Name, assignedCabin.Name,
				),
			})
		}
	}
	return unmet
}

func findUnmetCocounselorPreferences(snapshot SessionSnapshot, counselorCabin map[string]string, counselorsByID map[string]Counselor) []UnmetPreference {
	var unmet []UnmetPreference
	for counselorID, prefs := range snapshot.CocounselorPreferences {
		cabinID, assigned := counselorCabin[counselorID]
		if !assigned {
			continue
		}

		for _, pref := range prefs {
			prefCabinID, prefAssigned := counselorCabin[pref.TargetID]
			if !prefAssigned || prefCabinID != cabinID {
				prefName := counselorsByID[pref.TargetID].Name
				unmet = append(unmet, UnmetPreference{
					CounselorID: counselorID,
					Constraint:  "cocounselor_preference",
					Message: fmt.Sprintf(
						"preferred co-counselor %q but placed in different cabins",
						prefName,
					),
				})
			}
		}
	}
	return unmet
}
