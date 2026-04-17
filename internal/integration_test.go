//go:build integration

package internal_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"camp-scheduler/internal/config"
	"camp-scheduler/internal/server"
	"camp-scheduler/internal/testutil"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func mustSetupServer(t *testing.T) (*httptest.Server, *pgxpool.Pool) {
	t.Helper()

	pool := testutil.MustOpenDB(t)
	testutil.TruncateAll(t, pool)

	cfg := config.Config{
		Server: config.ServerConfig{Mode: "test"},
		JWT: config.JWTConfig{
			Secret:          "test-secret",
			AccessTokenTTL:  15 * time.Minute,
			RefreshTokenTTL: 168 * time.Hour,
		},
	}
	srv := server.NewWithPool(cfg, pool, nil)
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return ts, pool
}

const testPassword = "password"

// mustLogin creates a test user for the given camp and returns an access token.
func mustLogin(t *testing.T, ts *httptest.Server, pool *pgxpool.Pool, campID string) string {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hashing password: %v", err)
	}

	_, err = pool.Exec(context.Background(),
		`INSERT INTO users (camp_id, username, email, password_hash, first_name, last_name, role)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 ON CONFLICT (username) DO NOTHING`,
		campID, "admin-"+campID[:8], "admin-"+campID[:8]+"@test.com",
		string(hash), "Test", "Admin", "admin",
	)
	if err != nil {
		t.Fatalf("inserting test user: %v", err)
	}

	resp := doRequest(t, http.MethodPost, apiURL(ts, "/auth/login"),
		map[string]any{"username": "admin-" + campID[:8], "password": testPassword},
		http.StatusOK, "")

	token := str(resp, "access_token")
	if token == "" {
		t.Fatal("expected access_token in login response")
	}
	return token
}

// apiURL builds a full URL from the test server base and a path suffix.
func apiURL(ts *httptest.Server, path string) string {
	return ts.URL + "/api/v1" + path
}

func mustPost(t *testing.T, url string, body any, token string) map[string]any {
	t.Helper()
	return doRequest(t, http.MethodPost, url, body, http.StatusCreated, token)
}

func mustGet(t *testing.T, url string, token string) map[string]any {
	t.Helper()
	return doRequest(t, http.MethodGet, url, nil, http.StatusOK, token)
}

func mustGetList(t *testing.T, url string, token string) []any {
	t.Helper()
	resp := doRawRequest(t, http.MethodGet, url, nil, http.StatusOK, token)
	var result []any
	if err := json.Unmarshal(resp, &result); err != nil {
		t.Fatalf("unmarshaling list response: %v", err)
	}
	return result
}

func mustDelete(t *testing.T, url string, token string) {
	t.Helper()
	doRawRequest(t, http.MethodDelete, url, nil, http.StatusOK, token)
}

func doRequest(t *testing.T, method, url string, body any, expectedStatus int, token string) map[string]any {
	t.Helper()
	raw := doRawRequest(t, method, url, body, expectedStatus, token)
	var result map[string]any
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshaling response from %s %s: %v\nbody: %s", method, url, err, string(raw))
	}
	return result
}

func doRawRequest(t *testing.T, method, url string, body any, expectedStatus int, token string) []byte {
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
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
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
	t.Run("repeated_unmet_preference_boost", testRepeatedUnmetPreferenceBoost)
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

func TestActivitySchedulingSolver(t *testing.T) {
	t.Run("activity_scheduling", testActivityScheduling)
}

func TestAuth(t *testing.T) {
	t.Run("login_success", testLoginSuccess)
	t.Run("login_bad_password", testLoginBadPassword)
	t.Run("login_nonexistent_user", testLoginNonexistentUser)
	t.Run("refresh_success", testRefreshSuccess)
	t.Run("refresh_revokes_old_token", testRefreshRevokesOldToken)
	t.Run("protected_route_no_token", testProtectedRouteNoToken)
	t.Run("protected_route_invalid_token", testProtectedRouteInvalidToken)
	t.Run("cross_camp_isolation", testCrossCampIsolation)
}

func testLoginSuccess(t *testing.T) {
	ts, pool := mustSetupServer(t)

	var directCampID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, "Camp Auth Test").Scan(&directCampID)
	if err != nil {
		t.Fatalf("inserting camp: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("secret123"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hashing password: %v", err)
	}

	_, err = pool.Exec(context.Background(),
		`INSERT INTO users (camp_id, username, email, password_hash, first_name, last_name, role)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		directCampID, "testuser", "testuser@test.com", string(hash), "Test", "User", "admin",
	)
	if err != nil {
		t.Fatalf("inserting user: %v", err)
	}

	resp := doRequest(t, http.MethodPost, apiURL(ts, "/auth/login"),
		map[string]any{"username": "testuser", "password": "secret123"},
		http.StatusOK, "")

	if str(resp, "access_token") == "" {
		t.Fatal("expected access_token in response")
	}
	if str(resp, "refresh_token") == "" {
		t.Fatal("expected refresh_token in response")
	}
}

func testLoginBadPassword(t *testing.T) {
	ts, pool := mustSetupServer(t)

	var campID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, "Camp Bad PW").Scan(&campID)
	if err != nil {
		t.Fatalf("inserting camp: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hashing password: %v", err)
	}

	_, err = pool.Exec(context.Background(),
		`INSERT INTO users (camp_id, username, email, password_hash, first_name, last_name, role)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		campID, "badpwuser", "badpw@test.com", string(hash), "Test", "User", "admin",
	)
	if err != nil {
		t.Fatalf("inserting user: %v", err)
	}

	doRawRequest(t, http.MethodPost, apiURL(ts, "/auth/login"),
		map[string]any{"username": "badpwuser", "password": "wrong"},
		http.StatusUnauthorized, "")
}

func testLoginNonexistentUser(t *testing.T) {
	ts, _ := mustSetupServer(t)

	doRawRequest(t, http.MethodPost, apiURL(ts, "/auth/login"),
		map[string]any{"username": "ghost", "password": "whatever"},
		http.StatusUnauthorized, "")
}

func testRefreshSuccess(t *testing.T) {
	ts, pool := mustSetupServer(t)

	var campID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, "Camp Refresh").Scan(&campID)
	if err != nil {
		t.Fatalf("inserting camp: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hashing password: %v", err)
	}

	_, err = pool.Exec(context.Background(),
		`INSERT INTO users (camp_id, username, email, password_hash, first_name, last_name, role)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		campID, "refreshuser", "refresh@test.com", string(hash), "Test", "User", "admin",
	)
	if err != nil {
		t.Fatalf("inserting user: %v", err)
	}

	loginResp := doRequest(t, http.MethodPost, apiURL(ts, "/auth/login"),
		map[string]any{"username": "refreshuser", "password": testPassword},
		http.StatusOK, "")

	refreshToken := str(loginResp, "refresh_token")
	if refreshToken == "" {
		t.Fatal("expected refresh_token in login response")
	}

	refreshResp := doRequest(t, http.MethodPost, apiURL(ts, "/auth/refresh"),
		map[string]any{"refresh_token": refreshToken},
		http.StatusOK, "")

	if str(refreshResp, "access_token") == "" {
		t.Fatal("expected access_token in refresh response")
	}
	if str(refreshResp, "refresh_token") == "" {
		t.Fatal("expected refresh_token in refresh response")
	}

	// Verify the refreshed access token works on a protected endpoint.
	doRawRequest(t, http.MethodGet, apiURL(ts, "/camp"),
		nil, http.StatusOK, str(refreshResp, "access_token"))
}

func testRefreshRevokesOldToken(t *testing.T) {
	ts, pool := mustSetupServer(t)

	var campID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, "Camp Revoke").Scan(&campID)
	if err != nil {
		t.Fatalf("inserting camp: %v", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hashing password: %v", err)
	}

	_, err = pool.Exec(context.Background(),
		`INSERT INTO users (camp_id, username, email, password_hash, first_name, last_name, role)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		campID, "revokeuser", "revoke@test.com", string(hash), "Test", "User", "admin",
	)
	if err != nil {
		t.Fatalf("inserting user: %v", err)
	}

	loginResp := doRequest(t, http.MethodPost, apiURL(ts, "/auth/login"),
		map[string]any{"username": "revokeuser", "password": testPassword},
		http.StatusOK, "")

	refreshToken := str(loginResp, "refresh_token")

	// Use the refresh token once — should succeed.
	doRequest(t, http.MethodPost, apiURL(ts, "/auth/refresh"),
		map[string]any{"refresh_token": refreshToken},
		http.StatusOK, "")

	// Reuse the same refresh token — should fail (already revoked).
	doRawRequest(t, http.MethodPost, apiURL(ts, "/auth/refresh"),
		map[string]any{"refresh_token": refreshToken},
		http.StatusUnauthorized, "")
}

func testProtectedRouteNoToken(t *testing.T) {
	ts, _ := mustSetupServer(t)

	doRawRequest(t, http.MethodGet, apiURL(ts, "/camp"), nil, http.StatusUnauthorized, "")
}

func testProtectedRouteInvalidToken(t *testing.T) {
	ts, _ := mustSetupServer(t)

	doRawRequest(t, http.MethodGet, apiURL(ts, "/camp"), nil, http.StatusUnauthorized, "garbage-token")
}

