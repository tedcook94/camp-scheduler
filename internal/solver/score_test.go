package solver

import (
	"math"
	"testing"
)

func floatEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.0001
}

func TestCounselorWantsToReturn(t *testing.T) {
	tests := []struct {
		name     string
		snapshot SessionSnapshot
		want     bool
	}{
		{
			name: "no preferences means wants to return",
			snapshot: SessionSnapshot{
				CounselorPreviousPlacements: map[string]CounselorPreviousPlacement{
					"co1": {AgeGroupID: "ag1"},
				},
			},
			want: true,
		},
		{
			name: "no previous placement",
			snapshot: SessionSnapshot{
				AgeGroupPreferences: map[string][]RankedPreference{
					"co1": {{TargetID: "ag2", Rank: 1}},
				},
			},
			want: false,
		},
		{
			name: "rank 1 matches previous age group",
			snapshot: SessionSnapshot{
				CounselorPreviousPlacements: map[string]CounselorPreviousPlacement{
					"co1": {AgeGroupID: "ag1"},
				},
				AgeGroupPreferences: map[string][]RankedPreference{
					"co1": {
						{TargetID: "ag1", Rank: 1},
						{TargetID: "ag2", Rank: 2},
					},
				},
			},
			want: true,
		},
		{
			name: "rank 1 does not match previous age group",
			snapshot: SessionSnapshot{
				CounselorPreviousPlacements: map[string]CounselorPreviousPlacement{
					"co1": {AgeGroupID: "ag1"},
				},
				AgeGroupPreferences: map[string][]RankedPreference{
					"co1": {
						{TargetID: "ag2", Rank: 1},
						{TargetID: "ag1", Rank: 2},
					},
				},
			},
			want: false,
		},
		{
			name: "previous age group in preferences but not rank 1",
			snapshot: SessionSnapshot{
				CounselorPreviousPlacements: map[string]CounselorPreviousPlacement{
					"co1": {AgeGroupID: "ag1"},
				},
				AgeGroupPreferences: map[string][]RankedPreference{
					"co1": {
						{TargetID: "ag3", Rank: 1},
						{TargetID: "ag1", Rank: 2},
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := counselorWantsToReturn("co1", tt.snapshot)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScoreReturningAgeGroup(t *testing.T) {
	weights := DefaultWeights()

	tests := []struct {
		name       string
		snapshot   SessionSnapshot
		assignment Assignment
		wantCount  int
		wantScore  float64
	}{
		{
			name: "returning to same age group scores",
			snapshot: SessionSnapshot{
				Cabins: []Cabin{
					{ID: "c1", Name: "Pine", AgeGroupID: "ag1"},
					{ID: "c2", Name: "Oak", AgeGroupID: "ag2"},
				},
				CounselorPreviousPlacements: map[string]CounselorPreviousPlacement{
					"co1": {AgeGroupID: "ag1"},
				},
			},
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"co1"},
				},
			},
			wantCount: 1,
			wantScore: weights.ReturningAgeGroup,
		},
		{
			name: "returning to different age group",
			snapshot: SessionSnapshot{
				Cabins: []Cabin{
					{ID: "c1", Name: "Pine", AgeGroupID: "ag1"},
					{ID: "c2", Name: "Oak", AgeGroupID: "ag2"},
				},
				CounselorPreviousPlacements: map[string]CounselorPreviousPlacement{
					"co1": {AgeGroupID: "ag1"},
				},
			},
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c2": {"co1"},
				},
			},
			wantCount: 0,
			wantScore: 0,
		},
		{
			name: "counselor without history",
			snapshot: SessionSnapshot{
				Cabins: []Cabin{
					{ID: "c1", Name: "Pine", AgeGroupID: "ag1"},
				},
			},
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"co1"},
				},
			},
			wantCount: 0,
			wantScore: 0,
		},
		{
			name: "counselor wants change skips bonus",
			snapshot: SessionSnapshot{
				Cabins: []Cabin{
					{ID: "c1", Name: "Pine", AgeGroupID: "ag1"},
				},
				CounselorPreviousPlacements: map[string]CounselorPreviousPlacement{
					"co1": {AgeGroupID: "ag1"},
				},
				AgeGroupPreferences: map[string][]RankedPreference{
					"co1": {{TargetID: "ag2", Rank: 1}},
				},
			},
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"co1"},
				},
			},
			wantCount: 0,
			wantScore: 0,
		},
		{
			name: "rank 1 matches previous still scores",
			snapshot: SessionSnapshot{
				Cabins: []Cabin{
					{ID: "c1", Name: "Pine", AgeGroupID: "ag1"},
				},
				CounselorPreviousPlacements: map[string]CounselorPreviousPlacement{
					"co1": {AgeGroupID: "ag1"},
				},
				AgeGroupPreferences: map[string][]RankedPreference{
					"co1": {
						{TargetID: "ag1", Rank: 1},
						{TargetID: "ag2", Rank: 2},
					},
				},
			},
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"co1"},
				},
			},
			wantCount: 1,
			wantScore: weights.ReturningAgeGroup,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			components := scoreReturningAgeGroup(tt.snapshot, tt.assignment, weights)
			if len(components) != tt.wantCount {
				t.Fatalf("got %d components, want %d", len(components), tt.wantCount)
			}
			var total float64
			for _, c := range components {
				total += c.Score
				if c.Constraint != "returning_age_group" {
					t.Errorf("got constraint %q, want %q", c.Constraint, "returning_age_group")
				}
			}
			if !floatEqual(total, tt.wantScore) {
				t.Errorf("got total score %f, want %f", total, tt.wantScore)
			}
		})
	}
}

