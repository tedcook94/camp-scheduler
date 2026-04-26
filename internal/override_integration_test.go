//go:build integration

package internal_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAssignmentOverrides exercises the full override lifecycle: CRUD,
// validation rejection at save time, forced placement by the solver, and
// trigger-time revalidation when overrides go stale.
func TestAssignmentOverrides(t *testing.T) {
	t.Run("counselor_cabin_override_pins_solver_output", testCounselorCabinOverridePins)
	t.Run("camper_cabin_override_pins_solver_output", testCamperCabinOverridePins)
	t.Run("counselor_activity_override_pins_solver_output", testCounselorActivityOverridePins)
	t.Run("counselor_activity_override_pins_across_time_slots", testCounselorActivityOverrideAcrossTimeSlots)
	t.Run("counselor_cabin_override_rejects_gender_mismatch", testCounselorCabinOverrideGenderMismatch)
	t.Run("override_mutation_marks_run_stale", testOverrideMutationMarksRunStale)
	t.Run("trigger_rejects_stale_override", testTriggerRejectsStaleOverride)
	t.Run("session_cabin_gender_overrides_default", testSessionCabinGenderOverridesDefault)
}

// overrideCabinFixture sets up a session with two female cabins and several
// counselors / campers and returns the IDs needed to exercise overrides.
type overrideCabinFixture struct {
	campID        string
	token         string
	sessionID     string
	pineCabinID   string // cabins.id for Pine
	oakCabinID    string // cabins.id for Oak
	pineSAGCID    string // session_age_group_cabins.id for Pine in this session
	oakSAGCID     string // session_age_group_cabins.id for Oak in this session
	juniorsAGID   string
	sagJuniorsID  string
	counselorIDs  map[string]string // name -> counselors.id
	camperIDs     map[string]string // name -> campers.id
	enrollmentIDs map[string]string // name -> camper_session_enrollments.id
}