func testActivityScheduling(t *testing.T) {
	ts, pool := mustSetupServer(t)

	var campID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, "Camp Activities").Scan(&campID)
	if err != nil {
		t.Fatalf("inserting camp: %v", err)
	}
	token := mustLogin(t, ts, pool, campID)

	// Create certifications.
	lifeguard := mustPost(t, apiURL(ts, "/certifications"), map[string]any{
		"name": "Lifeguard",
	}, token)
	lifeguardID := str(lifeguard, "id")

	archeryInstructor := mustPost(t, apiURL(ts, "/certifications"), map[string]any{
		"name": "Archery Instructor",
	}, token)
	archeryInstructorID := str(archeryInstructor, "id")

	// Create activities.
	swimming := mustPost(t, apiURL(ts, "/activities"), map[string]any{
		"name": "Swimming",
	}, token)
	swimmingID := str(swimming, "id")

	archery := mustPost(t, apiURL(ts, "/activities"), map[string]any{
		"name": "Archery",
	}, token)
	archeryID := str(archery, "id")

	artsCrafts := mustPost(t, apiURL(ts, "/activities"), map[string]any{
		"name": "Arts & Crafts",
	}, token)
	artsCraftsID := str(artsCrafts, "id")

	// Add certifications to activities.
	mustPost(t, apiURL(ts, "/activities/"+swimmingID+"/certifications"), map[string]any{
		"certification_id": lifeguardID,
	}, token)
	mustPost(t, apiURL(ts, "/activities/"+archeryID+"/certifications"), map[string]any{
		"certification_id": archeryInstructorID,
	}, token)

	// Create counselors.
	type counselorInfo struct {
		id   string
		name string
	}
	counselorDefs := []struct {
		name   string
		junior bool
	}{
		{"Alice", false}, // lifeguard + archery
		{"Bob", false},   // lifeguard
		{"Carol", false}, // archery
		{"Dave", false},  // no certs
		{"Eve", false},   // no certs
	}
	counselors := make([]counselorInfo, len(counselorDefs))
	for i, c := range counselorDefs {
		resp := mustPost(t, apiURL(ts, "/counselors"), map[string]any{
			"name":             c.name,
			"junior_counselor": c.junior,
		}, token)
		counselors[i] = counselorInfo{id: str(resp, "id"), name: c.name}
	}

	alice, bob, carol, dave, eve := counselors[0], counselors[1], counselors[2], counselors[3], counselors[4]

	// Add certifications to counselors.
	// Alice: lifeguard + archery instructor
	mustPost(t, apiURL(ts, "/counselors/"+alice.id+"/certifications"), map[string]any{
		"certification_id": lifeguardID,
	}, token)
	mustPost(t, apiURL(ts, "/counselors/"+alice.id+"/certifications"), map[string]any{
		"certification_id": archeryInstructorID,
	}, token)
	// Bob: lifeguard
	mustPost(t, apiURL(ts, "/counselors/"+bob.id+"/certifications"), map[string]any{
		"certification_id": lifeguardID,
	}, token)
	// Carol: archery instructor
	mustPost(t, apiURL(ts, "/counselors/"+carol.id+"/certifications"), map[string]any{
		"certification_id": archeryInstructorID,
	}, token)

	// Create season and session.
	season := mustPost(t, apiURL(ts, "/seasons"), map[string]any{
		"name": "Summer 2026", "start_date": "2026-06-01", "end_date": "2026-08-31",
	}, token)
	seasonID := str(season, "id")

	session := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
		"name": "Week 1", "season_id": seasonID,
	}, token)
	sessionID := str(session, "id")
	sessionBase := "/sessions/" + sessionID

	// Create time slots.
	period1 := mustPost(t, apiURL(ts, "/time-slots"), map[string]any{
		"name": "Period 1",
	}, token)
	period1ID := str(period1, "id")

	period2 := mustPost(t, apiURL(ts, "/time-slots"), map[string]any{
		"name": "Period 2",
	}, token)
	period2ID := str(period2, "id")

	// Create session time slots.
	sts1 := mustPost(t, apiURL(ts, sessionBase+"/time-slots"), map[string]any{
		"time_slot_id": period1ID,
		"sort_order":   1,
	}, token)
	sts1ID := str(sts1, "id")

	sts2 := mustPost(t, apiURL(ts, sessionBase+"/time-slots"), map[string]any{
		"time_slot_id": period2ID,
		"sort_order":   2,
	}, token)
	sts2ID := str(sts2, "id")

	// Create session activities.
	// Swimming in Period 1 (requires lifeguard, 1 counselor, capacity 2)
	mustPost(t, apiURL(ts, sessionBase+"/time-slots/"+sts1ID+"/activities"), map[string]any{
		"activity_id":         swimmingID,
		"capacity":            2,
		"required_counselors": 1,
	}, token)

	// Archery in Period 1 (requires archery instructor, 1 counselor, capacity 2)
	mustPost(t, apiURL(ts, sessionBase+"/time-slots/"+sts1ID+"/activities"), map[string]any{
		"activity_id":         archeryID,
		"capacity":            2,
		"required_counselors": 1,
	}, token)

	// Arts & Crafts in Period 2 (no cert required, 1 counselor, capacity 3)
	mustPost(t, apiURL(ts, sessionBase+"/time-slots/"+sts2ID+"/activities"), map[string]any{
		"activity_id":         artsCraftsID,
		"capacity":            3,
		"required_counselors": 1,
	}, token)

	// Set activity preferences.
	prefBase := sessionBase + "/counselors/"
	// Alice prefers Swimming
	mustPost(t, apiURL(ts, prefBase+alice.id+"/activity-preferences"), map[string]any{
		"activity_id": swimmingID, "rank": 1,
	}, token)
	// Bob prefers Swimming
	mustPost(t, apiURL(ts, prefBase+bob.id+"/activity-preferences"), map[string]any{
		"activity_id": swimmingID, "rank": 1,
	}, token)
	// Carol prefers Archery
	mustPost(t, apiURL(ts, prefBase+carol.id+"/activity-preferences"), map[string]any{
		"activity_id": archeryID, "rank": 1,
	}, token)
	// Dave prefers Arts & Crafts
	mustPost(t, apiURL(ts, prefBase+dave.id+"/activity-preferences"), map[string]any{
		"activity_id": artsCraftsID, "rank": 1,
	}, token)
	// Eve prefers Arts & Crafts
	mustPost(t, apiURL(ts, prefBase+eve.id+"/activity-preferences"), map[string]any{
		"activity_id": artsCraftsID, "rank": 1,
	}, token)

	// Trigger activity_schedule run.
	runURL := apiURL(ts, sessionBase+"/assignment-runs")
	runResp := mustPost(t, runURL, map[string]any{
		"run_type": "activity_schedule",
	}, token)

	if str(runResp, "run_type") != "activity_schedule" {
		t.Fatalf("expected run_type activity_schedule, got %s", str(runResp, "run_type"))
	}
	if str(runResp, "status") != "completed" {
		t.Fatalf("expected status completed, got %s", str(runResp, "status"))
	}

	solutions := list(runResp, "solutions")
	if len(solutions) == 0 {
		t.Fatal("expected at least one solution")
	}

	// Get the top solution.
	topSolution := asMap(solutions[0])
	topSolutionID := str(topSolution, "id")
	runID := str(runResp, "id")

	solDetail := mustGet(t, runURL+"/"+runID+"/solutions/"+topSolutionID, token)

	assignments := list(solDetail, "assignments")
	if len(assignments) == 0 {
		t.Fatal("expected assignments in activity solution detail")
	}

	// Verify each assignment has counselor_id and session_activity_id.
	for _, a := range assignments {
		am := asMap(a)
		if str(am, "counselor_id") == "" {
			t.Fatal("expected counselor_id in activity assignment")
		}
		if str(am, "session_activity_id") == "" {
			t.Fatal("expected session_activity_id in activity assignment")
		}
	}

	explanations := list(solDetail, "explanations")
	if len(explanations) == 0 {
		t.Fatal("expected explanations in activity solution detail")
	}

	// Select the solution.
	selectResp := doRequest(t, http.MethodPost,
		runURL+"/"+runID+"/solutions/"+topSolutionID+"/select", nil, http.StatusOK, token)
	selectedID := str(selectResp, "selected_solution_id")
	if selectedID != topSolutionID {
		t.Fatalf("expected selected_solution_id %q, got %q", topSolutionID, selectedID)
	}
	if str(selectResp, "status") != "selected" {
		t.Fatalf("expected status 'selected', got %q", str(selectResp, "status"))
	}

	// Verify via GetRun.
	updatedRun := mustGet(t, runURL+"/"+runID, token)
	if str(updatedRun, "selected_solution_id") != topSolutionID {
		t.Fatal("run detail does not reflect selected activity solution")
	}

	// Clean up.
	mustDelete(t, runURL+"/"+runID, token)
}

