package solver

import "sort"

// SolveCamperCabin assigns campers to cabins using a hybrid approach:
// preference-bearing campers are placed first via backtracking search,
// then remaining campers are distributed evenly across eligible cabins.
func SolveCamperCabin(snapshot CamperCabinSnapshot, config CamperSolverConfig) []CamperSolution {
	if config.MaxSolutions <= 0 || config.MaxIterations <= 0 {
		return nil
	}

	cabinsByAgeGroup := groupCabinsByAgeGroup(snapshot)
	withPrefs, withoutPrefs := partitionCampers(snapshot)

	s := &camperSearchState{
		snapshot:         snapshot,
		config:           config,
		cabinsByAgeGroup: cabinsByAgeGroup,
		assignment:       make(map[string][]string),
		cabinCount:       make(map[string]int),
	}

	for _, cabin := range snapshot.Cabins {
		s.assignment[cabin.ID] = nil
		s.cabinCount[cabin.ID] = 0
	}

	// Phase 1: backtracking search over preference-bearing campers.
	ordered := orderCampersByConstraints(withPrefs, snapshot)
	s.search(ordered, 0)

	if len(s.solutions) == 0 {
		// Backtracking couldn't place preference campers; fall back to even
		// distribution for ALL campers so nobody is silently omitted.
		allCampers := append(withPrefs, withoutPrefs...)
		base := s.cloneAssignment()
		filled := fillRemaining(base, allCampers, snapshot, cabinsByAgeGroup)
		score := ScoreCamperSoftConstraints(snapshot, filled, config.Weights)
		violations := CheckCamperHardConstraints(snapshot, filled)
		if len(violations) == 0 {
			s.solutions = append(s.solutions, CamperSolution{
				Assignment: filled,
				Score:      score,
			})
		}
	}

	// Phase 2: for each solution from phase 1, fill remaining campers.
	var filled []CamperSolution
	for _, sol := range s.solutions {
		completed := fillRemaining(sol.Assignment, withoutPrefs, snapshot, cabinsByAgeGroup)
		violations := CheckCamperHardConstraints(snapshot, completed)
		if len(violations) > 0 {
			continue
		}
		score := ScoreCamperSoftConstraints(snapshot, completed, config.Weights)
		filled = append(filled, CamperSolution{
			Assignment: completed,
			Score:      score,
		})
	}

	sort.Slice(filled, func(i, j int) bool {
		return filled[i].Score.Total > filled[j].Score.Total
	})

	if len(filled) > config.MaxSolutions {
		filled = filled[:config.MaxSolutions]
	}

	return filled
}

type camperSearchState struct {
	snapshot         CamperCabinSnapshot
	config           CamperSolverConfig
	cabinsByAgeGroup map[string][]CamperCabin
	assignment       map[string][]string
	cabinCount       map[string]int
	solutions        []CamperSolution
	iterations       int
}

func (s *camperSearchState) search(campers []Camper, index int) {
	if s.iterations >= s.config.MaxIterations {
		return
	}
	s.iterations++

	if index == len(campers) {
		s.evaluateSolution()
		return
	}

	camper := campers[index]
	eligibleCabins := s.cabinsByAgeGroup[camper.AgeGroupID]

	// Filter out cabins whose gender doesn't match the camper's.
	gendered := make([]CamperCabin, 0, len(eligibleCabins))
	for _, c := range eligibleCabins {
		if c.Gender == camper.Gender {
			gendered = append(gendered, c)
		}
	}
	eligibleCabins = gendered

	// Order cabins by most-remaining-capacity first so backtracking
	// explores balanced placements before lopsided ones.
	ordered := make([]CamperCabin, len(eligibleCabins))
	copy(ordered, eligibleCabins)
	sort.SliceStable(ordered, func(i, j int) bool {
		ri := ordered[i].Capacity - s.cabinCount[ordered[i].ID]
		rj := ordered[j].Capacity - s.cabinCount[ordered[j].ID]
		return ri > rj
	})

	for _, cabin := range ordered {
		if s.cabinCount[cabin.ID] >= cabin.Capacity {
			continue
		}

		s.assignment[cabin.ID] = append(s.assignment[cabin.ID], camper.ID)
		s.cabinCount[cabin.ID]++

		s.search(campers, index+1)

		s.assignment[cabin.ID] = s.assignment[cabin.ID][:len(s.assignment[cabin.ID])-1]
		s.cabinCount[cabin.ID]--

		if s.iterations >= s.config.MaxIterations {
			return
		}
	}
}

