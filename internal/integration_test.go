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

func TestCamperSolverIntegration(t *testing.T) {
	t.Run("basic_camper_assignment", testBasicCamperAssignment)
	t.Run("camper_friend_preferences", testCamperFriendPreferences)
}

func TestReviewFixes(t *testing.T) {
	t.Run("enrollment_session_scoping", testEnrollmentSessionScoping)
	t.Run("enrollment_session_uniqueness", testEnrollmentSessionUniqueness)
	t.Run("get_solution_run_ownership", testGetSolutionRunOwnership)
	t.Run("get_solution_run_not_found", testGetSolutionRunNotFound)
	t.Run("select_solution_camper_run", testSelectSolutionCamperRun)
	t.Run("camper_run_invalid_session", testCamperRunInvalidSession)
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

// testBasicCamperAssignment verifies the full camper assignment flow:
// create camp entities, enroll campers, trigger a camper_cabin run,
// and verify assignments respect cabin capacity and age group constraints.
func testBasicCamperAssignment(t *testing.T) {
	ts := mustSetupServer(t)

	// Create camp.
	campResp := mustPost(t, apiURL(ts, "/camps"), map[string]any{
		"name": "Camp Lakeside",
	})
	campID := str(campResp, "id")
	base := "/camps/" + campID

	// Create age groups.
	juniors := mustPost(t, apiURL(ts, base+"/age-groups"), map[string]any{"name": "Young"})
	juniorsID := str(juniors, "id")

	// Create cabins.
	cabinA := mustPost(t, apiURL(ts, base+"/cabins"), map[string]any{
		"name": "Birch", "age_group_id": juniorsID,
	})
	cabinAID := str(cabinA, "id")

	cabinB := mustPost(t, apiURL(ts, base+"/cabins"), map[string]any{
		"name": "Elm", "age_group_id": juniorsID,
	})
	cabinBID := str(cabinB, "id")

	// Create season and session.
	season := mustPost(t, apiURL(ts, base+"/seasons"), map[string]any{
		"name": "Summer 2026", "start_date": "2026-06-01", "end_date": "2026-08-31",
	})
	seasonID := str(season, "id")

	session := mustPost(t, apiURL(ts, base+"/sessions"), map[string]any{
		"name": "Week 1", "season_id": seasonID,
	})
	sessionID := str(session, "id")
	sessionBase := base + "/sessions/" + sessionID

	// Configure session age group and cabins with capacity.
	sag := mustPost(t, apiURL(ts, sessionBase+"/age-groups"), map[string]any{
		"age_group_id": juniorsID, "group_size": 20,
	})
	sagID := str(sag, "id")

	mustPost(t, apiURL(ts, sessionBase+"/cabins"), map[string]any{
		"session_age_group_id": sagID, "cabin_id": cabinAID,
		"group_size": 4, "required_counselors": 1,
	})
	mustPost(t, apiURL(ts, sessionBase+"/cabins"), map[string]any{
		"session_age_group_id": sagID, "cabin_id": cabinBID,
		"group_size": 4, "required_counselors": 1,
	})

	// Create campers.
	camperNames := []string{"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank"}
	camperIDs := make([]string, len(camperNames))
	for i, name := range camperNames {
		resp := mustPost(t, apiURL(ts, base+"/campers"), map[string]any{"name": name})
		camperIDs[i] = str(resp, "id")
	}

	// Enroll all campers.
	for _, camperID := range camperIDs {
		mustPost(t, apiURL(ts, sessionBase+"/enrollments"), map[string]any{
			"camper_id": camperID, "session_age_group_id": sagID,
		})
	}

	// Trigger camper_cabin run.
	runURL := apiURL(ts, sessionBase+"/assignment-runs")
	runResp := mustPost(t, runURL, map[string]any{
		"run_type": "camper_cabin",
	})

	if str(runResp, "run_type") != "camper_cabin" {
		t.Fatalf("expected run_type camper_cabin, got %s", str(runResp, "run_type"))
	}
	if str(runResp, "status") != "completed" {
		t.Fatalf("expected status completed, got %s", str(runResp, "status"))
	}

	solutions := list(runResp, "solutions")
	if len(solutions) == 0 {
		t.Fatal("expected at least one solution")
	}

	// Check the top solution's assignments.
	topSolution := asMap(solutions[0])
	topSolutionID := str(topSolution, "id")
	runID := str(runResp, "id")

	solDetail := mustGet(t, runURL+"/"+runID+"/solutions/"+topSolutionID)
	assignments := list(solDetail, "assignments")
	if len(assignments) != 6 {
		t.Fatalf("expected 6 camper assignments, got %d", len(assignments))
	}

	// Verify assignments respect capacity: no cabin has more than 4.
	cabinCounts := make(map[string]int)
	for _, a := range assignments {
		am := asMap(a)
		cabinCounts[str(am, "cabin_id")]++
	}
	for cabinID, count := range cabinCounts {
		if count > 4 {
			t.Fatalf("cabin %s has %d campers, exceeding capacity of 4", cabinID, count)
		}
	}

	// Clean up.
	mustDelete(t, runURL+"/"+runID)
}

// testCamperFriendPreferences verifies that the solver optimizes for friend
// preferences and generates appropriate explanations.
func testCamperFriendPreferences(t *testing.T) {
	ts := mustSetupServer(t)

	// Create camp.
	campResp := mustPost(t, apiURL(ts, "/camps"), map[string]any{
		"name": "Camp Friendship",
	})
	campID := str(campResp, "id")
	base := "/camps/" + campID

	// Create age group and cabins.
	teens := mustPost(t, apiURL(ts, base+"/age-groups"), map[string]any{"name": "Teens"})
	teensID := str(teens, "id")

	cabin1 := mustPost(t, apiURL(ts, base+"/cabins"), map[string]any{
		"name": "Hawk", "age_group_id": teensID,
	})
	cabin1ID := str(cabin1, "id")

	cabin2 := mustPost(t, apiURL(ts, base+"/cabins"), map[string]any{
		"name": "Eagle", "age_group_id": teensID,
	})
	cabin2ID := str(cabin2, "id")

	// Create season and session.
	season := mustPost(t, apiURL(ts, base+"/seasons"), map[string]any{
		"name": "Summer 2026", "start_date": "2026-06-01", "end_date": "2026-08-31",
	})
	seasonID := str(season, "id")

	session := mustPost(t, apiURL(ts, base+"/sessions"), map[string]any{
		"name": "Week 1", "season_id": seasonID,
	})
	sessionID := str(session, "id")
	sessionBase := base + "/sessions/" + sessionID

	// Configure cabins with capacity 3 each.
	sag := mustPost(t, apiURL(ts, sessionBase+"/age-groups"), map[string]any{
		"age_group_id": teensID, "group_size": 6,
	})
	sagID := str(sag, "id")

	mustPost(t, apiURL(ts, sessionBase+"/cabins"), map[string]any{
		"session_age_group_id": sagID, "cabin_id": cabin1ID,
		"group_size": 3, "required_counselors": 1,
	})
	mustPost(t, apiURL(ts, sessionBase+"/cabins"), map[string]any{
		"session_age_group_id": sagID, "cabin_id": cabin2ID,
		"group_size": 3, "required_counselors": 1,
	})

	// Create 4 campers: Amy, Beth, Carol, Dana.
	amy := mustPost(t, apiURL(ts, base+"/campers"), map[string]any{"name": "Amy"})
	amyID := str(amy, "id")
	beth := mustPost(t, apiURL(ts, base+"/campers"), map[string]any{"name": "Beth"})
	bethID := str(beth, "id")
	carol := mustPost(t, apiURL(ts, base+"/campers"), map[string]any{"name": "Carol"})
	carolID := str(carol, "id")
	dana := mustPost(t, apiURL(ts, base+"/campers"), map[string]any{"name": "Dana"})
	danaID := str(dana, "id")

	// Enroll all.
	for _, id := range []string{amyID, bethID, carolID, danaID} {
		mustPost(t, apiURL(ts, sessionBase+"/enrollments"), map[string]any{
			"camper_id": id, "session_age_group_id": sagID,
		})
	}

	// Amy wants Beth, Beth wants Amy (mutual friends).
	mustPost(t, apiURL(ts, sessionBase+"/campers/"+amyID+"/friend-preferences"), map[string]any{
		"preferred_camper_id": bethID, "rank": 1,
	})
	mustPost(t, apiURL(ts, sessionBase+"/campers/"+bethID+"/friend-preferences"), map[string]any{
		"preferred_camper_id": amyID, "rank": 1,
	})

	// Carol wants Dana.
	mustPost(t, apiURL(ts, sessionBase+"/campers/"+carolID+"/friend-preferences"), map[string]any{
		"preferred_camper_id": danaID, "rank": 1,
	})

	// Trigger camper run.
	runURL := apiURL(ts, sessionBase+"/assignment-runs")
	runResp := mustPost(t, runURL, map[string]any{
		"run_type":      "camper_cabin",
		"max_solutions": 3,
	})

	solutions := list(runResp, "solutions")
	if len(solutions) == 0 {
		t.Fatal("expected at least one solution")
	}

	// Get the top solution.
	topSolution := asMap(solutions[0])
	topScore := num(topSolution, "score")
	if topScore <= 0 {
		t.Fatalf("expected positive score for solution with friend prefs, got %.1f", topScore)
	}

	// Check the solution detail.
	runID := str(runResp, "id")
	topSolutionID := str(topSolution, "id")
	solDetail := mustGet(t, runURL+"/"+runID+"/solutions/"+topSolutionID)

	// Verify score breakdown exists.
	breakdown := list(solDetail, "score_breakdown")
	if len(breakdown) == 0 {
		t.Fatal("expected non-empty score_breakdown")
	}

	// Verify Amy and Beth are in the same cabin in the top solution.
	assignments := list(solDetail, "assignments")
	camperCabinMap := make(map[string]string)
	for _, a := range assignments {
		am := asMap(a)
		camperCabinMap[str(am, "camper_id")] = str(am, "cabin_id")
	}

	if camperCabinMap[amyID] != camperCabinMap[bethID] {
		t.Fatal("expected Amy and Beth to be in the same cabin (mutual friends)")
	}

	// Verify explanations exist.
	explanations := list(solDetail, "explanations")
	if len(explanations) == 0 {
		t.Fatal("expected non-empty explanations")
	}

	// Check that there are reason-type explanations.
	hasReason := false
	for _, e := range explanations {
		em := asMap(e)
		if str(em, "explanation_type") == "reason" {
			hasReason = true
			break
		}
	}
	if !hasReason {
		t.Fatal("expected at least one 'reason' explanation")
	}

	// Clean up.
	mustDelete(t, runURL+"/"+runID)
}

// testEnrollmentSessionScoping verifies that GET and DELETE enrollment
// endpoints enforce session scoping: accessing an enrollment via the wrong
// session returns 404 even if the enrollment ID and camp ID are valid.
func testEnrollmentSessionScoping(t *testing.T) {
	ts := mustSetupServer(t)

	campResp := mustPost(t, apiURL(ts, "/camps"), map[string]any{"name": "Camp Scope"})
	campID := str(campResp, "id")
	base := "/camps/" + campID

	ag := mustPost(t, apiURL(ts, base+"/age-groups"), map[string]any{"name": "Kids"})
	agID := str(ag, "id")

	season := mustPost(t, apiURL(ts, base+"/seasons"), map[string]any{"name": "Summer", "start_date": "2026-06-01", "end_date": "2026-08-31"})
	seasonID := str(season, "id")

	s1 := mustPost(t, apiURL(ts, base+"/sessions"), map[string]any{"name": "S1", "season_id": seasonID})
	s1ID := str(s1, "id")

	s2 := mustPost(t, apiURL(ts, base+"/sessions"), map[string]any{
		"name": "S2", "season_id": seasonID, "previous_session_id": s1ID,
	})
	s2ID := str(s2, "id")

	sag := mustPost(t, apiURL(ts, base+"/sessions/"+s1ID+"/age-groups"), map[string]any{
		"age_group_id": agID,
	})
	sagID := str(sag, "id")

	camper := mustPost(t, apiURL(ts, base+"/campers"), map[string]any{"name": "Test Camper"})
	camperID := str(camper, "id")

	enrollment := mustPost(t, apiURL(ts, base+"/sessions/"+s1ID+"/enrollments"), map[string]any{
		"camper_id": camperID, "session_age_group_id": sagID,
	})
	enrollmentID := str(enrollment, "id")

	// GET via correct session should succeed.
	mustGet(t, apiURL(ts, base+"/sessions/"+s1ID+"/enrollments/"+enrollmentID))

	// GET via wrong session should return 404.
	doRawRequest(t, http.MethodGet,
		apiURL(ts, base+"/sessions/"+s2ID+"/enrollments/"+enrollmentID),
		nil, http.StatusNotFound)

	// DELETE via wrong session should return 404.
	doRawRequest(t, http.MethodDelete,
		apiURL(ts, base+"/sessions/"+s2ID+"/enrollments/"+enrollmentID),
		nil, http.StatusNotFound)

	// DELETE via correct session should succeed.
	mustDelete(t, apiURL(ts, base+"/sessions/"+s1ID+"/enrollments/"+enrollmentID))
}

// testEnrollmentSessionUniqueness verifies that a camper cannot be enrolled
// twice in the same session, even across different age groups.
func testEnrollmentSessionUniqueness(t *testing.T) {
	ts := mustSetupServer(t)

	campResp := mustPost(t, apiURL(ts, "/camps"), map[string]any{"name": "Camp Unique"})
	campID := str(campResp, "id")
	base := "/camps/" + campID

	ag1 := mustPost(t, apiURL(ts, base+"/age-groups"), map[string]any{"name": "Young"})
	ag1ID := str(ag1, "id")
	ag2 := mustPost(t, apiURL(ts, base+"/age-groups"), map[string]any{"name": "Old"})
	ag2ID := str(ag2, "id")

	season := mustPost(t, apiURL(ts, base+"/seasons"), map[string]any{"name": "Summer", "start_date": "2026-06-01", "end_date": "2026-08-31"})
	seasonID := str(season, "id")

	session := mustPost(t, apiURL(ts, base+"/sessions"), map[string]any{
		"name": "Week 1", "season_id": seasonID,
	})
	sessionID := str(session, "id")

	sag1 := mustPost(t, apiURL(ts, base+"/sessions/"+sessionID+"/age-groups"), map[string]any{
		"age_group_id": ag1ID,
	})
	sag1ID := str(sag1, "id")
	sag2 := mustPost(t, apiURL(ts, base+"/sessions/"+sessionID+"/age-groups"), map[string]any{
		"age_group_id": ag2ID,
	})
	sag2ID := str(sag2, "id")

	camper := mustPost(t, apiURL(ts, base+"/campers"), map[string]any{"name": "Duplicate Dan"})
	camperID := str(camper, "id")

	// First enrollment succeeds.
	mustPost(t, apiURL(ts, base+"/sessions/"+sessionID+"/enrollments"), map[string]any{
		"camper_id": camperID, "session_age_group_id": sag1ID,
	})

	// Second enrollment in the same session (different age group) should fail
	// due to UNIQUE(camper_id, session_id).
	doRawRequest(t, http.MethodPost,
		apiURL(ts, base+"/sessions/"+sessionID+"/enrollments"),
		map[string]any{"camper_id": camperID, "session_age_group_id": sag2ID},
		http.StatusConflict)
}

// testGetSolutionRunOwnership verifies that requesting a solution through a
// run that doesn't own it returns 404, preventing cross-run data leakage.
func testGetSolutionRunOwnership(t *testing.T) {
	ts := mustSetupServer(t)

	campResp := mustPost(t, apiURL(ts, "/camps"), map[string]any{"name": "Camp Ownership"})
	campID := str(campResp, "id")
	base := "/camps/" + campID

	ag := mustPost(t, apiURL(ts, base+"/age-groups"), map[string]any{"name": "Juniors"})
	agID := str(ag, "id")

	cabin := mustPost(t, apiURL(ts, base+"/cabins"), map[string]any{
		"name": "Pine", "age_group_id": agID,
	})
	cabinID := str(cabin, "id")

	season := mustPost(t, apiURL(ts, base+"/seasons"), map[string]any{"name": "Summer", "start_date": "2026-06-01", "end_date": "2026-08-31"})
	seasonID := str(season, "id")

	session := mustPost(t, apiURL(ts, base+"/sessions"), map[string]any{
		"name": "Week 1", "season_id": seasonID,
	})
	sessionID := str(session, "id")

	sag := mustPost(t, apiURL(ts, base+"/sessions/"+sessionID+"/age-groups"), map[string]any{
		"age_group_id": agID, "group_size": 10,
	})
	sagID := str(sag, "id")

	mustPost(t, apiURL(ts, base+"/sessions/"+sessionID+"/cabins"), map[string]any{
		"session_age_group_id": sagID, "cabin_id": cabinID,
		"group_size": 4, "required_counselors": 1,
	})

	// Create 2 campers and enroll them.
	for _, name := range []string{"Alice", "Bob"} {
		c := mustPost(t, apiURL(ts, base+"/campers"), map[string]any{"name": name})
		mustPost(t, apiURL(ts, base+"/sessions/"+sessionID+"/enrollments"), map[string]any{
			"camper_id": str(c, "id"), "session_age_group_id": sagID,
		})
	}

	// Trigger two camper runs to get solutions from different runs.
	runURL := apiURL(ts, base+"/sessions/"+sessionID+"/assignment-runs")
	run1 := mustPost(t, runURL, map[string]any{"run_type": "camper_cabin"})
	run1ID := str(run1, "id")
	sol1ID := str(asMap(list(run1, "solutions")[0]), "id")

	run2 := mustPost(t, runURL, map[string]any{"run_type": "camper_cabin"})
	run2ID := str(run2, "id")

	// Getting run1's solution through run1 should succeed.
	mustGet(t, runURL+"/"+run1ID+"/solutions/"+sol1ID)

	// Getting run1's solution through run2 should return 404.
	doRawRequest(t, http.MethodGet,
		runURL+"/"+run2ID+"/solutions/"+sol1ID,
		nil, http.StatusNotFound)

	// Clean up.
	mustDelete(t, runURL+"/"+run1ID)
	mustDelete(t, runURL+"/"+run2ID)
}

// testGetSolutionRunNotFound verifies that GetSolution returns 404 (not 500)
// when the run doesn't exist.
func testGetSolutionRunNotFound(t *testing.T) {
	ts := mustSetupServer(t)

	campResp := mustPost(t, apiURL(ts, "/camps"), map[string]any{"name": "Camp 404"})
	campID := str(campResp, "id")

	season := mustPost(t, apiURL(ts, "/camps/"+campID+"/seasons"), map[string]any{"name": "Summer", "start_date": "2026-06-01", "end_date": "2026-08-31"})
	seasonID := str(season, "id")

	session := mustPost(t, apiURL(ts, "/camps/"+campID+"/sessions"), map[string]any{
		"name": "Week 1", "season_id": seasonID,
	})
	sessionID := str(session, "id")

	fakeRunID := "00000000-0000-0000-0000-000000000001"
	fakeSolID := "00000000-0000-0000-0000-000000000002"

	runURL := apiURL(ts, "/camps/"+campID+"/sessions/"+sessionID+"/assignment-runs")

	// GetRun with nonexistent run ID should return 404.
	doRawRequest(t, http.MethodGet, runURL+"/"+fakeRunID, nil, http.StatusNotFound)

	// GetSolution with nonexistent run ID should return 404, not 500.
	doRawRequest(t, http.MethodGet,
		runURL+"/"+fakeRunID+"/solutions/"+fakeSolID,
		nil, http.StatusNotFound)
}

// testSelectSolutionCamperRun verifies that SelectSolution works for
// camper_cabin runs (not just counselor_cabin runs).
func testSelectSolutionCamperRun(t *testing.T) {
	ts := mustSetupServer(t)

	campResp := mustPost(t, apiURL(ts, "/camps"), map[string]any{"name": "Camp Select"})
	campID := str(campResp, "id")
	base := "/camps/" + campID

	ag := mustPost(t, apiURL(ts, base+"/age-groups"), map[string]any{"name": "Teens"})
	agID := str(ag, "id")

	cabin := mustPost(t, apiURL(ts, base+"/cabins"), map[string]any{
		"name": "Hawk", "age_group_id": agID,
	})
	cabinID := str(cabin, "id")

	season := mustPost(t, apiURL(ts, base+"/seasons"), map[string]any{"name": "Summer", "start_date": "2026-06-01", "end_date": "2026-08-31"})
	seasonID := str(season, "id")

	session := mustPost(t, apiURL(ts, base+"/sessions"), map[string]any{
		"name": "Week 1", "season_id": seasonID,
	})
	sessionID := str(session, "id")
	sessionBase := base + "/sessions/" + sessionID

	sag := mustPost(t, apiURL(ts, sessionBase+"/age-groups"), map[string]any{
		"age_group_id": agID, "group_size": 10,
	})
	sagID := str(sag, "id")

	mustPost(t, apiURL(ts, sessionBase+"/cabins"), map[string]any{
		"session_age_group_id": sagID, "cabin_id": cabinID,
		"group_size": 4, "required_counselors": 1,
	})

	// Create and enroll 2 campers.
	for _, name := range []string{"Alice", "Bob"} {
		c := mustPost(t, apiURL(ts, base+"/campers"), map[string]any{"name": name})
		mustPost(t, apiURL(ts, sessionBase+"/enrollments"), map[string]any{
			"camper_id": str(c, "id"), "session_age_group_id": sagID,
		})
	}

	// Trigger camper run.
	runURL := apiURL(ts, sessionBase+"/assignment-runs")
	runResp := mustPost(t, runURL, map[string]any{"run_type": "camper_cabin"})
	runID := str(runResp, "id")
	topSolutionID := str(asMap(list(runResp, "solutions")[0]), "id")

	// Select solution should succeed for a camper run.
	selectResp := doRequest(t, http.MethodPost,
		runURL+"/"+runID+"/solutions/"+topSolutionID+"/select", nil, http.StatusOK)

	selectedID := str(selectResp, "selected_solution_id")
	if selectedID != topSolutionID {
		t.Fatalf("expected selected_solution_id %q, got %q", topSolutionID, selectedID)
	}
	if str(selectResp, "status") != "selected" {
		t.Fatalf("expected status 'selected', got %q", str(selectResp, "status"))
	}

	// Verify via GetRun.
	updatedRun := mustGet(t, runURL+"/"+runID)
	if str(updatedRun, "selected_solution_id") != topSolutionID {
		t.Fatal("run detail does not reflect selected camper solution")
	}

	// Clean up.
	mustDelete(t, runURL+"/"+runID)
}

func testCamperRunInvalidSession(t *testing.T) {
	ts := mustSetupServer(t)

	campResp := mustPost(t, apiURL(ts, "/camps"), map[string]any{"name": "Camp Ghost"})
	campID := str(campResp, "id")

	fakeSessionID := "00000000-0000-0000-0000-000000000099"
	runURL := apiURL(ts, "/camps/"+campID+"/sessions/"+fakeSessionID+"/assignment-runs")

	doRawRequest(t, http.MethodPost, runURL, map[string]any{"run_type": "camper_cabin"}, http.StatusNotFound)
}