// testSimpleCamp exercises the full API -> DB -> Solver -> Results flow with a
// small, realistic camp: 2 age groups, 4 cabins, 6 counselors.
func testSimpleCamp(t *testing.T) {
	ts, pool := mustSetupServer(t)

	var campID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name, camp_location) VALUES ($1, $2) RETURNING id`,
		"Camp Pinebrook", "Vermont").Scan(&campID)
	if err != nil {
		t.Fatalf("inserting camp: %v", err)
	}
	token := mustLogin(t, ts, pool, campID)

	juniors := mustPost(t, apiURL(ts, "/age-groups"), map[string]any{
		"name": "Juniors",
	}, token)
	juniorsID := str(juniors, "id")

	seniors := mustPost(t, apiURL(ts, "/age-groups"), map[string]any{
		"name": "Seniors",
	}, token)
	seniorsID := str(seniors, "id")

	pine := mustPost(t, apiURL(ts, "/cabins"), map[string]any{
		"name":                 "Pine",
		"default_age_group_id": juniorsID,
	}, token)
	pineID := str(pine, "id")
	if got := str(pine, "default_age_group_name"); got != "Juniors" {
		t.Fatalf("expected default_age_group_name %q, got %q", "Juniors", got)
	}

	oak := mustPost(t, apiURL(ts, "/cabins"), map[string]any{
		"name":                 "Oak",
		"default_age_group_id": juniorsID,
	}, token)
	oakID := str(oak, "id")
	if got := str(oak, "default_age_group_name"); got != "Juniors" {
		t.Fatalf("expected default_age_group_name %q, got %q", "Juniors", got)
	}

	maple := mustPost(t, apiURL(ts, "/cabins"), map[string]any{
		"name":                 "Maple",
		"default_age_group_id": seniorsID,
	}, token)
	mapleID := str(maple, "id")
	if got := str(maple, "default_age_group_name"); got != "Seniors" {
		t.Fatalf("expected default_age_group_name %q, got %q", "Seniors", got)
	}

	cedar := mustPost(t, apiURL(ts, "/cabins"), map[string]any{
		"name":                 "Cedar",
		"default_age_group_id": seniorsID,
	}, token)
	cedarID := str(cedar, "id")
	if got := str(cedar, "default_age_group_name"); got != "Seniors" {
		t.Fatalf("expected default_age_group_name %q, got %q", "Seniors", got)
	}

	seasonResp := mustPost(t, apiURL(ts, "/seasons"), map[string]any{
		"name":       "Summer 2025",
		"start_date": "2025-06-01",
		"end_date":   "2025-08-31",
	}, token)
	seasonID := str(seasonResp, "id")

	session1 := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
		"name":      "Session 1",
		"season_id": seasonID,
	}, token)
	session1ID := str(session1, "id")

	session2 := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
		"name":                "Session 2",
		"season_id":           seasonID,
		"previous_session_id": session1ID,
	}, token)
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
		resp := mustPost(t, apiURL(ts, "/counselors"), map[string]any{
			"name":             c.name,
			"junior_counselor": c.junior,
		}, token)
		counselors = append(counselors, counselorInfo{id: str(resp, "id"), name: c.name})
	}

	alice, bob, carol := counselors[0], counselors[1], counselors[2]
	eve := counselors[4]

	sagJuniors := mustPost(t, apiURL(ts, "/sessions/"+session2ID+"/age-groups"), map[string]any{
		"age_group_id": juniorsID,
	}, token)
	sagJuniorsID := str(sagJuniors, "id")

	sagSeniors := mustPost(t, apiURL(ts, "/sessions/"+session2ID+"/age-groups"), map[string]any{
		"age_group_id": seniorsID,
	}, token)
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
		mustPost(t, apiURL(ts, "/sessions/"+session2ID+"/cabins"), map[string]any{
			"session_age_group_id": cfg.sagID,
			"cabin_id":             cfg.cabinID,
			"required_counselors":  reqCounselors,
		}, token)
	}

	// Alice and Bob were both in Pine (Juniors) in session 1, making them
	// co-counselors for the returning-counselor scoring. Carol was in Maple.
	historyBase := "/counselors/"
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
		mustPost(t, apiURL(ts, historyBase+h.counselorID+"/session-history"), body, token)
	}

	prefBase := "/sessions/" + session2ID + "/counselors/"
	mustPost(t, apiURL(ts, prefBase+alice.id+"/age-group-preferences"), map[string]any{
		"age_group_id": juniorsID,
		"rank":         1,
	}, token)
	mustPost(t, apiURL(ts, prefBase+bob.id+"/cocounselor-preferences"), map[string]any{
		"preferred_counselor_id": alice.id,
		"rank":                   1,
	}, token)
	mustPost(t, apiURL(ts, prefBase+carol.id+"/age-group-preferences"), map[string]any{
		"age_group_id": seniorsID,
		"rank":         1,
	}, token)

	runURL := apiURL(ts, "/sessions/"+session2ID+"/assignment-runs")
	triggerResp := mustPost(t, runURL, nil, token)

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

	runs := mustGetList(t, runURL, token)
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	if str(asMap(runs[0]), "id") != runID {
		t.Fatal("listed run ID does not match created run")
	}

	runDetail := mustGet(t, runURL+"/"+runID, token)
	detailSolutions := list(runDetail, "solutions")
	if len(detailSolutions) != len(solutions) {
		t.Fatalf("expected %d solutions in detail, got %d", len(solutions), len(detailSolutions))
	}

	solDetail := mustGet(t, runURL+"/"+runID+"/solutions/"+topSolutionID, token)

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
		runURL+"/"+runID+"/solutions/"+topSolutionID+"/select", nil, http.StatusOK, token)
	selectedID := str(selectResp, "selected_solution_id")
	if selectedID != topSolutionID {
		t.Fatalf("expected selected_solution_id %q, got %q", topSolutionID, selectedID)
	}

	updatedRun := mustGet(t, runURL+"/"+runID, token)
	if str(updatedRun, "selected_solution_id") != topSolutionID {
		t.Fatal("run detail does not reflect selected solution")
	}

	mustDelete(t, runURL+"/"+runID, token)

	runsAfterDelete := mustGetList(t, runURL, token)
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
	ts, pool := mustSetupServer(t)

	var campID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name, camp_location) VALUES ($1, $2) RETURNING id`,
		"Camp Ridgewood", "New Hampshire").Scan(&campID)
	if err != nil {
		t.Fatalf("inserting camp: %v", err)
	}
	token := mustLogin(t, ts, pool, campID)

	young := mustPost(t, apiURL(ts, "/age-groups"), map[string]any{"name": "Young"}, token)
	youngID := str(young, "id")

	middle := mustPost(t, apiURL(ts, "/age-groups"), map[string]any{"name": "Middle"}, token)
	middleID := str(middle, "id")

	teen := mustPost(t, apiURL(ts, "/age-groups"), map[string]any{"name": "Teen"}, token)
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
		resp := mustPost(t, apiURL(ts, "/cabins"), map[string]any{
			"name":                 cd.name,
			"default_age_group_id": cd.ageGroupID,
		}, token)
		cabins[i] = cabinInfo{id: str(resp, "id"), name: cd.name}
	}

	seasonResp := mustPost(t, apiURL(ts, "/seasons"), map[string]any{
		"name":       "Summer 2025",
		"start_date": "2025-06-01",
		"end_date":   "2025-08-31",
	}, token)
	seasonID := str(seasonResp, "id")

	session1 := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
		"name":      "Session 1",
		"season_id": seasonID,
	}, token)
	session1ID := str(session1, "id")

	session2 := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
		"name":                "Session 2",
		"season_id":           seasonID,
		"previous_session_id": session1ID,
	}, token)
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
		resp := mustPost(t, apiURL(ts, "/counselors"), map[string]any{
			"name":             c.name,
			"junior_counselor": c.junior,
		}, token)
		allCounselors[i] = counselorInfo{
			id:       str(resp, "id"),
			name:     c.name,
			isJunior: c.junior,
		}
	}

	gina, hank, iris, jake := allCounselors[0], allCounselors[1], allCounselors[2], allCounselors[3]
	kim := allCounselors[4]
	mia := allCounselors[6]

	sagYoung := mustPost(t, apiURL(ts, "/sessions/"+session2ID+"/age-groups"), map[string]any{
		"age_group_id": youngID,
	}, token)
	sagYoungID := str(sagYoung, "id")

	sagMiddle := mustPost(t, apiURL(ts, "/sessions/"+session2ID+"/age-groups"), map[string]any{
		"age_group_id": middleID,
	}, token)
	sagMiddleID := str(sagMiddle, "id")

	sagTeen := mustPost(t, apiURL(ts, "/sessions/"+session2ID+"/age-groups"), map[string]any{
		"age_group_id": teenID,
	}, token)
	sagTeenID := str(sagTeen, "id")

	sagIDs := []string{sagYoungID, sagYoungID, sagMiddleID, sagMiddleID, sagTeenID, sagTeenID}
	reqCounselors := int32(1)
	for i, cab := range cabins {
		mustPost(t, apiURL(ts, "/sessions/"+session2ID+"/cabins"), map[string]any{
			"session_age_group_id": sagIDs[i],
			"cabin_id":             cab.id,
			"required_counselors":  reqCounselors,
		}, token)
	}

	// Gina and Hank were both in Birch (Young) in session 1, making them
	// co-counselors. Iris was in Spruce (Middle), Jake was in Redwood (Teen).
	historyBase := "/counselors/"
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
		}, token)
	}

	prefBase := "/sessions/" + session2ID + "/counselors/"

	mustPost(t, apiURL(ts, prefBase+gina.id+"/age-group-preferences"), map[string]any{
		"age_group_id": youngID, "rank": 1,
	}, token)
	mustPost(t, apiURL(ts, prefBase+hank.id+"/age-group-preferences"), map[string]any{
		"age_group_id": youngID, "rank": 1,
	}, token)
	mustPost(t, apiURL(ts, prefBase+hank.id+"/cocounselor-preferences"), map[string]any{
		"preferred_counselor_id": gina.id, "rank": 1,
	}, token)
	mustPost(t, apiURL(ts, prefBase+iris.id+"/age-group-preferences"), map[string]any{
		"age_group_id": middleID, "rank": 1,
	}, token)
	mustPost(t, apiURL(ts, prefBase+jake.id+"/age-group-preferences"), map[string]any{
		"age_group_id": teenID, "rank": 1,
	}, token)
	mustPost(t, apiURL(ts, prefBase+jake.id+"/cocounselor-preferences"), map[string]any{
		"preferred_counselor_id": kim.id, "rank": 1,
	}, token)
	// Kim prefers Young, but Jake wants Kim as a Teen co-counselor -- can't both be satisfied.
	mustPost(t, apiURL(ts, prefBase+kim.id+"/age-group-preferences"), map[string]any{
		"age_group_id": youngID, "rank": 1,
	}, token)
	mustPost(t, apiURL(ts, prefBase+allCounselors[5].id+"/age-group-preferences"), map[string]any{
		"age_group_id": middleID, "rank": 1,
	}, token)
	mustPost(t, apiURL(ts, prefBase+mia.id+"/age-group-preferences"), map[string]any{
		"age_group_id": teenID, "rank": 1,
	}, token)
	mustPost(t, apiURL(ts, prefBase+allCounselors[7].id+"/cocounselor-preferences"), map[string]any{
		"preferred_counselor_id": mia.id, "rank": 1,
	}, token)

	runURL := apiURL(ts, "/sessions/"+session2ID+"/assignment-runs")
	triggerResp := mustPost(t, runURL, map[string]any{
		"max_solutions": 5,
	}, token)

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
	solDetail := mustGet(t, runURL+"/"+runID+"/solutions/"+topSolutionID, token)

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
		runURL+"/"+runID+"/solutions/"+topSolutionID+"/select", nil, http.StatusOK, token)
	if str(selectResp, "selected_solution_id") != topSolutionID {
		t.Fatal("selected_solution_id mismatch after selection")
	}

	mustDelete(t, runURL+"/"+runID, token)
	runsAfterDelete := mustGetList(t, runURL, token)
	if len(runsAfterDelete) != 0 {
		t.Fatalf("expected 0 runs after delete, got %d", len(runsAfterDelete))
	}
}

