package solver

type ActivitySlot struct {
	ID                     string
	ActivityID             string
	ActivityName           string
	TimeSlotID             string
	TimeSlotName           string
	RequiredCounselors     int
	Capacity               int
	RequiredCertifications []string
}

type ActivityCounselor struct {
	ID             string
	Name           string
	IsJunior       bool
	Certifications map[string]bool
}

type ActivitySnapshot struct {
	SessionID                string
	Slots                    []ActivitySlot
	Counselors               []ActivityCounselor
	ActivityPreferences      map[string][]RankedPreference
	UnmetActivityPreferences map[string]map[string]bool
	CertificationNames       map[string]string

	// eligibleTimeSlotsByCounselor[counselorID] is the set of time-slot
	// IDs where the counselor satisfies the required certifications for
	// at least one scheduled activity slot. Populated by
	// BuildActivitySnapshot and consumed by scoring + explanation paths
	// to avoid repeated (counselor × slot) scans during search.
	eligibleTimeSlotsByCounselor map[string]map[string]bool
}

// eligibleTimeSlots returns the per-counselor set of time slots with at
// least one eligible activity slot. The cached value is preferred when
// present (production path); the fallback recomputes from Counselors and
// Slots so test fixtures that build snapshot literals without going
// through BuildActivitySnapshot continue to work.
func (s ActivitySnapshot) eligibleTimeSlots() map[string]map[string]bool {
	if s.eligibleTimeSlotsByCounselor != nil {
		return s.eligibleTimeSlotsByCounselor
	}
	return computeEligibleTimeSlots(s.Counselors, s.Slots)
}

func computeEligibleTimeSlots(counselors []ActivityCounselor, slots []ActivitySlot) map[string]map[string]bool {
	result := make(map[string]map[string]bool, len(counselors))
	for _, c := range counselors {
		var set map[string]bool
		for _, slot := range slots {
			if !counselorHasCerts(c, slot) {
				continue
			}
			if set == nil {
				set = make(map[string]bool)
			}
			set[slot.TimeSlotID] = true
		}
		if set != nil {
			result[c.ID] = set
		}
	}
	return result
}

type ActivityAssignment struct {
	SlotCounselors map[string][]string
}

type ActivityWeights struct {
	ActivityPreference     float64
	RepeatedUnmetBoost     float64
	MissingTimeSlotPenalty float64
}

func DefaultActivityWeights() ActivityWeights {
	return ActivityWeights{
		ActivityPreference:     10.0,
		RepeatedUnmetBoost:     1.5,
		MissingTimeSlotPenalty: 10.0,
	}
}

type ActivitySolverConfig struct {
	MaxSolutions  int
	MaxIterations int
	Weights       ActivityWeights
}

func DefaultActivitySolverConfig() ActivitySolverConfig {
	return ActivitySolverConfig{
		MaxSolutions:  5,
		MaxIterations: 100_000,
		Weights:       DefaultActivityWeights(),
	}
}

type ActivitySolution struct {
	Assignment ActivityAssignment
	Score      ScoreResult
}

type ActivityExplanation struct {
	Assignments            []ActivityAssignmentExplanation
	UnmetPreferences       []ActivityUnmetPreference
	IneligiblePreferences  []ActivityUnmetPreference
}

type ActivityAssignmentExplanation struct {
	CounselorID string
	SlotID      string
	Reasons     []AssignmentReason
}

type ActivityUnmetPreference struct {
	CounselorID string
	Constraint  string
	Rank        int
	Message     string
}
