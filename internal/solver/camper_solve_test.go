package solver

import (
	"slices"
	"testing"
)

func TestUnassignableCampers(t *testing.T) {
	t.Run("returns campers without a viable cabin", func(t *testing.T) {
		// Two cabins in age group ag1 (capacity 1 each), three campers in
		// ag1 — one camper has no viable placement.
		snapshot := CamperCabinSnapshot{
			Cabins: []CamperCabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", Capacity: 1, Gender: "female"},
				{ID: "c2", Name: "Oak", AgeGroupID: "ag1", Capacity: 1, Gender: "female"},
			},
			Campers: []Camper{
				{ID: "k1", Name: "Alice", AgeGroupID: "ag1", Gender: "female"},
				{ID: "k2", Name: "Beth", AgeGroupID: "ag1", Gender: "female"},
				{ID: "k3", Name: "Carol", AgeGroupID: "ag1", Gender: "female"},
			},
		}

		unassigned := UnassignableCampers(snapshot)
		if len(unassigned) != 1 {
			t.Fatalf("expected exactly 1 unassignable camper, got %d (%+v)", len(unassigned), unassigned)
		}
		// The specific camper depends on greedy ordering, but it should be
		// one of the three.
		ids := []string{"k1", "k2", "k3"}
		if !slices.Contains(ids, unassigned[0].ID) {
			t.Errorf("got unexpected camper %+v", unassigned[0])
		}
	})

	t.Run("returns empty when all fit", func(t *testing.T) {
		snapshot := CamperCabinSnapshot{
			Cabins: []CamperCabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", Capacity: 5, Gender: "female"},
			},
			Campers: []Camper{
				{ID: "k1", Name: "Alice", AgeGroupID: "ag1", Gender: "female"},
				{ID: "k2", Name: "Beth", AgeGroupID: "ag1", Gender: "female"},
			},
		}
		unassigned := UnassignableCampers(snapshot)
		if len(unassigned) != 0 {
			t.Errorf("expected no unassignable campers, got %+v", unassigned)
		}
	})
}