// testBasicCamperAssignment verifies the full camper assignment flow:
// create camp entities, enroll campers, trigger a camper_cabin run,
// and verify assignments respect cabin capacity and age group constraints.
func testBasicCamperAssignment(t *testing.T) {
	ts, pool := mustSetupServer(t)

	var campID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, "Camp Lakeside").Scan(&campID)
	if err != nil {
		t.Fatalf("inserting camp: %v", err)
	}
	token := mustLogin(t, ts, pool, campID)

	// Create age groups.
	juniors := mustPost(t, apiURL(ts, "/age-groups"), map[string]any{"name": "Young"}, token)
	juniorsID := str(juniors, "id")

	// Create cabins.
	cabinA := mustPost(t, apiURL(ts, "/cabins"), map[string]any{
		"name": "Birch", "default_age_group_id": juniorsID,
	}, token)
	cabinAID := str(cabinA, "id")
	if got := str(cabinA, "default_age_group_name"); got != "Young" {
		t.Fatalf("expected default_age_group_name %q, got %q", "Young", got)
	}

	cabinB := mustPost(t, apiURL(ts, "/cabins"), map[string]any{
		"name": "Elm", "default_age_group_id": juniorsID,
	}, token)
	cabinBID := str(cabinB, "id")
	if got := str(cabinB, "default_age_group_name"); got != "Young" {
		t.Fatalf("expected default_age_group_name %q, got %q", "Young", got)
	}

	// Create season and session.
	season := mustPost(t, apiURL(ts, "/seasons"), map[string]any{
		"name": "Summer 2026", "start_date": "2026-06-01", "end_date": "2026-08-31",
	}, token)
	seasonID := str(season, "id")

	session := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
		"name": "Week 1", "season_id": seasonID,
	}, token)
	sessionID := str(session, "id")
	sessionBase := "/sessions/" + sessionID

	// Configure session age group and cabins with capacity.
	sag := mustPost(t, apiURL(ts, sessionBase+"/age-groups"), map[string]any{
		"age_group_id": juniorsID, "group_size": 20,
	}, token)
	sagID := str(sag, "id")

	mustPost(t, apiURL(ts, sessionBase+"/cabins"), map[string]any{
		"session_age_group_id": sagID, "cabin_id": cabinAID,
		"group_size": 4, "required_counselors": 1,
	}, token)
	mustPost(t, apiURL(ts, sessionBase+"/cabins"), map[string]any{
		"session_age_group_id": sagID, "cabin_id": cabinBID,
		"group_size": 4, "required_counselors": 1,
	}, token)

	// Create campers.
	camperNames := []string{"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank"}
	camperIDs := make([]string, len(camperNames))
	for i, name := range camperNames {
		resp := mustPost(t, apiURL(ts, "/campers"), map[string]any{"name": name}, token)
		camperIDs[i] = str(resp, "id")
	}

	// Enroll all campers.
	for _, camperID := range camperIDs {
		mustPost(t, apiURL(ts, sessionBase+"/enrollments"), map[string]any{
			"camper_id": camperID, "session_age_group_id": sagID,
		}, token)
	}

	// Trigger camper_cabin run.
	runURL := apiURL(ts, sessionBase+"/assignment-runs")
	runResp := mustPost(t, runURL, map[string]any{
		"run_type": "camper_cabin",
	}, token)

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

	solDetail := mustGet(t, runURL+"/"+runID+"/solutions/"+topSolutionID, token)
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
	mustDelete(t, runURL+"/"+runID, token)
}

// testCamperFriendPreferences verifies that the solver optimizes for friend
// preferences and generates appropriate explanations.
func testCamperFriendPreferences(t *testing.T) {
	ts, pool := mustSetupServer(t)

	var campID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, "Camp Friendship").Scan(&campID)
	if err != nil {
		t.Fatalf("inserting camp: %v", err)
	}
	token := mustLogin(t, ts, pool, campID)

	// Create age group and cabins.
	teens := mustPost(t, apiURL(ts, "/age-groups"), map[string]any{"name": "Teens"}, token)
	teensID := str(teens, "id")

	cabin1 := mustPost(t, apiURL(ts, "/cabins"), map[string]any{
		"name": "Hawk", "default_age_group_id": teensID,
	}, token)
	cabin1ID := str(cabin1, "id")
	if got := str(cabin1, "default_age_group_name"); got != "Teens" {
		t.Fatalf("expected default_age_group_name %q, got %q", "Teens", got)
	}

	cabin2 := mustPost(t, apiURL(ts, "/cabins"), map[string]any{
		"name": "Eagle", "default_age_group_id": teensID,
	}, token)
	cabin2ID := str(cabin2, "id")
	if got := str(cabin2, "default_age_group_name"); got != "Teens" {
		t.Fatalf("expected default_age_group_name %q, got %q", "Teens", got)
	}

	// Create season and session.
	season := mustPost(t, apiURL(ts, "/seasons"), map[string]any{
		"name": "Summer 2026", "start_date": "2026-06-01", "end_date": "2026-08-31",
	}, token)
	seasonID := str(season, "id")

	session := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
		"name": "Week 1", "season_id": seasonID,
	}, token)
	sessionID := str(session, "id")
	sessionBase := "/sessions/" + sessionID

	// Configure cabins with capacity 3 each.
	sag := mustPost(t, apiURL(ts, sessionBase+"/age-groups"), map[string]any{
		"age_group_id": teensID, "group_size": 6,
	}, token)
	sagID := str(sag, "id")

	mustPost(t, apiURL(ts, sessionBase+"/cabins"), map[string]any{
		"session_age_group_id": sagID, "cabin_id": cabin1ID,
		"group_size": 3, "required_counselors": 1,
	}, token)
	mustPost(t, apiURL(ts, sessionBase+"/cabins"), map[string]any{
		"session_age_group_id": sagID, "cabin_id": cabin2ID,
		"group_size": 3, "required_counselors": 1,
	}, token)

	// Create 4 campers: Amy, Beth, Carol, Dana.
	amy := mustPost(t, apiURL(ts, "/campers"), map[string]any{"name": "Amy"}, token)
	amyID := str(amy, "id")
	beth := mustPost(t, apiURL(ts, "/campers"), map[string]any{"name": "Beth"}, token)
	bethID := str(beth, "id")
	carol := mustPost(t, apiURL(ts, "/campers"), map[string]any{"name": "Carol"}, token)
	carolID := str(carol, "id")
	dana := mustPost(t, apiURL(ts, "/campers"), map[string]any{"name": "Dana"}, token)
	danaID := str(dana, "id")

	// Enroll all.
	for _, id := range []string{amyID, bethID, carolID, danaID} {
		mustPost(t, apiURL(ts, sessionBase+"/enrollments"), map[string]any{
			"camper_id": id, "session_age_group_id": sagID,
		}, token)
	}

	// Amy wants Beth, Beth wants Amy (mutual friends).
	mustPost(t, apiURL(ts, sessionBase+"/campers/"+amyID+"/friend-preferences"), map[string]any{
		"preferred_camper_id": bethID, "rank": 1,
	}, token)
	mustPost(t, apiURL(ts, sessionBase+"/campers/"+bethID+"/friend-preferences"), map[string]any{
		"preferred_camper_id": amyID, "rank": 1,
	}, token)

	// Carol wants Dana.
	mustPost(t, apiURL(ts, sessionBase+"/campers/"+carolID+"/friend-preferences"), map[string]any{
		"preferred_camper_id": danaID, "rank": 1,
	}, token)

	// Trigger camper run.
	runURL := apiURL(ts, sessionBase+"/assignment-runs")
	runResp := mustPost(t, runURL, map[string]any{
		"run_type":      "camper_cabin",
		"max_solutions": 3,
	}, token)

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
	solDetail := mustGet(t, runURL+"/"+runID+"/solutions/"+topSolutionID, token)

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
	mustDelete(t, runURL+"/"+runID, token)
}