func (s *camperSearchState) evaluateSolution() {
	assignment := s.cloneAssignment()
	score := ScoreCamperSoftConstraints(s.snapshot, assignment, s.config.Weights)

	if len(s.solutions) < s.config.MaxSolutions {
		s.solutions = append(s.solutions, CamperSolution{
			Assignment: assignment,
			Score:      score,
		})
		return
	}

	worstIdx := 0
	for i, sol := range s.solutions {
		if sol.Score.Total < s.solutions[worstIdx].Score.Total {
			worstIdx = i
		}
	}
	if score.Total > s.solutions[worstIdx].Score.Total {
		s.solutions[worstIdx] = CamperSolution{
			Assignment: assignment,
			Score:      score,
		}
	}
}

func (s *camperSearchState) cloneAssignment() CamperAssignment {
	clone := make(map[string][]string, len(s.assignment))
	for cabinID, camperIDs := range s.assignment {
		if len(camperIDs) == 0 {
			continue
		}
		c := make([]string, len(camperIDs))
		copy(c, camperIDs)
		clone[cabinID] = c
	}
	return CamperAssignment{CabinCampers: clone}
}

func fillRemaining(base CamperAssignment, campers []Camper, snapshot CamperCabinSnapshot, cabinsByAgeGroup map[string][]CamperCabin) CamperAssignment {
	result := make(map[string][]string, len(base.CabinCampers))
	counts := make(map[string]int)
	capacities := make(map[string]int)

	for _, cabin := range snapshot.Cabins {
		capacities[cabin.ID] = cabin.Capacity
	}

	for cabinID, camperIDs := range base.CabinCampers {
		cp := make([]string, len(camperIDs))
		copy(cp, camperIDs)
		result[cabinID] = cp
		counts[cabinID] = len(cp)
	}

	for _, camper := range campers {
		cabins := cabinsByAgeGroup[camper.AgeGroupID]
		if len(cabins) == 0 {
			continue
		}

		// Pick the cabin with the most remaining capacity that matches the
		// camper's gender.
		bestCabin := ""
		bestRemaining := -1
		for _, cabin := range cabins {
			if cabin.Gender != camper.Gender {
				continue
			}
			remaining := capacities[cabin.ID] - counts[cabin.ID]
			if remaining > bestRemaining {
				bestRemaining = remaining
				bestCabin = cabin.ID
			}
		}

		if bestCabin != "" && bestRemaining > 0 {
			result[bestCabin] = append(result[bestCabin], camper.ID)
			counts[bestCabin]++
		}
	}

	return CamperAssignment{CabinCampers: result}
}

func groupCabinsByAgeGroup(snapshot CamperCabinSnapshot) map[string][]CamperCabin {
	groups := make(map[string][]CamperCabin)
	for _, cabin := range snapshot.Cabins {
		groups[cabin.AgeGroupID] = append(groups[cabin.AgeGroupID], cabin)
	}
	return groups
}

func partitionCampers(snapshot CamperCabinSnapshot) (withPrefs, withoutPrefs []Camper) {
	for _, c := range snapshot.Campers {
		if _, has := snapshot.FriendPreferences[c.ID]; has {
			withPrefs = append(withPrefs, c)
		} else {
			withoutPrefs = append(withoutPrefs, c)
		}
	}
	return
}

func orderCampersByConstraints(campers []Camper, snapshot CamperCabinSnapshot) []Camper {
	type camperPriority struct {
		camper      Camper
		constraints int
	}

	priorities := make([]camperPriority, len(campers))
	for i, c := range campers {
		constraints := 0
		if prefs, ok := snapshot.FriendPreferences[c.ID]; ok {
			constraints += len(prefs) * 10
		}
		priorities[i] = camperPriority{camper: c, constraints: constraints}
	}

	sort.SliceStable(priorities, func(i, j int) bool {
		return priorities[i].constraints > priorities[j].constraints
	})

	result := make([]Camper, len(priorities))
	for i, p := range priorities {
		result[i] = p.camper
	}
	return result
}

// UnassignableCampers reports campers that cannot be placed in any cabin
// even via greedy fill against the supplied snapshot. Intended as a
// diagnostic when SolveCabin returns no solutions: it surfaces which
// specific campers (by ID and name) are blocked by capacity or by having
// no eligible cabin in their age group / gender.
func UnassignableCampers(snapshot CamperCabinSnapshot) []Camper {
	cabinsByAgeGroup := groupCabinsByAgeGroup(snapshot)
	filled := fillRemaining(CamperAssignment{CabinCampers: map[string][]string{}}, snapshot.Campers, snapshot, cabinsByAgeGroup)

	assigned := make(map[string]bool)
	for _, ids := range filled.CabinCampers {
		for _, id := range ids {
			assigned[id] = true
		}
	}

	var unassigned []Camper
	for _, c := range snapshot.Campers {
		if !assigned[c.ID] {
			unassigned = append(unassigned, c)
		}
	}
	return unassigned
}
