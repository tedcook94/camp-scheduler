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
	SessionID           string
	Slots               []ActivitySlot
	Counselors          []ActivityCounselor
	ActivityPreferences map[string][]RankedPreference
}

type ActivityAssignment struct {
	SlotCounselors map[string][]string
}

type ActivityWeights struct {
	ActivityPreference float64
}

func DefaultActivityWeights() ActivityWeights {
	return ActivityWeights{
		ActivityPreference: 10.0,
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
	Assignments      []ActivityAssignmentExplanation
	UnmetPreferences []ActivityUnmetPreference
}

type ActivityAssignmentExplanation struct {
	CounselorID string
	SlotID      string
	Reasons     []string
}

type ActivityUnmetPreference struct {
	CounselorID string
	Constraint  string
	Message     string
}
