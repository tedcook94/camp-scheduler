package solver

import "fmt"

func ScoreSoftConstraints(snapshot SessionSnapshot, assignment Assignment, weights Weights) ScoreResult {
	var result ScoreResult

	scorers := []func(SessionSnapshot, Assignment, Weights) []ScoreComponent{
		scoreReturningAgeGroup,
		scoreReturningCabin,
		scoreCocounselorPreference,
		scoreAgeGroupPreference,
		scoreMultipleSeniors,
	}

	for _, scorer := range scorers {
		components := scorer(snapshot, assignment, weights)
		for _, c := range components {
			result.Total += c.Score
			result.Breakdown = append(result.Breakdown, c)
		}
	}

	return result
}

func scoreReturningAgeGroup(snapshot SessionSnapshot, assignment Assignment, weights Weights) []ScoreComponent {
	if weights.ReturningAgeGroup == 0 {
		return nil
	}

	cabinsByID := indexCabins(snapshot)
	counselorCabin := invertAssignment(assignment)

	var components []ScoreComponent
	for counselorID, placement := range snapshot.CounselorPreviousPlacements {
		if !counselorWantsToReturn(counselorID, snapshot) {
			continue
		}
		cabinID, ok := counselorCabin[counselorID]
		if !ok {
			continue
		}
		cabin := cabinsByID[cabinID]
		if cabin.AgeGroupID == placement.AgeGroupID {
			components = append(components, ScoreComponent{
				Constraint:  "returning_age_group",
				Score:       weights.ReturningAgeGroup,
				CounselorID: counselorID,
				CabinID:     cabinID,
				Message:     fmt.Sprintf("counselor returning to same age group in cabin %q", cabin.Name),
			})
		}
	}
	return components
}

func scoreReturningCabin(snapshot SessionSnapshot, assignment Assignment, weights Weights) []ScoreComponent {
	if weights.ReturningCabin == 0 {
		return nil
	}

	cabinsByID := indexCabins(snapshot)
	counselorCabin := invertAssignment(assignment)

	var components []ScoreComponent
	for counselorID, placement := range snapshot.CounselorPreviousPlacements {
		if placement.CabinID == "" {
			continue
		}
		if !counselorWantsToReturn(counselorID, snapshot) {
			continue
		}
		cabinID, ok := counselorCabin[counselorID]
		if !ok {
			continue
		}
		if cabinID == placement.CabinID {
			cabin := cabinsByID[cabinID]
			components = append(components, ScoreComponent{
				Constraint:  "returning_cabin",
				Score:       weights.ReturningCabin,
				CounselorID: counselorID,
				CabinID:     cabinID,
				Message:     fmt.Sprintf("counselor returning to same cabin %q", cabin.Name),
			})
		}
	}
	return components
}

func scoreCocounselorPreference(snapshot SessionSnapshot, assignment Assignment, weights Weights) []ScoreComponent {
	if weights.CocounselorPreference == 0 {
		return nil
	}

	counselorCabin := invertAssignment(assignment)

	var components []ScoreComponent
	for counselorID, prefs := range snapshot.CocounselorPreferences {
		cabinID, ok := counselorCabin[counselorID]
		if !ok {
			continue
		}
		for _, pref := range prefs {
			if pref.Rank <= 0 {
				continue
			}
			prefCabinID, ok := counselorCabin[pref.TargetID]
			if !ok {
				continue
			}
			if cabinID == prefCabinID {
				score := weights.CocounselorPreference / float64(pref.Rank)
				components = append(components, ScoreComponent{
					Constraint:  "cocounselor_preference",
					Score:       score,
					CounselorID: counselorID,
					CabinID:     cabinID,
					Message: fmt.Sprintf(
						"paired with preferred co-counselor (rank %d)",
						pref.Rank,
					),
				})
			}
		}
	}
	return components
}

func scoreAgeGroupPreference(snapshot SessionSnapshot, assignment Assignment, weights Weights) []ScoreComponent {
	if weights.AgeGroupPreference == 0 {
		return nil
	}

	cabinsByID := indexCabins(snapshot)
	counselorCabin := invertAssignment(assignment)

	var components []ScoreComponent
	for counselorID, prefs := range snapshot.AgeGroupPreferences {
		cabinID, ok := counselorCabin[counselorID]
		if !ok {
			continue
		}
		cabin := cabinsByID[cabinID]
		for _, pref := range prefs {
			if pref.Rank <= 0 {
				continue
			}
			if cabin.AgeGroupID == pref.TargetID {
				score := weights.AgeGroupPreference / float64(pref.Rank)
				components = append(components, ScoreComponent{
					Constraint:  "age_group_preference",
					Score:       score,
					CounselorID: counselorID,
					CabinID:     cabinID,
					Message: fmt.Sprintf(
						"assigned to preferred age group (rank %d) in cabin %q",
						pref.Rank, cabin.Name,
					),
				})
				break
			}
		}
	}
	return components
}

func scoreMultipleSeniors(snapshot SessionSnapshot, assignment Assignment, weights Weights) []ScoreComponent {
	if weights.MultipleSeniors == 0 {
		return nil
	}

	counselorsByID := indexCounselors(snapshot)

	var components []ScoreComponent
	for _, cabin := range snapshot.Cabins {
		counselorIDs := assignment.CabinCounselors[cabin.ID]
		seniorCount := 0
		for _, cID := range counselorIDs {
			if !counselorsByID[cID].IsJunior {
				seniorCount++
			}
		}
		if seniorCount >= 2 {
			components = append(components, ScoreComponent{
				Constraint: "multiple_seniors",
				Score:      weights.MultipleSeniors,
				CabinID:    cabin.ID,
				Message: fmt.Sprintf(
					"cabin %q has %d senior counselors",
					cabin.Name, seniorCount,
				),
			})
		}
	}
	return components
}

// Returns true when the counselor has no age-group preferences or when their
// rank-1 preference matches their previous age group.
func counselorWantsToReturn(counselorID string, snapshot SessionSnapshot) bool {
	prefs, hasPrefs := snapshot.AgeGroupPreferences[counselorID]
	if !hasPrefs {
		return true
	}

	placement, hasPlacement := snapshot.CounselorPreviousPlacements[counselorID]
	if !hasPlacement {
		return false
	}

	for _, p := range prefs {
		if p.Rank == 1 {
			return p.TargetID == placement.AgeGroupID
		}
	}
	return false
}
