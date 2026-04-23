package solver

import (
	"slices"
	"testing"
)

func TestSolve(t *testing.T) {
	t.Run("simple valid assignment", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 1, Capacity: 10},
				{ID: "c2", Name: "Oak", AgeGroupID: "ag2", RequiredCounselors: 1, Capacity: 10},
			},
			Counselors: []Counselor{
				{ID: "sr1", Name: "Counselor 1", IsJunior: false},
				{ID: "sr2", Name: "Counselor 2", IsJunior: false},
			},
		}

		solutions := Solve(snapshot, DefaultSolverConfig())
		if len(solutions) == 0 {
			t.Fatal("expected at least one solution")
		}

		for _, sol := range solutions {
			violations := CheckHardConstraints(snapshot, sol.Assignment)
			if len(violations) > 0 {
				t.Errorf("solution has hard constraint violations: %v", violations)
			}
		}
	})

	t.Run("no valid solution with all juniors", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 1, Capacity: 10, HasCampers: true},
			},
			Counselors: []Counselor{
				{ID: "jr1", Name: "Junior 1", IsJunior: true},
			},
		}

		solutions := Solve(snapshot, DefaultSolverConfig())
		if len(solutions) != 0 {
			t.Errorf("expected no solutions, got %d", len(solutions))
		}
	})

	t.Run("solutions ranked by descending score", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 1, Capacity: 10},
				{ID: "c2", Name: "Oak", AgeGroupID: "ag2", RequiredCounselors: 1, Capacity: 10},
			},
			Counselors: []Counselor{
				{ID: "sr1", Name: "Counselor 1", IsJunior: false},
				{ID: "sr2", Name: "Counselor 2", IsJunior: false},
			},
			AgeGroupPreferences: map[string][]RankedPreference{
				"sr1": {{TargetID: "ag1", Rank: 1}},
				"sr2": {{TargetID: "ag2", Rank: 1}},
			},
		}

		solutions := Solve(snapshot, DefaultSolverConfig())
		if len(solutions) < 2 {
			t.Fatalf("expected at least 2 solutions, got %d", len(solutions))
		}

		for i := 1; i < len(solutions); i++ {
			if solutions[i].Score.Total > solutions[i-1].Score.Total {
				t.Errorf("solutions not sorted: index %d score %f > index %d score %f",
					i, solutions[i].Score.Total, i-1, solutions[i-1].Score.Total)
			}
		}
	})

	t.Run("max solutions respected", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 1, Capacity: 10},
				{ID: "c2", Name: "Oak", AgeGroupID: "ag2", RequiredCounselors: 1, Capacity: 10},
			},
			Counselors: []Counselor{
				{ID: "sr1", Name: "Counselor 1", IsJunior: false},
				{ID: "sr2", Name: "Counselor 2", IsJunior: false},
				{ID: "sr3", Name: "Counselor 3", IsJunior: false},
			},
		}

		config := DefaultSolverConfig()
		config.MaxSolutions = 2
		solutions := Solve(snapshot, config)
		if len(solutions) > 2 {
			t.Errorf("expected at most 2 solutions, got %d", len(solutions))
		}
	})

	t.Run("max iterations terminates search", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 1, Capacity: 10},
				{ID: "c2", Name: "Oak", AgeGroupID: "ag2", RequiredCounselors: 1, Capacity: 10},
				{ID: "c3", Name: "Elm", AgeGroupID: "ag1", RequiredCounselors: 1, Capacity: 10},
			},
			Counselors: []Counselor{
				{ID: "sr1", Name: "Counselor 1", IsJunior: false},
				{ID: "sr2", Name: "Counselor 2", IsJunior: false},
				{ID: "sr3", Name: "Counselor 3", IsJunior: false},
				{ID: "sr4", Name: "Counselor 4", IsJunior: false},
				{ID: "sr5", Name: "Counselor 5", IsJunior: false},
			},
		}

		config := DefaultSolverConfig()
		config.MaxIterations = 10

		// Should terminate without panicking or hanging.
		Solve(snapshot, config)
	})

	t.Run("extra counselors can be left unassigned", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 1, Capacity: 10},
			},
			Counselors: []Counselor{
				{ID: "sr1", Name: "Counselor 1", IsJunior: false},
				{ID: "sr2", Name: "Counselor 2", IsJunior: false},
				{ID: "sr3", Name: "Counselor 3", IsJunior: false},
			},
		}

		solutions := Solve(snapshot, DefaultSolverConfig())
		if len(solutions) == 0 {
			t.Fatal("expected at least one solution")
		}

		for _, sol := range solutions {
			violations := CheckHardConstraints(snapshot, sol.Assignment)
			if len(violations) > 0 {
				t.Errorf("solution has hard constraint violations: %v", violations)
			}
		}
	})

	t.Run("preferences influence best solution", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 1, Capacity: 10},
				{ID: "c2", Name: "Oak", AgeGroupID: "ag2", RequiredCounselors: 1, Capacity: 10},
			},
			Counselors: []Counselor{
				{ID: "sr1", Name: "Counselor 1", IsJunior: false},
				{ID: "sr2", Name: "Counselor 2", IsJunior: false},
			},
			CounselorPreviousPlacements: map[string]CounselorPreviousPlacement{
				"sr1": {AgeGroupID: "ag1", CabinID: "c1"},
			},
		}

		solutions := Solve(snapshot, DefaultSolverConfig())
		if len(solutions) == 0 {
			t.Fatal("expected at least one solution")
		}

		best := solutions[0]
		counselors := best.Assignment.CabinCounselors["c1"]
		found := slices.Contains(counselors, "sr1")
		if !found {
			t.Error("expected best solution to place sr1 in c1 (returning to same cabin/age group)")
		}
	})

	t.Run("junior plus senior satisfies constraints", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 2, Capacity: 10},
			},
			Counselors: []Counselor{
				{ID: "sr1", Name: "Counselor 1", IsJunior: false},
				{ID: "jr1", Name: "Junior 1", IsJunior: true},
			},
		}

		solutions := Solve(snapshot, DefaultSolverConfig())
		if len(solutions) == 0 {
			t.Fatal("expected at least one solution")
		}

		for _, sol := range solutions {
			violations := CheckHardConstraints(snapshot, sol.Assignment)
			if len(violations) > 0 {
				t.Errorf("solution has hard constraint violations: %v", violations)
			}
		}
	})
}

