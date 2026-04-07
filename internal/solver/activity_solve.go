package solver

import "sort"

func SolveActivity(snapshot ActivitySnapshot, config ActivitySolverConfig) []ActivitySolution {
	if config.MaxSolutions <= 0 || config.MaxIterations <= 0 {
		return nil
	}

	slotsByID := indexActivitySlots(snapshot)

	// Build eligibility: for each counselor, which slots they can serve.
	eligibleSlots := make(map[string][]string)
	for _, c := range snapshot.Counselors {
		for _, slot := range snapshot.Slots {
			if counselorHasCerts(c, slot) {
				eligibleSlots[c.ID] = append(eligibleSlots[c.ID], slot.ID)
			}
		}
	}

	// Build time slot grouping used to partition eligible slots.
	slotTimeSlot := make(map[string]string)
	for _, slot := range snapshot.Slots {
		slotTimeSlot[slot.ID] = slot.TimeSlotID
	}

	// Group eligible slots by time slot for each counselor. This powers the
	// combinatorial search: for each time slot a counselor could serve, they
	// either pick one eligible slot in that time slot or skip it.
	eligibleByTimeSlot := make(map[string]map[string][]string)
	for _, c := range snapshot.Counselors {
		byTS := make(map[string][]string)
		for _, slotID := range eligibleSlots[c.ID] {
			tsID := slotTimeSlot[slotID]
			byTS[tsID] = append(byTS[tsID], slotID)
		}
		eligibleByTimeSlot[c.ID] = byTS
	}

	// Collect ordered time slot IDs so iteration is deterministic.
	timeSlotOrder := orderedTimeSlots(snapshot)

	s := &activitySearchState{
		snapshot:           snapshot,
		config:             config,
		slotsByID:          slotsByID,
		eligibleSlots:      eligibleSlots,
		eligibleByTimeSlot: eligibleByTimeSlot,
		timeSlotOrder:      timeSlotOrder,
		assignment:         make(map[string][]string),
		counselorSlots:     make(map[string][]string),
	}

	for _, slot := range snapshot.Slots {
		s.assignment[slot.ID] = nil
	}

	counselors := orderActivityCounselors(snapshot, eligibleSlots)
	s.search(counselors, 0)

	sort.Slice(s.solutions, func(i, j int) bool {
		return s.solutions[i].Score.Total > s.solutions[j].Score.Total
	})

	return s.solutions
}

type activitySearchState struct {
	snapshot           ActivitySnapshot
	config             ActivitySolverConfig
	slotsByID          map[string]ActivitySlot
	eligibleSlots      map[string][]string
	eligibleByTimeSlot map[string]map[string][]string
	timeSlotOrder      []string
	assignment         map[string][]string
	counselorSlots     map[string][]string
	solutions          []ActivitySolution
	iterations         int
}

func (s *activitySearchState) search(counselors []ActivityCounselor, index int) {
	if s.iterations >= s.config.MaxIterations {
		return
	}
	s.iterations++

	if index == len(counselors) {
		s.evaluateSolution()
		return
	}

	counselor := counselors[index]

	// Try all valid slot combinations for this counselor: at most one slot
	// per time slot. We iterate time slots in order and for each one either
	// pick an eligible available slot or skip that time slot entirely.
	s.searchCounselorSlots(counselors, index, counselor.ID, 0)
}

