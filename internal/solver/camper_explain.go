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
	unmet = append(unmet, findUnmetFriendPreferences(snapshot, camperCabin, campersByID)...)

	sort.Slice(unmet, func(i, j int) bool {
		if unmet[i].CamperID != unmet[j].CamperID {
			return unmet[i].CamperID < unmet[j].CamperID
		}
		if unmet[i].Constraint != unmet[j].Constraint {
			return unmet[i].Constraint < unmet[j].Constraint
		}
		return unmet[i].Message < unmet[j].Message
	})

	return CamperExplanation{
		Assignments:      assignments,
		UnmetPreferences: unmet,
	}
}

func buildCamperReasonMap(breakdown []ScoreComponent) map[string][]string {
	reasons := make(map[string][]string)
	for _, c := range breakdown {
		if c.CamperID == "" {
			continue
		}
		reason := fmt.Sprintf("%s (+%.1f)", c.Message, c.Score)
		reasons[c.CamperID] = append(reasons[c.CamperID], reason)
	}
	return reasons
}

func findUnmetFriendPreferences(snapshot CamperCabinSnapshot, camperCabin map[string]string, campersByID map[string]Camper) []CamperUnmetPreference {
	var unmet []CamperUnmetPreference
	for camperID, prefs := range snapshot.FriendPreferences {
		cabinID, assigned := camperCabin[camperID]
		if !assigned {
			continue
		}

		for _, pref := range prefs {
			prefCabinID, prefAssigned := camperCabin[pref.TargetID]
			if !prefAssigned || prefCabinID != cabinID {
				prefName := pref.TargetID
				if c, ok := campersByID[pref.TargetID]; ok && c.Name != "" {
					prefName = c.Name
				}
				unmet = append(unmet, CamperUnmetPreference{
					CamperID:   camperID,
					Constraint: "friend_preference",
					Message: fmt.Sprintf(
						"preferred friend %q (rank %d) but placed in different cabins",
						prefName, pref.Rank,
					),
				})
			}
		}
	}
	return unmet
}
