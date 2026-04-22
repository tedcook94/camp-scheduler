package solver

import (
	"fmt"
	"sort"
)

func ScoreCamperSoftConstraints(snapshot CamperCabinSnapshot, assignment CamperAssignment, weights CamperWeights) ScoreResult {
	var result ScoreResult

	components := scoreFriendPreference(snapshot, assignment, weights)
	for _, c := range components {
		result.Total += c.Score
		result.Breakdown = append(result.Breakdown, c)
	}

	sort.Slice(result.Breakdown, func(i, j int) bool {
		a, b := result.Breakdown[i], result.Breakdown[j]
		if a.CamperID != b.CamperID {
			return a.CamperID < b.CamperID
		}
		if a.CabinID != b.CabinID {
			return a.CabinID < b.CabinID
		}
		if a.Constraint != b.Constraint {
			return a.Constraint < b.Constraint
		}
		return a.Message < b.Message
	})

	return result
}

func scoreFriendPreference(snapshot CamperCabinSnapshot, assignment CamperAssignment, weights CamperWeights) []ScoreComponent {
	if weights.FriendPreference == 0 {
		return nil
	}

	camperCabin := invertCamperAssignment(assignment)
	cabinsByID := indexCamperCabins(snapshot)
	campersByID := indexCampersByID(snapshot)

	var components []ScoreComponent
	for camperID, prefs := range snapshot.FriendPreferences {
		cabinID, ok := camperCabin[camperID]
		if !ok {
			continue
		}
		for _, pref := range prefs {
			if pref.Rank <= 0 {
				continue
			}
			prefCabinID, ok := camperCabin[pref.TargetID]
			if !ok {
				continue
			}
			if cabinID == prefCabinID {
				score := weights.FriendPreference / float64(pref.Rank)
				friendName := pref.TargetID
				if c, ok := campersByID[pref.TargetID]; ok {
					friendName = c.Name
				}
				cabinName := cabinID
				if cb, ok := cabinsByID[cabinID]; ok {
					cabinName = cb.Name
				}
				components = append(components, ScoreComponent{
					Constraint: "friend_preference",
					Score:      score,
					Rank:       pref.Rank,
					CamperID:   camperID,
					CabinID:    cabinID,
					Message: fmt.Sprintf(
						"placed with preferred friend %s (rank %d) in cabin %q",
						friendName, pref.Rank, cabinName,
					),
				})
			}
		}
	}
	return components
}

func invertCamperAssignment(assignment CamperAssignment) map[string]string {
	m := make(map[string]string)
	for cabinID, camperIDs := range assignment.CabinCampers {
		for _, camperID := range camperIDs {
			m[camperID] = cabinID
		}
	}
	return m
}
