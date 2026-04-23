package solver

import "sort"

func Solve(snapshot SessionSnapshot, config SolverConfig) []Solution {
	if config.MaxSolutions <= 0 || config.MaxIterations <= 0 {
		return nil
	}

	counselorsByID := indexCounselors(snapshot)

	s := &searchState{
		snapshot:       snapshot,
		config:         config,
		counselorsByID: counselorsByID,
		cabinIDs:       make([]string, len(snapshot.Cabins)),
		cabinIndexByID: make(map[string]int, len(snapshot.Cabins)),
		assignment:     make(map[string][]string, len(snapshot.Cabins)),
		iterations:     0,
	}

	for i, c := range snapshot.Cabins {
		s.cabinIDs[i] = c.ID
		s.cabinIndexByID[c.ID] = i
	}

	counselors := orderCounselors(snapshot)
	s.search(counselors, 0)

	sort.Slice(s.solutions, func(i, j int) bool {
		return s.solutions[i].Score.Total > s.solutions[j].Score.Total
	})

	return s.solutions
}

type searchState struct {
	snapshot       SessionSnapshot
	config         SolverConfig
	counselorsByID map[string]Counselor
	cabinIDs       []string
	cabinIndexByID map[string]int
	assignment     map[string][]string
	solutions      []Solution
	iterations     int
}

func (s *searchState) search(counselors []Counselor, index int) {
	if s.iterations >= s.config.MaxIterations {
		return
	}
	s.iterations++

	if index == len(counselors) {
		s.evaluateSolution()
		return
	}

	remaining := counselors[index:]
	counselor := remaining[0]

	for _, cabinID := range s.cabinIDs {
		cabin := s.snapshot.Cabins[s.cabinIndexByID[cabinID]]
		if cabin.Gender != counselor.Gender {
			continue
		}
		if len(s.assignment[cabinID]) >= cabin.Capacity {
			continue
		}
		s.assignment[cabinID] = append(s.assignment[cabinID], counselor.ID)

		if s.feasible(remaining[1:]) {
			s.search(counselors, index+1)
		}

		s.assignment[cabinID] = s.assignment[cabinID][:len(s.assignment[cabinID])-1]

		if s.iterations >= s.config.MaxIterations {
			return
		}
	}

	// Also try not placing this counselor at all. The search must keep this
	// branch so it can explore configurations where dropping a low-value
	// counselor frees a slot for a higher-value one encountered later. The
	// post-search fill pass and unassigned-counselor scoring penalty handle
	// leftovers and bias ranking toward fuller solutions.
	if s.feasible(remaining[1:]) {
		s.search(counselors, index+1)
	}
}

// feasible checks whether the current partial assignment can still lead to a
// valid solution given the counselors remaining to be placed. It prunes
// branches that are guaranteed to violate hard constraints.
func (s *searchState) feasible(remaining []Counselor) bool {
	remainingSeniors := 0
	for _, c := range remaining {
		if !c.IsJunior {
			remainingSeniors++
		}
	}

	cabinsNeedingSenior := 0
	for _, cabin := range s.snapshot.Cabins {
		if !cabin.HasCampers {
			// No campers => no senior requirement; skip senior pruning.
			continue
		}
		assigned := s.assignment[cabin.ID]
		if len(assigned) == 0 {
			// Empty cabins with no staffing requirement can stay empty --
			// the hard constraint checker skips them for the senior check.
			if cabin.RequiredCounselors > 0 {
				cabinsNeedingSenior++
			}
			continue
		}

		hasSenior := false
		for _, cID := range assigned {
			if !s.counselorsByID[cID].IsJunior {
				hasSenior = true
				break
			}
		}

		if !hasSenior {
			// Cabin is staffed with only juniors -- it still needs a senior.
			cabinsNeedingSenior++
		}
	}

	if remainingSeniors < cabinsNeedingSenior {
		return false
	}

	// Check that each cabin with campers can still reach its
	// required_counselors count. Cabins without campers have no minimum.
	totalRemaining := len(remaining)
	deficit := 0
	for _, cabin := range s.snapshot.Cabins {
		if !cabin.HasCampers {
			continue
		}
		need := cabin.RequiredCounselors - len(s.assignment[cabin.ID])
		if need > 0 {
			deficit += need
		}
	}
	if totalRemaining < deficit {
		return false
	}

	return true
}

