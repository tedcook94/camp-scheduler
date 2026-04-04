package solver

func indexCabins(snapshot SessionSnapshot) map[string]Cabin {
	cabins := make(map[string]Cabin, len(snapshot.Cabins))
	for _, c := range snapshot.Cabins {
		cabins[c.ID] = c
	}
	return cabins
}

func indexCounselors(snapshot SessionSnapshot) map[string]Counselor {
	counselors := make(map[string]Counselor, len(snapshot.Counselors))
	for _, c := range snapshot.Counselors {
		counselors[c.ID] = c
	}
	return counselors
}

func invertAssignment(assignment Assignment) map[string]string {
	result := make(map[string]string)
	for cabinID, counselorIDs := range assignment.CabinCounselors {
		for _, cID := range counselorIDs {
			result[cID] = cabinID
		}
	}
	return result
}
