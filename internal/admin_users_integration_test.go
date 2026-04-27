//go:build integration

package internal_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"camp-scheduler/internal/admin"
	"camp-scheduler/internal/auth/authtest"
	"camp-scheduler/internal/config"
	"camp-scheduler/internal/server"
	"camp-scheduler/internal/testutil"

	"github.com/jackc/pgx/v5/pgxpool"
)

// stubAuthServer captures calls made by the OrgSyncer and returns scripted
// responses. It mimics the subset of the auth-server's /internal/* surface
// that the Go side calls today.
type stubAuthServer struct {
	*httptest.Server

	mu        sync.Mutex
	calls     []stubCall
	responses map[string]stubResponse
}

type stubCall struct {
	Method string
	Path   string
	Query  string
	Body   []byte
}

type stubResponse struct {
	Status int
	Body   string
}

func newStubAuthServer(t *testing.T) *stubAuthServer {
	t.Helper()
	s := &stubAuthServer{responses: map[string]stubResponse{}}
	s.Server = httptest.NewServer(http.HandlerFunc(s.handle))
	t.Cleanup(s.Server.Close)
	return s
}

func (s *stubAuthServer) handle(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	s.mu.Lock()
	s.calls = append(s.calls, stubCall{
		Method: r.Method,
		Path:   r.URL.Path,
		Query:  r.URL.RawQuery,
		Body:   body,
	})
	resp, ok := s.responses[r.Method+" "+r.URL.Path]
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	if !ok {
		_, _ = w.Write([]byte(`{}`))
		return
	}
	w.WriteHeader(resp.Status)
	_, _ = w.Write([]byte(resp.Body))
}

func (s *stubAuthServer) setResponse(method, path string, status int, body string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responses[method+" "+path] = stubResponse{Status: status, Body: body}
}

func (s *stubAuthServer) recordedCalls() []stubCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]stubCall, len(s.calls))
	copy(out, s.calls)
	return out
}

// mustSetupServerWithAuthStub wires the test server to a stub auth-server so
// AddMember/RemoveMember/UpdateUser/SetUserRole can be exercised end-to-end
// without standing up the real BetterAuth service.
func mustSetupServerWithAuthStub(t *testing.T) (*testServer, *pgxpool.Pool, *stubAuthServer) {
	t.Helper()

	pool := testutil.MustOpenDB(t)
	testutil.TruncateAll(t, pool)

	stub := newStubAuthServer(t)
	a := authtest.New()
	orgSync := admin.NewOrgSyncer(admin.OrgSyncerConfig{
		BaseURL: stub.URL + "/internal",
		Secret:  "test-secret",
	})

	cfg := config.Config{Server: config.ServerConfig{Mode: "test"}}
	srv := server.NewWithDeps(cfg, pool, nil, server.Deps{
		Authenticator: a,
		OrgSyncer:     orgSync,
	})
	ts := httptest.NewServer(srv)
	t.Cleanup(ts.Close)
	return &testServer{Server: ts, auth: a}, pool, stub
}

func mustInsertCamp(t *testing.T, pool *pgxpool.Pool, name string) string {
	t.Helper()
	var id string
	err := pool.QueryRow(context.Background(),
		`INSERT INTO camps (camp_name) VALUES ($1) RETURNING id`, name).Scan(&id)
	if err != nil {
		t.Fatalf("inserting camp %q: %v", name, err)
	}
	return id
}

func TestAdminUserEndpoints(t *testing.T) {
	t.Run("add_member_happy_path", testAddMemberHappyPath)
	t.Run("add_member_default_role_when_body_empty", testAddMemberDefaultRole)
	t.Run("add_member_malformed_body_returns_400", testAddMemberMalformedBody)
	t.Run("add_member_unknown_camp_returns_404", testAddMemberUnknownCamp)
	t.Run("remove_member_happy_path", testRemoveMemberHappyPath)
	t.Run("update_user_partial_patch", testUpdateUserPartialPatch)
	t.Run("update_user_upstream_409_passthrough", testUpdateUserUpstreamConflict)
	t.Run("set_user_role_upstream_5xx_passthrough", testSetUserRoleUpstream5xx)
	t.Run("admin_routes_reject_non_super_admin", testAdminUserRoutesRejectAdmin)
}