// testEnrollmentSessionScoping verifies that GET and DELETE enrollment
// endpoints enforce session scoping: accessing an enrollment via the wrong
// session returns 404 even if the enrollment ID and camp ID are valid.
func testEnrollmentSessionScoping(t *testing.T) {
	ts, pool := mustSetupServer(t)

	var campID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, "Camp Scope").Scan(&campID)
	if err != nil {
		t.Fatalf("inserting camp: %v", err)
	}
	token := mustLogin(t, ts, pool, campID)

	ag := mustPost(t, apiURL(ts, "/age-groups"), map[string]any{"name": "Kids"}, token)
	agID := str(ag, "id")

	season := mustPost(t, apiURL(ts, "/seasons"), map[string]any{"name": "Summer", "start_date": "2026-06-01", "end_date": "2026-08-31"}, token)
	seasonID := str(season, "id")

	s1 := mustPost(t, apiURL(ts, "/sessions"), map[string]any{"name": "S1", "season_id": seasonID}, token)
	s1ID := str(s1, "id")

	s2 := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
		"name": "S2", "season_id": seasonID, "previous_session_id": s1ID,
	}, token)
	s2ID := str(s2, "id")

	sag := mustPost(t, apiURL(ts, "/sessions/"+s1ID+"/age-groups"), map[string]any{
		"age_group_id": agID,
	}, token)
	sagID := str(sag, "id")

	camper := mustPost(t, apiURL(ts, "/campers"), map[string]any{"name": "Test Camper"}, token)
	camperID := str(camper, "id")

	enrollment := mustPost(t, apiURL(ts, "/sessions/"+s1ID+"/enrollments"), map[string]any{
		"camper_id": camperID, "session_age_group_id": sagID,
	}, token)
	enrollmentID := str(enrollment, "id")

	// GET via correct session should succeed.
	mustGet(t, apiURL(ts, "/sessions/"+s1ID+"/enrollments/"+enrollmentID), token)

	// GET via wrong session should return 404.
	doRawRequest(t, http.MethodGet,
		apiURL(ts, "/sessions/"+s2ID+"/enrollments/"+enrollmentID),
		nil, http.StatusNotFound, token)

	// DELETE via wrong session should return 404.
	doRawRequest(t, http.MethodDelete,
		apiURL(ts, "/sessions/"+s2ID+"/enrollments/"+enrollmentID),
		nil, http.StatusNotFound, token)

	// DELETE via correct session should succeed.
	mustDelete(t, apiURL(ts, "/sessions/"+s1ID+"/enrollments/"+enrollmentID), token)
}

// testEnrollmentSessionUniqueness verifies that a camper cannot be enrolled
// twice in the same session, even across different age groups.
func testEnrollmentSessionUniqueness(t *testing.T) {
	ts, pool := mustSetupServer(t)

	var campID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, "Camp Unique").Scan(&campID)
	if err != nil {
		t.Fatalf("inserting camp: %v", err)
	}
	token := mustLogin(t, ts, pool, campID)

	ag1 := mustPost(t, apiURL(ts, "/age-groups"), map[string]any{"name": "Young"}, token)
	ag1ID := str(ag1, "id")
	ag2 := mustPost(t, apiURL(ts, "/age-groups"), map[string]any{"name": "Old"}, token)
	ag2ID := str(ag2, "id")

	season := mustPost(t, apiURL(ts, "/seasons"), map[string]any{"name": "Summer", "start_date": "2026-06-01", "end_date": "2026-08-31"}, token)
	seasonID := str(season, "id")

	session := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
		"name": "Week 1", "season_id": seasonID,
	}, token)
	sessionID := str(session, "id")

	sag1 := mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/age-groups"), map[string]any{
		"age_group_id": ag1ID,
	}, token)
	sag1ID := str(sag1, "id")
	sag2 := mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/age-groups"), map[string]any{
		"age_group_id": ag2ID,
	}, token)
	sag2ID := str(sag2, "id")

	camper := mustPost(t, apiURL(ts, "/campers"), map[string]any{"name": "Duplicate Dan"}, token)
	camperID := str(camper, "id")

	// First enrollment succeeds.
	mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/enrollments"), map[string]any{
		"camper_id": camperID, "session_age_group_id": sag1ID,
	}, token)

	// Second enrollment in the same session (different age group) should fail
	// due to UNIQUE(camper_id, session_id).
	doRawRequest(t, http.MethodPost,
		apiURL(ts, "/sessions/"+sessionID+"/enrollments"),
		map[string]any{"camper_id": camperID, "session_age_group_id": sag2ID},
		http.StatusConflict, token)
}

// testGetSolutionRunOwnership verifies that requesting a solution through a
// run that doesn't own it returns 404, preventing cross-run data leakage.
func testGetSolutionRunOwnership(t *testing.T) {
	ts, pool := mustSetupServer(t)

	var campID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, "Camp Ownership").Scan(&campID)
	if err != nil {
		t.Fatalf("inserting camp: %v", err)
	}
	token := mustLogin(t, ts, pool, campID)

	ag := mustPost(t, apiURL(ts, "/age-groups"), map[string]any{"name": "Juniors"}, token)
	agID := str(ag, "id")

	cabin := mustPost(t, apiURL(ts, "/cabins"), map[string]any{
		"name": "Pine", "default_age_group_id": agID,
	}, token)
	cabinID := str(cabin, "id")

	season := mustPost(t, apiURL(ts, "/seasons"), map[string]any{"name": "Summer", "start_date": "2026-06-01", "end_date": "2026-08-31"}, token)
	seasonID := str(season, "id")

	session := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
		"name": "Week 1", "season_id": seasonID,
	}, token)
	sessionID := str(session, "id")

	sag := mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/age-groups"), map[string]any{
		"age_group_id": agID, "group_size": 10,
	}, token)
	sagID := str(sag, "id")

	mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/cabins"), map[string]any{
		"session_age_group_id": sagID, "cabin_id": cabinID,
		"group_size": 4, "required_counselors": 1,
	}, token)

	// Create 2 campers and enroll them.
	for _, name := range []string{"Alice", "Bob"} {
		c := mustPost(t, apiURL(ts, "/campers"), map[string]any{"name": name}, token)
		mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/enrollments"), map[string]any{
			"camper_id": str(c, "id"), "session_age_group_id": sagID,
		}, token)
	}

	// Trigger two camper runs to get solutions from different runs.
	runURL := apiURL(ts, "/sessions/"+sessionID+"/assignment-runs")
	run1 := mustPost(t, runURL, map[string]any{"run_type": "camper_cabin"}, token)
	run1ID := str(run1, "id")
	sol1ID := str(asMap(list(run1, "solutions")[0]), "id")

	run2 := mustPost(t, runURL, map[string]any{"run_type": "camper_cabin"}, token)
	run2ID := str(run2, "id")

	// Getting run1's solution through run1 should succeed.
	mustGet(t, runURL+"/"+run1ID+"/solutions/"+sol1ID, token)

	// Getting run1's solution through run2 should return 404.
	doRawRequest(t, http.MethodGet,
		runURL+"/"+run2ID+"/solutions/"+sol1ID,
		nil, http.StatusNotFound, token)

	// Clean up.
	mustDelete(t, runURL+"/"+run1ID, token)
	mustDelete(t, runURL+"/"+run2ID, token)
}

// testGetSolutionRunNotFound verifies that GetSolution returns 404 (not 500)
// when the run doesn't exist.
func testGetSolutionRunNotFound(t *testing.T) {
	ts, pool := mustSetupServer(t)

	var campID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, "Camp 404").Scan(&campID)
	if err != nil {
		t.Fatalf("inserting camp: %v", err)
	}
	token := mustLogin(t, ts, pool, campID)

	season := mustPost(t, apiURL(ts, "/seasons"), map[string]any{"name": "Summer", "start_date": "2026-06-01", "end_date": "2026-08-31"}, token)
	seasonID := str(season, "id")

	session := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
		"name": "Week 1", "season_id": seasonID,
	}, token)
	sessionID := str(session, "id")

	fakeRunID := "00000000-0000-0000-0000-000000000001"
	fakeSolID := "00000000-0000-0000-0000-000000000002"

	runURL := apiURL(ts, "/sessions/"+sessionID+"/assignment-runs")

	// GetRun with nonexistent run ID should return 404.
	doRawRequest(t, http.MethodGet, runURL+"/"+fakeRunID, nil, http.StatusNotFound, token)

	// GetSolution with nonexistent run ID should return 404, not 500.
	doRawRequest(t, http.MethodGet,
		runURL+"/"+fakeRunID+"/solutions/"+fakeSolID,
		nil, http.StatusNotFound, token)
}

// testSelectSolutionCamperRun verifies that SelectSolution works for
// camper_cabin runs (not just counselor_cabin runs).
func testSelectSolutionCamperRun(t *testing.T) {
	ts, pool := mustSetupServer(t)

	var campID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, "Camp Select").Scan(&campID)
	if err != nil {
		t.Fatalf("inserting camp: %v", err)
	}
	token := mustLogin(t, ts, pool, campID)

	ag := mustPost(t, apiURL(ts, "/age-groups"), map[string]any{"name": "Teens"}, token)
	agID := str(ag, "id")

	cabin := mustPost(t, apiURL(ts, "/cabins"), map[string]any{
		"name": "Hawk", "default_age_group_id": agID,
	}, token)
	cabinID := str(cabin, "id")

	season := mustPost(t, apiURL(ts, "/seasons"), map[string]any{"name": "Summer", "start_date": "2026-06-01", "end_date": "2026-08-31"}, token)
	seasonID := str(season, "id")

	session := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
		"name": "Week 1", "season_id": seasonID,
	}, token)
	sessionID := str(session, "id")
	sessionBase := "/sessions/" + sessionID

	sag := mustPost(t, apiURL(ts, sessionBase+"/age-groups"), map[string]any{
		"age_group_id": agID, "group_size": 10,
	}, token)
	sagID := str(sag, "id")

	mustPost(t, apiURL(ts, sessionBase+"/cabins"), map[string]any{
		"session_age_group_id": sagID, "cabin_id": cabinID,
		"group_size": 4, "required_counselors": 1,
	}, token)

	// Create and enroll 2 campers.
	for _, name := range []string{"Alice", "Bob"} {
		c := mustPost(t, apiURL(ts, "/campers"), map[string]any{"name": name}, token)
		mustPost(t, apiURL(ts, sessionBase+"/enrollments"), map[string]any{
			"camper_id": str(c, "id"), "session_age_group_id": sagID,
		}, token)
	}

	// Trigger camper run.
	runURL := apiURL(ts, sessionBase+"/assignment-runs")
	runResp := mustPost(t, runURL, map[string]any{"run_type": "camper_cabin"}, token)
	runID := str(runResp, "id")
	topSolutionID := str(asMap(list(runResp, "solutions")[0]), "id")

	// Select solution should succeed for a camper run.
	selectResp := doRequest(t, http.MethodPost,
		runURL+"/"+runID+"/solutions/"+topSolutionID+"/select", nil, http.StatusOK, token)

	selectedID := str(selectResp, "selected_solution_id")
	if selectedID != topSolutionID {
		t.Fatalf("expected selected_solution_id %q, got %q", topSolutionID, selectedID)
	}
	if str(selectResp, "status") != "selected" {
		t.Fatalf("expected status 'selected', got %q", str(selectResp, "status"))
	}

	// Verify via GetRun.
	updatedRun := mustGet(t, runURL+"/"+runID, token)
	if str(updatedRun, "selected_solution_id") != topSolutionID {
		t.Fatal("run detail does not reflect selected camper solution")
	}

	// Clean up.
	mustDelete(t, runURL+"/"+runID, token)
}

