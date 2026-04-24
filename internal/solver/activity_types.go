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
}

type ActivityAssignment struct {
	SlotCounselors map[string][]string
}

type ActivityWeights struct {
	ActivityPreference float64
	RepeatedUnmetBoost float64
}

func DefaultActivityWeights() ActivityWeights {
	return ActivityWeights{
		ActivityPreference: 10.0,
		RepeatedUnmetBoost: 1.5,
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
	Assignment           ActivityAssignment
	Score                ScoreResult
	UnassignedCounselors []UnassignedCounselorSlots
}

// UnassignedCounselorSlots records a counselor that the activity solver
// could not place in every time slot they were eligible for. MissingTimeSlotIDs
// is the ordered list of session time slot IDs the counselor was not
// assigned to despite having at least one eligible activity slot in that
// time slot. A counselor with zero eligible time slots (e.g. lacking every
// required certification) does not appear here -- there was nothing the
// solver could do for them.
type UnassignedCounselorSlots struct {
	CounselorID        string
	MissingTimeSlotIDs []string
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
