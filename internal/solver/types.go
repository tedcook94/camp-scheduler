package solver

type Cabin struct {
	ID                     string
	Name                   string
	AgeGroupID             string
	AgeGroupName           string
	SessionAgeGroupCabinID string
	RequiredCounselors     int
	// Capacity is the total occupancy cap for the cabin: counselors + campers
	// combined cannot exceed this number.
	Capacity int
	Gender   string
}

type Counselor struct {
	ID        string
	FirstName string
	LastName  string
	Name      string
	IsJunior  bool
	Gender    string
}

type CounselorPreviousPlacement struct {
	AgeGroupID     string
	CabinID        string
	CocounselorIDs []string
}

type RankedPreference struct {
	TargetID string
	Rank     int
}

type SessionSnapshot struct {
	SessionID                   string
	Cabins                      []Cabin
	Counselors                  []Counselor
	CounselorPreviousPlacements map[string]CounselorPreviousPlacement
	CocounselorPreferences      map[string][]RankedPreference
	AgeGroupPreferences         map[string][]RankedPreference
	UnmetAgeGroupPreferences    map[string]map[string]bool
	UnmetCocounselorPreferences map[string]map[string]bool
}

type Assignment struct {
	CabinCounselors map[string][]string
}

type Violation struct {
	Constraint  string
	Message     string
	CabinID     string
	CounselorID string
	CamperID    string
}

type Weights struct {
	ReturningAgeGroup     float64
	ReturningCabin        float64
	CocounselorPreference float64
	AgeGroupPreference    float64
	MultipleSeniors       float64
	RepeatedUnmetBoost    float64
}

func DefaultWeights() Weights {
	return Weights{
		ReturningAgeGroup:     10.0,
		ReturningCabin:        5.0,
		CocounselorPreference: 5.0,
		AgeGroupPreference:    5.0,
		MultipleSeniors:       2.0,
		RepeatedUnmetBoost:    1.5,
	}
}

type ScoreResult struct {
	Total     float64
	Breakdown []ScoreComponent
}

type ScoreComponent struct {
	Constraint  string
	Score       float64
	Message     string
	Rank        int
	CabinID     string
	CounselorID string
	CamperID    string
	SlotID      string
}

type SolverConfig struct {
	MaxSolutions  int
	MaxIterations int
	Weights       Weights
}

func DefaultSolverConfig() SolverConfig {
	return SolverConfig{
		MaxSolutions:  5,
		MaxIterations: 100_000,
		Weights:       DefaultWeights(),
	}
}

type Solution struct {
	Assignment           Assignment
	Score                ScoreResult
	UnassignedCounselors []string
}

type Explanation struct {
	Assignments           []AssignmentExplanation
	UnmetPreferences      []UnmetPreference
	IneligiblePreferences []IneligiblePreference
}

type AssignmentExplanation struct {
	CounselorID string
	CabinID     string
	Reasons     []AssignmentReason
}

// AssignmentReason is one scoring reason for an assignment, carrying the
// constraint type, preference rank (0 when not applicable), the slot it
// applies to (only used by the activity-schedule solver), and the
// human-readable message (already formatted with the "(+N.N)" score suffix).
type AssignmentReason struct {
	Constraint string
	Rank       int
	SlotID     string
	Message    string
}

type UnmetPreference struct {
	CounselorID string
	Constraint  string
	Rank        int
	Message     string
}

// IneligiblePreference describes a preference that cannot be satisfied for a
// structural reason (e.g. target counselor is not on the session roster, or
// would require sharing a gender-segregated cabin with a counselor of a
// different gender). Distinct from UnmetPreference, which is a preference
// that could have been satisfied but wasn't in the chosen solution.
type IneligiblePreference struct {
	CounselorID string
	Constraint  string
	Rank        int
	Message     string
}