func testCamperRunInvalidSession(t *testing.T) {
	ts, pool := mustSetupServer(t)

	var campID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, "Camp Ghost").Scan(&campID)
	if err != nil {
		t.Fatalf("inserting camp: %v", err)
	}
	token := mustLogin(t, ts, pool, campID)

	fakeSessionID := "00000000-0000-0000-0000-000000000099"
	runURL := apiURL(ts, "/sessions/"+fakeSessionID+"/assignment-runs")

	doRawRequest(t, http.MethodPost, runURL, map[string]any{"run_type": "camper_cabin"}, http.StatusNotFound, token)
}

// testRepeatedUnmetPreferenceBoost exercises the full flow of:
// 1. Run session 1 solver with conflicting preferences (some go unmet)
// 2. Select the session 1 solution
// 3. Run session 2 solver with the same preferences
// 4. Verify the session 2 score breakdown reflects the "(previously unmet)" boost
//
// Scenario: 2 age groups, 2 cabins, 3 counselors. Alice and Bob both prefer
// Young, but only one Young cabin exists, so one must go to Old. In session 2,
// the counselor whose Young preference was unmet gets a score boost.
func testRepeatedUnmetPreferenceBoost(t *testing.T) {
	ts, pool := mustSetupServer(t)

	var campID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, "Camp Boost").Scan(&campID)
	if err != nil {
		t.Fatalf("inserting camp: %v", err)
	}
	token := mustLogin(t, ts, pool, campID)

	// Create age groups.
	young := mustPost(t, apiURL(ts, "/age-groups"), map[string]any{"name": "Young"}, token)
	youngID := str(young, "id")

	old := mustPost(t, apiURL(ts, "/age-groups"), map[string]any{"name": "Old"}, token)
	oldID := str(old, "id")

	// Create cabins.
	pine := mustPost(t, apiURL(ts, "/cabins"), map[string]any{
		"name": "Pine", "default_age_group_id": youngID,
	}, token)
	pineID := str(pine, "id")

	oak := mustPost(t, apiURL(ts, "/cabins"), map[string]any{
		"name": "Oak", "default_age_group_id": oldID,
	}, token)
	oakID := str(oak, "id")

	// Create season and sessions.
	season := mustPost(t, apiURL(ts, "/seasons"), map[string]any{
		"name": "Summer 2025", "start_date": "2025-06-01", "end_date": "2025-08-31",
	}, token)
	seasonID := str(season, "id")

	session1 := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
		"name": "Session 1", "season_id": seasonID,
	}, token)
	session1ID := str(session1, "id")

	session2 := mustPost(t, apiURL(ts, "/sessions"), map[string]any{
		"name":                "Session 2",
		"season_id":           seasonID,
		"previous_session_id": session1ID,
	}, token)
	session2ID := str(session2, "id")

	// Create 2 senior counselors. Both prefer Young, but only one Young cabin
	// exists so the solver must leave one preference unmet.
	type counselorInfo struct {
		id   string
		name string
	}
	var counselors []counselorInfo
	for _, name := range []string{"Alice", "Bob"} {
		resp := mustPost(t, apiURL(ts, "/counselors"), map[string]any{
			"name":             name,
			"junior_counselor": false,
		}, token)
		counselors = append(counselors, counselorInfo{id: str(resp, "id"), name: name})
	}
	alice, bob := counselors[0], counselors[1]

	// Configure session 1 cabins (1 required counselor each).
	configureCabins := func(sessionID string) {
		sagYoung := mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/age-groups"), map[string]any{
			"age_group_id": youngID,
		}, token)
		sagOld := mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/age-groups"), map[string]any{
			"age_group_id": oldID,
		}, token)
		mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/cabins"), map[string]any{
			"session_age_group_id": str(sagYoung, "id"),
			"cabin_id":             pineID,
			"required_counselors":  1,
		}, token)
		mustPost(t, apiURL(ts, "/sessions/"+sessionID+"/cabins"), map[string]any{
			"session_age_group_id": str(sagOld, "id"),
			"cabin_id":             oakID,
			"required_counselors":  1,
		}, token)
	}

	configureCabins(session1ID)
	configureCabins(session2ID)

	// Set preferences: both Alice and Bob want Young.
	setPreferences := func(sessionID string) {
		prefBase := "/sessions/" + sessionID + "/counselors/"
		mustPost(t, apiURL(ts, prefBase+alice.id+"/age-group-preferences"), map[string]any{
			"age_group_id": youngID, "rank": 1,
		}, token)
		mustPost(t, apiURL(ts, prefBase+bob.id+"/age-group-preferences"), map[string]any{
			"age_group_id": youngID, "rank": 1,
		}, token)
	}

	setPreferences(session1ID)

	// Trigger session 1 run.
	run1URL := apiURL(ts, "/sessions/"+session1ID+"/assignment-runs")
	run1Resp := mustPost(t, run1URL, nil, token)
	run1ID := str(run1Resp, "id")

	solutions1 := list(run1Resp, "solutions")
	if len(solutions1) == 0 {
		t.Fatal("expected at least one solution for session 1")
	}
	sol1ID := str(asMap(solutions1[0]), "id")

	// Select the top solution for session 1.
	doRequest(t, http.MethodPost,
		run1URL+"/"+run1ID+"/solutions/"+sol1ID+"/select", nil, http.StatusOK, token)

	// Set the same preferences for session 2.
	setPreferences(session2ID)

	// Trigger session 2 run.
	run2URL := apiURL(ts, "/sessions/"+session2ID+"/assignment-runs")
	run2Resp := mustPost(t, run2URL, nil, token)
	run2ID := str(run2Resp, "id")

	solutions2 := list(run2Resp, "solutions")
	if len(solutions2) == 0 {
		t.Fatal("expected at least one solution for session 2")
	}
	sol2ID := str(asMap(solutions2[0]), "id")

	// Fetch the session 2 solution detail and verify the boost.
	sol2Detail := mustGet(t, run2URL+"/"+run2ID+"/solutions/"+sol2ID, token)

	breakdown := list(sol2Detail, "score_breakdown")
	if len(breakdown) == 0 {
		t.Fatal("expected non-empty score_breakdown")
	}

	hasPreviouslyUnmet := false
	for _, b := range breakdown {
		bm := asMap(b)
		msg := str(bm, "Message")
		if strings.Contains(msg, "previously unmet") {
			hasPreviouslyUnmet = true
			break
		}
	}
	if !hasPreviouslyUnmet {
		t.Fatal("expected at least one score component with '(previously unmet)' boost in session 2")
	}

	// Clean up.
	mustDelete(t, run1URL+"/"+run1ID, token)
	mustDelete(t, run2URL+"/"+run2ID, token)
}

func testCrossCampIsolation(t *testing.T) {
	ts, pool := mustSetupServer(t)

	var campAID, campBID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, "Camp Alpha").Scan(&campAID)
	if err != nil {
		t.Fatalf("inserting camp A: %v", err)
	}
	err = pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, "Camp Bravo").Scan(&campBID)
	if err != nil {
		t.Fatalf("inserting camp B: %v", err)
	}

	tokenA := mustLogin(t, ts, pool, campAID)
	tokenB := mustLogin(t, ts, pool, campBID)

	// GET /camp returns each user's own camp.
	campA := mustGet(t, apiURL(ts, "/camp"), tokenA)
	campB := mustGet(t, apiURL(ts, "/camp"), tokenB)
	if str(campA, "id") != campAID {
		t.Fatalf("expected camp A id %s, got %s", campAID, str(campA, "id"))
	}
	if str(campB, "id") != campBID {
		t.Fatalf("expected camp B id %s, got %s", campBID, str(campB, "id"))
	}

	// Create resources in camp A.
	ageGroup := mustPost(t, apiURL(ts, "/age-groups"), map[string]any{"name": "Juniors"}, tokenA)
	ageGroupID := str(ageGroup, "id")

	cabin := mustPost(t, apiURL(ts, "/cabins"), map[string]any{
		"name": "Pine", "default_age_group_id": ageGroupID,
	}, tokenA)
	cabinID := str(cabin, "id")

	counselor := mustPost(t, apiURL(ts, "/counselors"), map[string]any{
		"name": "Alice", "junior_counselor": false,
	}, tokenA)
	counselorID := str(counselor, "id")

	season := mustPost(t, apiURL(ts, "/seasons"), map[string]any{
		"name": "Summer 2026", "start_date": "2026-06-01", "end_date": "2026-08-31",
	}, tokenA)
	seasonID := str(season, "id")

	// Token B must not see camp A's resources in list endpoints.
	for _, tc := range []struct {
		name string
		path string
	}{
		{"age-groups", "/age-groups"},
		{"cabins", "/cabins"},
		{"counselors", "/counselors"},
		{"seasons", "/seasons"},
	} {
		items := mustGetList(t, apiURL(ts, tc.path), tokenB)
		if len(items) != 0 {
			t.Errorf("token B listed %d %s, expected 0", len(items), tc.name)
		}
	}

	// Token B must get 404 when fetching camp A resources by ID.
	for _, tc := range []struct {
		name string
		path string
	}{
		{"age-group", "/age-groups/" + ageGroupID},
		{"cabin", "/cabins/" + cabinID},
		{"counselor", "/counselors/" + counselorID},
		{"season", "/seasons/" + seasonID},
	} {
		doRawRequest(t, http.MethodGet, apiURL(ts, tc.path), nil, http.StatusNotFound, tokenB)
	}
}