func (s *searchState) evaluateSolution() {
	assignment := s.cloneAssignment()
	assignment = fillRemainingCounselors(s.snapshot, assignment)

	violations := CheckHardConstraints(s.snapshot, assignment)
	if len(violations) > 0 {
		return
	}

	score := ScoreSoftConstraints(s.snapshot, assignment, s.config.Weights)

	if len(s.solutions) < s.config.MaxSolutions {
		s.solutions = append(s.solutions, Solution{
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
		s.solutions[worstIdx] = Solution{
			Assignment: assignment,
			Score:      score,
		}
	}
}

func (s *searchState) cloneAssignment() Assignment {
	clone := make(map[string][]string, len(s.assignment))
	for cabinID, counselorIDs := range s.assignment {
		if len(counselorIDs) == 0 {
			continue
		}
		c := make([]string, len(counselorIDs))
		copy(c, counselorIDs)
		clone[cabinID] = c
	}
	return Assignment{CabinCounselors: clone}
}

func orderCounselors(snapshot SessionSnapshot) []Counselor {
	type counselorPriority struct {
		counselor   Counselor
		constraints int
	}

	priorities := make([]counselorPriority, len(snapshot.Counselors))
	for i, c := range snapshot.Counselors {
		constraints := 0

		// Seniors are more constrained: they must fill the senior requirement.
		if !c.IsJunior {
			constraints += 100
		}

		if _, ok := snapshot.CounselorPreviousPlacements[c.ID]; ok {
			constraints += 10
		}
		if _, ok := snapshot.AgeGroupPreferences[c.ID]; ok {
			constraints += 10
		}
		if _, ok := snapshot.CocounselorPreferences[c.ID]; ok {
			constraints += 10
		}

		priorities[i] = counselorPriority{counselor: c, constraints: constraints}
	}

	sort.SliceStable(priorities, func(i, j int) bool {
		return priorities[i].constraints > priorities[j].constraints
	})

	result := make([]Counselor, len(priorities))
	for i, p := range priorities {
		result[i] = p.counselor
	}
	return result
}

// fillRemainingCounselors places any counselor not already in the assignment
// into a gender-matching cabin with remaining capacity. For each counselor,
// eligible cabins are considered greedily by age-group preference (best
// rank first), then by remaining counselor-slot capacity (highest first).
// The fill pass refuses placements that would create a hard-constraint
// violation (specifically, dropping a junior into a cabin with campers
// that has no senior present). A counselor with no eligible cabin is left
// out — the unassigned-counselor scoring penalty will surface that.
func fillRemainingCounselors(snapshot SessionSnapshot, assignment Assignment) Assignment {
	result := make(map[string][]string, len(assignment.CabinCounselors))
	counts := make(map[string]int, len(snapshot.Cabins))
	placed := make(map[string]bool)
	for cabinID, counselorIDs := range assignment.CabinCounselors {
		cp := make([]string, len(counselorIDs))
		copy(cp, counselorIDs)
		result[cabinID] = cp
		counts[cabinID] = len(cp)
		for _, cID := range counselorIDs {
			placed[cID] = true
		}
	}

	counselorsByID := indexCounselors(snapshot)

	cabinHasSenior := func(cabinID string) bool {
		for _, existingID := range result[cabinID] {
			if existing, ok := counselorsByID[existingID]; ok && !existing.IsJunior {
				return true
			}
		}
		return false
	}

	prefRank := make(map[string]map[string]int, len(snapshot.AgeGroupPreferences))
	for cID, prefs := range snapshot.AgeGroupPreferences {
		m := make(map[string]int, len(prefs))
		for _, p := range prefs {
			if p.Rank > 0 {
				m[p.TargetID] = p.Rank
			}
		}
		prefRank[cID] = m
	}

	for _, counselor := range snapshot.Counselors {
		if placed[counselor.ID] {
			continue
		}

		bestCabinID := ""
		bestRank := 0
		bestRemaining := -1
		for _, cabin := range snapshot.Cabins {
			if cabin.Gender != counselor.Gender {
				continue
			}
			remaining := cabin.Capacity - counts[cabin.ID]
			if remaining <= 0 {
				continue
			}
			// A junior cannot be the only counselor in a campered cabin
			// (would violate the senior-presence rule). The senior may
			// be either pre-placed by the search or already added to
			// this cabin earlier in the fill pass.
			if cabin.HasCampers && counselor.IsJunior && !cabinHasSenior(cabin.ID) {
				continue
			}
			rank := prefRank[counselor.ID][cabin.AgeGroupID]

			if bestCabinID == "" {
				bestCabinID = cabin.ID
				bestRank = rank
				bestRemaining = remaining
				continue
			}

			// Prefer ranked age-group matches (lower rank wins; rank 0
			// means no preference and loses to any ranked match).
			currentHasPref := bestRank > 0
			candHasPref := rank > 0
			if candHasPref && !currentHasPref {
				bestCabinID = cabin.ID
				bestRank = rank
				bestRemaining = remaining
				continue
			}
			if currentHasPref && !candHasPref {
				continue
			}
			if candHasPref && currentHasPref && rank != bestRank {
				if rank < bestRank {
					bestCabinID = cabin.ID
					bestRank = rank
					bestRemaining = remaining
				}
				continue
			}

			// Tie on preference: balance by remaining capacity.
			if remaining > bestRemaining {
				bestCabinID = cabin.ID
				bestRank = rank
				bestRemaining = remaining
			}
		}

		if bestCabinID == "" {
			continue
		}
		result[bestCabinID] = append(result[bestCabinID], counselor.ID)
		counts[bestCabinID]++
	}

	return Assignment{CabinCounselors: result}
}
