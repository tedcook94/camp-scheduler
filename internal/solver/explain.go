package solver

import (
	"fmt"
	"sort"
)

func Explain(snapshot SessionSnapshot, solution Solution) Explanation {
	cabinsByID := indexCabins(snapshot)
	ageGroupNames := indexAgeGroupNames(snapshot)
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
			continue
		}

		assignments = append(assignments, AssignmentExplanation{
			CounselorID: counselor.ID,
			CabinID:     cabinID,
			Reasons:     reasons,
		})
	}

	var unmet []UnmetPreference
	unmet = append(unmet, findUnmetAgeGroupPreferences(snapshot, counselorCabin, cabinsByID, ageGroupNames)...)
	unmet = append(unmet, findUnmetReturningAgeGroup(snapshot, counselorCabin, cabinsByID, ageGroupNames)...)
	unmet = append(unmet, findUnmetReturningCabin(snapshot, counselorCabin, cabinsByID)...)

	rosterSet := buildRosterSet(snapshot.Counselors)
	unmetCo, ineligibleCo := findUnmetCocounselorPreferences(snapshot, counselorCabin, counselorsByID, rosterSet)
	unmet = append(unmet, unmetCo...)

	sort.Slice(unmet, func(i, j int) bool {
		if unmet[i].CounselorID != unmet[j].CounselorID {
			return unmet[i].CounselorID < unmet[j].CounselorID
		}
		if unmet[i].Constraint != unmet[j].Constraint {
			return unmet[i].Constraint < unmet[j].Constraint
		}
		return unmet[i].Message < unmet[j].Message
	})

	sort.Slice(ineligibleCo, func(i, j int) bool {
		if ineligibleCo[i].CounselorID != ineligibleCo[j].CounselorID {
			return ineligibleCo[i].CounselorID < ineligibleCo[j].CounselorID
		}
		if ineligibleCo[i].Constraint != ineligibleCo[j].Constraint {
			return ineligibleCo[i].Constraint < ineligibleCo[j].Constraint
		}
		return ineligibleCo[i].Message < ineligibleCo[j].Message
	})

	return Explanation{
		Assignments:           assignments,
		UnmetPreferences:      unmet,
		IneligiblePreferences: ineligibleCo,
	}
}

func indexAgeGroupNames(snapshot SessionSnapshot) map[string]string {
	names := make(map[string]string)
	for _, c := range snapshot.Cabins {
		if c.AgeGroupID != "" && c.AgeGroupName != "" {
			names[c.AgeGroupID] = c.AgeGroupName
		}
	}
	return names
}

func ageGroupLabel(id string, names map[string]string) string {
	if name, ok := names[id]; ok {
		return name
	}
	return id
}

func buildReasonMap(breakdown []ScoreComponent) map[string][]AssignmentReason {
	reasons := make(map[string][]AssignmentReason)
	for _, c := range breakdown {
		if c.CounselorID == "" {
			continue
		}
		reasons[c.CounselorID] = append(reasons[c.CounselorID], AssignmentReason{
			Constraint: c.Constraint,
			Rank:       c.Rank,
			Message:    fmt.Sprintf("%s (+%.1f)", c.Message, c.Score),
		})
	}
	return reasons
}

func findUnmetAgeGroupPreferences(snapshot SessionSnapshot, counselorCabin map[string]string, cabinsByID map[string]Cabin, ageGroupNames map[string]string) []UnmetPreference {
	var unmet []UnmetPreference
	for counselorID, prefs := range snapshot.AgeGroupPreferences {
		cabinID, assigned := counselorCabin[counselorID]

		var assignedAgeGroup string
		if assigned {
			assignedAgeGroup = cabinsByID[cabinID].AgeGroupID
		}

		var topPref RankedPreference
		for _, p := range prefs {
			if p.Rank == 1 {
				topPref = p
				break
			}
		}

		if assignedAgeGroup == topPref.TargetID {
			continue
		}

		var detail string
		if !assigned {
			detail = "not assigned to any cabin"
		} else {
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
			detail = fmt.Sprintf("assigned to age group %q (%s)", ageGroupLabel(assignedAgeGroup, ageGroupNames), rankMsg)
		}

		unmet = append(unmet, UnmetPreference{
			CounselorID: counselorID,
			Constraint:  "age_group_preference",
			Rank:        topPref.Rank,
			Message: fmt.Sprintf(
				"preferred age group %q but %s",
				ageGroupLabel(topPref.TargetID, ageGroupNames), detail,
			),
		})
	}
	return unmet
}

func findUnmetReturningAgeGroup(snapshot SessionSnapshot, counselorCabin map[string]string, cabinsByID map[string]Cabin, ageGroupNames map[string]string) []UnmetPreference {
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
					ageGroupLabel(placement.AgeGroupID, ageGroupNames),
					ageGroupLabel(cabin.AgeGroupID, ageGroupNames),
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

func findUnmetCocounselorPreferences(snapshot SessionSnapshot, counselorCabin map[string]string, counselorsByID map[string]Counselor, rosterSet map[string]bool) ([]UnmetPreference, []IneligiblePreference) {
	var unmet []UnmetPreference
	var ineligible []IneligiblePreference
	for counselorID, prefs := range snapshot.CocounselorPreferences {
		source, sourceKnown := counselorsByID[counselorID]
		cabinID := counselorCabin[counselorID]

		for _, pref := range prefs {
			target, targetKnown := counselorsByID[pref.TargetID]

			if !targetKnown || !rosterSet[pref.TargetID] {
				prefName := pref.TargetID
				if targetKnown && target.Name != "" {
					prefName = target.Name
				}
				ineligible = append(ineligible, IneligiblePreference{
					CounselorID: counselorID,
					Constraint:  "cocounselor_preference_ineligible",
					Rank:        pref.Rank,
					Message: fmt.Sprintf(
						"preferred co-counselor %q is not on this session's roster",
						prefName,
					),
				})
				continue
			}

			if sourceKnown && source.Gender != "" && target.Gender != "" && source.Gender != target.Gender {
				ineligible = append(ineligible, IneligiblePreference{
					CounselorID: counselorID,
					Constraint:  "cocounselor_preference_ineligible",
					Rank:        pref.Rank,
					Message: fmt.Sprintf(
						"preferred co-counselor %q cannot share a cabin (gender mismatch)",
						target.Name,
					),
				})
				continue
			}

			prefCabinID := counselorCabin[pref.TargetID]
			if cabinID == "" || prefCabinID == "" || prefCabinID != cabinID {
				unmet = append(unmet, UnmetPreference{
					CounselorID: counselorID,
					Constraint:  "cocounselor_preference",
					Rank:        pref.Rank,
					Message: fmt.Sprintf(
						"preferred co-counselor %q but not placed together",
						target.Name,
					),
				})
			}
		}
	}
	return unmet, ineligible
}