func TestAdminCampOrgSync(t *testing.T) {
	t.Run("create_camp_orgsync_failure_rolls_back_camp", testCreateCampOrgSyncFailureRollsBack)
	t.Run("delete_camp_orgsync_failure_returns_error", testDeleteCampOrgSyncFailureReturnsError)
}

func testAddMemberHappyPath(t *testing.T) {
	ts, pool, stub := mustSetupServerWithAuthStub(t)
	token := mustLoginSuperAdmin(t, ts, pool)

	campID := mustInsertCamp(t, pool, "Camp AddMember")
	userID := "11111111-1111-1111-1111-111111111111"

	url := apiURL(ts, "/admin/users/"+userID+"/camps/"+campID)
	doRawRequest(t, http.MethodPost, url, map[string]any{"role": "owner"}, http.StatusOK, token)

	calls := stub.recordedCalls()
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 auth-server call, got %d: %+v", len(calls), calls)
	}
	if calls[0].Method != http.MethodPost || calls[0].Path != "/internal/members" {
		t.Fatalf("expected POST /internal/members, got %s %s", calls[0].Method, calls[0].Path)
	}
	var payload map[string]any
	if err := json.Unmarshal(calls[0].Body, &payload); err != nil {
		t.Fatalf("decoding upstream payload: %v", err)
	}
	if payload["userId"] != userID {
		t.Fatalf("expected userId %q in payload, got %v", userID, payload["userId"])
	}
	if payload["organizationId"] != campID {
		t.Fatalf("expected organizationId %q in payload, got %v", campID, payload["organizationId"])
	}
	if payload["role"] != "owner" {
		t.Fatalf("expected role %q in payload, got %v", "owner", payload["role"])
	}
}

func testAddMemberDefaultRole(t *testing.T) {
	ts, pool, stub := mustSetupServerWithAuthStub(t)
	token := mustLoginSuperAdmin(t, ts, pool)

	campID := mustInsertCamp(t, pool, "Camp DefaultRole")
	userID := "22222222-2222-2222-2222-222222222222"

	url := apiURL(ts, "/admin/users/"+userID+"/camps/"+campID)
	// nil body — controller should default the role to "admin".
	doRawRequest(t, http.MethodPost, url, nil, http.StatusOK, token)

	calls := stub.recordedCalls()
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 auth-server call, got %d", len(calls))
	}
	var payload map[string]any
	if err := json.Unmarshal(calls[0].Body, &payload); err != nil {
		t.Fatalf("decoding upstream payload: %v", err)
	}
	if payload["role"] != "admin" {
		t.Fatalf("expected default role %q, got %v", "admin", payload["role"])
	}
}

func testAddMemberUnknownCamp(t *testing.T) {
	ts, pool, stub := mustSetupServerWithAuthStub(t)
	token := mustLoginSuperAdmin(t, ts, pool)

	userID := "33333333-3333-3333-3333-333333333333"
	missingCampID := "99999999-9999-9999-9999-999999999999"
	url := apiURL(ts, "/admin/users/"+userID+"/camps/"+missingCampID)
	doRawRequest(t, http.MethodPost, url, map[string]any{"role": "admin"}, http.StatusNotFound, token)

	if got := len(stub.recordedCalls()); got != 0 {
		t.Fatalf("expected no auth-server call when camp is missing, got %d", got)
	}
}

func testRemoveMemberHappyPath(t *testing.T) {
	ts, pool, stub := mustSetupServerWithAuthStub(t)
	token := mustLoginSuperAdmin(t, ts, pool)

	campID := mustInsertCamp(t, pool, "Camp RemoveMember")
	userID := "44444444-4444-4444-4444-444444444444"

	url := apiURL(ts, "/admin/users/"+userID+"/camps/"+campID)
	doRawRequest(t, http.MethodDelete, url, nil, http.StatusOK, token)

	calls := stub.recordedCalls()
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 auth-server call, got %d", len(calls))
	}
	if calls[0].Method != http.MethodDelete || calls[0].Path != "/internal/members" {
		t.Fatalf("expected DELETE /internal/members, got %s %s", calls[0].Method, calls[0].Path)
	}
	if !strings.Contains(calls[0].Query, "userId="+userID) {
		t.Fatalf("expected userId in query, got %q", calls[0].Query)
	}
	if !strings.Contains(calls[0].Query, "organizationId="+campID) {
		t.Fatalf("expected organizationId in query, got %q", calls[0].Query)
	}
}

