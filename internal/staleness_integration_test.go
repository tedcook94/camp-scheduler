//go:build integration

package internal_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// boolField extracts a bool field from a decoded JSON map. Fails the test if
// the key is missing or the value is not a bool — assertions like "this run
// should not be stale" must not silently pass when the API forgets to return
// the field.
func boolField(t *testing.T, m map[string]any, key string) bool {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("expected key %q in response, got: %v", key, m)
	}
	b, ok := v.(bool)
	if !ok {
		t.Fatalf("expected key %q to be bool, got %T: %v", key, v, v)
	}
	return b
}

func findRun(runs []any, id string) map[string]any {
	for _, r := range runs {
		m := asMap(r)
		if str(m, "id") == id {
			return m
		}
	}
	return nil
}

type stalenessFixture struct {
	ts            *httptest.Server
	pool          *pgxpool.Pool
	campID        string
	token         string
	sessionID     string
	sagID         string
	cabinRunID    string
	activityRunID string
	counselor1ID  string
	camperID      string
}

func setupStalenessFixture(t *testing.T, name string) *stalenessFixture {
	t.Helper()
	ts, pool := mustSetupServer(t)

	var campID string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, name).Scan(&campID); err != nil {
		t.Fatalf("inserting camp: %v", err)
	}
	token := mustLogin(t, ts, pool, campID)

	season := mustPost(t, apiURL(ts, "/seasons"), map[string]any{
		"name": "Summer", "start_date": "2026-06-01", "end_date": "2026-08-31",
	}, token)
	seasonID := str(season, "id")

	ag := mustPost(t, apiURL(ts, "/age-groups"), map[string]any{"name": "Juniors"}, token)
	agID := str(ag, "id")

	cabin := mustPost(t, apiURL(ts, "/cabins"), map[string]any{
		"name": "Pine", "default_age_group_id": agID,
		"default_group_size": 8, "default_required_counselors": 1, "gender": "female",
	}, token)
	cabinID := str(cabin, "id")

	session := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
		"name": "S1", "season_id": seasonID,
	}, token)
	sessionID := str(session, "id")
	sessBase := "/sessions/" + sessionID

	sag := mustPost(t, apiURL(ts, sessBase+"/age-groups"), map[string]any{
		"age_group_id": agID,
	}, token)
	sagID := str(sag, "id")

	mustPost(t, apiURL(ts, sessBase+"/cabins"), map[string]any{
		"session_age_group_id": sagID, "cabin_id": cabinID,
		"group_size": 8, "required_counselors": 1,
		"gender":             "female",
	}, token)

	camper := mustPost(t, apiURL(ts, "/campers"), map[string]any{
		"first_name": "Cam", "last_name": "Per", "gender": "female",
	}, token)
	camperID := str(camper, "id")
	mustPost(t, apiURL(ts, sessBase+"/enrollments"), map[string]any{
		"camper_id": camperID, "session_age_group_id": sagID,
	}, token)

	c1 := mustPost(t, apiURL(ts, "/counselors"), map[string]any{
		"first_name": "Alice", "last_name": "Test",
		"junior_counselor": false, "gender": "female",
	}, token)
	c1ID := str(c1, "id")
	mustPost(t, apiURL(ts, "/counselors"), map[string]any{
		"first_name": "Bob", "last_name": "Test",
		"junior_counselor": false, "gender": "female",
	}, token)

	swim := mustPost(t, apiURL(ts, "/activities"), map[string]any{"name": "Swim"}, token)
	swimID := str(swim, "id")
	period := mustPost(t, apiURL(ts, "/time-slots"), map[string]any{"name": "Morning"}, token)
	periodID := str(period, "id")
	sts := mustPost(t, apiURL(ts, sessBase+"/time-slots"), map[string]any{
		"time_slot_id": periodID, "sort_order": 1,
	}, token)
	stsID := str(sts, "id")
	mustPost(t, apiURL(ts, sessBase+"/time-slots/"+stsID+"/activities"), map[string]any{
		"activity_id": swimID, "capacity": 4, "required_counselors": 1,
	}, token)

	cabinRunURL := apiURL(ts, sessBase+"/assignment-runs")
	cabinRun := mustPost(t, cabinRunURL, map[string]any{"run_type": "cabin"}, token)
	cabinRunID := str(cabinRun, "id")
	if len(list(cabinRun, "solutions")) == 0 {
		t.Fatal("expected cabin solutions")
	}

	activityRun := mustPost(t, cabinRunURL, map[string]any{"run_type": "activity_schedule"}, token)
	activityRunID := str(activityRun, "id")
	if len(list(activityRun, "solutions")) == 0 {
		t.Fatal("expected activity solutions")
	}

	f := &stalenessFixture{
		ts: ts, pool: pool, campID: campID, token: token,
		sessionID: sessionID, sagID: sagID,
		cabinRunID: cabinRunID, activityRunID: activityRunID,
		counselor1ID: c1ID, camperID: camperID,
	}

	cabinDetail, activityDetail := f.runs(t)
	if boolField(t, cabinDetail, "is_stale") {
		t.Fatalf("cabin run unexpectedly stale at setup")
	}
	if boolField(t, activityDetail, "is_stale") {
		t.Fatalf("activity run unexpectedly stale at setup")
	}
	return f
}

