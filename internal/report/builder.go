package report

import (
	"sort"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/assignment"
	"camp-scheduler/internal/camper"
	"camp-scheduler/internal/db"
)

// buildCabinReport groups counselor and camper assignments under their
// cabin and folds in cabin metadata (gender, capacity, required staff).
// Empty cabins in the session config are still rendered so directors can
// see at a glance which cabins were skipped.
func buildCabinReport(
	detail assignment.SolutionDetailResponse,
	cabins []db.ListSessionCabinsWithCapacityRow,
	counselors map[string]db.Counselor,
	campers map[string]db.Camper,
) *CabinReport {
	type cabinKey = string // session_age_group_cabin_id is unique per cabin slot

	groups := make(map[string]*CabinGroup, len(cabins))
	order := make([]string, 0, len(cabins))
	for _, c := range cabins {
		key := api.UUIDToString(c.SessionAgeGroupCabinID)
		// Capacity is total occupancy (counselors + campers).
		groups[key] = &CabinGroup{
			AgeGroupName:       c.AgeGroupName,
			CabinName:          c.CabinName,
			Gender:             c.Gender,
			Capacity:           c.GroupSize,
			RequiredCounselors: c.RequiredCounselors,
		}
		order = append(order, key)
	}

	// The assignment loader returns assignments keyed by the cabin's primary
	// key (cabins.id) via the join in ListCounselorCabinAssignmentsBySolution.
	// We need to group by (age_group_name, cabin_name) since the same
	// physical cabin may appear under different age groups in a session — but
	// in practice it is unique per session. Map by the human-readable name
	// pair rather than the cabin UUID we don't have on the assignment row.
	nameKey := func(ageGroup, cabin string) string {
		return ageGroup + "\x00" + cabin
	}
	groupByName := make(map[string]*CabinGroup, len(groups))
	for _, g := range groups {
		groupByName[nameKey(g.AgeGroupName, g.CabinName)] = g
	}

	for _, a := range detail.Assignments {
		g, ok := groupByName[nameKey(a.AgeGroupName, a.CabinName)]
		if !ok {
			// Assignment references a cabin no longer in the session config.
			// Surface it as its own ad-hoc group so the data isn't lost.
			g = &CabinGroup{
				AgeGroupName: a.AgeGroupName,
				CabinName:    a.CabinName,
			}
			key := nameKey(a.AgeGroupName, a.CabinName)
			groupByName[key] = g
			synthKey := "synthetic\x00" + key
			groups[synthKey] = g
			order = append(order, synthKey)
		}
		switch {
		case a.CounselorID != "":
			g.Counselors = append(g.Counselors, counselorRow(a.CounselorID, a.CounselorName, counselors))
		case a.CamperID != "":
			g.Campers = append(g.Campers, camperRow(a.CamperID, a.CamperName, campers))
		}
	}

	// Stable, human-friendly ordering: age group, then cabin.
	sort.SliceStable(order, func(i, j int) bool {
		gi := groups[order[i]]
		gj := groups[order[j]]
		if gi.AgeGroupName != gj.AgeGroupName {
			return gi.AgeGroupName < gj.AgeGroupName
		}
		return gi.CabinName < gj.CabinName
	})

	out := &CabinReport{
		Cabins: make([]CabinGroup, 0, len(order)),
	}
	for _, k := range order {
		g := groups[k]
		sort.SliceStable(g.Counselors, func(i, j int) bool { return g.Counselors[i].Name < g.Counselors[j].Name })
		sort.SliceStable(g.Campers, func(i, j int) bool { return g.Campers[i].Name < g.Campers[j].Name })
		out.Cabins = append(out.Cabins, *g)
	}

	for _, u := range detail.UnassignedCounselors {
		out.Unassigned.Counselors = append(out.Unassigned.Counselors, counselorRow(u.CounselorID, u.CounselorName, counselors))
	}
	sort.SliceStable(out.Unassigned.Counselors, func(i, j int) bool {
		return out.Unassigned.Counselors[i].Name < out.Unassigned.Counselors[j].Name
	})
	// Unassigned campers aren't tracked by the solver as a discrete list
	// (capacity shortages are caught up-front), so leave that bucket empty
	// for now. If the solver starts emitting unassigned campers, surface
	// them here.

	return out
}

