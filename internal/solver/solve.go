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
		assignment:     make(map[string][]string, len(snapshot.Cabins)),
		iterations:     0,
	}

	for i, c := range snapshot.Cabins {
		s.cabinIDs[i] = c.ID
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
		s.assignment[cabinID] = append(s.assignment[cabinID], counselor.ID)

		if s.feasible(remaining[1:]) {
			s.search(counselors, index+1)
		}

		s.assignment[cabinID] = s.assignment[cabinID][:len(s.assignment[cabinID])-1]

		if s.iterations >= s.config.MaxIterations {
			return
		}
	}

	// Also try not placing this counselor at all.
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

	// Check that each cabin can still reach its required_counselors count.
	totalRemaining := len(remaining)
	deficit := 0
	for _, cabin := range s.snapshot.Cabins {
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