func testUpdateUserPartialPatch(t *testing.T) {
	ts, pool, stub := mustSetupServerWithAuthStub(t)
	token := mustLoginSuperAdmin(t, ts, pool)

	userID := "55555555-5555-5555-5555-555555555555"
	url := apiURL(ts, "/admin/users/"+userID)
	// Send the snake_case wire shape the controller exposes; expect the
	// upstream auth-server call to receive the camelCase shape BetterAuth
	// expects. This pins the controller→orgsync field translation.
	body := map[string]any{"first_name": "Updated", "email": "new@example.com"}
	doRawRequest(t, http.MethodPut, url, body, http.StatusOK, token)

	calls := stub.recordedCalls()
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 auth-server call, got %d", len(calls))
	}
	if calls[0].Method != http.MethodPut || calls[0].Path != "/internal/users/"+userID {
		t.Fatalf("expected PUT /internal/users/%s, got %s %s", userID, calls[0].Method, calls[0].Path)
	}
	var payload map[string]any
	if err := json.Unmarshal(calls[0].Body, &payload); err != nil {
		t.Fatalf("decoding upstream payload: %v", err)
	}
	if payload["email"] != "new@example.com" {
		t.Fatalf("expected email in upstream payload, got %v", payload["email"])
	}
	if payload["firstName"] != "Updated" {
		t.Fatalf("expected firstName=%q in upstream payload, got %v", "Updated", payload["firstName"])
	}
	// snake_case key from the inbound request must NOT leak through.
	if _, present := payload["first_name"]; present {
		t.Fatalf("expected first_name to be translated, got %v", payload["first_name"])
	}
	if _, present := payload["lastName"]; present {
		t.Fatalf("expected lastName to be omitted, got %v", payload["lastName"])
	}
}

func testUpdateUserUpstreamConflict(t *testing.T) {
	ts, pool, stub := mustSetupServerWithAuthStub(t)
	token := mustLoginSuperAdmin(t, ts, pool)

	userID := "66666666-6666-6666-6666-666666666666"
	stub.setResponse(http.MethodPut, "/internal/users/"+userID, http.StatusBadRequest,
		`{"error":"User with this email already exists.","code":"USER_ALREADY_EXISTS_USE_ANOTHER_EMAIL"}`)

	url := apiURL(ts, "/admin/users/"+userID)
	respBody := doRawRequest(t, http.MethodPut, url,
		map[string]any{"email": "dup@example.com"}, http.StatusBadRequest, token)

	var resp map[string]any
	if err := json.Unmarshal(respBody, &resp); err != nil {
		t.Fatalf("decoding controller response: %v", err)
	}
	if resp["error"] != "User with this email already exists." {
		t.Fatalf("expected upstream error message to pass through, got %v", resp["error"])
	}
	if resp["code"] != "USER_ALREADY_EXISTS_USE_ANOTHER_EMAIL" {
		t.Fatalf("expected upstream error code to pass through, got %v", resp["code"])
	}
}

func testSetUserRoleUpstream5xx(t *testing.T) {
	ts, pool, stub := mustSetupServerWithAuthStub(t)
	token := mustLoginSuperAdmin(t, ts, pool)

	userID := "77777777-7777-7777-7777-777777777777"
	stub.setResponse(http.MethodPost, "/internal/users/"+userID+"/role", http.StatusServiceUnavailable,
		`{"error":"db unreachable"}`)

	url := apiURL(ts, "/admin/users/"+userID+"/role")
	respBody := doRawRequest(t, http.MethodPost, url,
		map[string]any{"role": "super_admin"}, http.StatusServiceUnavailable, token)

	var resp map[string]any
	if err := json.Unmarshal(respBody, &resp); err != nil {
		t.Fatalf("decoding controller response: %v", err)
	}
	if resp["error"] != "db unreachable" {
		t.Fatalf("expected upstream 5xx error to pass through, got %v", resp["error"])
	}
}

