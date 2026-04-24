package solver

import (
	"testing"
)

func TestCamperShortages(t *testing.T) {
	t.Run("over-capacity bucket reports overflow", func(t *testing.T) {
		snapshot := CamperCabinSnapshot{
			Cabins: []CamperCabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", AgeGroupName: "Juniors", Capacity: 1, Gender: "female"},
				{ID: "c2", Name: "Oak", AgeGroupID: "ag1", AgeGroupName: "Juniors", Capacity: 1, Gender: "female"},
			},
			Campers: []Camper{
				{ID: "k1", Name: "Alice", AgeGroupID: "ag1", Gender: "female"},
				{ID: "k2", Name: "Beth", AgeGroupID: "ag1", Gender: "female"},
				{ID: "k3", Name: "Carol", AgeGroupID: "ag1", Gender: "female"},
			},
		}

		shortages := CamperShortages(snapshot)
		if len(shortages) != 1 {
			t.Fatalf("expected 1 shortage, got %d (%+v)", len(shortages), shortages)
		}
		got := shortages[0]
		if got.Reason != ShortageOverCapacity {
			t.Errorf("expected over_capacity, got %q", got.Reason)
		}
		if got.Count != 1 {
			t.Errorf("expected overflow count 1, got %d", got.Count)
		}
		if got.AgeGroupName != "Juniors" || got.Gender != "female" {
			t.Errorf("unexpected bucket: %+v", got)
		}
	})

	t.Run("no matching cabin reports full bucket", func(t *testing.T) {
		// Only female cabins exist but campers include males.
		snapshot := CamperCabinSnapshot{
			Cabins: []CamperCabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", AgeGroupName: "Juniors", Capacity: 5, Gender: "female"},
			},
			Campers: []Camper{
				{ID: "k1", Name: "Liam", AgeGroupID: "ag1", Gender: "male"},
				{ID: "k2", Name: "Noah", AgeGroupID: "ag1", Gender: "male"},
			},
		}

		shortages := CamperShortages(snapshot)
		if len(shortages) != 1 {
			t.Fatalf("expected 1 shortage, got %d (%+v)", len(shortages), shortages)
		}
		got := shortages[0]
		if got.Reason != ShortageNoMatchingCabin {
			t.Errorf("expected no_matching_cabin, got %q", got.Reason)
		}
		if got.Count != 2 {
			t.Errorf("expected count of all 2 campers in bucket, got %d", got.Count)
		}
		if got.Gender != "male" {
			t.Errorf("expected male bucket, got %q", got.Gender)
		}
	})

	t.Run("no shortages when all fit", func(t *testing.T) {
		snapshot := CamperCabinSnapshot{
			Cabins: []CamperCabin{
				{ID: "c1", Name: "Pine", AgeGroupID: "ag1", AgeGroupName: "Juniors", Capacity: 5, Gender: "female"},
			},
			Campers: []Camper{
				{ID: "k1", Name: "Alice", AgeGroupID: "ag1", Gender: "female"},
				{ID: "k2", Name: "Beth", AgeGroupID: "ag1", Gender: "female"},
			},
		}
		if got := CamperShortages(snapshot); len(got) != 0 {
			t.Errorf("expected no shortages, got %+v", got)
		}
	})

	t.Run("multiple buckets sorted by age group then gender", func(t *testing.T) {
		snapshot := CamperCabinSnapshot{
			Cabins: []CamperCabin{
				{ID: "fb1", Name: "Birch", AgeGroupID: "ag-b", AgeGroupName: "Bears", Capacity: 1, Gender: "female"},
				{ID: "fa1", Name: "Aspen", AgeGroupID: "ag-a", AgeGroupName: "Antelopes", Capacity: 1, Gender: "female"},
				{ID: "ma1", Name: "Alder", AgeGroupID: "ag-a", AgeGroupName: "Antelopes", Capacity: 1, Gender: "male"},
			},
			Campers: []Camper{
				// Antelopes / female: 2 campers, capacity 1 -> overflow 1
				{ID: "k1", AgeGroupID: "ag-a", Gender: "female"},
				{ID: "k2", AgeGroupID: "ag-a", Gender: "female"},
				// Antelopes / male: 2 campers, capacity 1 -> overflow 1
				{ID: "k3", AgeGroupID: "ag-a", Gender: "male"},
				{ID: "k4", AgeGroupID: "ag-a", Gender: "male"},
				// Bears / female: 2 campers, capacity 1 -> overflow 1
				{ID: "k5", AgeGroupID: "ag-b", Gender: "female"},
				{ID: "k6", AgeGroupID: "ag-b", Gender: "female"},
			},
		}

		shortages := CamperShortages(snapshot)
		if len(shortages) != 3 {
			t.Fatalf("expected 3 shortages, got %d (%+v)", len(shortages), shortages)
		}
		// Sort order: Antelopes/female, Antelopes/male, Bears/female
		want := []struct {
			ag, gender string
		}{
			{"Antelopes", "female"},
			{"Antelopes", "male"},
			{"Bears", "female"},
		}
		for i, w := range want {
			if shortages[i].AgeGroupName != w.ag || shortages[i].Gender != w.gender {
				t.Errorf("shortage[%d]: got %s/%s, want %s/%s",
					i, shortages[i].AgeGroupName, shortages[i].Gender, w.ag, w.gender)
			}
		}
	})
}