// buildActivityReport groups activity assignments by time slot and
// activity, preserving the time-slot sort_order from the schedule.
//
// The schedule rows seed the report so configured time slots / activities
// with zero counselors still appear in the output (otherwise empty
// activities would silently disappear from CSV/PDF exports — exactly the
// gaps directors most want to see).
func buildActivityReport(
	detail assignment.SolutionDetailResponse,
	schedule []db.ListSessionActivitiesWithDetailsRow,
	counselors map[string]db.Counselor,
) *ActivityReport {
	type slotBuilder struct {
		group      *TimeSlotGroup
		activities map[string]*ActivityRowGroup
	}
	slots := make(map[string]*slotBuilder)
	slotOrder := []string{}

	ensureSlot := func(name string, sortOrder int32) *slotBuilder {
		sb, ok := slots[name]
		if ok {
			return sb
		}
		sb = &slotBuilder{
			group: &TimeSlotGroup{
				Name:      name,
				SortOrder: sortOrder,
			},
			activities: make(map[string]*ActivityRowGroup),
		}
		slots[name] = sb
		slotOrder = append(slotOrder, name)
		return sb
	}
	ensureActivity := func(sb *slotBuilder, name string) *ActivityRowGroup {
		ag, ok := sb.activities[name]
		if ok {
			return ag
		}
		ag = &ActivityRowGroup{Name: name}
		sb.activities[name] = ag
		return ag
	}

	// Seed from the configured schedule first so empty cells still render.
	for _, row := range schedule {
		sb := ensureSlot(row.TimeSlotName, row.SortOrder)
		ensureActivity(sb, row.ActivityName)
	}

	// Overlay assignments. If an assignment references a slot/activity that
	// isn't in the schedule (defensive — shouldn't happen), synthesize it
	// rather than dropping the data.
	for _, a := range detail.Assignments {
		sb := ensureSlot(a.TimeSlotName, a.SortOrder)
		ag := ensureActivity(sb, a.ActivityName)
		ag.Counselors = append(ag.Counselors, counselorRow(a.CounselorID, a.CounselorName, counselors))
	}

	sort.SliceStable(slotOrder, func(i, j int) bool {
		si := slots[slotOrder[i]].group
		sj := slots[slotOrder[j]].group
		if si.SortOrder != sj.SortOrder {
			return si.SortOrder < sj.SortOrder
		}
		return si.Name < sj.Name
	})

	out := &ActivityReport{
		TimeSlots: make([]TimeSlotGroup, 0, len(slotOrder)),
	}
	for _, name := range slotOrder {
		sb := slots[name]
		actNames := make([]string, 0, len(sb.activities))
		for n := range sb.activities {
			actNames = append(actNames, n)
		}
		sort.Strings(actNames)
		for _, n := range actNames {
			ag := sb.activities[n]
			sort.SliceStable(ag.Counselors, func(i, j int) bool { return ag.Counselors[i].Name < ag.Counselors[j].Name })
			sb.group.Activities = append(sb.group.Activities, *ag)
		}
		out.TimeSlots = append(out.TimeSlots, *sb.group)
	}

	for _, u := range detail.UnassignedCounselors {
		row := ActivityUnassignedRow{
			CounselorRow: counselorRow(u.CounselorID, u.CounselorName, counselors),
		}
		for _, ts := range u.MissingTimeSlots {
			row.MissingTimeSlots = append(row.MissingTimeSlots, ts.TimeSlotName)
		}
		out.Unassigned = append(out.Unassigned, row)
	}
	sort.SliceStable(out.Unassigned, func(i, j int) bool {
		return out.Unassigned[i].Name < out.Unassigned[j].Name
	})

	return out
}

func counselorRow(id, fallbackName string, counselors map[string]db.Counselor) CounselorRow {
	row := CounselorRow{ID: id, Name: fallbackName}
	if c, ok := counselors[id]; ok {
		row.Name = c.CounselorName
		row.Junior = c.JuniorCounselor
		row.Gender = c.Gender
	}
	return row
}

func camperRow(id, fallbackName string, campers map[string]db.Camper) CamperRow {
	row := CamperRow{ID: id, Name: fallbackName}
	if c, ok := campers[id]; ok {
		row.Name = camper.FullName(c.FirstName, c.LastName)
		row.Gender = c.Gender
	}
	return row
}
