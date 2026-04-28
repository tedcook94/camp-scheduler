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

func TestRepeatedUnmetAgeGroupBoost(t *testing.T) {
	weights := DefaultWeights()

	snapshot := SessionSnapshot{
		Cabins: []Cabin{
			{ID: "c1", Name: "Pine", AgeGroupID: "ag1"},
			{ID: "c2", Name: "Oak", AgeGroupID: "ag2"},
		},
		AgeGroupPreferences: map[string][]RankedPreference{
			"co1": {{TargetID: "ag1", Rank: 1}},
			"co2": {{TargetID: "ag2", Rank: 1}},
		},
		UnmetAgeGroupPreferences: map[string]map[string]bool{
			"co1": {"ag1": true},
		},
	}

	assignment := Assignment{
		CabinCounselors: map[string][]string{
			"c1": {"co1"},
			"c2": {"co2"},
		},
	}

	components := scoreAgeGroupPreference(snapshot, assignment, weights)
	if len(components) != 2 {
		t.Fatalf("got %d components, want 2", len(components))
	}

	// Find scores by counselor.
	scores := make(map[string]float64)
	messages := make(map[string]string)
	for _, c := range components {
		scores[c.CounselorID] = c.Score
		messages[c.CounselorID] = c.Message
	}

	// co1 had a previously unmet preference -- should be boosted.
	wantBoosted := weights.AgeGroupPreference * weights.RepeatedUnmetBoost
	if !floatEqual(scores["co1"], wantBoosted) {
		t.Errorf("co1: got score %f, want %f (boosted)", scores["co1"], wantBoosted)
	}
	if got := messages["co1"]; got == "" || !contains(got, "previously unmet") {
		t.Errorf("co1: expected message to contain 'previously unmet', got %q", got)
	}

	// co2 had no unmet preference -- should be normal.
	if !floatEqual(scores["co2"], weights.AgeGroupPreference) {
		t.Errorf("co2: got score %f, want %f (normal)", scores["co2"], weights.AgeGroupPreference)
	}
	if got := messages["co2"]; contains(got, "previously unmet") {
		t.Errorf("co2: expected message without 'previously unmet', got %q", got)
	}
}

func TestRepeatedUnmetCocounselorBoost(t *testing.T) {
	weights := DefaultWeights()

	snapshot := SessionSnapshot{
		Cabins: []Cabin{
			{ID: "c1", Name: "Pine"},
		},
		CocounselorPreferences: map[string][]RankedPreference{
			"co1": {{TargetID: "co2", Rank: 1}},
		},
		UnmetCocounselorPreferences: map[string]map[string]bool{
			"co1": {"co2": true},
		},
	}

	assignment := Assignment{
		CabinCounselors: map[string][]string{
			"c1": {"co1", "co2"},
		},
	}

	components := scoreCocounselorPreference(snapshot, assignment, weights)
	if len(components) != 1 {
		t.Fatalf("got %d components, want 1", len(components))
	}

	wantScore := weights.CocounselorPreference * weights.RepeatedUnmetBoost
	if !floatEqual(components[0].Score, wantScore) {
		t.Errorf("got score %f, want %f", components[0].Score, wantScore)
	}
	if !contains(components[0].Message, "previously unmet") {
		t.Errorf("expected message to contain 'previously unmet', got %q", components[0].Message)
	}
}

func TestRepeatedUnmetActivityBoost(t *testing.T) {
	weights := DefaultActivityWeights()

	snapshot := ActivitySnapshot{
		Slots: []ActivitySlot{
			{ID: "s1", ActivityID: "act1", ActivityName: "Swimming", TimeSlotID: "ts1"},
		},
		ActivityPreferences: map[string][]RankedPreference{
			"co1": {{TargetID: "act1", Rank: 1}},
			"co2": {{TargetID: "act1", Rank: 1}},
		},
		UnmetActivityPreferences: map[string]map[string]bool{
			"co1": {"act1": true},
		},
	}

	assignment := CounselorActivityAssignment{
		SlotCounselors: map[string][]string{
			"s1": {"co1", "co2"},
		},
	}

	components := scoreActivityPreference(snapshot, assignment, weights)
	if len(components) != 2 {
		t.Fatalf("got %d components, want 2", len(components))
	}

	scores := make(map[string]float64)
	messages := make(map[string]string)
	for _, c := range components {
		scores[c.CounselorID] = c.Score
		messages[c.CounselorID] = c.Message
	}

	wantBoosted := weights.ActivityPreference * weights.RepeatedUnmetBoost
	if !floatEqual(scores["co1"], wantBoosted) {
		t.Errorf("co1: got score %f, want %f (boosted)", scores["co1"], wantBoosted)
	}
	if !contains(messages["co1"], "previously unmet") {
		t.Errorf("co1: expected message to contain 'previously unmet', got %q", messages["co1"])
	}

	if !floatEqual(scores["co2"], weights.ActivityPreference) {
		t.Errorf("co2: got score %f, want %f (normal)", scores["co2"], weights.ActivityPreference)
	}
}