func (f *stalenessFixture) runs(t *testing.T) (map[string]any, map[string]any) {
	t.Helper()
	cabin := mustGet(t, apiURL(f.ts, "/assignment-runs/"+f.cabinRunID), f.token)
	activity := mustGet(t, apiURL(f.ts, "/assignment-runs/"+f.activityRunID), f.token)
	return cabin, activity
}

func TestStalenessFlagging(t *testing.T) {
	t.Run("camper update marks cabin only", func(t *testing.T) {
		f := setupStalenessFixture(t, "Camp Stale Camper")

		mustPut(t, apiURL(f.ts, "/campers/"+f.camperID), map[string]any{
			"first_name": "Cam2", "last_name": "Per", "gender": "female",
		}, f.token)

		cabin, activity := f.runs(t)
		if !boolField(t, cabin, "is_stale") {
			t.Fatalf("expected cabin run stale after camper update")
		}
		if boolField(t, activity, "is_stale") {
			t.Fatalf("activity run should not be stale after camper-only mutation")
		}
	})

	t.Run("activity delete marks activity only", func(t *testing.T) {
		f := setupStalenessFixture(t, "Camp Stale Activity")

		extra := mustPost(t, apiURL(f.ts, "/activities"), map[string]any{"name": "Extra"}, f.token)
		mustDelete(t, apiURL(f.ts, "/activities/"+str(extra, "id")), f.token)

		cabin, activity := f.runs(t)
		if boolField(t, cabin, "is_stale") {
			t.Fatalf("cabin run should not be stale after activity-only mutation")
		}
		if !boolField(t, activity, "is_stale") {
			t.Fatalf("expected activity run stale after activity delete")
		}
	})

	t.Run("counselor update marks both run types", func(t *testing.T) {
		f := setupStalenessFixture(t, "Camp Stale Counselor")

		mustPut(t, apiURL(f.ts, "/counselors/"+f.counselor1ID), map[string]any{
			"first_name": "Alice2", "last_name": "Test",
			"junior_counselor": false, "gender": "female",
		}, f.token)

		cabin, activity := f.runs(t)
		if !boolField(t, cabin, "is_stale") {
			t.Fatalf("expected cabin run stale after counselor update")
		}
		if !boolField(t, activity, "is_stale") {
			t.Fatalf("expected activity run stale after counselor update")
		}
	})

	t.Run("enrollment marks cabin only", func(t *testing.T) {
		f := setupStalenessFixture(t, "Camp Stale Enroll")

		c := mustPost(t, apiURL(f.ts, "/campers"), map[string]any{
			"first_name": "Newby", "last_name": "Test", "gender": "female",
		}, f.token)
		mustPost(t, apiURL(f.ts, "/sessions/"+f.sessionID+"/enrollments"), map[string]any{
			"camper_id": str(c, "id"), "session_age_group_id": f.sagID,
		}, f.token)

		cabin, activity := f.runs(t)
		if !boolField(t, cabin, "is_stale") {
			t.Fatalf("expected cabin run stale after enrollment add")
		}
		if boolField(t, activity, "is_stale") {
			t.Fatalf("activity run should not be stale after enrollment add")
		}
	})

	t.Run("triggering a new run resets stale flag", func(t *testing.T) {
		f := setupStalenessFixture(t, "Camp Stale Reset")

		mustPut(t, apiURL(f.ts, "/campers/"+f.camperID), map[string]any{
			"first_name": "Cam2", "last_name": "Per", "gender": "female",
		}, f.token)

		cabin, _ := f.runs(t)
		if !boolField(t, cabin, "is_stale") {
			t.Fatalf("setup: expected cabin stale after camper update")
		}

		runURL := apiURL(f.ts, "/sessions/"+f.sessionID+"/assignment-runs")
		newRun := mustPost(t, runURL, map[string]any{"run_type": "cabin"}, f.token)
		if boolField(t, newRun, "is_stale") {
			t.Fatalf("freshly triggered run should not be stale")
		}
		fresh := mustGet(t, apiURL(f.ts, "/assignment-runs/"+str(newRun, "id")), f.token)
		if boolField(t, fresh, "is_stale") {
			t.Fatalf("freshly triggered run via GET should not be stale")
		}
	})

	t.Run("is_stale present in list endpoint", func(t *testing.T) {
		f := setupStalenessFixture(t, "Camp Stale List")

		mustPut(t, apiURL(f.ts, "/campers/"+f.camperID), map[string]any{
			"first_name": "Cam2", "last_name": "Per", "gender": "female",
		}, f.token)

		runs := mustGetList(t, apiURL(f.ts, "/sessions/"+f.sessionID+"/assignment-runs"), f.token)
		cabin := findRun(runs, f.cabinRunID)
		activity := findRun(runs, f.activityRunID)
		if cabin == nil || activity == nil {
			t.Fatalf("missing cabin/activity in list response")
		}
		if !boolField(t, cabin, "is_stale") {
			t.Fatalf("expected cabin stale in list response")
		}
		if boolField(t, activity, "is_stale") {
			t.Fatalf("activity should not be stale in list response")
		}
	})

	t.Run("selecting solution cascades to dependent session", func(t *testing.T) {
		ts, pool := mustSetupServer(t)

		var campID string
		if err := pool.QueryRow(context.Background(),
			`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, "Camp Stale Cascade").Scan(&campID); err != nil {
			t.Fatalf("inserting camp: %v", err)
		}
		token := mustLogin(t, ts, pool, campID)

		season := mustPost(t, apiURL(ts, "/seasons"), map[string]any{
			"name": "Summer", "start_date": "2026-06-01", "end_date": "2026-08-31",
		}, token)
		seasonID := str(season, "id")

		ag := mustPost(t, apiURL(ts, "/age-groups"), map[string]any{"name": "J"}, token)
		agID := str(ag, "id")
		cab := mustPost(t, apiURL(ts, "/cabins"), map[string]any{
			"name": "P", "default_age_group_id": agID,
			"default_group_size": 8, "default_required_counselors": 1, "gender": "female",
		}, token)
		cabID := str(cab, "id")

		s1 := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
			"name": "S1", "season_id": seasonID,
		}, token)
		s1ID := str(s1, "id")
		s2 := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
			"name": "S2", "season_id": seasonID, "previous_session_id": s1ID,
		}, token)
		s2ID := str(s2, "id")

		for _, sid := range []string{s1ID, s2ID} {
			sag := mustPost(t, apiURL(ts, "/sessions/"+sid+"/age-groups"), map[string]any{
				"age_group_id": agID,
			}, token)
			sagID := str(sag, "id")
			mustPost(t, apiURL(ts, "/sessions/"+sid+"/cabins"), map[string]any{
				"session_age_group_id": sagID, "cabin_id": cabID,
				"group_size": 8, "required_counselors": 1,
				"gender":             "female",
			}, token)
		}

		mustPost(t, apiURL(ts, "/counselors"), map[string]any{
			"first_name": "Alice", "last_name": "Test",
			"junior_counselor": false, "gender": "female",
		}, token)
		mustPost(t, apiURL(ts, "/counselors"), map[string]any{
			"first_name": "Bob", "last_name": "Test",
			"junior_counselor": false, "gender": "female",
		}, token)

		s1Run := mustPost(t, apiURL(ts, "/sessions/"+s1ID+"/assignment-runs"),
			map[string]any{"run_type": "cabin"}, token)
		s1RunID := str(s1Run, "id")
		s1SolID := str(asMap(list(s1Run, "solutions")[0]), "id")

		s2Run := mustPost(t, apiURL(ts, "/sessions/"+s2ID+"/assignment-runs"),
			map[string]any{"run_type": "cabin"}, token)
		s2RunID := str(s2Run, "id")

		s2Detail := mustGet(t, apiURL(ts, "/assignment-runs/"+s2RunID), token)
		if boolField(t, s2Detail, "is_stale") {
			t.Fatalf("s2 cabin run should start fresh")
		}

		doRequest(t, http.MethodPost,
			apiURL(ts, "/sessions/"+s1ID+"/assignment-runs/"+s1RunID+"/solutions/"+s1SolID+"/select"),
			nil, http.StatusOK, token)

		s2Detail = mustGet(t, apiURL(ts, "/assignment-runs/"+s2RunID), token)
		if !boolField(t, s2Detail, "is_stale") {
			t.Fatalf("expected s2 cabin run stale after selecting s1 solution")
		}

		s1Detail := mustGet(t, apiURL(ts, "/assignment-runs/"+s1RunID), token)
		if boolField(t, s1Detail, "is_stale") {
			t.Fatalf("s1 run should not be stale after its own selection")
		}
	})

	t.Run("history update across sessions marks both old and new dependents", func(t *testing.T) {
		ts, pool := mustSetupServer(t)

		var campID string
		if err := pool.QueryRow(context.Background(),
			`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, "Camp History Move").Scan(&campID); err != nil {
			t.Fatalf("inserting camp: %v", err)
		}
		token := mustLogin(t, ts, pool, campID)

		season := mustPost(t, apiURL(ts, "/seasons"), map[string]any{
			"name": "Summer", "start_date": "2026-06-01", "end_date": "2026-08-31",
		}, token)
		seasonID := str(season, "id")

		ag := mustPost(t, apiURL(ts, "/age-groups"), map[string]any{"name": "J"}, token)
		agID := str(ag, "id")
		cab := mustPost(t, apiURL(ts, "/cabins"), map[string]any{
			"name": "P", "default_age_group_id": agID,
			"default_group_size": 8, "default_required_counselors": 1, "gender": "female",
		}, token)
		cabID := str(cab, "id")

		// Two source sessions A and C, each with a dependent (DA -> A, DC -> C).
		sa := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
			"name": "A", "season_id": seasonID,
		}, token)
		saID := str(sa, "id")
		sc := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
			"name": "C", "season_id": seasonID,
		}, token)
		scID := str(sc, "id")
		da := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
			"name": "DA", "season_id": seasonID, "previous_session_id": saID,
		}, token)
		daID := str(da, "id")
		dc := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
			"name": "DC", "season_id": seasonID, "previous_session_id": scID,
		}, token)
		dcID := str(dc, "id")

		for _, sid := range []string{daID, dcID} {
			sag := mustPost(t, apiURL(ts, "/sessions/"+sid+"/age-groups"), map[string]any{
				"age_group_id": agID,
			}, token)
			sagID := str(sag, "id")
			mustPost(t, apiURL(ts, "/sessions/"+sid+"/cabins"), map[string]any{
				"session_age_group_id": sagID, "cabin_id": cabID,
				"group_size": 8, "required_counselors": 1,
				"gender":             "female",
			}, token)
		}

		c1 := mustPost(t, apiURL(ts, "/counselors"), map[string]any{
			"first_name": "Alice", "last_name": "Test",
			"junior_counselor": false, "gender": "female",
		}, token)
		c1ID := str(c1, "id")
		mustPost(t, apiURL(ts, "/counselors"), map[string]any{
			"first_name": "Bob", "last_name": "Test",
			"junior_counselor": false, "gender": "female",
		}, token)

		// Seed a history entry on session A for c1.
		entry := mustPost(t, apiURL(ts, "/counselors/"+c1ID+"/session-history"), map[string]any{
			"session_id":   saID,
			"age_group_id": agID,
		}, token)
		entryID := str(entry, "id")

		daRun := mustPost(t, apiURL(ts, "/sessions/"+daID+"/assignment-runs"),
			map[string]any{"run_type": "cabin"}, token)
		daRunID := str(daRun, "id")
		dcRun := mustPost(t, apiURL(ts, "/sessions/"+dcID+"/assignment-runs"),
			map[string]any{"run_type": "cabin"}, token)
		dcRunID := str(dcRun, "id")

		if boolField(t, mustGet(t, apiURL(ts, "/assignment-runs/"+daRunID), token), "is_stale") {
			t.Fatalf("DA cabin run should start fresh")
		}
		if boolField(t, mustGet(t, apiURL(ts, "/assignment-runs/"+dcRunID), token), "is_stale") {
			t.Fatalf("DC cabin run should start fresh")
		}

		// Move the history entry from A to C — both DA (lost an input) and DC
		// (gained one) should now be stale.
		doRequest(t, http.MethodPut,
			apiURL(ts, "/counselors/"+c1ID+"/session-history/"+entryID),
			map[string]any{
				"session_id":   scID,
				"age_group_id": agID,
			}, http.StatusOK, token)

		if !boolField(t, mustGet(t, apiURL(ts, "/assignment-runs/"+daRunID), token), "is_stale") {
			t.Fatalf("expected DA cabin run stale after moving history away from session A")
		}
		if !boolField(t, mustGet(t, apiURL(ts, "/assignment-runs/"+dcRunID), token), "is_stale") {
			t.Fatalf("expected DC cabin run stale after moving history onto session C")
		}
	})

	t.Run("re-triggering a selected source run marks dependents stale", func(t *testing.T) {
		ts, token, s1ID, s2ID, s1RunID, s2RunID, s1SolID := setupCascadeFixture(t, "Camp Stale Retrigger")

		// Select s1's solution. Cascade marks s2 stale.
		doRequest(t, http.MethodPost,
			apiURL(ts, "/sessions/"+s1ID+"/assignment-runs/"+s1RunID+"/solutions/"+s1SolID+"/select"),
			nil, http.StatusOK, token)

		// Re-run s2 to clear the cascade-induced staleness.
		freshS2 := mustPost(t, apiURL(ts, "/sessions/"+s2ID+"/assignment-runs"),
			map[string]any{"run_type": "cabin"}, token)
		s2RunID = str(freshS2, "id")
		if boolField(t, mustGet(t, apiURL(ts, "/assignment-runs/"+s2RunID), token), "is_stale") {
			t.Fatalf("freshly re-run s2 should not be stale before retrigger")
		}

		// Re-trigger s1's cabin run. The prior selected solution disappears,
		// so dependent s2 must be marked stale.
		newS1 := mustPost(t, apiURL(ts, "/sessions/"+s1ID+"/assignment-runs"),
			map[string]any{"run_type": "cabin"}, token)
		newS1RunID := str(newS1, "id")
		if boolField(t, mustGet(t, apiURL(ts, "/assignment-runs/"+newS1RunID), token), "is_stale") {
			t.Fatalf("freshly triggered s1 run should not be stale")
		}
		if !boolField(t, mustGet(t, apiURL(ts, "/assignment-runs/"+s2RunID), token), "is_stale") {
			t.Fatalf("expected s2 stale after re-triggering selected s1 run")
		}
	})

	t.Run("deleting a selected source run marks dependents stale", func(t *testing.T) {
		ts, token, s1ID, s2ID, s1RunID, s2RunID, s1SolID := setupCascadeFixture(t, "Camp Stale Delete")

		doRequest(t, http.MethodPost,
			apiURL(ts, "/sessions/"+s1ID+"/assignment-runs/"+s1RunID+"/solutions/"+s1SolID+"/select"),
			nil, http.StatusOK, token)

		freshS2 := mustPost(t, apiURL(ts, "/sessions/"+s2ID+"/assignment-runs"),
			map[string]any{"run_type": "cabin"}, token)
		s2RunID = str(freshS2, "id")
		if boolField(t, mustGet(t, apiURL(ts, "/assignment-runs/"+s2RunID), token), "is_stale") {
			t.Fatalf("freshly re-run s2 should not be stale before delete")
		}

		// Delete s1's selected run. Dependent s2 must be marked stale.
		doRawRequest(t, http.MethodDelete,
			apiURL(ts, "/sessions/"+s1ID+"/assignment-runs/"+s1RunID),
			nil, http.StatusOK, token)

		if !boolField(t, mustGet(t, apiURL(ts, "/assignment-runs/"+s2RunID), token), "is_stale") {
			t.Fatalf("expected s2 stale after deleting selected s1 run")
		}
	})
}