func TestScoreReturningCabin(t *testing.T) {
	weights := DefaultWeights()

	tests := []struct {
		name       string
		snapshot   SessionSnapshot
		assignment Assignment
		wantCount  int
		wantScore  float64
	}{
		{
			name: "returning to same cabin scores",
			snapshot: SessionSnapshot{
				Cabins: []Cabin{
					{ID: "c1", Name: "Pine", AgeGroupID: "ag1"},
				},
				CounselorPreviousPlacements: map[string]CounselorPreviousPlacement{
					"co1": {AgeGroupID: "ag1", CabinID: "c1"},
				},
			},
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"co1"},
				},
			},
			wantCount: 1,
			wantScore: weights.ReturningCabin,
		},
		{
			name: "returning to different cabin",
			snapshot: SessionSnapshot{
				Cabins: []Cabin{
					{ID: "c1", Name: "Pine", AgeGroupID: "ag1"},
					{ID: "c2", Name: "Oak", AgeGroupID: "ag2"},
				},
				CounselorPreviousPlacements: map[string]CounselorPreviousPlacement{
					"co1": {AgeGroupID: "ag1", CabinID: "c1"},
				},
			},
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c2": {"co1"},
				},
			},
			wantCount: 0,
			wantScore: 0,
		},
		{
			name: "empty cabin ID in history skipped",
			snapshot: SessionSnapshot{
				Cabins: []Cabin{
					{ID: "c1", Name: "Pine", AgeGroupID: "ag1"},
				},
				CounselorPreviousPlacements: map[string]CounselorPreviousPlacement{
					"co1": {AgeGroupID: "ag1", CabinID: ""},
				},
			},
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"co1"},
				},
			},
			wantCount: 0,
			wantScore: 0,
		},
		{
			name: "counselor wants change skips cabin bonus",
			snapshot: SessionSnapshot{
				Cabins: []Cabin{
					{ID: "c1", Name: "Pine", AgeGroupID: "ag1"},
				},
				CounselorPreviousPlacements: map[string]CounselorPreviousPlacement{
					"co1": {AgeGroupID: "ag1", CabinID: "c1"},
				},
				AgeGroupPreferences: map[string][]RankedPreference{
					"co1": {{TargetID: "ag2", Rank: 1}},
				},
			},
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"co1"},
				},
			},
			wantCount: 0,
			wantScore: 0,
		},
		{
			name: "rank 1 matches previous still scores cabin bonus",
			snapshot: SessionSnapshot{
				Cabins: []Cabin{
					{ID: "c1", Name: "Pine", AgeGroupID: "ag1"},
				},
				CounselorPreviousPlacements: map[string]CounselorPreviousPlacement{
					"co1": {AgeGroupID: "ag1", CabinID: "c1"},
				},
				AgeGroupPreferences: map[string][]RankedPreference{
					"co1": {{TargetID: "ag1", Rank: 1}},
				},
			},
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"co1"},
				},
			},
			wantCount: 1,
			wantScore: weights.ReturningCabin,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			components := scoreReturningCabin(tt.snapshot, tt.assignment, weights)
			if len(components) != tt.wantCount {
				t.Fatalf("got %d components, want %d", len(components), tt.wantCount)
			}
			var total float64
			for _, c := range components {
				total += c.Score
				if c.Constraint != "returning_cabin" {
					t.Errorf("got constraint %q, want %q", c.Constraint, "returning_cabin")
				}
			}
			if !floatEqual(total, tt.wantScore) {
				t.Errorf("got total score %f, want %f", total, tt.wantScore)
			}
		})
	}
}