// mustLoginSuperAdmin creates a super-admin user (no camp association) and
// returns an access token.
func mustLoginSuperAdmin(t *testing.T, ts *httptest.Server, pool *pgxpool.Pool) string {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hashing password: %v", err)
	}

	_, err = pool.Exec(context.Background(),
		`INSERT INTO users (camp_id, username, email, password_hash, first_name, last_name, role)
		 VALUES (NULL, $1, $2, $3, $4, $5, $6)
		 ON CONFLICT (username) DO NOTHING`,
		"superadmin", "superadmin@test.com",
		string(hash), "Super", "Admin", "super_admin",
	)
	if err != nil {
		t.Fatalf("inserting super-admin user: %v", err)
	}

	resp := doRequest(t, http.MethodPost, apiURL(ts, "/auth/login"),
		map[string]any{"username": "superadmin", "password": testPassword},
		http.StatusOK, "")

	token := str(resp, "access_token")
	if token == "" {
		t.Fatal("expected access_token in login response")
	}
	return token
}

func TestSuperAdminCampCRUD(t *testing.T) {
	t.Run("full_lifecycle", testSuperAdminCampLifecycle)
	t.Run("admin_rejected_from_admin_routes", testAdminRejectedFromAdminRoutes)
	t.Run("super_admin_rejected_from_camp_routes", testSuperAdminRejectedFromCampRoutes)
}

func testSuperAdminCampLifecycle(t *testing.T) {
	ts, pool := mustSetupServer(t)
	token := mustLoginSuperAdmin(t, ts, pool)

	// Create a camp.
	location := "Lake Tahoe"
	created := mustPost(t, apiURL(ts, "/admin/camps"),
		map[string]any{"name": "Camp Lifecycle", "location": location}, token)

	campID := str(created, "id")
	if campID == "" {
		t.Fatal("expected id in create response")
	}
	if str(created, "name") != "Camp Lifecycle" {
		t.Fatalf("expected name 'Camp Lifecycle', got %q", str(created, "name"))
	}
	if str(created, "location") != "Lake Tahoe" {
		t.Fatalf("expected location 'Lake Tahoe', got %q", str(created, "location"))
	}

	// List camps — the created camp must appear.
	camps := mustGetList(t, apiURL(ts, "/admin/camps"), token)
	if len(camps) < 1 {
		t.Fatal("expected at least 1 camp in list")
	}
	found := false
	for _, c := range camps {
		m := c.(map[string]any)
		if str(m, "id") == campID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("created camp %s not found in list", campID)
	}

	// Get camp by ID.
	got := mustGet(t, apiURL(ts, "/admin/camps/"+campID), token)
	if str(got, "name") != "Camp Lifecycle" {
		t.Fatalf("expected name 'Camp Lifecycle', got %q", str(got, "name"))
	}

	// Update camp.
	newLocation := "Yosemite"
	updated := doRequest(t, http.MethodPut, apiURL(ts, "/admin/camps/"+campID),
		map[string]any{"name": "Camp Updated", "location": newLocation, "enabled": false},
		http.StatusOK, token)
	if str(updated, "name") != "Camp Updated" {
		t.Fatalf("expected name 'Camp Updated', got %q", str(updated, "name"))
	}
	if str(updated, "location") != "Yosemite" {
		t.Fatalf("expected location 'Yosemite', got %q", str(updated, "location"))
	}

	// Verify update persisted.
	gotAfterUpdate := mustGet(t, apiURL(ts, "/admin/camps/"+campID), token)
	if str(gotAfterUpdate, "name") != "Camp Updated" {
		t.Fatalf("expected persisted name 'Camp Updated', got %q", str(gotAfterUpdate, "name"))
	}

	// Delete camp.
	mustDelete(t, apiURL(ts, "/admin/camps/"+campID), token)

	// Verify deletion — GET should return 404.
	doRawRequest(t, http.MethodGet, apiURL(ts, "/admin/camps/"+campID), nil, http.StatusNotFound, token)
}

func testAdminRejectedFromAdminRoutes(t *testing.T) {
	ts, pool := mustSetupServer(t)

	var campID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, "Camp Admin Reject").Scan(&campID)
	if err != nil {
		t.Fatalf("inserting camp: %v", err)
	}

	adminToken := mustLogin(t, ts, pool, campID)

	// Regular admin should get 403 on all admin endpoints.
	doRawRequest(t, http.MethodGet, apiURL(ts, "/admin/camps"), nil, http.StatusForbidden, adminToken)
	doRawRequest(t, http.MethodPost, apiURL(ts, "/admin/camps"),
		map[string]any{"name": "Sneaky Camp"}, http.StatusForbidden, adminToken)
}

func testSuperAdminRejectedFromCampRoutes(t *testing.T) {
	ts, pool := mustSetupServer(t)
	superToken := mustLoginSuperAdmin(t, ts, pool)

	// Super-admin should get 403 on camp-scoped routes.
	doRawRequest(t, http.MethodGet, apiURL(ts, "/camp"), nil, http.StatusForbidden, superToken)
	doRawRequest(t, http.MethodGet, apiURL(ts, "/age-groups"), nil, http.StatusForbidden, superToken)
	doRawRequest(t, http.MethodGet, apiURL(ts, "/cabins"), nil, http.StatusForbidden, superToken)
}

func TestSuperAdminUserCRUD(t *testing.T) {
	t.Run("full_lifecycle", testSuperAdminUserLifecycle)
	t.Run("admin_rejected_from_user_routes", testAdminRejectedFromUserRoutes)
	t.Run("constraint_violations", testUserConstraintViolations)
	t.Run("password_change_revokes_tokens", testPasswordChangeRevokesTokens)
}

func testSuperAdminUserLifecycle(t *testing.T) {
	ts, pool := mustSetupServer(t)
	token := mustLoginSuperAdmin(t, ts, pool)

	// Create a camp to associate with admin users.
	camp := mustPost(t, apiURL(ts, "/admin/camps"),
		map[string]any{"name": "User Test Camp", "location": "Somewhere"}, token)
	campID := str(camp, "id")

	// Create an admin user.
	created := mustPost(t, apiURL(ts, "/admin/users"), map[string]any{
		"camp_id":    campID,
		"username":   "testadmin",
		"email":      "testadmin@example.com",
		"password":   "securepass123",
		"first_name": "Test",
		"last_name":  "Admin",
		"role":       "admin",
	}, token)

	userID := str(created, "id")
	if userID == "" {
		t.Fatal("expected id in create response")
	}
	if str(created, "username") != "testadmin" {
		t.Fatalf("expected username 'testadmin', got %q", str(created, "username"))
	}
	if str(created, "role") != "admin" {
		t.Fatalf("expected role 'admin', got %q", str(created, "role"))
	}
	if str(created, "camp_id") != campID {
		t.Fatalf("expected camp_id %q, got %q", campID, str(created, "camp_id"))
	}

	// Create a super_admin user.
	superUser := mustPost(t, apiURL(ts, "/admin/users"), map[string]any{
		"username":   "newsuper",
		"email":      "newsuper@example.com",
		"password":   "securepass123",
		"first_name": "New",
		"last_name":  "Super",
		"role":       "super_admin",
	}, token)
	superUserID := str(superUser, "id")
	if str(superUser, "camp_id") != "" {
		t.Fatalf("expected empty camp_id for super_admin, got %q", str(superUser, "camp_id"))
	}

	// List all users — both created users (and the seeded super-admin) must appear.
	users := mustGetList(t, apiURL(ts, "/admin/users"), token)
	if len(users) < 3 {
		t.Fatalf("expected at least 3 users in list, got %d", len(users))
	}

	// List users filtered by camp_id.
	campUsers := mustGetList(t, apiURL(ts, "/admin/users?camp_id="+campID), token)
	if len(campUsers) != 1 {
		t.Fatalf("expected 1 user for camp, got %d", len(campUsers))
	}
	if str(asMap(campUsers[0]), "id") != userID {
		t.Fatalf("expected user %s in camp list, got %s", userID, str(asMap(campUsers[0]), "id"))
	}

	// Get user by ID.
	got := mustGet(t, apiURL(ts, "/admin/users/"+userID), token)
	if str(got, "username") != "testadmin" {
		t.Fatalf("expected username 'testadmin', got %q", str(got, "username"))
	}

	// Update user.
	updated := doRequest(t, http.MethodPut, apiURL(ts, "/admin/users/"+userID),
		map[string]any{
			"camp_id":    campID,
			"username":   "updatedadmin",
			"email":      "updated@example.com",
			"first_name": "Updated",
			"last_name":  "Admin",
			"role":       "admin",
		}, http.StatusOK, token)
	if str(updated, "username") != "updatedadmin" {
		t.Fatalf("expected username 'updatedadmin', got %q", str(updated, "username"))
	}
	if str(updated, "email") != "updated@example.com" {
		t.Fatalf("expected email 'updated@example.com', got %q", str(updated, "email"))
	}

	// Verify update persisted.
	gotAfterUpdate := mustGet(t, apiURL(ts, "/admin/users/"+userID), token)
	if str(gotAfterUpdate, "first_name") != "Updated" {
		t.Fatalf("expected persisted first_name 'Updated', got %q", str(gotAfterUpdate, "first_name"))
	}

	// Update password.
	doRawRequest(t, http.MethodPut, apiURL(ts, "/admin/users/"+userID+"/password"),
		map[string]any{"password": "newpassword123"}, http.StatusOK, token)

	// Verify new password works by logging in.
	doRequest(t, http.MethodPost, apiURL(ts, "/auth/login"),
		map[string]any{"username": "updatedadmin", "password": "newpassword123"},
		http.StatusOK, "")

	// Update password for nonexistent user returns 404.
	doRawRequest(t, http.MethodPut,
		apiURL(ts, "/admin/users/00000000-0000-0000-0000-000000000000/password"),
		map[string]any{"password": "newpassword123"}, http.StatusNotFound, token)

	// Get nonexistent user returns 404.
	doRawRequest(t, http.MethodGet,
		apiURL(ts, "/admin/users/00000000-0000-0000-0000-000000000000"),
		nil, http.StatusNotFound, token)

	// Update nonexistent user returns 404.
	doRawRequest(t, http.MethodPut,
		apiURL(ts, "/admin/users/00000000-0000-0000-0000-000000000000"),
		map[string]any{
			"camp_id":    campID,
			"username":   "ghost",
			"email":      "ghost@example.com",
			"first_name": "Ghost",
			"last_name":  "User",
			"role":       "admin",
		}, http.StatusNotFound, token)

	// Delete nonexistent user returns 404.
	doRawRequest(t, http.MethodDelete,
		apiURL(ts, "/admin/users/00000000-0000-0000-0000-000000000000"),
		nil, http.StatusNotFound, token)

	// Delete user.
	mustDelete(t, apiURL(ts, "/admin/users/"+userID), token)

	// Verify deletion.
	doRawRequest(t, http.MethodGet, apiURL(ts, "/admin/users/"+userID),
		nil, http.StatusNotFound, token)

	// Clean up the other created user.
	mustDelete(t, apiURL(ts, "/admin/users/"+superUserID), token)
}

