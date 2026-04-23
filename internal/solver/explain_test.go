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
				if !strings.Contains(r.Message, "+") {
					t.Errorf("reason missing score: %q", r.Message)
				}
			}
		}
	})

	t.Run("no preferences produces no assignment explanation", func(t *testing.T) {
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

		if len(explanation.Assignments) != 0 {
			t.Errorf("expected no assignment explanations when there are no scoring reasons, got %d", len(explanation.Assignments))
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

		if len(explanation.Assignments) != 0 {
			t.Errorf("expected 0 assignment explanations (sr1 has no scoring reasons), got %d", len(explanation.Assignments))
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

	t.Run("cross-gender cocounselor preference is ineligible", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "cf", AgeGroupID: "ag", Gender: "female", RequiredCounselors: 1, Capacity: 4},
				{ID: "cm", AgeGroupID: "ag", Gender: "male", RequiredCounselors: 1, Capacity: 4},
			},
			Counselors: []Counselor{
				{ID: "f1", Name: "Sarah", Gender: "female"},
				{ID: "m1", Name: "Mike", Gender: "male"},
			},
			CocounselorPreferences: map[string][]RankedPreference{
				"f1": {{TargetID: "m1", Rank: 1}},
			},
		}

		assignment := Assignment{
			CabinCounselors: map[string][]string{
				"cf": {"f1"},
				"cm": {"m1"},
			},
		}
		score := ScoreSoftConstraints(snapshot, assignment, DefaultWeights())
		explanation := Explain(snapshot, Solution{Assignment: assignment, Score: score})

		for _, u := range explanation.UnmetPreferences {
			if u.CounselorID == "f1" && u.Constraint == "cocounselor_preference" {
				t.Errorf("did not expect unmet cocounselor entry for cross-gender pref, got %v", u)
			}
		}

		var ip IneligiblePreference
		for _, x := range explanation.IneligiblePreferences {
			if x.CounselorID == "f1" {
				ip = x
				break
			}
		}
		if ip.Constraint != "cocounselor_preference_ineligible" {
			t.Fatalf("expected cocounselor_preference_ineligible for f1, got %v", explanation.IneligiblePreferences)
		}
		if !strings.Contains(ip.Message, "Mike") || !strings.Contains(ip.Message, "gender mismatch") {
			t.Errorf("unexpected message: %q", ip.Message)
		}
	})

	t.Run("cocounselor preference target off roster is ineligible", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "cf", AgeGroupID: "ag", Gender: "female", RequiredCounselors: 1, Capacity: 4},
			},
			Counselors: []Counselor{
				{ID: "f1", Name: "Sarah", Gender: "female"},
			},
			CocounselorPreferences: map[string][]RankedPreference{
				"f1": {{TargetID: "ghost", Rank: 1}},
			},
		}

		assignment := Assignment{
			CabinCounselors: map[string][]string{"cf": {"f1"}},
		}
		score := ScoreSoftConstraints(snapshot, assignment, DefaultWeights())
		explanation := Explain(snapshot, Solution{Assignment: assignment, Score: score})

		for _, u := range explanation.UnmetPreferences {
			if u.CounselorID == "f1" && u.Constraint == "cocounselor_preference" {
				t.Errorf("did not expect unmet cocounselor entry for off-roster target, got %v", u)
			}
		}

		var ip IneligiblePreference
		for _, x := range explanation.IneligiblePreferences {
			if x.CounselorID == "f1" {
				ip = x
				break
			}
		}
		if ip.Constraint != "cocounselor_preference_ineligible" {
			t.Fatalf("expected cocounselor_preference_ineligible for f1, got %v", explanation.IneligiblePreferences)
		}
		if !strings.Contains(ip.Message, "not on this session's roster") {
			t.Errorf("unexpected message: %q", ip.Message)
		}
	})

	t.Run("unassigned counselor surfaced as unmet preference", func(t *testing.T) {
		snapshot := SessionSnapshot{
			Cabins: []Cabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", RequiredCounselors: 1, Capacity: 1, Gender: "male", HasCampers: true},
			},
			Counselors: []Counselor{
				{ID: "sr1", Name: "Counselor 1", IsJunior: false, Gender: "male"},
				{ID: "sr2", Name: "Counselor 2", IsJunior: false, Gender: "male"},
			},
		}

		assignment := Assignment{
			CabinCounselors: map[string][]string{
				"c1": {"sr1"},
			},
		}
		score := ScoreSoftConstraints(snapshot, assignment, DefaultWeights())
		explanation := Explain(snapshot, Solution{Assignment: assignment, Score: score})

		var found bool
		for _, u := range explanation.UnmetPreferences {
			if u.Constraint == "unassigned_counselor" && u.CounselorID == "sr2" {
				found = true
				if !strings.Contains(u.Message, "not assigned to any cabin") {
					t.Errorf("unexpected message: %q", u.Message)
				}
			}
		}
		if !found {
			t.Errorf("expected unassigned_counselor entry for sr2, got %+v", explanation.UnmetPreferences)
		}
	})
}