func TestScoreCocounselorPreference(t *testing.T) {
	weights := DefaultWeights()

	snapshot := SessionSnapshot{
		Cabins: []Cabin{
			{ID: "c1", Name: "Pine"},
			{ID: "c2", Name: "Oak"},
		},
		Counselors: []Counselor{
			{ID: "co1", Name: "Alice"},
			{ID: "co2", Name: "Bob"},
			{ID: "co3", Name: "Carol"},
		},
		CocounselorPreferences: map[string][]RankedPreference{
			"co1": {
				{TargetID: "co2", Rank: 1},
				{TargetID: "co3", Rank: 2},
			},
		},
	}

	tests := []struct {
		name       string
		assignment Assignment
		wantCount  int
		wantScore  float64
	}{
		{
			name: "rank 1 preference satisfied",
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"co1", "co2"},
					"c2": {"co3"},
				},
			},
			wantCount: 1,
			wantScore: weights.CocounselorPreference, // 5.0 / 1
		},
		{
			name: "rank 2 preference satisfied",
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"co1", "co3"},
					"c2": {"co2"},
				},
			},
			wantCount: 1,
			wantScore: weights.CocounselorPreference / 2, // 5.0 / 2
		},
		{
			name: "both preferences satisfied in same cabin",
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"co1", "co2", "co3"},
				},
			},
			wantCount: 2,
			wantScore: weights.CocounselorPreference + weights.CocounselorPreference/2,
		},
		{
			name: "no preference satisfied",
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"co1"},
					"c2": {"co2", "co3"},
				},
			},
			wantCount: 0,
			wantScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			components := scoreCocounselorPreference(snapshot, tt.assignment, weights)
			if len(components) != tt.wantCount {
				t.Fatalf("got %d components, want %d", len(components), tt.wantCount)
			}
			var total float64
			for _, c := range components {
				total += c.Score
				if c.Constraint != "cocounselor_preference" {
					t.Errorf("got constraint %q, want %q", c.Constraint, "cocounselor_preference")
				}
			}
			if !floatEqual(total, tt.wantScore) {
				t.Errorf("got total score %f, want %f", total, tt.wantScore)
			}
		})
	}
}

func TestScoreAgeGroupPreference(t *testing.T) {
	weights := DefaultWeights()

	snapshot := SessionSnapshot{
		Cabins: []Cabin{
			{ID: "c1", Name: "Pine", AgeGroupID: "ag1"},
			{ID: "c2", Name: "Oak", AgeGroupID: "ag2"},
		},
		Counselors: []Counselor{
			{ID: "co1", Name: "Alice"},
			{ID: "co2", Name: "Bob"},
		},
		AgeGroupPreferences: map[string][]RankedPreference{
			"co1": {
				{TargetID: "ag2", Rank: 1},
				{TargetID: "ag1", Rank: 2},
			},
			"co2": {
				{TargetID: "ag1", Rank: 1},
			},
		},
	}

	tests := []struct {
		name       string
		assignment Assignment
		wantCount  int
		wantScore  float64
	}{
		{
			name: "rank 1 age group preference",
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c2": {"co1"},
					"c1": {"co2"},
				},
			},
			wantCount: 2,
			wantScore: weights.AgeGroupPreference + weights.AgeGroupPreference,
		},
		{
			name: "rank 2 age group preference",
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"co1"},
				},
			},
			wantCount: 1,
			wantScore: weights.AgeGroupPreference / 2,
		},
		{
			name: "no preference match",
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c2": {"co2"},
				},
			},
			wantCount: 0,
			wantScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			components := scoreAgeGroupPreference(snapshot, tt.assignment, weights)
			if len(components) != tt.wantCount {
				t.Fatalf("got %d components, want %d", len(components), tt.wantCount)
			}
			var total float64
			for _, c := range components {
				total += c.Score
				if c.Constraint != "age_group_preference" {
					t.Errorf("got constraint %q, want %q", c.Constraint, "age_group_preference")
				}
			}
			if !floatEqual(total, tt.wantScore) {
				t.Errorf("got total score %f, want %f", total, tt.wantScore)
			}
		})
	}
}