func testAdminRejectedFromUserRoutes(t *testing.T) {
	ts, pool := mustSetupServer(t)

	var campID string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, "Camp User Reject").Scan(&campID)
	if err != nil {
		t.Fatalf("inserting camp: %v", err)
	}

	adminToken := mustLogin(t, ts, pool, campID)

	doRawRequest(t, http.MethodGet, apiURL(ts, "/admin/users"), nil, http.StatusForbidden, adminToken)
	doRawRequest(t, http.MethodPost, apiURL(ts, "/admin/users"),
		map[string]any{
			"camp_id":    campID,
			"username":   "sneaky",
			"email":      "sneaky@example.com",
			"password":   "securepass123",
			"first_name": "Sneaky",
			"last_name":  "User",
			"role":       "admin",
		}, http.StatusForbidden, adminToken)
}

func testUserConstraintViolations(t *testing.T) {
	ts, pool := mustSetupServer(t)
	token := mustLoginSuperAdmin(t, ts, pool)

	camp := mustPost(t, apiURL(ts, "/admin/camps"),
		map[string]any{"name": "Constraint Camp", "location": "Here"}, token)
	campID := str(camp, "id")

	// Create a user to test conflicts against.
	mustPost(t, apiURL(ts, "/admin/users"), map[string]any{
		"camp_id":    campID,
		"username":   "existing",
		"email":      "existing@example.com",
		"password":   "securepass123",
		"first_name": "Existing",
		"last_name":  "User",
		"role":       "admin",
	}, token)

	// Duplicate username → 409.
	doRawRequest(t, http.MethodPost, apiURL(ts, "/admin/users"),
		map[string]any{
			"camp_id":    campID,
			"username":   "existing",
			"email":      "different@example.com",
			"password":   "securepass123",
			"first_name": "Dup",
			"last_name":  "User",
			"role":       "admin",
		}, http.StatusConflict, token)

	// Duplicate email → 409.
	doRawRequest(t, http.MethodPost, apiURL(ts, "/admin/users"),
		map[string]any{
			"camp_id":    campID,
			"username":   "different",
			"email":      "existing@example.com",
			"password":   "securepass123",
			"first_name": "Dup",
			"last_name":  "User",
			"role":       "admin",
		}, http.StatusConflict, token)

	// admin role with no camp_id → 400 (check violation).
	doRawRequest(t, http.MethodPost, apiURL(ts, "/admin/users"),
		map[string]any{
			"username":   "nocampuser",
			"email":      "nocampuser@example.com",
			"password":   "securepass123",
			"first_name": "No",
			"last_name":  "Camp",
			"role":       "admin",
		}, http.StatusBadRequest, token)

	// super_admin role with camp_id → 400 (check violation).
	doRawRequest(t, http.MethodPost, apiURL(ts, "/admin/users"),
		map[string]any{
			"camp_id":    campID,
			"username":   "badsuperadmin",
			"email":      "badsuperadmin@example.com",
			"password":   "securepass123",
			"first_name": "Bad",
			"last_name":  "Super",
			"role":       "super_admin",
		}, http.StatusBadRequest, token)

	// Invalid camp_id (FK violation) → 400.
	doRawRequest(t, http.MethodPost, apiURL(ts, "/admin/users"),
		map[string]any{
			"camp_id":    "00000000-0000-0000-0000-000000000000",
			"username":   "badcamp",
			"email":      "badcamp@example.com",
			"password":   "securepass123",
			"first_name": "Bad",
			"last_name":  "Camp",
			"role":       "admin",
		}, http.StatusBadRequest, token)

	// Missing required fields → 400.
	doRawRequest(t, http.MethodPost, apiURL(ts, "/admin/users"),
		map[string]any{"username": "incomplete"}, http.StatusBadRequest, token)

	// Short password → 400.
	doRawRequest(t, http.MethodPost, apiURL(ts, "/admin/users"),
		map[string]any{
			"camp_id":    campID,
			"username":   "shortpw",
			"email":      "shortpw@example.com",
			"password":   "short",
			"first_name": "Short",
			"last_name":  "Pw",
			"role":       "admin",
		}, http.StatusBadRequest, token)

	// Short password on update → 400.
	doRawRequest(t, http.MethodPut,
		apiURL(ts, "/admin/users/00000000-0000-0000-0000-000000000000/password"),
		map[string]any{"password": "short"}, http.StatusBadRequest, token)
}

func testPasswordChangeRevokesTokens(t *testing.T) {
	ts, pool := mustSetupServer(t)
	superToken := mustLoginSuperAdmin(t, ts, pool)

	// Create a camp and an admin user via the super-admin API.
	camp := mustPost(t, apiURL(ts, "/admin/camps"),
		map[string]any{"name": "Token Revoke Camp", "location": "Somewhere"}, superToken)
	campID := str(camp, "id")

	created := mustPost(t, apiURL(ts, "/admin/users"), map[string]any{
		"camp_id":    campID,
		"username":   "tokenuser",
		"email":      "tokenuser@example.com",
		"password":   "oldpassword123",
		"first_name": "Token",
		"last_name":  "User",
		"role":       "admin",
	}, superToken)
	userID := str(created, "id")

	// Login twice to simulate two devices with independent tokens.
	loginResp1 := doRequest(t, http.MethodPost, apiURL(ts, "/auth/login"),
		map[string]any{"username": "tokenuser", "password": "oldpassword123"},
		http.StatusOK, "")
	accessToken1 := str(loginResp1, "access_token")
	refreshToken1 := str(loginResp1, "refresh_token")
	if accessToken1 == "" {
		t.Fatal("expected access_token in first login response")
	}
	if refreshToken1 == "" {
		t.Fatal("expected refresh_token in first login response")
	}

	loginResp2 := doRequest(t, http.MethodPost, apiURL(ts, "/auth/login"),
		map[string]any{"username": "tokenuser", "password": "oldpassword123"},
		http.StatusOK, "")
	accessToken2 := str(loginResp2, "access_token")
	refreshToken2 := str(loginResp2, "refresh_token")
	if accessToken2 == "" {
		t.Fatal("expected access_token in second login response")
	}
	if refreshToken2 == "" {
		t.Fatal("expected refresh_token in second login response")
	}

	// Verify old access tokens work before password change.
	doRawRequest(t, http.MethodGet, apiURL(ts, "/camp"),
		nil, http.StatusOK, accessToken1)

	// Change the admin user's password via the super-admin endpoint.
	doRawRequest(t, http.MethodPut,
		apiURL(ts, "/admin/users/"+userID+"/password"),
		map[string]any{"password": "newpassword123"}, http.StatusOK, superToken)

	// Both access tokens should now be rejected (token version mismatch).
	doRawRequest(t, http.MethodGet, apiURL(ts, "/camp"),
		nil, http.StatusUnauthorized, accessToken1)

	doRawRequest(t, http.MethodGet, apiURL(ts, "/camp"),
		nil, http.StatusUnauthorized, accessToken2)

	// Both refresh tokens should also be revoked.
	doRawRequest(t, http.MethodPost, apiURL(ts, "/auth/refresh"),
		map[string]any{"refresh_token": refreshToken1},
		http.StatusUnauthorized, "")

	doRawRequest(t, http.MethodPost, apiURL(ts, "/auth/refresh"),
		map[string]any{"refresh_token": refreshToken2},
		http.StatusUnauthorized, "")

	// Login with the new password should succeed and produce a valid access token.
	newLogin := doRequest(t, http.MethodPost, apiURL(ts, "/auth/login"),
		map[string]any{"username": "tokenuser", "password": "newpassword123"},
		http.StatusOK, "")
	newAccessToken := str(newLogin, "access_token")
	if newAccessToken == "" {
		t.Fatal("expected access_token in new login response")
	}

	doRawRequest(t, http.MethodGet, apiURL(ts, "/camp"),
		nil, http.StatusOK, newAccessToken)
}