func TestEffectiveBoost(t *testing.T) {
	if got := effectiveBoost(0); got != 1.0 {
		t.Errorf("effectiveBoost(0) = %f, want 1.0", got)
	}
	if got := effectiveBoost(1.5); got != 1.5 {
		t.Errorf("effectiveBoost(1.5) = %f, want 1.5", got)
	}
	if got := effectiveBoost(2.0); got != 2.0 {
		t.Errorf("effectiveBoost(2.0) = %f, want 2.0", got)
	}
}

func TestDiffAgeGroupPreferences(t *testing.T) {
	prefs := map[string][]RankedPreference{
		"co1": {
			{TargetID: "ag1", Rank: 1},
			{TargetID: "ag2", Rank: 2},
		},
		"co2": {
			{TargetID: "ag1", Rank: 1},
		},
		"co3": {
			{TargetID: "ag2", Rank: 1},
			{TargetID: "ag1", Rank: 2},
		},
	}
	counselorCabin := map[string]string{
		"co1": "c1",
		"co2": "c2",
		"co3": "c1",
	}
	cabinAgeGroup := map[string]string{
		"c1": "ag1",
		"c2": "ag1",
	}

	result := diffAgeGroupPreferences(prefs, counselorCabin, cabinAgeGroup)

	// co1 got ag1 which is their rank-1 pref -> nothing unmet.
	if result["co1"] != nil {
		t.Errorf("expected co1 to have no unmet prefs, got %v", result["co1"])
	}
	// co2 got ag1 which is their rank-1 pref -> nothing unmet.
	if result["co2"] != nil {
		t.Errorf("expected co2 to have no unmet prefs, got %v", result["co2"])
	}
	// co3 got ag1 but their rank-1 pref is ag2 -> ag2 is unmet.
	if !result["co3"]["ag2"] {
		t.Error("expected co3 ag2 to be unmet")
	}
}

func TestDiffAgeGroupPreferencesUnassigned(t *testing.T) {
	prefs := map[string][]RankedPreference{
		"co1": {
			{TargetID: "ag1", Rank: 1},
			{TargetID: "ag2", Rank: 2},
		},
	}
	counselorCabin := map[string]string{}
	cabinAgeGroup := map[string]string{}

	result := diffAgeGroupPreferences(prefs, counselorCabin, cabinAgeGroup)

	// Only rank-1 preference is tracked as unmet.
	if !result["co1"]["ag1"] {
		t.Error("expected co1 ag1 (rank 1) to be unmet when unassigned")
	}
	if result["co1"]["ag2"] {
		t.Error("expected co1 ag2 (rank 2) to NOT be tracked as unmet")
	}
}

func TestDiffCocounselorPreferences(t *testing.T) {
	prefs := map[string][]RankedPreference{
		"co1": {
			{TargetID: "co2", Rank: 1},
			{TargetID: "co3", Rank: 2},
		},
	}
	counselorCabin := map[string]string{
		"co1": "c1",
		"co2": "c1",
		"co3": "c2",
	}

	result := diffCocounselorPreferences(prefs, counselorCabin)

	// co1 and co2 are in the same cabin, so co2 pref is met.
	if result["co1"]["co2"] {
		t.Error("expected co1->co2 to NOT be unmet")
	}
	// co1 and co3 are in different cabins, so co3 pref is unmet.
	if !result["co1"]["co3"] {
		t.Error("expected co1->co3 to be unmet")
	}
}

func TestDiffCocounselorPreferencesUnassigned(t *testing.T) {
	prefs := map[string][]RankedPreference{
		"co1": {
			{TargetID: "co2", Rank: 1},
		},
	}
	counselorCabin := map[string]string{
		"co2": "c1",
	}

	result := diffCocounselorPreferences(prefs, counselorCabin)

	if !result["co1"]["co2"] {
		t.Error("expected co1->co2 to be unmet when co1 is unassigned")
	}
}

func TestDiffActivityPreferences(t *testing.T) {
	prefs := map[string][]RankedPreference{
		"co1": {
			{TargetID: "act1", Rank: 1},
			{TargetID: "act2", Rank: 2},
		},
	}
	counselorActivities := map[string]map[string]bool{
		"co1": {"act1": true},
	}

	result := diffActivityPreferences(prefs, counselorActivities)

	if result["co1"]["act1"] {
		t.Error("expected co1->act1 to NOT be unmet (assigned)")
	}
	if !result["co1"]["act2"] {
		t.Error("expected co1->act2 to be unmet (not assigned)")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
