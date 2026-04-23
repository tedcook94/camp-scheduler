package solver

import (
	"fmt"
	"sort"
)

func ExplainCamper(snapshot CamperCabinSnapshot, solution CamperSolution) CamperExplanation {
	camperCabin := invertCamperAssignment(solution.Assignment)
	campersByID := indexCampersByID(snapshot)

	reasonsByCamper := buildCamperReasonMap(solution.Score.Breakdown)

	var assignments []CamperAssignmentExplanation
	for _, camper := range snapshot.Campers {
		cabinID, assigned := camperCabin[camper.ID]
		if !assigned {
			continue
		}

		reasons := reasonsByCamper[camper.ID]
		if len(reasons) == 0 {
			continue
		}

		assignments = append(assignments, CamperAssignmentExplanation{
			CamperID: camper.ID,
			CabinID:  cabinID,
			Reasons:  reasons,
		})
	}

	var unmet []CamperUnmetPreference
	var ineligible []CamperIneligiblePreference
	unmetFriend, ineligibleFriend := findUnmetFriendPreferences(snapshot, camperCabin, campersByID)
	unmet = append(unmet, unmetFriend...)
	ineligible = append(ineligible, ineligibleFriend...)

	sort.Slice(unmet, func(i, j int) bool {
		if unmet[i].CamperID != unmet[j].CamperID {
			return unmet[i].CamperID < unmet[j].CamperID
		}
		if unmet[i].Constraint != unmet[j].Constraint {
			return unmet[i].Constraint < unmet[j].Constraint
		}
		return unmet[i].Message < unmet[j].Message
	})

	sort.Slice(ineligible, func(i, j int) bool {
		if ineligible[i].CamperID != ineligible[j].CamperID {
			return ineligible[i].CamperID < ineligible[j].CamperID
		}
		if ineligible[i].Constraint != ineligible[j].Constraint {
			return ineligible[i].Constraint < ineligible[j].Constraint
		}
		return ineligible[i].Message < ineligible[j].Message
	})

	return CamperExplanation{
		Assignments:           assignments,
		UnmetPreferences:      unmet,
		IneligiblePreferences: ineligible,
	}
}

func buildCamperReasonMap(breakdown []ScoreComponent) map[string][]AssignmentReason {
	reasons := make(map[string][]AssignmentReason)
	for _, c := range breakdown {
		if c.CamperID == "" {
			continue
		}
		reasons[c.CamperID] = append(reasons[c.CamperID], AssignmentReason{
			Constraint: c.Constraint,
			Rank:       c.Rank,
			Message:    fmt.Sprintf("%s (+%.1f)", c.Message, c.Score),
		})
	}
	return reasons
}

func findUnmetFriendPreferences(snapshot CamperCabinSnapshot, camperCabin map[string]string, campersByID map[string]Camper) ([]CamperUnmetPreference, []CamperIneligiblePreference) {
	var unmet []CamperUnmetPreference
	var ineligible []CamperIneligiblePreference
	for camperID, prefs := range snapshot.FriendPreferences {
		source, sourceKnown := campersByID[camperID]
		if !sourceKnown {
			continue
		}

		cabinID, assigned := camperCabin[camperID]

		for _, pref := range prefs {
			target, targetKnown := campersByID[pref.TargetID]

			if !targetKnown {
				ineligible = append(ineligible, CamperIneligiblePreference{
					CamperID:   camperID,
					Constraint: "friend_preference_ineligible",
					Rank:       pref.Rank,
					Message: fmt.Sprintf(
						"preferred friend %q is not enrolled in this session",
						pref.TargetID,
					),
				})
				continue
			}

			if source.Gender != "" && target.Gender != "" && source.Gender != target.Gender {
				ineligible = append(ineligible, CamperIneligiblePreference{
					CamperID:   camperID,
					Constraint: "friend_preference_ineligible",
					Rank:       pref.Rank,
					Message: fmt.Sprintf(
						"preferred friend %q cannot share a cabin (gender mismatch)",
						target.Name,
					),
				})
				continue
			}

			if source.AgeGroupID != "" && target.AgeGroupID != "" && source.AgeGroupID != target.AgeGroupID {
				ineligible = append(ineligible, CamperIneligiblePreference{
					CamperID:   camperID,
					Constraint: "friend_preference_ineligible",
					Rank:       pref.Rank,
					Message: fmt.Sprintf(
						"preferred friend %q cannot share a cabin (different age group)",
						target.Name,
					),
				})
				continue
			}

			if !assigned {
				continue
			}

			prefCabinID, prefAssigned := camperCabin[pref.TargetID]
			if !prefAssigned || prefCabinID != cabinID {
				unmet = append(unmet, CamperUnmetPreference{
					CamperID:   camperID,
					Constraint: "friend_preference",
					Rank:       pref.Rank,
					Message: fmt.Sprintf(
						"preferred friend %q (rank %d) but placed in different cabins",
						target.Name, pref.Rank,
					),
				})
			}
		}
	}
	return unmet, ineligible
}
