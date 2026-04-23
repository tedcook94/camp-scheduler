package solver

import (
	"reflect"
	"testing"
)

func TestFilterMapByRoster(t *testing.T) {
	roster := map[string]bool{"a": true, "b": true}

	t.Run("drops keys not on roster", func(t *testing.T) {
		in := map[string][]RankedPreference{
			"a": {{TargetID: "x", Rank: 1}},
			"b": {{TargetID: "y", Rank: 1}},
			"c": {{TargetID: "z", Rank: 1}},
		}
		out := filterMapByRoster(in, roster)

		if _, ok := out["c"]; ok {
			t.Errorf("expected c to be dropped, got %v", out["c"])
		}
		if len(out) != 2 {
			t.Errorf("expected 2 entries, got %d", len(out))
		}
	})

	t.Run("empty input returns empty", func(t *testing.T) {
		var in map[string]CounselorPreviousPlacement
		out := filterMapByRoster(in, roster)
		if len(out) != 0 {
			t.Errorf("expected empty result, got %v", out)
		}
	})
}

func TestFilterCocounselorPrefsBySource(t *testing.T) {
	roster := map[string]bool{"a": true, "b": true}

	in := map[string][]RankedPreference{
		"a": {
			{TargetID: "b", Rank: 1},
			{TargetID: "c", Rank: 2}, // off-roster target
		},
		"c": { // off-roster owner
			{TargetID: "a", Rank: 1},
		},
	}
	out := filterCocounselorPrefsBySource(in, roster)

	if _, ok := out["c"]; ok {
		t.Errorf("expected off-roster owner c to be dropped")
	}
	got, ok := out["a"]
	if !ok {
		t.Fatalf("expected on-roster owner a to be kept")
	}
	want := []RankedPreference{
		{TargetID: "b", Rank: 1},
		{TargetID: "c", Rank: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected off-roster targets retained for ineligibility classification, got %v want %v", got, want)
	}
}

func TestFilterCocounselorUnmetByRoster(t *testing.T) {
	roster := map[string]bool{"a": true, "b": true}

	in := map[string]map[string]bool{
		"a": {"b": true, "c": true}, // c is off-roster target
		"c": {"a": true},             // off-roster owner
	}
	out := filterCocounselorUnmetByRoster(in, roster)

	if _, ok := out["c"]; ok {
		t.Errorf("expected off-roster owner c to be dropped")
	}
	got, ok := out["a"]
	if !ok {
		t.Fatalf("expected on-roster owner a to be kept")
	}
	want := map[string]bool{"b": true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("expected off-roster targets pruned, got %v want %v", got, want)
	}
}

func TestMarkCabinsWithCampers(t *testing.T) {
	cabins := []Cabin{
		{ID: "c1", AgeGroupID: "ag1", Gender: "male"},
		{ID: "c2", AgeGroupID: "ag1", Gender: "female"},
		{ID: "c3", AgeGroupID: "ag2", Gender: "male"},
	}
	campers := []Camper{
		{ID: "k1", AgeGroupID: "ag1", Gender: "male"},
		{ID: "k2", AgeGroupID: "ag2", Gender: "male"},
	}

	got := markCabinsWithCampers(cabins, campers)

	want := map[string]bool{"c1": true, "c2": false, "c3": true}
	for _, c := range got {
		if c.HasCampers != want[c.ID] {
			t.Errorf("cabin %q HasCampers = %v, want %v", c.ID, c.HasCampers, want[c.ID])
		}
	}
}
