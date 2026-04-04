package solver

import "sort"

func Solve(snapshot SessionSnapshot, config SolverConfig) []Solution {
	s := &searchState{
		snapshot:   snapshot,
		config:     config,
		cabinIDs:   make([]string, len(snapshot.Cabins)),
		assignment: make(map[string][]string, len(snapshot.Cabins)),
		iterations: 0,
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
	snapshot   SessionSnapshot
	config     SolverConfig
	cabinIDs   []string
	assignment map[string][]string
	solutions  []Solution
	iterations int
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

	counselor := counselors[index]

	for _, cabinID := range s.cabinIDs {
		s.assignment[cabinID] = append(s.assignment[cabinID], counselor.ID)
		s.search(counselors, index+1)
		s.assignment[cabinID] = s.assignment[cabinID][:len(s.assignment[cabinID])-1]

		if s.iterations >= s.config.MaxIterations {
			return
		}
	}

	// Also try not placing this counselor at all.
	s.search(counselors, index+1)
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