func setupOverrideCabinFixture(t *testing.T) (*httptest.Server, overrideCabinFixture) {
	t.Helper()
	ts, pool := mustSetupServer(t)

	var campID string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name, camp_location) VALUES ($1, $2) RETURNING id`,
		"Camp Override", "Test").Scan(&campID); err != nil {
		t.Fatalf("inserting camp: %v", err)
	}
	token := mustLogin(t, ts, pool, campID)

	juniors := mustPost(t, apiURL(ts, "/age-groups"), map[string]any{"name": "Juniors"}, token)
	juniorsAGID := str(juniors, "id")

	pine := mustPost(t, apiURL(ts, "/cabins"), map[string]any{
		"name": "Pine", "default_age_group_id": juniorsAGID,
		"default_group_size": 8, "default_required_counselors": 1, "gender": "female",
	}, token)
	oak := mustPost(t, apiURL(ts, "/cabins"), map[string]any{
		"name": "Oak", "default_age_group_id": juniorsAGID,
		"default_group_size": 8, "default_required_counselors": 1, "gender": "female",
	}, token)

	season := mustPost(t, apiURL(ts, "/seasons"), map[string]any{
		"name": "Summer 2025", "start_date": "2025-06-01", "end_date": "2025-08-31",
	}, token)
	session := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
		"name": "Session 1", "season_id": str(season, "id"),
	}, token)
	sessionID := str(session, "id")

	sagJuniors := mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/age-groups"), map[string]any{
		"age_group_id": juniorsAGID,
	}, token)
	sagJuniorsID := str(sagJuniors, "id")

	pineSAGC := mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/cabins"), map[string]any{
		"session_age_group_id": sagJuniorsID, "cabin_id": str(pine, "id"),
		"group_size": 6, "required_counselors": 1, "gender": "female",
	}, token)
	oakSAGC := mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/cabins"), map[string]any{
		"session_age_group_id": sagJuniorsID, "cabin_id": str(oak, "id"),
		"group_size": 6, "required_counselors": 1, "gender": "female",
	}, token)

	counselorIDs := map[string]string{}
	for _, c := range []struct {
		name   string
		junior bool
	}{
		{"Alice", false},
		{"Bea", false},
	} {
		resp := mustPost(t, apiURL(ts, "/counselors"), map[string]any{
			"first_name": c.name, "last_name": "Test",
			"junior_counselor": c.junior, "gender": "female",
		}, token)
		counselorIDs[c.name] = str(resp, "id")
	}

	camperIDs := map[string]string{}
	enrollmentIDs := map[string]string{}
	for _, name := range []string{"Kai", "Lin"} {
		c := mustPost(t, apiURL(ts, "/campers"), map[string]any{
			"first_name": name, "last_name": "Test", "gender": "female",
		}, token)
		camperIDs[name] = str(c, "id")
		enr := mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/enrollments"), map[string]any{
			"camper_id": camperIDs[name], "session_age_group_id": sagJuniorsID,
		}, token)
		enrollmentIDs[name] = str(enr, "id")
	}

	return ts, overrideCabinFixture{
		campID:        campID,
		token:         token,
		sessionID:     sessionID,
		pineCabinID:   str(pine, "id"),
		oakCabinID:    str(oak, "id"),
		pineSAGCID:    str(pineSAGC, "id"),
		oakSAGCID:     str(oakSAGC, "id"),
		juniorsAGID:   juniorsAGID,
		sagJuniorsID:  sagJuniorsID,
		counselorIDs:  counselorIDs,
		camperIDs:     camperIDs,
		enrollmentIDs: enrollmentIDs,
	}
}

func testCounselorCabinOverridePins(t *testing.T) {
	ts, f := setupOverrideCabinFixture(t)

	// Pin Alice to Oak.
	mustPost(t, apiURL(ts, "/sessions/"+f.sessionID+"/overrides/counselor-cabin"), map[string]any{
		"counselor_id":               f.counselorIDs["Alice"],
		"session_age_group_cabin_id": f.oakSAGCID,
	}, f.token)

	run := mustPost(t, apiURL(ts, "/sessions/"+f.sessionID+"/assignment-runs"),
		map[string]any{"run_type": "cabin"}, f.token)
	runID := str(run, "id")
	solutions := list(run, "solutions")
	if len(solutions) == 0 {
		t.Fatal("expected at least one solution")
	}
	topID := str(asMap(solutions[0]), "id")

	detail := mustGet(t, apiURL(ts, "/sessions/"+f.sessionID+"/assignment-runs/"+runID+"/solutions/"+topID), f.token)
	for _, a := range list(detail, "assignments") {
		am := asMap(a)
		if str(am, "counselor_id") == f.counselorIDs["Alice"] {
			if got := str(am, "cabin_id"); got != f.oakCabinID {
				t.Errorf("Alice pinned to Oak (%s) but assigned to %s", f.oakCabinID, got)
			}
		}
	}
}

func testCamperCabinOverridePins(t *testing.T) {
	ts, f := setupOverrideCabinFixture(t)

	// Pin Kai to Oak.
	mustPost(t, apiURL(ts, "/sessions/"+f.sessionID+"/overrides/camper-cabin"), map[string]any{
		"camper_id":                  f.camperIDs["Kai"],
		"session_age_group_cabin_id": f.oakSAGCID,
	}, f.token)

	run := mustPost(t, apiURL(ts, "/sessions/"+f.sessionID+"/assignment-runs"),
		map[string]any{"run_type": "cabin"}, f.token)
	runID := str(run, "id")
	topID := str(asMap(list(run, "solutions")[0]), "id")

	detail := mustGet(t, apiURL(ts, "/sessions/"+f.sessionID+"/assignment-runs/"+runID+"/solutions/"+topID), f.token)
	for _, a := range list(detail, "assignments") {
		am := asMap(a)
		if str(am, "camper_id") == f.camperIDs["Kai"] {
			if got := str(am, "cabin_id"); got != f.oakCabinID {
				t.Errorf("Kai pinned to Oak (%s) but assigned to %s", f.oakCabinID, got)
			}
		}
	}
}

func testCounselorActivityOverridePins(t *testing.T) {
	ts, pool := mustSetupServer(t)

	var campID string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name, camp_location) VALUES ($1, $2) RETURNING id`,
		"Camp Activity Override", "Test").Scan(&campID); err != nil {
		t.Fatalf("inserting camp: %v", err)
	}
	token := mustLogin(t, ts, pool, campID)

	season := mustPost(t, apiURL(ts, "/seasons"), map[string]any{
		"name": "Summer", "start_date": "2025-06-01", "end_date": "2025-08-31",
	}, token)
	session := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
		"name": "Session 1", "season_id": str(season, "id"),
	}, token)
	sessionID := str(session, "id")

	morning := mustPost(t, apiURL(ts, "/time-slots"), map[string]any{"name": "Morning"}, token)
	arts := mustPost(t, apiURL(ts, "/activities"), map[string]any{"name": "Arts"}, token)
	canoe := mustPost(t, apiURL(ts, "/activities"), map[string]any{"name": "Canoeing"}, token)

	stsMorning := mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/time-slots"), map[string]any{
		"time_slot_id": str(morning, "id"), "sort_order": 1,
	}, token)

	saArts := mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/time-slots/"+str(stsMorning, "id")+"/activities"), map[string]any{
		"activity_id": str(arts, "id"),
		"capacity":    5, "required_counselors": 1,
	}, token)
	mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/time-slots/"+str(stsMorning, "id")+"/activities"), map[string]any{
		"activity_id": str(canoe, "id"),
		"capacity":    5, "required_counselors": 1,
	}, token)

	// Two counselors so the solver still has choices.
	c1 := mustPost(t, apiURL(ts, "/counselors"), map[string]any{
		"first_name": "Alice", "last_name": "Test",
		"junior_counselor": false, "gender": "female",
	}, token)
	mustPost(t, apiURL(ts, "/counselors"), map[string]any{
		"first_name": "Bea", "last_name": "Test",
		"junior_counselor": false, "gender": "female",
	}, token)

	// Express a competing preference for Canoeing — override should win.
	mustPut(t, apiURL(ts, "/sessions/"+sessionID+"/counselors/"+str(c1, "id")+"/activity-preferences"),
		[]map[string]any{{"activity_id": str(canoe, "id"), "rank": 1}}, token)

	// Pin Alice to Arts.
	mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/overrides/counselor-activity"), map[string]any{
		"counselor_id":        str(c1, "id"),
		"session_activity_id": str(saArts, "id"),
	}, token)

	run := mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/assignment-runs"),
		map[string]any{"run_type": "activity_schedule"}, token)
	runID := str(run, "id")
	topID := str(asMap(list(run, "solutions")[0]), "id")

	detail := mustGet(t, apiURL(ts, "/sessions/"+sessionID+"/assignment-runs/"+runID+"/solutions/"+topID), token)
	found := false
	for _, a := range list(detail, "assignments") {
		am := asMap(a)
		if str(am, "counselor_id") != str(c1, "id") {
			continue
		}
		if str(am, "session_activity_id") == str(saArts, "id") {
			found = true
		}
	}
	if !found {
		t.Errorf("Alice was not pinned to the Arts activity slot")
	}
}

