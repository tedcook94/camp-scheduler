package solver

type CamperCabin struct {
	ID                     string
	Name                   string
	AgeGroupID             string
	AgeGroupName           string
	SessionAgeGroupCabinID string
	Capacity               int
	Gender                 string
}

type Camper struct {
	ID         string
	FirstName  string
	LastName   string
	Name       string
	AgeGroupID string
	Gender     string
}

type CamperCabinSnapshot struct {
	SessionID         string
	Cabins            []CamperCabin
	Campers           []Camper
	FriendPreferences map[string][]RankedPreference
	// Overrides pin a camper to a specific cabin. Keyed by camper ID; the
	// value is the cabins.id key used internally by the solver (already
	// remapped from the persisted session_age_group_cabin_id during snapshot
	// load). The solver must place these campers in their pinned cabin and
	// must not place them anywhere else.
	Overrides map[string]string
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
	Assignments           []CamperAssignmentExplanation
	UnmetPreferences      []CamperUnmetPreference
	IneligiblePreferences []CamperIneligiblePreference
}

type CamperAssignmentExplanation struct {
	CamperID string
	CabinID  string
	Reasons  []AssignmentReason
}

type CamperUnmetPreference struct {
	CamperID   string
	Constraint string
	Rank       int
	Message    string
}

// CamperIneligiblePreference describes a friend preference that cannot be
// satisfied for a structural reason (target not enrolled in this session,
// or would require sharing a gender-segregated/age-grouped cabin with a
// camper of a different gender or age group). Distinct from
// CamperUnmetPreference, which is a preference that could have been
// satisfied but wasn't in the chosen solution.
type CamperIneligiblePreference struct {
	CamperID   string
	Constraint string
	Rank       int
	Message    string
}
