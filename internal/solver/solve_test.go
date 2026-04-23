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

	t.Run("all counselors placed when capacity allows", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 1, Capacity: 10, HasCampers: true},
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

			placed := make(map[string]bool)
			for _, ids := range sol.Assignment.CabinCounselors {
				for _, id := range ids {
					placed[id] = true
				}
			}
			for _, c := range snapshot.Counselors {
				if !placed[c.ID] {
					t.Errorf("expected counselor %q to be placed; got %v", c.ID, sol.Assignment.CabinCounselors)
				}
			}
		}
	})

	t.Run("over-subscribed roster: leftovers penalized but solution still returned", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 1, Capacity: 2, Gender: "male", HasCampers: true},
			},
			Counselors: []Counselor{
				{ID: "sr1", Name: "Counselor 1", IsJunior: false, Gender: "male"},
				{ID: "sr2", Name: "Counselor 2", IsJunior: false, Gender: "male"},
				{ID: "sr3", Name: "Counselor 3", IsJunior: false, Gender: "male"},
			},
		}

		solutions := Solve(snapshot, DefaultSolverConfig())
		if len(solutions) == 0 {
			t.Fatal("expected at least one solution")
		}

		best := solutions[0]
		// Capacity 2 with 3 counselors leaves exactly one unplaced.
		placed := 0
		for _, ids := range best.Assignment.CabinCounselors {
			placed += len(ids)
		}
		if placed != 2 {
			t.Errorf("expected 2 counselors placed (capacity), got %d", placed)
		}

		penalty := DefaultWeights().UnassignedCounselorPenalty
		if best.Score.Total >= 0 {
			t.Errorf("expected negative total score from unassigned counselor, got %v", best.Score.Total)
		}

		var penaltyComponents int
		for _, comp := range best.Score.Breakdown {
			if comp.Constraint == "unassigned_counselor" {
				penaltyComponents++
				if comp.Score != -penalty {
					t.Errorf("expected penalty score %v, got %v", -penalty, comp.Score)
				}
			}
		}
		if penaltyComponents != 1 {
			t.Errorf("expected 1 unassigned_counselor breakdown entry, got %d", penaltyComponents)
		}
	})

	t.Run("junior leftover allowed in cabin without campers", func(t *testing.T) {
		// Two cabins: one with campers (needs senior + min), one without
		// campers (relaxed). One senior fills the campered cabin; the
		// junior leftover gets placed in the empty no-campers cabin
		// without a senior — and that's valid.
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 1, Capacity: 2, Gender: "male", HasCampers: true},
				{ID: "c2", Name: "Oak", AgeGroupID: "ag2", RequiredCounselors: 0, Capacity: 2, Gender: "male", HasCampers: false},
			},
			Counselors: []Counselor{
				{ID: "sr1", Name: "Senior", IsJunior: false, Gender: "male"},
				{ID: "jr1", Name: "Junior", IsJunior: true, Gender: "male"},
			},
		}

		solutions := Solve(snapshot, DefaultSolverConfig())
		if len(solutions) == 0 {
			t.Fatal("expected at least one solution")
		}

		// At least one solution should place both counselors.
		fullPlacement := false
		for _, sol := range solutions {
			placed := 0
			for _, ids := range sol.Assignment.CabinCounselors {
				placed += len(ids)
			}
			if placed == 2 {
				fullPlacement = true
				violations := CheckHardConstraints(snapshot, sol.Assignment)
				if len(violations) > 0 {
					t.Errorf("full-placement solution has violations: %v", violations)
				}
			}
		}
		if !fullPlacement {
			t.Errorf("expected at least one solution placing both counselors, got %v", solutions)
		}
	})

	t.Run("junior leftover stays unplaced when only campered cabin available", func(t *testing.T) {
		// Single campered cabin already filled with a senior to capacity;
		// the junior leftover cannot be placed (no other cabin) and
		// receives the unassigned-counselor penalty.
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 1, Capacity: 1, Gender: "male", HasCampers: true},
			},
			Counselors: []Counselor{
				{ID: "sr1", Name: "Senior", IsJunior: false, Gender: "male"},
				{ID: "jr1", Name: "Junior", IsJunior: true, Gender: "male"},
			},
		}

		solutions := Solve(snapshot, DefaultSolverConfig())
		if len(solutions) == 0 {
			t.Fatal("expected at least one solution")
		}

		best := solutions[0]
		violations := CheckHardConstraints(snapshot, best.Assignment)
		if len(violations) > 0 {
			t.Errorf("best solution has violations: %v", violations)
		}

		var penaltyHits int
		for _, comp := range best.Score.Breakdown {
			if comp.Constraint == "unassigned_counselor" && comp.CounselorID == "jr1" {
				penaltyHits++
			}
		}
		if penaltyHits != 1 {
			t.Errorf("expected junior to be flagged as unassigned, got %d hits", penaltyHits)
		}
	})

	t.Run("junior leftover not dropped into campered cabin without senior", func(t *testing.T) {
		// Single campered cabin with no minimum and no existing senior.
		// The fill pass must NOT place the junior here (would create a
		// senior-rule violation). Junior should stay unassigned.
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 0, Capacity: 2, Gender: "male", HasCampers: true},
			},
			Counselors: []Counselor{
				{ID: "jr1", Name: "Junior", IsJunior: true, Gender: "male"},
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
			if len(sol.Assignment.CabinCounselors["c1"]) != 0 {
				t.Errorf("expected junior to be left unassigned, got %v", sol.Assignment.CabinCounselors)
			}
		}

		// Penalty must fire for the unassigned junior.
		var penaltyHits int
		for _, comp := range solutions[0].Score.Breakdown {
			if comp.Constraint == "unassigned_counselor" && comp.CounselorID == "jr1" {
				penaltyHits++
			}
		}
		if penaltyHits != 1 {
			t.Errorf("expected junior to be flagged as unassigned, got %d hits", penaltyHits)
		}
	})

	t.Run("junior leftover joins campered cabin once a senior is there", func(t *testing.T) {
		// Single campered cabin (cap 2, req 1) is the only option for
		// both counselors. Senior placed by search; junior fill pass
		// must accept the cabin since a senior is now present.
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 1, Capacity: 2, Gender: "male", HasCampers: true},
			},
			Counselors: []Counselor{
				{ID: "sr1", Name: "Senior", IsJunior: false, Gender: "male"},
				{ID: "jr1", Name: "Junior", IsJunior: true, Gender: "male"},
			},
		}

		solutions := Solve(snapshot, DefaultSolverConfig())
		if len(solutions) == 0 {
			t.Fatal("expected at least one solution")
		}

		fullPlacement := false
		for _, sol := range solutions {
			violations := CheckHardConstraints(snapshot, sol.Assignment)
			if len(violations) > 0 {
				t.Errorf("solution has hard constraint violations: %v", violations)
			}
			if len(sol.Assignment.CabinCounselors["c1"]) == 2 {
				fullPlacement = true
			}
		}
		if !fullPlacement {
			t.Errorf("expected at least one solution placing both counselors in c1, got %v", solutions)
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
