package solver

import (
	"strings"
	"testing"
)

func TestExplain(t *testing.T) {
	t.Run("all preferences satisfied", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 1},
				{ID: "c2", Name: "Oak", AgeGroupID: "ag2", RequiredCounselors: 1},
			},
			Counselors: []Counselor{
				{ID: "sr1", Name: "Counselor 1", IsJunior: false},
				{ID: "sr2", Name: "Counselor 2", IsJunior: false},
			},
			CounselorPreviousPlacements: map[string]CounselorPreviousPlacement{
				"sr1": {AgeGroupID: "ag1", CabinID: "c1"},
			},
			AgeGroupPreferences: map[string][]RankedPreference{
				"sr2": {{TargetID: "ag2", Rank: 1}},
			},
		}

		assignment := Assignment{
			CabinCounselors: map[string][]string{
				"c1": {"sr1"},
				"c2": {"sr2"},
			},
		}
		score := ScoreSoftConstraints(snapshot, assignment, DefaultWeights())
		solution := Solution{Assignment: assignment, Score: score}

		explanation := Explain(snapshot, solution)

		if len(explanation.Assignments) != 2 {
			t.Fatalf("expected 2 assignment explanations, got %d", len(explanation.Assignments))
		}
		if len(explanation.UnmetPreferences) != 0 {
			t.Errorf("expected no unmet preferences, got %d: %v", len(explanation.UnmetPreferences), explanation.UnmetPreferences)
		}

		for _, ae := range explanation.Assignments {
			if len(ae.Reasons) == 0 {
				t.Errorf("counselor %s has no reasons", ae.CounselorID)
			}
			for _, r := range ae.Reasons {
				if !strings.Contains(r, "+") {
					t.Errorf("reason missing score: %q", r)
				}
			}
		}
	})

	t.Run("no preferences gives default reason", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 1},
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
		score := ScoreSoftConstraints(snapshot, assignment, DefaultWeights())
		solution := Solution{Assignment: assignment, Score: score}

		explanation := Explain(snapshot, solution)

		if len(explanation.Assignments) != 1 {
			t.Fatalf("expected 1 assignment explanation, got %d", len(explanation.Assignments))
		}
		if explanation.Assignments[0].Reasons[0] != "assigned to fill cabin requirement" {
			t.Errorf("expected default reason, got %q", explanation.Assignments[0].Reasons[0])
		}
	})

	t.Run("unmet age group preference with ranking info", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 1},
				{ID: "c2", Name: "Oak", AgeGroupID: "ag2", RequiredCounselors: 1},
			},
			Counselors: []Counselor{
				{ID: "sr1", Name: "Counselor 1", IsJunior: false},
				{ID: "sr2", Name: "Counselor 2", IsJunior: false},
			},
			AgeGroupPreferences: map[string][]RankedPreference{
				"sr1": {
					{TargetID: "ag2", Rank: 1},
					{TargetID: "ag1", Rank: 2},
				},
			},
		}

		assignment := Assignment{
			CabinCounselors: map[string][]string{
				"c1": {"sr1"},
				"c2": {"sr2"},
			},
		}
		score := ScoreSoftConstraints(snapshot, assignment, DefaultWeights())
		solution := Solution{Assignment: assignment, Score: score}

		explanation := Explain(snapshot, solution)

		found := false
		for _, u := range explanation.UnmetPreferences {
			if u.CounselorID == "sr1" && u.Constraint == "age_group_preference" {
				found = true
				if !strings.Contains(u.Message, "rank 2 of 2 preferences") {
					t.Errorf("expected ranking info in message, got %q", u.Message)
				}
				if !strings.Contains(u.Message, "preferred age group") {
					t.Errorf("expected preferred age group in message, got %q", u.Message)
				}
			}
		}
		if !found {
			t.Error("expected unmet age group preference for sr1")
		}
	})

	t.Run("unmet age group preference not ranked", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 1},
				{ID: "c2", Name: "Oak", AgeGroupID: "ag2", RequiredCounselors: 1},
			},
			Counselors: []Counselor{
				{ID: "sr1", Name: "Counselor 1", IsJunior: false},
				{ID: "sr2", Name: "Counselor 2", IsJunior: false},
			},
			AgeGroupPreferences: map[string][]RankedPreference{
				"sr1": {
					{TargetID: "ag2", Rank: 1},
				},
			},
		}

		assignment := Assignment{
			CabinCounselors: map[string][]string{
				"c1": {"sr1"},
				"c2": {"sr2"},
			},
		}
		score := ScoreSoftConstraints(snapshot, assignment, DefaultWeights())
		solution := Solution{Assignment: assignment, Score: score}

		explanation := Explain(snapshot, solution)

		found := false
		for _, u := range explanation.UnmetPreferences {
			if u.CounselorID == "sr1" && u.Constraint == "age_group_preference" {
				found = true
				if !strings.Contains(u.Message, "not ranked") {
					t.Errorf("expected 'not ranked' in message, got %q", u.Message)
				}
			}
		}
		if !found {
			t.Error("expected unmet age group preference for sr1")
		}
	})

	t.Run("unmet returning age group", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 1},
				{ID: "c2", Name: "Oak", AgeGroupID: "ag2", RequiredCounselors: 1},
			},
			Counselors: []Counselor{
				{ID: "sr1", Name: "Counselor 1", IsJunior: false},
				{ID: "sr2", Name: "Counselor 2", IsJunior: false},
			},
			CounselorPreviousPlacements: map[string]CounselorPreviousPlacement{
				"sr1": {AgeGroupID: "ag1"},
			},
		}

		assignment := Assignment{
			CabinCounselors: map[string][]string{
				"c1": {"sr2"},
				"c2": {"sr1"},
			},
		}
		score := ScoreSoftConstraints(snapshot, assignment, DefaultWeights())
		solution := Solution{Assignment: assignment, Score: score}

		explanation := Explain(snapshot, solution)

		found := false
		for _, u := range explanation.UnmetPreferences {
			if u.CounselorID == "sr1" && u.Constraint == "returning_age_group" {
				found = true
				if !strings.Contains(u.Message, "previously in age group") {
					t.Errorf("unexpected message: %q", u.Message)
				}
			}
		}
		if !found {
			t.Error("expected unmet returning age group for sr1")
		}
	})

	t.Run("unmet returning cabin", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 1},
				{ID: "c2", Name: "Oak", AgeGroupID: "ag1", RequiredCounselors: 1},
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
				"c1": {"sr2"},
				"c2": {"sr1"},
			},
		}
		score := ScoreSoftConstraints(snapshot, assignment, DefaultWeights())
		solution := Solution{Assignment: assignment, Score: score}

		explanation := Explain(snapshot, solution)

		found := false
		for _, u := range explanation.UnmetPreferences {
			if u.CounselorID == "sr1" && u.Constraint == "returning_cabin" {
				found = true
				if !strings.Contains(u.Message, `previously in cabin "Pine"`) {
					t.Errorf("unexpected message: %q", u.Message)
				}
				if !strings.Contains(u.Message, `assigned to cabin "Oak"`) {
					t.Errorf("unexpected message: %q", u.Message)
				}
			}
		}
		if !found {
			t.Error("expected unmet returning cabin for sr1")
		}
	})

	t.Run("unmet cocounselor preference", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 1},
				{ID: "c2", Name: "Oak", AgeGroupID: "ag2", RequiredCounselors: 1},
			},
			Counselors: []Counselor{
				{ID: "sr1", Name: "Counselor 1", IsJunior: false},
				{ID: "sr2", Name: "Counselor 2", IsJunior: false},
			},
			CocounselorPreferences: map[string][]RankedPreference{
				"sr1": {{TargetID: "sr2", Rank: 1}},
			},
		}

		assignment := Assignment{
			CabinCounselors: map[string][]string{
				"c1": {"sr1"},
				"c2": {"sr2"},
			},
		}
		score := ScoreSoftConstraints(snapshot, assignment, DefaultWeights())
		solution := Solution{Assignment: assignment, Score: score}

		explanation := Explain(snapshot, solution)

		found := false
		for _, u := range explanation.UnmetPreferences {
			if u.CounselorID == "sr1" && u.Constraint == "cocounselor_preference" {
				found = true
				if !strings.Contains(u.Message, `"Counselor 2"`) {
					t.Errorf("expected co-counselor name in message, got %q", u.Message)
				}
			}
		}
		if !found {
			t.Error("expected unmet cocounselor preference for sr1")
		}
	})

	t.Run("unassigned counselors have unmet preferences", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 1},
			},
			Counselors: []Counselor{
				{ID: "sr1", Name: "Counselor 1", IsJunior: false},
				{ID: "sr2", Name: "Counselor 2", IsJunior: false},
			},
			AgeGroupPreferences: map[string][]RankedPreference{
				"sr2": {{TargetID: "ag1", Rank: 1}},
			},
		}

		assignment := Assignment{
			CabinCounselors: map[string][]string{
				"c1": {"sr1"},
			},
		}
		score := ScoreSoftConstraints(snapshot, assignment, DefaultWeights())
		solution := Solution{Assignment: assignment, Score: score}

		explanation := Explain(snapshot, solution)

		if len(explanation.Assignments) != 1 {
			t.Errorf("expected 1 assignment explanation, got %d", len(explanation.Assignments))
		}
		found := false
		for _, u := range explanation.UnmetPreferences {
			if u.CounselorID == "sr2" && u.Constraint == "age_group_preference" {
				found = true
				if !strings.Contains(u.Message, "not assigned to any cabin") {
					t.Errorf("expected unassigned detail in message, got %q", u.Message)
				}
			}
		}
		if !found {
			t.Error("expected unmet age group preference for unassigned counselor sr2")
		}
	})
}