// setupCascadeFixture builds two sessions S1, S2 (S2.previous = S1) with the
// minimum cabin-run inputs and triggers a cabin run on each. Returns the test
// server, auth token, session IDs, run IDs, and S1's first solution ID — the
// shape needed by the previous-session cascade tests.
func setupCascadeFixture(t *testing.T, name string) (
	ts *httptest.Server,
	token, s1ID, s2ID, s1RunID, s2RunID, s1SolID string,
) {
	t.Helper()
	ts, pool := mustSetupServer(t)

	var campID string
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, name).Scan(&campID); err != nil {
		t.Fatalf("inserting camp: %v", err)
	}
	token = mustLogin(t, ts, pool, campID)

	season := mustPost(t, apiURL(ts, "/seasons"), map[string]any{
		"name": "Summer", "start_date": "2026-06-01", "end_date": "2026-08-31",
	}, token)
	seasonID := str(season, "id")

	ag := mustPost(t, apiURL(ts, "/age-groups"), map[string]any{"name": "J"}, token)
	agID := str(ag, "id")
	cab := mustPost(t, apiURL(ts, "/cabins"), map[string]any{
		"name": "P", "default_age_group_id": agID,
		"default_group_size": 8, "default_required_counselors": 1, "gender": "female",
	}, token)
	cabID := str(cab, "id")

	s1 := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
		"name": "S1", "season_id": seasonID,
	}, token)
	s1ID = str(s1, "id")
	s2 := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
		"name": "S2", "season_id": seasonID, "previous_session_id": s1ID,
	}, token)
	s2ID = str(s2, "id")

	for _, sid := range []string{s1ID, s2ID} {
		sag := mustPost(t, apiURL(ts, "/sessions/"+sid+"/age-groups"), map[string]any{
			"age_group_id": agID,
		}, token)
		sagID := str(sag, "id")
		mustPost(t, apiURL(ts, "/sessions/"+sid+"/cabins"), map[string]any{
			"session_age_group_id": sagID, "cabin_id": cabID,
			"group_size": 8, "required_counselors": 1,
			"gender":             "female",
		}, token)
	}

	mustPost(t, apiURL(ts, "/counselors"), map[string]any{
		"first_name": "Alice", "last_name": "Test",
		"junior_counselor": false, "gender": "female",
	}, token)
	mustPost(t, apiURL(ts, "/counselors"), map[string]any{
		"first_name": "Bob", "last_name": "Test",
		"junior_counselor": false, "gender": "female",
	}, token)

	s1Run := mustPost(t, apiURL(ts, "/sessions/"+s1ID+"/assignment-runs"),
		map[string]any{"run_type": "cabin"}, token)
	s1RunID = str(s1Run, "id")
	s1SolID = str(asMap(list(s1Run, "solutions")[0]), "id")

	s2Run := mustPost(t, apiURL(ts, "/sessions/"+s2ID+"/assignment-runs"),
		map[string]any{"run_type": "cabin"}, token)
	s2RunID = str(s2Run, "id")

	return ts, token, s1ID, s2ID, s1RunID, s2RunID, s1SolID
}
