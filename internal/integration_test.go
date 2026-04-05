//go:build integration

package internal_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"camp-scheduler/internal/config"
	"camp-scheduler/internal/server"
	"camp-scheduler/internal/testutil"
)

func mustSetupServer(t *testing.T) *httptest.Server {
	t.Helper()

	pool := testutil.MustOpenDB(t)
	testutil.TruncateAll(t, pool)

	cfg := config.Config{
		Server: config.ServerConfig{Mode: "test"},
	}
	srv := server.NewWithPool(cfg, pool)
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return ts
}

// apiURL builds a full URL from the test server base and a path suffix.
func apiURL(ts *httptest.Server, path string) string {
	return ts.URL + "/api/v1" + path
}

func mustPost(t *testing.T, url string, body any) map[string]any {
	t.Helper()
	return doRequest(t, http.MethodPost, url, body, http.StatusCreated)
}

func mustGet(t *testing.T, url string) map[string]any {
	t.Helper()
	return doRequest(t, http.MethodGet, url, nil, http.StatusOK)
}

func mustGetList(t *testing.T, url string) []any {
	t.Helper()
	resp := doRawRequest(t, http.MethodGet, url, nil, http.StatusOK)
	var result []any
	if err := json.Unmarshal(resp, &result); err != nil {
		t.Fatalf("unmarshaling list response: %v", err)
	}
	return result
}

func mustDelete(t *testing.T, url string) {
	t.Helper()
	doRawRequest(t, http.MethodDelete, url, nil, http.StatusOK)
}

func doRequest(t *testing.T, method, url string, body any, expectedStatus int) map[string]any {
	t.Helper()
	raw := doRawRequest(t, method, url, body, expectedStatus)
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshaling response from %s %s: %v\nbody: %s", method, url, err, string(raw))
	}
	return result
}

func doRawRequest(t *testing.T, method, url string, body any, expectedStatus int) []byte {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshaling request body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("executing %s %s: %v", method, url, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("reading response body: %v", err)
	}

	if resp.StatusCode != expectedStatus {
		t.Fatalf("%s %s: expected status %d, got %d\nbody: %s",
			method, url, expectedStatus, resp.StatusCode, string(respBody))
	}
	return respBody
}

func str(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}

func num(m map[string]any, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	n, ok := v.(float64)
	if !ok {
		return 0
	}
	return n
}

func list(m map[string]any, key string) []any {
	v, ok := m[key]
	if !ok {
		return nil
	}
	l, ok := v.([]any)
	if !ok {
		return nil
	}
	return l
}

func asMap(v any) map[string]any {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	return m
}

func TestSolverIntegration(t *testing.T) {
	t.Run("simple_camp", testSimpleCamp)
	t.Run("complex_camp", testComplexCamp)
}

