package solver

type Cabin struct {
	ID                 string
	Name               string
	AgeGroupID         string
	RequiredCounselors int
}

type Counselor struct {
	ID       string
	Name     string
	IsJunior bool
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
}

type Assignment struct {
	CabinCounselors map[string][]string
}

type Violation struct {
	Constraint  string
	Message     string
	CabinID     string
	CounselorID string
}

type Weights struct {
	ReturningAgeGroup     float64
	ReturningCabin        float64
	CocounselorPreference float64
	AgeGroupPreference    float64
	MultipleSeniors       float64
}

func DefaultWeights() Weights {
	return Weights{
		ReturningAgeGroup:     10.0,
		ReturningCabin:        5.0,
		CocounselorPreference: 5.0,
		AgeGroupPreference:    5.0,
		MultipleSeniors:       2.0,
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
	CabinID     string
	CounselorID string
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
	Assignment Assignment
	Score      ScoreResult
}
