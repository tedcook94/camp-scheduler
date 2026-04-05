package solver

type CamperCabin struct {
	ID         string
	Name       string
	AgeGroupID string
	Capacity   int
}

type Camper struct {
	ID         string
	Name       string
	AgeGroupID string
}

type CamperCabinSnapshot struct {
	SessionID         string
	Cabins            []CamperCabin
	Campers           []Camper
	FriendPreferences map[string][]RankedPreference
}

type CamperAssignment struct {
	CabinCampers map[string][]string
}

type CamperWeights struct {
	FriendPreference float64
}

func DefaultCamperWeights() CamperWeights {
	return CamperWeights{
		FriendPreference: 10.0,
	}
}

type CamperSolverConfig struct {
	MaxSolutions  int
	MaxIterations int
	Weights       CamperWeights
}

func DefaultCamperSolverConfig() CamperSolverConfig {
	return CamperSolverConfig{
		MaxSolutions:  5,
		MaxIterations: 100_000,
		Weights:       DefaultCamperWeights(),
	}
}

type CamperSolution struct {
	Assignment CamperAssignment
	Score      ScoreResult
}

type CamperExplanation struct {
	Assignments      []CamperAssignmentExplanation
	UnmetPreferences []CamperUnmetPreference
}

type CamperAssignmentExplanation struct {
	CamperID string
	CabinID  string
	Reasons  []string
}

type CamperUnmetPreference struct {
	CamperID   string
	Constraint string
	Message    string
}
