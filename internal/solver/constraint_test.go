package solver

import (
	"testing"
)

func TestCheckCabinMinimumCounselors(t *testing.T) {
	tests := []struct {
		name       string
		snapshot   SessionSnapshot
		assignment Assignment
		wantCount  int
		wantCabins []string // cabin IDs expected in violations
	}{
		{
			name: "all cabins meet minimum",
			snapshot: SessionSnapshot{
				Cabins: []Cabin{
					{ID: "c1", Name: "Pine", RequiredCounselors: 2, Capacity: 10, HasCampers: true},
					{ID: "c2", Name: "Oak", RequiredCounselors: 1, Capacity: 10, HasCampers: true},
				},
			},
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"a", "b"},
					"c2": {"c"},
				},
			},
			wantCount: 0,
		},
		{
			name: "cabin exceeds minimum is fine",
			snapshot: SessionSnapshot{
				Cabins: []Cabin{
					{ID: "c1", Name: "Pine", RequiredCounselors: 1, Capacity: 10, HasCampers: true},
				},
			},
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"a", "b", "c"},
				},
			},
			wantCount: 0,
		},
		{
			name: "one cabin under minimum",
			snapshot: SessionSnapshot{
				Cabins: []Cabin{
					{ID: "c1", Name: "Pine", RequiredCounselors: 2, Capacity: 10, HasCampers: true},
					{ID: "c2", Name: "Oak", RequiredCounselors: 1, Capacity: 10, HasCampers: true},
				},
			},
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"a"},
					"c2": {"b"},
				},
			},
			wantCount:  1,
			wantCabins: []string{"c1"},
		},
		{
			name: "multiple cabins under minimum",
			snapshot: SessionSnapshot{
				Cabins: []Cabin{
					{ID: "c1", Name: "Pine", RequiredCounselors: 2, Capacity: 10, HasCampers: true},
					{ID: "c2", Name: "Oak", RequiredCounselors: 3, Capacity: 10, HasCampers: true},
				},
			},
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"a"},
					"c2": {"b"},
				},
			},
			wantCount:  2,
			wantCabins: []string{"c1", "c2"},
		},
		{
			name: "cabin not in assignment map treated as zero",
			snapshot: SessionSnapshot{
				Cabins: []Cabin{
					{ID: "c1", Name: "Pine", RequiredCounselors: 1, Capacity: 10, HasCampers: true},
				},
			},
			assignment: Assignment{
				CabinCounselors: map[string][]string{},
			},
			wantCount:  1,
			wantCabins: []string{"c1"},
		},
		{
			name: "zero required with zero assigned is fine",
			snapshot: SessionSnapshot{
				Cabins: []Cabin{
					{ID: "c1", Name: "Pine", RequiredCounselors: 0, Capacity: 10, HasCampers: true},
				},
			},
			assignment: Assignment{
				CabinCounselors: map[string][]string{},
			},
			wantCount: 0,
		},
		{
			name: "cabin without campers exempt from minimum",
			snapshot: SessionSnapshot{
				Cabins: []Cabin{
					{ID: "c1", Name: "Pine", RequiredCounselors: 2, Capacity: 10, HasCampers: false},
				},
			},
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"a"},
				},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations := checkCabinMinimumCounselors(tt.snapshot, tt.assignment)
			if len(violations) != tt.wantCount {
				t.Fatalf("got %d violations, want %d", len(violations), tt.wantCount)
			}
			for i, cabinID := range tt.wantCabins {
				if violations[i].CabinID != cabinID {
					t.Errorf("violation[%d].CabinID = %q, want %q", i, violations[i].CabinID, cabinID)
				}
				if violations[i].Constraint != "cabin_minimum_counselors" {
					t.Errorf("violation[%d].Constraint = %q, want %q", i, violations[i].Constraint, "cabin_minimum_counselors")
				}
			}
		})
	}
}

func TestCheckCabinWithoutSeniorCounselor(t *testing.T) {
	snapshot := SessionSnapshot{
		Counselors: []Counselor{
			{ID: "jr1", Name: "Junior 1", IsJunior: true},
			{ID: "jr2", Name: "Junior 2", IsJunior: true},
			{ID: "sr1", Name: "Counselor 1", IsJunior: false},
			{ID: "sr2", Name: "Counselor 2", IsJunior: false},
		},
		Cabins: []Cabin{
			{ID: "c1", Name: "Pine", HasCampers: true},
			{ID: "c2", Name: "Oak", HasCampers: true},
		},
	}

	tests := []struct {
		name       string
		cabins     []Cabin
		assignment Assignment
		wantCount  int
		wantCabins []string
	}{
		{
			name: "single senior is fine",
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"sr1"},
				},
			},
			wantCount: 0,
		},
		{
			name: "single junior violates",
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"jr1"},
				},
			},
			wantCount:  1,
			wantCabins: []string{"c1"},
		},
		{
			name: "two juniors and no senior violates",
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"jr1", "jr2"},
				},
			},
			wantCount:  1,
			wantCabins: []string{"c1"},
		},
		{
			name: "junior plus senior is fine",
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"jr1", "sr1"},
				},
			},
			wantCount: 0,
		},
		{
			name: "empty cabin not flagged",
			assignment: Assignment{
				CabinCounselors: map[string][]string{},
			},
			wantCount: 0,
		},
		{
			name: "mixed cabins only violating one flagged",
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"jr1"},
					"c2": {"sr1", "jr2"},
				},
			},
			wantCount:  1,
			wantCabins: []string{"c1"},
		},
		{
			name: "junior-only cabin without campers exempt",
			cabins: []Cabin{
				{ID: "c1", Name: "Pine", HasCampers: false},
			},
			assignment: Assignment{
				CabinCounselors: map[string][]string{
					"c1": {"jr1"},
				},
			},
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snap := snapshot
			if tt.cabins != nil {
				snap.Cabins = tt.cabins
			}
			violations := checkCabinWithoutSeniorCounselor(snap, tt.assignment)
			if len(violations) != tt.wantCount {
				t.Fatalf("got %d violations, want %d", len(violations), tt.wantCount)
			}
			for i, cabinID := range tt.wantCabins {
				if violations[i].CabinID != cabinID {
					t.Errorf("violation[%d].CabinID = %q, want %q", i, violations[i].CabinID, cabinID)
				}
				if violations[i].Constraint != "cabin_without_senior_counselor" {
					t.Errorf("violation[%d].Constraint = %q, want %q", i, violations[i].Constraint, "cabin_without_senior_counselor")
				}
			}
		})
	}
}