func testAdminUserRoutesRejectAdmin(t *testing.T) {
	ts, pool, _ := mustSetupServerWithAuthStub(t)

	campID := mustInsertCamp(t, pool, "Camp RejectAdmin")
	adminToken := mustLogin(t, ts, pool, campID)
	userID := "88888888-8888-8888-8888-888888888888"

	doRawRequest(t, http.MethodPost,
		apiURL(ts, "/admin/users/"+userID+"/camps/"+campID),
		map[string]any{"role": "admin"}, http.StatusForbidden, adminToken)
	doRawRequest(t, http.MethodDelete,
		apiURL(ts, "/admin/users/"+userID+"/camps/"+campID),
		nil, http.StatusForbidden, adminToken)
	doRawRequest(t, http.MethodPut,
		apiURL(ts, "/admin/users/"+userID),
		map[string]any{"firstName": "x"}, http.StatusForbidden, adminToken)
	doRawRequest(t, http.MethodPost,
		apiURL(ts, "/admin/users/"+userID+"/role"),
		map[string]any{"role": "user"}, http.StatusForbidden, adminToken)

	// Unauthenticated must produce 401.
	doRawRequest(t, http.MethodPost,
		apiURL(ts, "/admin/users/"+userID+"/role"),
		map[string]any{"role": "user"}, http.StatusUnauthorized, "")
}

// testAddMemberMalformedBody verifies that the AddMember handler distinguishes
// "no body" (default-role path) from a malformed body. Sending raw bytes that
// aren't valid JSON must produce a 400, not a successful default-role call.
func testAddMemberMalformedBody(t *testing.T) {
	ts, pool, stub := mustSetupServerWithAuthStub(t)
	token := mustLoginSuperAdmin(t, ts, pool)

	campID := mustInsertCamp(t, pool, "Camp MalformedBody")
	userID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	url := apiURL(ts, "/admin/users/"+userID+"/camps/"+campID)

	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader("{"))
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("executing request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", resp.StatusCode)
	}
	if got := len(stub.recordedCalls()); got != 0 {
		t.Fatalf("expected no auth-server call when body is malformed, got %d", got)
	}
}

// testCreateCampOrgSyncFailureRollsBack verifies the Service.Create rollback
// contract: when the auth-server returns an error on POST /internal/organizations,
// the just-inserted camp row must be removed so the two systems stay aligned.
func testCreateCampOrgSyncFailureRollsBack(t *testing.T) {
	ts, _, stub := mustSetupServerWithAuthStub(t)
	token := mustLoginSuperAdmin(t, ts, nil)

	stub.setResponse(http.MethodPost, "/internal/organizations", http.StatusInternalServerError,
		`{"error":"db unreachable"}`)

	doRawRequest(t, http.MethodPost, apiURL(ts, "/admin/camps"),
		map[string]any{"name": "Camp Rollback"}, http.StatusInternalServerError, token)

	calls := stub.recordedCalls()
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 auth-server call, got %d", len(calls))
	}

	// Camp row must not have survived the failed sync.
	camps := mustGetList(t, apiURL(ts, "/admin/camps"), token)
	if len(camps) != 0 {
		t.Fatalf("expected 0 camps after rollback, got %d", len(camps))
	}
}

// testDeleteCampOrgSyncFailureReturnsError verifies that if the auth-server
// fails to delete the organization after the camp row has already been
// removed, the controller surfaces the failure to the caller rather than
// silently leaving the auth-server organization orphaned.
func testDeleteCampOrgSyncFailureReturnsError(t *testing.T) {
	ts, pool, stub := mustSetupServerWithAuthStub(t)
	token := mustLoginSuperAdmin(t, ts, nil)

	// Insert directly so the create path doesn't touch the stub; we only want
	// to exercise the delete path's failure mode.
	campID := mustInsertCamp(t, pool, "Camp DeleteFail")

	stub.setResponse(http.MethodDelete, "/internal/organizations/"+campID,
		http.StatusInternalServerError, `{"error":"db unreachable"}`)

	doRawRequest(t, http.MethodDelete, apiURL(ts, "/admin/camps/"+campID),
		nil, http.StatusInternalServerError, token)

	calls := stub.recordedCalls()
	if len(calls) != 1 {
		t.Fatalf("expected exactly 1 auth-server call, got %d", len(calls))
	}

	// The camp row was removed before the sync attempt; the failure surfaces
	// to the caller but PG state matches the failed sync intent.
	doRawRequest(t, http.MethodGet, apiURL(ts, "/admin/camps/"+campID),
		nil, http.StatusNotFound, token)
}