// searchCounselorSlots enumerates valid slot combinations for a single
// counselor across time slots. tsIndex is the position in s.timeSlotOrder.
func (s *activitySearchState) searchCounselorSlots(counselors []ActivityCounselor, counselorIndex int, counselorID string, tsIndex int) {
	if s.iterations >= s.config.MaxIterations {
		return
	}
	s.iterations++

	if tsIndex == len(s.timeSlotOrder) {
		// All time slots considered for this counselor — recurse to next.
		if s.feasible(counselors[counselorIndex+1:]) {
			s.search(counselors, counselorIndex+1)
		}
		return
	}

	tsID := s.timeSlotOrder[tsIndex]
	eligible := s.eligibleByTimeSlot[counselorID][tsID]

	// Try each eligible slot in this time slot.
	for _, slotID := range eligible {
		slot := s.slotsByID[slotID]
		if len(s.assignment[slotID]) >= slot.Capacity {
			continue
		}

		s.assignment[slotID] = append(s.assignment[slotID], counselorID)
		s.counselorSlots[counselorID] = append(s.counselorSlots[counselorID], slotID)

		s.searchCounselorSlots(counselors, counselorIndex, counselorID, tsIndex+1)

		s.assignment[slotID] = s.assignment[slotID][:len(s.assignment[slotID])-1]
		s.counselorSlots[counselorID] = s.counselorSlots[counselorID][:len(s.counselorSlots[counselorID])-1]

		if s.iterations >= s.config.MaxIterations {
			return
		}
	}

	// Also try skipping this time slot (don't assign counselor here).
	s.searchCounselorSlots(counselors, counselorIndex, counselorID, tsIndex+1)
}

func (s *activitySearchState) feasible(remaining []ActivityCounselor) bool {
	// Check that remaining counselors can fill unfilled required slots.
	for _, slot := range s.snapshot.Slots {
		need := slot.RequiredCounselors - len(s.assignment[slot.ID])
		if need <= 0 {
			continue
		}

		// Count remaining counselors who could fill this slot.
		available := 0
		for _, c := range remaining {
			if counselorHasCerts(c, slot) {
				available++
			}
		}
		if available < need {
			return false
		}
	}
	return true
}

func (s *activitySearchState) evaluateSolution() {
	assignment := s.cloneAssignment()

	violations := CheckActivityHardConstraints(s.snapshot, assignment)
	if len(violations) > 0 {
		return
	}

	score := ScoreActivitySoftConstraints(s.snapshot, assignment, s.config.Weights)

	if len(s.solutions) < s.config.MaxSolutions {
		s.solutions = append(s.solutions, ActivitySolution{
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
		s.solutions[worstIdx] = ActivitySolution{
			Assignment: assignment,
			Score:      score,
		}
	}
}

func (s *activitySearchState) cloneAssignment() ActivityAssignment {
	clone := make(map[string][]string, len(s.assignment))
	for slotID, counselorIDs := range s.assignment {
		if len(counselorIDs) == 0 {
			continue
		}
		c := make([]string, len(counselorIDs))
		copy(c, counselorIDs)
		clone[slotID] = c
	}
	return ActivityAssignment{SlotCounselors: clone}
}

func counselorHasCerts(c ActivityCounselor, slot ActivitySlot) bool {
	for _, certID := range slot.RequiredCertifications {
		if !c.Certifications[certID] {
			return false
		}
	}
	return true
}

func orderActivityCounselors(snapshot ActivitySnapshot, eligibleSlots map[string][]string) []ActivityCounselor {
	type counselorPriority struct {
		counselor   ActivityCounselor
		constraints int
	}

	priorities := make([]counselorPriority, len(snapshot.Counselors))
	for i, c := range snapshot.Counselors {
		constraints := 0

		// Fewer eligible slots means more constrained.
		eligible := len(eligibleSlots[c.ID])
		if eligible > 0 {
			constraints += 1000 / eligible
		}

		// Has certifications = more constrained (needed for specific slots).
		if len(c.Certifications) > 0 {
			constraints += 100
		}

		if _, ok := snapshot.ActivityPreferences[c.ID]; ok {
			constraints += 10
		}

		priorities[i] = counselorPriority{counselor: c, constraints: constraints}
	}

	sort.SliceStable(priorities, func(i, j int) bool {
		return priorities[i].constraints > priorities[j].constraints
	})

	result := make([]ActivityCounselor, len(priorities))
	for i, p := range priorities {
		result[i] = p.counselor
	}
	return result
}

func orderedTimeSlots(snapshot ActivitySnapshot) []string {
	seen := make(map[string]bool)
	var order []string
	for _, slot := range snapshot.Slots {
		if !seen[slot.TimeSlotID] {
			seen[slot.TimeSlotID] = true
			order = append(order, slot.TimeSlotID)
		}
	}
	return order
}