func TestOrderCounselors(t *testing.T) {
	t.Run("seniors ordered before juniors", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Counselors: []Counselor{
				{ID: "jr1", Name: "Junior 1", IsJunior: true},
				{ID: "sr1", Name: "Counselor 1", IsJunior: false},
				{ID: "jr2", Name: "Junior 2", IsJunior: true},
			},
		}

		ordered := orderCounselors(snapshot)
		if ordered[0].ID != "sr1" {
			t.Errorf("expected senior first, got %q", ordered[0].ID)
		}
	})

	t.Run("counselors with preferences ordered before those without", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Counselors: []Counselor{
				{ID: "sr1", Name: "Counselor 1", IsJunior: false},
				{ID: "sr2", Name: "Counselor 2", IsJunior: false},
			},
			AgeGroupPreferences: map[string][]RankedPreference{
				"sr2": {{TargetID: "ag1", Rank: 1}},
			},
		}

		ordered := orderCounselors(snapshot)
		if ordered[0].ID != "sr2" {
			t.Errorf("expected counselor with preferences first, got %q", ordered[0].ID)
		}
	})

	t.Run("counselors with history ordered before those without", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Counselors: []Counselor{
				{ID: "sr1", Name: "Counselor 1", IsJunior: false},
				{ID: "sr2", Name: "Counselor 2", IsJunior: false},
			},
			CounselorPreviousPlacements: map[string]CounselorPreviousPlacement{
				"sr2": {AgeGroupID: "ag1"},
			},
		}

		ordered := orderCounselors(snapshot)
		if ordered[0].ID != "sr2" {
			t.Errorf("expected counselor with history first, got %q", ordered[0].ID)
		}
	})
}