// testCounselorActivityOverrideAcrossTimeSlots regression test for the bug
// where multiple activity overrides for the same counselor (one per time
// slot) collapsed in the solver snapshot, causing trigger-time validation
// to falsely reject the run as stale.
func testCounselorActivityOverrideAcrossTimeSlots(t *testing.T) {
	ts, pool := mustSetupServer(t)

	var campID string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name, camp_location) VALUES ($1, $2) RETURNING id`,
		"Camp Multi Slot Override", "Test").Scan(&campID); err != nil {
		t.Fatalf("inserting camp: %v", err)
	}
	token := mustLogin(t, ts, pool, campID)

	season := mustPost(t, apiURL(ts, "/seasons"), map[string]any{
		"name": "Summer", "start_date": "2025-06-01", "end_date": "2025-08-31",
	}, token)
	session := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
		"name": "Session 1", "season_id": str(season, "id"),
	}, token)
	sessionID := str(session, "id")

	morning1 := mustPost(t, apiURL(ts, "/time-slots"), map[string]any{"name": "Morning 1"}, token)
	morning2 := mustPost(t, apiURL(ts, "/time-slots"), map[string]any{"name": "Morning 2"}, token)
	hiking := mustPost(t, apiURL(ts, "/activities"), map[string]any{"name": "Nature Hiking"}, token)
	cooking := mustPost(t, apiURL(ts, "/activities"), map[string]any{"name": "Campfire Cooking"}, token)

	stsM1 := mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/time-slots"), map[string]any{
		"time_slot_id": str(morning1, "id"), "sort_order": 1,
	}, token)
	stsM2 := mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/time-slots"), map[string]any{
		"time_slot_id": str(morning2, "id"), "sort_order": 2,
	}, token)

	saHikingM1 := mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/time-slots/"+str(stsM1, "id")+"/activities"), map[string]any{
		"activity_id": str(hiking, "id"), "capacity": 5, "required_counselors": 1,
	}, token)
	mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/time-slots/"+str(stsM1, "id")+"/activities"), map[string]any{
		"activity_id": str(cooking, "id"), "capacity": 5, "required_counselors": 1,
	}, token)
	saCookingM2 := mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/time-slots/"+str(stsM2, "id")+"/activities"), map[string]any{
		"activity_id": str(cooking, "id"), "capacity": 5, "required_counselors": 1,
	}, token)
	mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/time-slots/"+str(stsM2, "id")+"/activities"), map[string]any{
		"activity_id": str(hiking, "id"), "capacity": 5, "required_counselors": 1,
	}, token)

	mike := mustPost(t, apiURL(ts, "/counselors"), map[string]any{
		"first_name": "Mike", "last_name": "Chen",
		"junior_counselor": false, "gender": "male",
	}, token)
	mustPost(t, apiURL(ts, "/counselors"), map[string]any{
		"first_name": "Bea", "last_name": "Smith",
		"junior_counselor": false, "gender": "female",
	}, token)

	// Pin Mike to Hiking in Morning 1 AND Cooking in Morning 2.
	mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/overrides/counselor-activity"), map[string]any{
		"counselor_id":        str(mike, "id"),
		"session_activity_id": str(saHikingM1, "id"),
	}, token)
	mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/overrides/counselor-activity"), map[string]any{
		"counselor_id":        str(mike, "id"),
		"session_activity_id": str(saCookingM2, "id"),
	}, token)

	// Trigger should succeed — multiple per-time-slot pins are valid.
	run := mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/assignment-runs"),
		map[string]any{"run_type": "activity_schedule"}, token)
	runID := str(run, "id")
	topID := str(asMap(list(run, "solutions")[0]), "id")

	detail := mustGet(t, apiURL(ts, "/sessions/"+sessionID+"/assignment-runs/"+runID+"/solutions/"+topID), token)
	hikingPinned, cookingPinned := false, false
	for _, a := range list(detail, "assignments") {
		am := asMap(a)
		if str(am, "counselor_id") != str(mike, "id") {
			continue
		}
		switch str(am, "session_activity_id") {
		case str(saHikingM1, "id"):
			hikingPinned = true
		case str(saCookingM2, "id"):
			cookingPinned = true
		}
	}
	if !hikingPinned {
		t.Errorf("Mike was not pinned to Hiking in Morning 1")
	}
	if !cookingPinned {
		t.Errorf("Mike was not pinned to Cooking in Morning 2")
	}
}

func testCounselorCabinOverrideGenderMismatch(t *testing.T) {
	ts, f := setupOverrideCabinFixture(t)

	// Add a male counselor, then attempt to pin them to a female cabin.
	male := mustPost(t, apiURL(ts, "/counselors"), map[string]any{
		"first_name": "Sam", "last_name": "Test",
		"junior_counselor": false, "gender": "male",
	}, f.token)

	doRawRequest(t, http.MethodPost, apiURL(ts, "/sessions/"+f.sessionID+"/overrides/counselor-cabin"),
		map[string]any{
			"counselor_id":               str(male, "id"),
			"session_age_group_cabin_id": f.oakSAGCID,
		}, http.StatusUnprocessableEntity, f.token)
}

func testOverrideMutationMarksRunStale(t *testing.T) {
	ts, f := setupOverrideCabinFixture(t)

	run := mustPost(t, apiURL(ts, "/sessions/"+f.sessionID+"/assignment-runs"),
		map[string]any{"run_type": "cabin"}, f.token)
	runID := str(run, "id")
	if v, _ := run["is_stale"].(bool); v {
		t.Fatal("fresh run should not be stale")
	}

	mustPost(t, apiURL(ts, "/sessions/"+f.sessionID+"/overrides/counselor-cabin"), map[string]any{
		"counselor_id":               f.counselorIDs["Alice"],
		"session_age_group_cabin_id": f.pineSAGCID,
	}, f.token)

	after := mustGet(t, apiURL(ts, "/assignment-runs/"+runID), f.token)
	if v, _ := after["is_stale"].(bool); !v {
		t.Errorf("run should be stale after override mutation; got is_stale=%v", after["is_stale"])
	}
}

func testTriggerRejectsStaleOverride(t *testing.T) {
	ts, f := setupOverrideCabinFixture(t)

	mustPost(t, apiURL(ts, "/sessions/"+f.sessionID+"/overrides/counselor-cabin"), map[string]any{
		"counselor_id":               f.counselorIDs["Alice"],
		"session_age_group_cabin_id": f.pineSAGCID,
	}, f.token)

	// Archive Alice — her override now references an off-roster counselor.
	doRawRequest(t, http.MethodPost, apiURL(ts, "/counselors/"+f.counselorIDs["Alice"]+"/archive"),
		nil, http.StatusOK, f.token)

	doRawRequest(t, http.MethodPost, apiURL(ts, "/sessions/"+f.sessionID+"/assignment-runs"),
		map[string]any{"run_type": "cabin"}, http.StatusUnprocessableEntity, f.token)
}

// testSessionCabinGenderOverridesDefault verifies that a session cabin's
// gender takes precedence over the global cabin's gender for both override
// validation and solver placement. The global cabin is created as female,
// then the session cabin is created with gender=male. A male counselor pin
// should succeed and the run should complete.
func testSessionCabinGenderOverridesDefault(t *testing.T) {
	ts, pool := mustSetupServer(t)

	var campID string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name, camp_location) VALUES ($1, $2) RETURNING id`,
		"Camp Gender", "Test").Scan(&campID); err != nil {
		t.Fatalf("inserting camp: %v", err)
	}
	token := mustLogin(t, ts, pool, campID)

	juniors := mustPost(t, apiURL(ts, "/age-groups"), map[string]any{"name": "Juniors"}, token)
	juniorsAGID := str(juniors, "id")

	// Global cabin defaults to female.
	pine := mustPost(t, apiURL(ts, "/cabins"), map[string]any{
		"name": "Pine", "default_age_group_id": juniorsAGID,
		"default_group_size": 8, "default_required_counselors": 1, "gender": "female",
	}, token)

	season := mustPost(t, apiURL(ts, "/seasons"), map[string]any{
		"name": "S", "start_date": "2025-06-01", "end_date": "2025-08-31",
	}, token)
	session := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
		"name": "S1", "season_id": str(season, "id"),
	}, token)
	sessionID := str(session, "id")

	sag := mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/age-groups"), map[string]any{
		"age_group_id": juniorsAGID,
	}, token)

	// Per-session, override Pine to be a male cabin.
	pineSAGC := mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/cabins"), map[string]any{
		"session_age_group_id": str(sag, "id"), "cabin_id": str(pine, "id"),
		"group_size": 6, "required_counselors": 1, "gender": "male",
	}, token)

	// Need a senior male counselor.
	mike := mustPost(t, apiURL(ts, "/counselors"), map[string]any{
		"first_name": "Mike", "last_name": "Test",
		"junior_counselor": false, "gender": "male",
	}, token)

	// Pin succeeds against per-session male gender.
	mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/overrides/counselor-cabin"), map[string]any{
		"counselor_id":               str(mike, "id"),
		"session_age_group_cabin_id": str(pineSAGC, "id"),
	}, token)

	// Run completes — solver respects per-session male gender.
	run := mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/assignment-runs"),
		map[string]any{"run_type": "cabin"}, token)
	if str(run, "status") != "completed" {
		t.Fatalf("expected status completed, got %q", str(run, "status"))
	}
}
