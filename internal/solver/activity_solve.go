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

	// Order eligible slots so that preferred activities (by rank) are tried
	// first. Ties break by highest remaining-capacity fraction so the search
	// naturally lands on balanced, preference-respecting solutions early.
	ordered := make([]string, len(eligible))
	copy(ordered, eligible)
	prefs := s.snapshot.ActivityPreferences[counselorID]
	rankByActivity := make(map[string]int, len(prefs))
	for _, p := range prefs {
		if p.Rank > 0 {
			rankByActivity[p.TargetID] = p.Rank
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		si := s.slotsByID[ordered[i]]
		sj := s.slotsByID[ordered[j]]
		ri, iPref := rankByActivity[si.ActivityID]
		rj, jPref := rankByActivity[sj.ActivityID]
		if iPref != jPref {
			return iPref
		}
		if iPref && jPref && ri != rj {
			return ri < rj
		}
		remI := si.Capacity - len(s.assignment[ordered[i]])
		remJ := sj.Capacity - len(s.assignment[ordered[j]])
		fi := float64(remI) / float64(si.Capacity)
		fj := float64(remJ) / float64(sj.Capacity)
		return fi > fj
	})

	// Try each eligible slot in this time slot.
	for _, slotID := range ordered {
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
	assignment = fillRemainingActivity(s.snapshot, assignment, s.eligibleByTimeSlot, s.timeSlotOrder, s.slotsByID)

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

// fillRemainingActivity places any counselor not yet assigned in a time slot
// into an eligible slot within that time slot. Preference is given to the
// counselor's ranked activity preferences; otherwise it picks the slot with
// the highest remaining-capacity fraction (round-robin balancing).
func fillRemainingActivity(
	snapshot ActivitySnapshot,
	assignment ActivityAssignment,
	eligibleByTimeSlot map[string]map[string][]string,
	timeSlotOrder []string,
	slotsByID map[string]ActivitySlot,
) ActivityAssignment {
	result := make(map[string][]string, len(assignment.SlotCounselors))
	counts := make(map[string]int)
	for slotID, counselorIDs := range assignment.SlotCounselors {
		cp := make([]string, len(counselorIDs))
		copy(cp, counselorIDs)
		result[slotID] = cp
		counts[slotID] = len(cp)
	}

	// Build counselor -> set of time slots already served.
	counselorTimeSlots := make(map[string]map[string]bool)
	for slotID, counselorIDs := range result {
		tsID := slotsByID[slotID].TimeSlotID
		for _, cID := range counselorIDs {
			if counselorTimeSlots[cID] == nil {
				counselorTimeSlots[cID] = make(map[string]bool)
			}
			counselorTimeSlots[cID][tsID] = true
		}
	}

	// Build counselor -> activityID -> rank for quick preference lookup.
	prefRank := make(map[string]map[string]int)
	for cID, prefs := range snapshot.ActivityPreferences {
		m := make(map[string]int, len(prefs))
		for _, p := range prefs {
			if p.Rank > 0 {
				m[p.TargetID] = p.Rank
			}
		}
		prefRank[cID] = m
	}

	pickSlot := func(cID, tsID string) string {
		eligible := eligibleByTimeSlot[cID][tsID]
		if len(eligible) == 0 {
			return ""
		}

		// First try: pick the preferred eligible slot with lowest rank (best).
		prefs := prefRank[cID]
		bestPrefSlot := ""
		bestPrefRank := 0
		for _, slotID := range eligible {
			slot := slotsByID[slotID]
			if counts[slotID] >= slot.Capacity {
				continue
			}
			rank, ok := prefs[slot.ActivityID]
			if !ok {
				continue
			}
			if bestPrefSlot == "" || rank < bestPrefRank {
				bestPrefSlot = slotID
				bestPrefRank = rank
			}
		}
		if bestPrefSlot != "" {
			return bestPrefSlot
		}

		// Fallback: pick slot with highest remaining-capacity fraction
		// so counselors spread evenly across activities.
		bestSlot := ""
		bestFraction := 0.0
		for _, slotID := range eligible {
			slot := slotsByID[slotID]
			remaining := slot.Capacity - counts[slotID]
			if remaining <= 0 {
				continue
			}
			fraction := float64(remaining) / float64(slot.Capacity)
			if bestSlot == "" || fraction > bestFraction {
				bestSlot = slotID
				bestFraction = fraction
			}
		}
		return bestSlot
	}

	for _, c := range snapshot.Counselors {
		for _, tsID := range timeSlotOrder {
			if counselorTimeSlots[c.ID][tsID] {
				continue
			}
			slotID := pickSlot(c.ID, tsID)
			if slotID == "" {
				continue
			}
			result[slotID] = append(result[slotID], c.ID)
			counts[slotID]++
			if counselorTimeSlots[c.ID] == nil {
				counselorTimeSlots[c.ID] = make(map[string]bool)
			}
			counselorTimeSlots[c.ID][tsID] = true
		}
	}

	return ActivityAssignment{SlotCounselors: result}
}
