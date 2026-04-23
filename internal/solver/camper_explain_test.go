package solver

import (
	"strings"
	"testing"
)

func TestExplainCamperIneligibility(t *testing.T) {
	t.Run("cross-gender friend preference is ineligible", func(t *testing.T) {
		snapshot := CamperCabinSnapshot{
			Cabins: []CamperCabin{
				{ID: "fcab", Name: "Pine", AgeGroupID: "ag1", Capacity: 10, Gender: "female"},
				{ID: "mcab", Name: "Cedar", AgeGroupID: "ag1", Capacity: 10, Gender: "male"},
			},
			Campers: []Camper{
				{ID: "f1", Name: "Emma", AgeGroupID: "ag1", Gender: "female"},
				{ID: "m1", Name: "Liam", AgeGroupID: "ag1", Gender: "male"},
			},
			FriendPreferences: map[string][]RankedPreference{
				"f1": {{TargetID: "m1", Rank: 1}},
			},
		}
		assignment := CamperAssignment{
			CabinCampers: map[string][]string{
				"fcab": {"f1"},
				"mcab": {"m1"},
			},
		}
		solution := CamperSolution{
			Assignment: assignment,
			Score:      ScoreCamperSoftConstraints(snapshot, assignment, DefaultCamperWeights()),
		}
		explanation := ExplainCamper(snapshot, solution)

		for _, u := range explanation.UnmetPreferences {
			if u.CamperID == "f1" {
				t.Errorf("cross-gender pref should not appear as unmet, got %v", u)
			}
		}

		if len(explanation.IneligiblePreferences) != 1 {
			t.Fatalf("expected 1 ineligible, got %v", explanation.IneligiblePreferences)
		}
		ip := explanation.IneligiblePreferences[0]
		if ip.Constraint != "friend_preference_ineligible" {
			t.Errorf("expected friend_preference_ineligible, got %q", ip.Constraint)
		}
		if !strings.Contains(ip.Message, "Liam") || !strings.Contains(ip.Message, "gender mismatch") {
			t.Errorf("unexpected message: %q", ip.Message)
		}
	})

	t.Run("friend preference target not enrolled is ineligible", func(t *testing.T) {
		snapshot := CamperCabinSnapshot{
			Cabins: []CamperCabin{
				{ID: "fcab", Name: "Pine", AgeGroupID: "ag1", Capacity: 10, Gender: "female"},
			},
			Campers: []Camper{
				{ID: "f1", Name: "Emma", AgeGroupID: "ag1", Gender: "female"},
			},
			FriendPreferences: map[string][]RankedPreference{
				"f1": {{TargetID: "ghost", Rank: 1}},
			},
		}
		assignment := CamperAssignment{CabinCampers: map[string][]string{"fcab": {"f1"}}}
		solution := CamperSolution{
			Assignment: assignment,
			Score:      ScoreCamperSoftConstraints(snapshot, assignment, DefaultCamperWeights()),
		}
		explanation := ExplainCamper(snapshot, solution)

		if len(explanation.UnmetPreferences) != 0 {
			t.Errorf("not-enrolled target should not appear as unmet, got %v", explanation.UnmetPreferences)
		}

		if len(explanation.IneligiblePreferences) != 1 {
			t.Fatalf("expected 1 ineligible, got %v", explanation.IneligiblePreferences)
		}
		ip := explanation.IneligiblePreferences[0]
		if !strings.Contains(ip.Message, "not enrolled") {
			t.Errorf("unexpected message: %q", ip.Message)
		}
	})

	t.Run("cross-age-group friend preference is ineligible", func(t *testing.T) {
		snapshot := CamperCabinSnapshot{
			Cabins: []CamperCabin{
				{ID: "ag1cab", Name: "Pine", AgeGroupID: "ag1", Capacity: 10, Gender: "female"},
				{ID: "ag2cab", Name: "Oak", AgeGroupID: "ag2", Capacity: 10, Gender: "female"},
			},
			Campers: []Camper{
				{ID: "f1", Name: "Emma", AgeGroupID: "ag1", Gender: "female"},
				{ID: "f2", Name: "Charlotte", AgeGroupID: "ag2", Gender: "female"},
			},
			FriendPreferences: map[string][]RankedPreference{
				"f1": {{TargetID: "f2", Rank: 1}},
			},
		}
		assignment := CamperAssignment{
			CabinCampers: map[string][]string{
				"ag1cab": {"f1"},
				"ag2cab": {"f2"},
			},
		}
		solution := CamperSolution{
			Assignment: assignment,
			Score:      ScoreCamperSoftConstraints(snapshot, assignment, DefaultCamperWeights()),
		}
		explanation := ExplainCamper(snapshot, solution)

		if len(explanation.UnmetPreferences) != 0 {
			t.Errorf("cross-age-group target should not appear as unmet, got %v", explanation.UnmetPreferences)
		}
		if len(explanation.IneligiblePreferences) != 1 {
			t.Fatalf("expected 1 ineligible, got %v", explanation.IneligiblePreferences)
		}
		if !strings.Contains(explanation.IneligiblePreferences[0].Message, "different age group") {
			t.Errorf("unexpected message: %q", explanation.IneligiblePreferences[0].Message)
		}
	})

	t.Run("eligible-but-unsatisfied friend preference is unmet not ineligible", func(t *testing.T) {
		snapshot := CamperCabinSnapshot{
			Cabins: []CamperCabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", Capacity: 10, Gender: "female"},
				{ID: "c2", Name: "Oak", AgeGroupID: "ag1", Capacity: 10, Gender: "female"},
			},
			Campers: []Camper{
				{ID: "f1", Name: "Emma", AgeGroupID: "ag1", Gender: "female"},
				{ID: "f2", Name: "Olivia", AgeGroupID: "ag1", Gender: "female"},
			},
			FriendPreferences: map[string][]RankedPreference{
				"f1": {{TargetID: "f2", Rank: 1}},
			},
		}
		assignment := CamperAssignment{
			CabinCampers: map[string][]string{
				"c1": {"f1"},
				"c2": {"f2"},
			},
		}
		solution := CamperSolution{
			Assignment: assignment,
			Score:      ScoreCamperSoftConstraints(snapshot, assignment, DefaultCamperWeights()),
		}
		explanation := ExplainCamper(snapshot, solution)

		if len(explanation.IneligiblePreferences) != 0 {
			t.Errorf("same-gender same-age-group should not be ineligible, got %v", explanation.IneligiblePreferences)
		}
		if len(explanation.UnmetPreferences) != 1 {
			t.Fatalf("expected 1 unmet, got %v", explanation.UnmetPreferences)
		}
		if explanation.UnmetPreferences[0].Constraint != "friend_preference" {
			t.Errorf("expected friend_preference, got %q", explanation.UnmetPreferences[0].Constraint)
		}
	})
}
