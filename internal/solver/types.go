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

type SessionSnapshot struct {
	SessionID  string
	Cabins     []Cabin
	Counselors []Counselor
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