func TestCheckHardConstraints(t *testing.T) {
	t.Run("valid assignment returns no violations", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", RequiredCounselors: 1, Capacity: 10, HasCampers: true},
			},
			Counselors: []Counselor{
				{ID: "sr1", Name: "Counselor 1", IsJunior: false},
			},
		}
		assignment := Assignment{
			CabinCounselors: map[string][]string{
				"c1": {"sr1"},
			},
		}

		violations := CheckHardConstraints(snapshot, assignment)
		if len(violations) != 0 {
			t.Fatalf("got %d violations, want 0", len(violations))
		}
	})

	t.Run("both constraint types violated", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", RequiredCounselors: 2, Capacity: 10, HasCampers: true},
				{ID: "c2", Name: "Oak", RequiredCounselors: 1, Capacity: 10, HasCampers: true},
			},
			Counselors: []Counselor{
				{ID: "jr1", Name: "Junior 1", IsJunior: true},
				{ID: "sr1", Name: "Counselor 1", IsJunior: false},
			},
		}
		assignment := Assignment{
			CabinCounselors: map[string][]string{
				"c1": {"sr1"},
				"c2": {"jr1"},
			},
		}

		violations := CheckHardConstraints(snapshot, assignment)

		constraintCounts := map[string]int{}
		for _, v := range violations {
			constraintCounts[v.Constraint]++
		}
		if constraintCounts["cabin_minimum_counselors"] != 1 {
			t.Errorf("expected 1 cabin_minimum_counselors violation, got %d", constraintCounts["cabin_minimum_counselors"])
		}
		if constraintCounts["cabin_without_senior_counselor"] != 1 {
			t.Errorf("expected 1 cabin_without_senior_counselor violation, got %d", constraintCounts["cabin_without_senior_counselor"])
		}
	})
}

func TestCheckCabinGenderMatchCounselors(t *testing.T) {
	snapshot := SessionSnapshot{
		Counselors: []Counselor{
			{ID: "fc1", Name: "Female Counselor 1", Gender: "female"},
			{ID: "fc2", Name: "Female Counselor 2", Gender: "female"},
			{ID: "mc1", Name: "Male Counselor 1", Gender: "male"},
		},
		Cabins: []Cabin{
			{ID: "fcab", Name: "Pine", Gender: "female"},
			{ID: "mcab", Name: "Cedar", Gender: "male"},
		},
	}

	t.Run("matching genders pass", func(t *testing.T) {
		assignment := Assignment{
			CabinCounselors: map[string][]string{
				"fcab": {"fc1", "fc2"},
				"mcab": {"mc1"},
			},
		}
		v := checkCabinGenderMatchCounselors(snapshot, assignment)
		if len(v) != 0 {
			t.Fatalf("got %d violations, want 0", len(v))
		}
	})

	t.Run("male counselor in female cabin violates", func(t *testing.T) {
		assignment := Assignment{
			CabinCounselors: map[string][]string{
				"fcab": {"fc1", "mc1"},
			},
		}
		v := checkCabinGenderMatchCounselors(snapshot, assignment)
		if len(v) != 1 {
			t.Fatalf("got %d violations, want 1", len(v))
		}
		if v[0].Constraint != "cabin_gender_mismatch" {
			t.Errorf("got constraint %q, want cabin_gender_mismatch", v[0].Constraint)
		}
		if v[0].CounselorID != "mc1" || v[0].CabinID != "fcab" {
			t.Errorf("got counselor=%q cabin=%q", v[0].CounselorID, v[0].CabinID)
		}
	})

	t.Run("female counselor in male cabin violates", func(t *testing.T) {
		assignment := Assignment{
			CabinCounselors: map[string][]string{
				"mcab": {"fc1"},
			},
		}
		v := checkCabinGenderMatchCounselors(snapshot, assignment)
		if len(v) != 1 {
			t.Fatalf("got %d violations, want 1", len(v))
		}
	})
}