func TestScoreMultipleSeniors(t *testing.T) {
	weights := DefaultWeights()

	snapshot := SessionSnapshot{
		Cabins: []Cabin{
			{ID: "c1", Name: "Pine"},
			{ID: "c2", Name: "Oak"},
		},
		Counselors: []Counselor{
			{ID: "sr1", Name: "Counselor 1", IsJunior: false},
			{ID: "sr2", Name: "Counselor 2", IsJunior: false},
			{ID: "sr3", Name: "Counselor 3", IsJunior: false},
			{ID: "jr1", Name: "Junior 1", IsJunior: true},
		},
	}

	tests := []struct {
		name       string
		assignment Assignment
		wantCount  int
		wantScore  float64
	}{
		{
			name: "two seniors in cabin scores",
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"sr1", "sr2"},
				},
			},
			wantCount: 1,
			wantScore: weights.MultipleSeniors,
		},
		{
			name: "three seniors in cabin scores once",
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"sr1", "sr2", "sr3"},
				},
			},
			wantCount: 1,
			wantScore: weights.MultipleSeniors,
		},
		{
			name: "one senior and one junior no score",
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"sr1", "jr1"},
				},
			},
			wantCount: 0,
			wantScore: 0,
		},
		{
			name: "single senior no score",
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"sr1"},
				},
			},
			wantCount: 0,
			wantScore: 0,
		},
		{
			name: "both cabins with multiple seniors",
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"sr1", "sr2"},
					"c2": {"sr3", "sr1"},
				},
			},
			wantCount: 2,
			wantScore: weights.MultipleSeniors * 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			components := scoreMultipleSeniors(snapshot, tt.assignment, weights)
			if len(components) != tt.wantCount {
				t.Fatalf("got %d components, want %d", len(components), tt.wantCount)
			}
			var total float64
			for _, c := range components {
				total += c.Score
				if c.Constraint != "multiple_seniors" {
					t.Errorf("got constraint %q, want %q", c.Constraint, "multiple_seniors")
				}
			}
			if !floatEqual(total, tt.wantScore) {
				t.Errorf("got total score %f, want %f", total, tt.wantScore)
			}
		})
	}
}

func TestScoreSoftConstraints(t *testing.T) {
	weights := DefaultWeights()

	t.Run("aggregates all scorers", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1"},
			},
			Counselors: []Counselor{
				{ID: "sr1", Name: "Counselor 1", IsJunior: false},
				{ID: "sr2", Name: "Counselor 2", IsJunior: false},
			},
			CounselorPreviousPlacements: map[string]CounselorPreviousPlacement{
				"sr1": {AgeGroupID: "ag1", CabinID: "c1"},
			},
			AgeGroupPreferences: map[string][]RankedPreference{
				"sr2": {{TargetID: "ag1", Rank: 1}},
			},
		}
		assignment := Assignment{
			CabinCounselors: map[string][]string{
				"c1": {"sr1", "sr2"},
			},
		}

		result := ScoreSoftConstraints(snapshot, assignment, weights)

		wantTotal := weights.ReturningAgeGroup + weights.ReturningCabin + weights.AgeGroupPreference + weights.MultipleSeniors
		if !floatEqual(result.Total, wantTotal) {
			t.Errorf("got total %f, want %f", result.Total, wantTotal)
		}
		if len(result.Breakdown) != 4 {
			t.Errorf("got %d breakdown components, want 4", len(result.Breakdown))
		}
	})

	t.Run("zero weight disables constraint", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1"},
			},
			Counselors: []Counselor{
				{ID: "sr1", Name: "Counselor 1", IsJunior: false},
				{ID: "sr2", Name: "Counselor 2", IsJunior: false},
			},
			CounselorPreviousPlacements: map[string]CounselorPreviousPlacement{
				"sr1": {AgeGroupID: "ag1", CabinID: "c1"},
			},
		}
		assignment := Assignment{
			CabinCounselors: map[string][]string{
				"c1": {"sr1", "sr2"},
			},
		}

		zeroWeights := Weights{}
		result := ScoreSoftConstraints(snapshot, assignment, zeroWeights)
		if result.Total != 0 {
			t.Errorf("got total %f, want 0", result.Total)
		}
		if len(result.Breakdown) != 0 {
			t.Errorf("got %d breakdown components, want 0", len(result.Breakdown))
		}
	})
}