// testSimpleCamp exercises the full API -> DB -> Solver -> Results flow with a
// small, realistic camp: 2 age groups, 4 cabins, 6 counselors.
func testSimpleCamp(t *testing.T) {
	ts := mustSetupServer(t)

	campResp := mustPost(t, apiURL(ts, "/camps"), map[string]any{
		"name":     "Camp Pinebrook",
		"location": "Vermont",
	})
	campID := str(campResp, "id")

	juniors := mustPost(t, apiURL(ts, "/camps/"+campID+"/age-groups"), map[string]any{
		"name": "Juniors",
	})
	juniorsID := str(juniors, "id")

	seniors := mustPost(t, apiURL(ts, "/camps/"+campID+"/age-groups"), map[string]any{
		"name": "Seniors",
	})
	seniorsID := str(seniors, "id")

	pine := mustPost(t, apiURL(ts, "/camps/"+campID+"/cabins"), map[string]any{
		"name":         "Pine",
		"age_group_id": juniorsID,
	})
	pineID := str(pine, "id")

	oak := mustPost(t, apiURL(ts, "/camps/"+campID+"/cabins"), map[string]any{
		"name":         "Oak",
		"age_group_id": juniorsID,
	})
	oakID := str(oak, "id")

	maple := mustPost(t, apiURL(ts, "/camps/"+campID+"/cabins"), map[string]any{
		"name":         "Maple",
		"age_group_id": seniorsID,
	})
	mapleID := str(maple, "id")

	cedar := mustPost(t, apiURL(ts, "/camps/"+campID+"/cabins"), map[string]any{
		"name":         "Cedar",
		"age_group_id": seniorsID,
	})
	cedarID := str(cedar, "id")

	seasonResp := mustPost(t, apiURL(ts, "/camps/"+campID+"/seasons"), map[string]any{
		"name":       "Summer 2025",
		"start_date": "2025-06-01",
		"end_date":   "2025-08-31",
	})
	seasonID := str(seasonResp, "id")

	session1 := mustPost(t, apiURL(ts, "/camps/"+campID+"/sessions"), map[string]any{
		"name":      "Session 1",
		"season_id": seasonID,
	})
	session1ID := str(session1, "id")

	session2 := mustPost(t, apiURL(ts, "/camps/"+campID+"/sessions"), map[string]any{
		"name":                "Session 2",
		"season_id":           seasonID,
		"previous_session_id": session1ID,
	})
	session2ID := str(session2, "id")

	type counselorInfo struct {
		id   string
		name string
	}
	counselors := make([]counselorInfo, 0, 6)
	for _, c := range []struct {
		name   string
		junior bool
	}{
		{"Alice", false},
		{"Bob", false},
		{"Carol", false},
		{"Dave", false},
		{"Eve", true},
		{"Frank", true},
	} {
		resp := mustPost(t, apiURL(ts, "/camps/"+campID+"/counselors"), map[string]any{
			"name":             c.name,
			"junior_counselor": c.junior,
		})
		counselors = append(counselors, counselorInfo{id: str(resp, "id"), name: c.name})
	}

	alice, bob, carol := counselors[0], counselors[1], counselors[2]
	eve := counselors[4]

	sagJuniors := mustPost(t, apiURL(ts, "/camps/"+campID+"/sessions/"+session2ID+"/age-groups"), map[string]any{
		"age_group_id": juniorsID,
	})
	sagJuniorsID := str(sagJuniors, "id")

	sagSeniors := mustPost(t, apiURL(ts, "/camps/"+campID+"/sessions/"+session2ID+"/age-groups"), map[string]any{
		"age_group_id": seniorsID,
	})
	sagSeniorsID := str(sagSeniors, "id")

	reqCounselors := int32(1)
	for _, cfg := range []struct {
		sagID   string
		cabinID string
	}{
		{sagJuniorsID, pineID},
		{sagJuniorsID, oakID},
		{sagSeniorsID, mapleID},
		{sagSeniorsID, cedarID},
	} {
		mustPost(t, apiURL(ts, "/camps/"+campID+"/sessions/"+session2ID+"/cabins"), map[string]any{
			"session_age_group_id": cfg.sagID,
			"cabin_id":             cfg.cabinID,
			"required_counselors":  reqCounselors,
		})
	}

	// Alice and Bob were both in Pine (Juniors) in session 1, making them
	// co-counselors for the returning-counselor scoring. Carol was in Maple.
	historyBase := "/camps/" + campID + "/counselors/"
	for _, h := range []struct {
		counselorID string
		sessionID   string
		ageGroupID  string
		cabinID     *string
	}{
		{alice.id, session1ID, juniorsID, &pineID},
		{bob.id, session1ID, juniorsID, &pineID},
		{carol.id, session1ID, seniorsID, &mapleID},
	} {
		body := map[string]any{
			"session_id":   h.sessionID,
			"age_group_id": h.ageGroupID,
		}
		if h.cabinID != nil {
			body["cabin_id"] = *h.cabinID
		}
		mustPost(t, apiURL(ts, historyBase+h.counselorID+"/session-history"), body)
	}

	prefBase := "/camps/" + campID + "/sessions/" + session2ID + "/counselors/"
	mustPost(t, apiURL(ts, prefBase+alice.id+"/age-group-preferences"), map[string]any{
		"age_group_id": juniorsID,
		"rank":         1,
	})
	mustPost(t, apiURL(ts, prefBase+bob.id+"/cocounselor-preferences"), map[string]any{
		"preferred_counselor_id": alice.id,
		"rank":                   1,
	})
	mustPost(t, apiURL(ts, prefBase+carol.id+"/age-group-preferences"), map[string]any{
		"age_group_id": seniorsID,
		"rank":         1,
	})

	runURL := apiURL(ts, "/camps/"+campID+"/sessions/"+session2ID+"/assignment-runs")
	triggerResp := mustPost(t, runURL, nil)

	runID := str(triggerResp, "id")
	if runID == "" {
		t.Fatal("expected run ID in trigger response")
	}
	if str(triggerResp, "status") != "completed" {
		t.Fatalf("expected status 'completed', got %q", str(triggerResp, "status"))
	}
	if str(triggerResp, "run_type") != "counselor_cabin" {
		t.Fatalf("expected run_type 'counselor_cabin', got %q", str(triggerResp, "run_type"))
	}

	solutions := list(triggerResp, "solutions")
	if len(solutions) == 0 {
		t.Fatal("expected at least one solution")
	}

	topSolution := asMap(solutions[0])
	topScore := num(topSolution, "score")
	if topScore <= 0 {
		t.Fatalf("expected top solution score > 0, got %f", topScore)
	}
	topSolutionID := str(topSolution, "id")

	for i := 1; i < len(solutions); i++ {
		prev := num(asMap(solutions[i-1]), "score")
		curr := num(asMap(solutions[i]), "score")
		if curr > prev {
			t.Fatalf("solutions not ranked by score: index %d (%.2f) > index %d (%.2f)", i, curr, i-1, prev)
		}
	}

	runs := mustGetList(t, runURL)
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	if str(asMap(runs[0]), "id") != runID {
		t.Fatal("listed run ID does not match created run")
	}

	runDetail := mustGet(t, runURL+"/"+runID)
	detailSolutions := list(runDetail, "solutions")
	if len(detailSolutions) != len(solutions) {
		t.Fatalf("expected %d solutions in detail, got %d", len(solutions), len(detailSolutions))
	}

	solDetail := mustGet(t, runURL+"/"+runID+"/solutions/"+topSolutionID)

	assignments := list(solDetail, "assignments")
	if len(assignments) == 0 {
		t.Fatal("expected assignments in solution detail")
	}

	cabinCounselors := make(map[string][]string)
	counselorCabin := make(map[string]string)
	for _, a := range assignments {
		am := asMap(a)
		cID := str(am, "counselor_id")
		cabID := str(am, "cabin_id")
		cabinCounselors[cabID] = append(cabinCounselors[cabID], cID)
		counselorCabin[cID] = cabID
	}

	allCabinIDs := []string{pineID, oakID, mapleID, cedarID}
	for _, cabID := range allCabinIDs {
		if len(cabinCounselors[cabID]) < 1 {
			t.Fatalf("cabin %s has %d counselors, expected at least 1", cabID, len(cabinCounselors[cabID]))
		}
	}

	juniorSet := map[string]bool{eve.id: true, counselors[5].id: true}
	for _, cabID := range allCabinIDs {
		hasSenior := false
		for _, cID := range cabinCounselors[cabID] {
			if !juniorSet[cID] {
				hasSenior = true
				break
			}
		}
		if !hasSenior {
			t.Fatalf("cabin %s has no senior counselor assigned", cabID)
		}
	}

	explanations := list(solDetail, "explanations")
	if len(explanations) == 0 {
		t.Fatal("expected explanations in solution detail")
	}

	selectResp := doRequest(t, http.MethodPost,
		runURL+"/"+runID+"/solutions/"+topSolutionID+"/select", nil, http.StatusOK)
	selectedID := str(selectResp, "selected_solution_id")
	if selectedID != topSolutionID {
		t.Fatalf("expected selected_solution_id %q, got %q", topSolutionID, selectedID)
	}

	updatedRun := mustGet(t, runURL+"/"+runID)
	if str(updatedRun, "selected_solution_id") != topSolutionID {
		t.Fatal("run detail does not reflect selected solution")
	}

	mustDelete(t, runURL+"/"+runID)

	runsAfterDelete := mustGetList(t, runURL)
	if len(runsAfterDelete) != 0 {
		t.Fatalf("expected 0 runs after delete, got %d", len(runsAfterDelete))
	}
}

