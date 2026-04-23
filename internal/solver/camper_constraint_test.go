package solver

import "testing"

func TestCheckCabinGenderMatchCampers(t *testing.T) {
	snapshot := CamperCabinSnapshot{
		Cabins: []CamperCabin{
			{ID: "fcab", Name: "Pine", AgeGroupID: "ag1", Capacity: 10, Gender: "female"},
			{ID: "mcab", Name: "Cedar", AgeGroupID: "ag1", Capacity: 10, Gender: "male"},
		},
		Campers: []Camper{
			{ID: "fc1", Name: "Emma", AgeGroupID: "ag1", Gender: "female"},
			{ID: "mc1", Name: "Liam", AgeGroupID: "ag1", Gender: "male"},
		},
	}

	t.Run("matching genders pass", func(t *testing.T) {
		assignment := CamperAssignment{
			CabinCampers: map[string][]string{
				"fcab": {"fc1"},
				"mcab": {"mc1"},
			},
		}
		v := checkCabinGenderMatchCampers(snapshot, assignment)
		if len(v) != 0 {
			t.Fatalf("got %d violations, want 0", len(v))
		}
	})

	t.Run("male camper in female cabin violates", func(t *testing.T) {
		assignment := CamperAssignment{
			CabinCampers: map[string][]string{
				"fcab": {"mc1"},
			},
		}
		v := checkCabinGenderMatchCampers(snapshot, assignment)
		if len(v) != 1 {
			t.Fatalf("got %d violations, want 1", len(v))
		}
		if v[0].Constraint != "cabin_gender_mismatch" {
			t.Errorf("got constraint %q, want cabin_gender_mismatch", v[0].Constraint)
		}
		if v[0].CamperID != "mc1" || v[0].CabinID != "fcab" {
			t.Errorf("got camper=%q cabin=%q", v[0].CamperID, v[0].CabinID)
		}
	})
}