// testComplexCamp exercises a larger scenario with 3 age groups, 6 cabins,
// 10 counselors, conflicting preferences, and richer history data. The
// preference conflicts (e.g. Kim prefers Young but Jake wants Kim as a
// co-counselor in Teen) ensure the solver produces unmet_preference
// explanations and that solutions are meaningfully ranked.
func testComplexCamp(t *testing.T) {
	ts := mustSetupServer(t)

	campResp := mustPost(t, apiURL(ts, "/camps"), map[string]any{
		"name":     "Camp Ridgewood",
		"location": "New Hampshire",
	})
	campID := str(campResp, "id")

	young := mustPost(t, apiURL(ts, "/camps/"+campID+"/age-groups"), map[string]any{"name": "Young"})
	youngID := str(young, "id")

	middle := mustPost(t, apiURL(ts, "/camps/"+campID+"/age-groups"), map[string]any{"name": "Middle"})
	middleID := str(middle, "id")

	teen := mustPost(t, apiURL(ts, "/camps/"+campID+"/age-groups"), map[string]any{"name": "Teen"})
	teenID := str(teen, "id")

	type cabinInfo struct {
		id   string
		name string
	}
	cabinDefs := []struct {
		name       string
		ageGroupID string
	}{
		{"Birch", youngID},
		{"Elm", youngID},
		{"Spruce", middleID},
		{"Willow", middleID},
		{"Redwood", teenID},
		{"Sequoia", teenID},
	}
	cabins := make([]cabinInfo, len(cabinDefs))
	for i, cd := range cabinDefs {
		resp := mustPost(t, apiURL(ts, "/camps/"+campID+"/cabins"), map[string]any{
			"name":         cd.name,
			"age_group_id": cd.ageGroupID,
		})
		cabins[i] = cabinInfo{id: str(resp, "id"), name: cd.name}
	}

	seasonResp := mustPost(t, apiURL(ts, "/camps/"+campID+"/seasons"), map[string]any{
		"name":       "Summer 2025",
		"start_date": "2025-06-01",
		"end_date":   "2025-08-31",
	})
	seasonID := str(seasonResp, "id")

	session1 := mustPost(t, apiURL(ts, "/camps/"+campID+"/sessions"), map[string]any{
		"name":      "Session 1",
		"season_id": seasonID,
	})
	session1ID := str(session1, "id")

	session2 := mustPost(t, apiURL(ts, "/camps/"+campID+"/sessions"), map[string]any{
		"name":                "Session 2",
		"season_id":           seasonID,
		"previous_session_id": session1ID,
	})
	session2ID := str(session2, "id")

	type counselorInfo struct {
		id       string
		name     string
		isJunior bool
	}
	counselorDefs := []struct {
		name   string
		junior bool
	}{
		{"Gina", false},
		{"Hank", false},
		{"Iris", false},
		{"Jake", false},
		{"Kim", false},
		{"Leo", false},
		{"Mia", true},
		{"Nate", true},
		{"Olive", true},
		{"Pat", true},
	}
	allCounselors := make([]counselorInfo, len(counselorDefs))
	for i, c := range counselorDefs {
		resp := mustPost(t, apiURL(ts, "/camps/"+campID+"/counselors"), map[string]any{
			"name":             c.name,
			"junior_counselor": c.junior,
		})
		allCounselors[i] = counselorInfo{
			id:       str(resp, "id"),
			name:     c.name,
			isJunior: c.junior,
		}
	}

	gina, hank, iris, jake := allCounselors[0], allCounselors[1], allCounselors[2], allCounselors[3]
	kim := allCounselors[4]
	mia := allCounselors[6]

	sagYoung := mustPost(t, apiURL(ts, "/camps/"+campID+"/sessions/"+session2ID+"/age-groups"), map[string]any{
		"age_group_id": youngID,
	})
	sagYoungID := str(sagYoung, "id")

	sagMiddle := mustPost(t, apiURL(ts, "/camps/"+campID+"/sessions/"+session2ID+"/age-groups"), map[string]any{
		"age_group_id": middleID,
	})
	sagMiddleID := str(sagMiddle, "id")

	sagTeen := mustPost(t, apiURL(ts, "/camps/"+campID+"/sessions/"+session2ID+"/age-groups"), map[string]any{
		"age_group_id": teenID,
	})
	sagTeenID := str(sagTeen, "id")

	sagIDs := []string{sagYoungID, sagYoungID, sagMiddleID, sagMiddleID, sagTeenID, sagTeenID}
	reqCounselors := int32(1)
	for i, cab := range cabins {
		mustPost(t, apiURL(ts, "/camps/"+campID+"/sessions/"+session2ID+"/cabins"), map[string]any{
			"session_age_group_id": sagIDs[i],
			"cabin_id":             cab.id,
			"required_counselors":  reqCounselors,
		})
	}

	// Gina and Hank were both in Birch (Young) in session 1, making them
	// co-counselors. Iris was in Spruce (Middle), Jake was in Redwood (Teen).
	historyBase := "/camps/" + campID + "/counselors/"
	for _, h := range []struct {
		counselorID string
		ageGroupID  string
		cabinID     string
	}{
		{gina.id, youngID, cabins[0].id},
		{hank.id, youngID, cabins[0].id},
		{iris.id, middleID, cabins[2].id},
		{jake.id, teenID, cabins[4].id},
	} {
		mustPost(t, apiURL(ts, historyBase+h.counselorID+"/session-history"), map[string]any{
			"session_id":   session1ID,
			"age_group_id": h.ageGroupID,
			"cabin_id":     h.cabinID,
		})
	}

	prefBase := "/camps/" + campID + "/sessions/" + session2ID + "/counselors/"

	mustPost(t, apiURL(ts, prefBase+gina.id+"/age-group-preferences"), map[string]any{
		"age_group_id": youngID, "rank": 1,
	})
	mustPost(t, apiURL(ts, prefBase+hank.id+"/age-group-preferences"), map[string]any{
		"age_group_id": youngID, "rank": 1,
	})
	mustPost(t, apiURL(ts, prefBase+hank.id+"/cocounselor-preferences"), map[string]any{
		"preferred_counselor_id": gina.id, "rank": 1,
	})
	mustPost(t, apiURL(ts, prefBase+iris.id+"/age-group-preferences"), map[string]any{
		"age_group_id": middleID, "rank": 1,
	})
	mustPost(t, apiURL(ts, prefBase+jake.id+"/age-group-preferences"), map[string]any{
		"age_group_id": teenID, "rank": 1,
	})
	mustPost(t, apiURL(ts, prefBase+jake.id+"/cocounselor-preferences"), map[string]any{
		"preferred_counselor_id": kim.id, "rank": 1,
	})
	// Kim prefers Young, but Jake wants Kim as a Teen co-counselor -- can't both be satisfied.
	mustPost(t, apiURL(ts, prefBase+kim.id+"/age-group-preferences"), map[string]any{
		"age_group_id": youngID, "rank": 1,
	})
	mustPost(t, apiURL(ts, prefBase+allCounselors[5].id+"/age-group-preferences"), map[string]any{
		"age_group_id": middleID, "rank": 1,
	})
	mustPost(t, apiURL(ts, prefBase+mia.id+"/age-group-preferences"), map[string]any{
		"age_group_id": teenID, "rank": 1,
	})
	mustPost(t, apiURL(ts, prefBase+allCounselors[7].id+"/cocounselor-preferences"), map[string]any{
		"preferred_counselor_id": mia.id, "rank": 1,
	})

	runURL := apiURL(ts, "/camps/"+campID+"/sessions/"+session2ID+"/assignment-runs")
	triggerResp := mustPost(t, runURL, map[string]any{
		"max_solutions": 5,
	})

	runID := str(triggerResp, "id")
	if runID == "" {
		t.Fatal("expected run ID in trigger response")
	}
	if str(triggerResp, "status") != "completed" {
		t.Fatalf("expected status 'completed', got %q", str(triggerResp, "status"))
	}

	solutions := list(triggerResp, "solutions")
	if len(solutions) == 0 {
		t.Fatal("expected at least one solution")
	}
	if len(solutions) > 5 {
		t.Fatalf("expected at most 5 solutions, got %d", len(solutions))
	}

	for i := 1; i < len(solutions); i++ {
		prev := num(asMap(solutions[i-1]), "score")
		curr := num(asMap(solutions[i]), "score")
		if curr > prev {
			t.Fatalf("solutions not ranked by score: index %d (%.2f) > index %d (%.2f)", i, curr, i-1, prev)
		}
	}

	topSolutionID := str(asMap(solutions[0]), "id")
	solDetail := mustGet(t, runURL+"/"+runID+"/solutions/"+topSolutionID)

	assignments := list(solDetail, "assignments")
	if len(assignments) == 0 {
		t.Fatal("expected assignments in solution detail")
	}

	cabinCounselors := make(map[string][]string)
	for _, a := range assignments {
		am := asMap(a)
		cabID := str(am, "cabin_id")
		cID := str(am, "counselor_id")
		cabinCounselors[cabID] = append(cabinCounselors[cabID], cID)
	}

	for _, cab := range cabins {
		if len(cabinCounselors[cab.id]) < 1 {
			t.Fatalf("cabin %s (%s) has %d counselors, expected at least 1", cab.name, cab.id, len(cabinCounselors[cab.id]))
		}
	}

	juniorSet := make(map[string]bool)
	for _, c := range allCounselors {
		if c.isJunior {
			juniorSet[c.id] = true
		}
	}
	for _, cab := range cabins {
		hasSenior := false
		for _, cID := range cabinCounselors[cab.id] {
			if !juniorSet[cID] {
				hasSenior = true
				break
			}
		}
		if !hasSenior {
			t.Fatalf("cabin %s (%s) has no senior counselor", cab.name, cab.id)
		}
	}

	explanations := list(solDetail, "explanations")
	if len(explanations) == 0 {
		t.Fatal("expected explanations in solution detail")
	}

	hasUnmetPref := false
	for _, e := range explanations {
		em := asMap(e)
		if str(em, "explanation_type") == "unmet_preference" {
			hasUnmetPref = true
			if str(em, "message") == "" {
				t.Fatal("unmet_preference explanation should have a message")
			}
			break
		}
	}
	if !hasUnmetPref {
		t.Fatal("expected at least one unmet_preference explanation given conflicting preferences")
	}

	topScore := num(asMap(solutions[0]), "score")
	if topScore <= 0 {
		t.Fatalf("expected positive top score, got %f", topScore)
	}

	breakdown := asMap(solutions[0])["score_breakdown"]
	if breakdown == nil {
		t.Fatal("expected score_breakdown in solution")
	}

	selectResp := doRequest(t, http.MethodPost,
		runURL+"/"+runID+"/solutions/"+topSolutionID+"/select", nil, http.StatusOK)
	if str(selectResp, "selected_solution_id") != topSolutionID {
		t.Fatal("selected_solution_id mismatch after selection")
	}

	mustDelete(t, runURL+"/"+runID)
	runsAfterDelete := mustGetList(t, runURL)
	if len(runsAfterDelete) != 0 {
		t.Fatalf("expected 0 runs after delete, got %d", len(runsAfterDelete))
	}
}
