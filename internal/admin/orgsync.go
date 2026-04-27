package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// OrgSyncer keeps the auth-server's `organization` table in sync with
// the Go-owned `camps` table. Each camp has a 1:1 organization with the
// same id; this client calls the auth-server's internal admin API
// (protected by the shared bearer secret) on camp create/delete.
type OrgSyncer struct {
	baseURL string
	secret  string
	client  *http.Client
}

type OrgSyncerConfig struct {
	BaseURL string // e.g. http://localhost:9101/internal
	Secret  string
	Timeout time.Duration
}

func NewOrgSyncer(cfg OrgSyncerConfig) *OrgSyncer {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &OrgSyncer{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"),
		secret:  cfg.Secret,
		client:  &http.Client{Timeout: timeout},
	}
}

type orgPayload struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// CreateOrg upserts an organization in the auth-server with id == campID.
func (s *OrgSyncer) CreateOrg(ctx context.Context, campID, name, slug string) error {
	body, err := json.Marshal(orgPayload{ID: campID, Name: name, Slug: slug})
	if err != nil {
		return fmt.Errorf("error marshaling org payload: %w", err)
	}
	return s.do(ctx, http.MethodPost, "/organizations", body)
}

// DeleteOrg removes the organization with the given id from the auth-server.
func (s *OrgSyncer) DeleteOrg(ctx context.Context, campID string) error {
	return s.do(ctx, http.MethodDelete, "/organizations/"+campID, nil)
}

type memberPayload struct {
	UserID         string `json:"userId"`
	OrganizationID string `json:"organizationId"`
	Role           string `json:"role,omitempty"`
}

// AddMember adds a user to a camp's organization on the auth-server.
// role defaults to "admin" when empty.
func (s *OrgSyncer) AddMember(ctx context.Context, userID, campID, role string) error {
	body, err := json.Marshal(memberPayload{UserID: userID, OrganizationID: campID, Role: role})
	if err != nil {
		return fmt.Errorf("error marshaling member payload: %w", err)
	}
	return s.do(ctx, http.MethodPost, "/members", body)
}

// RemoveMember removes a user from a camp's organization on the auth-server.
func (s *OrgSyncer) RemoveMember(ctx context.Context, userID, campID string) error {
	path := fmt.Sprintf("/members?userId=%s&organizationId=%s",
		url.QueryEscape(userID), url.QueryEscape(campID))
	return s.do(ctx, http.MethodDelete, path, nil)
}

type CreateUserRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	Role      string `json:"role,omitempty"`
}

type CreateUserResponse struct {
	UserID string `json:"userId"`
}

// CreateUser provisions a user on the auth-server (used by seed scripts).
func (s *OrgSyncer) CreateUser(ctx context.Context, req CreateUserRequest) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("error marshaling user payload: %w", err)
	}
	resp, err := s.doRead(ctx, http.MethodPost, "/users", body)
	if err != nil {
		return "", err
	}
	var out CreateUserResponse
	if err := json.Unmarshal(resp, &out); err != nil {
		return "", fmt.Errorf("error decoding create user response: %w", err)
	}
	return out.UserID, nil
}

// DeleteUserByEmail removes a user from the auth-server by email
// (used by seed scripts to make seeding idempotent).
func (s *OrgSyncer) DeleteUserByEmail(ctx context.Context, email string) error {
	path := "/users?email=" + url.QueryEscape(email)
	return s.do(ctx, http.MethodDelete, path, nil)
}

type UpdateUserRequest struct {
	FirstName *string `json:"firstName,omitempty"`
	LastName  *string `json:"lastName,omitempty"`
	Email     *string `json:"email,omitempty"`
}

// UpdateUser patches a user's mutable profile fields on the auth-server.
// Only non-nil pointer fields are sent, letting the auth-server distinguish
// "not provided" from "set to empty string".
func (s *OrgSyncer) UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshaling user update payload: %w", err)
	}
	return s.do(ctx, http.MethodPut, "/users/"+url.PathEscape(userID), body)
}

// SetUserRole updates a user's global role on the auth-server.
func (s *OrgSyncer) SetUserRole(ctx context.Context, userID, role string) error {
	body, err := json.Marshal(struct {
		Role string `json:"role"`
	}{Role: role})
	if err != nil {
		return fmt.Errorf("error marshaling role payload: %w", err)
	}
	return s.do(ctx, http.MethodPost, "/users/"+url.PathEscape(userID)+"/role", body)
}

func (s *OrgSyncer) doRead(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("error building auth-server request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.secret)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error calling auth-server %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return respBody, nil
	}
	return nil, formatAuthServerError(method, path, resp.StatusCode, respBody)
}

func (s *OrgSyncer) do(ctx context.Context, method, path string, body []byte) error {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, s.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("error building auth-server request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.secret)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("error calling auth-server %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return formatAuthServerError(method, path, resp.StatusCode, respBody)
}

// AuthServerError carries the upstream auth-server's HTTP status, structured
// error payload, and request context so controllers can map upstream failures
// to appropriate HTTP responses (e.g. 404 → 404, 400 USER_ALREADY_EXISTS → 400)
// instead of collapsing every non-2xx into a generic 500.
type AuthServerError struct {
	Status  int
	Message string
	Code    string
	Method  string
	Path    string
	RawBody []byte
}

func (e *AuthServerError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("auth-server: %s", e.Message)
	}
	return fmt.Sprintf("auth-server %s %s returned %d: %s", e.Method, e.Path, e.Status, string(e.RawBody))
}

// formatAuthServerError surfaces the auth-server's structured `{error, code}`
// payload as a clean message when present, falling back to the raw body so
// debugging unstructured failures (network errors, panic dumps) stays useful.
func formatAuthServerError(method, path string, status int, body []byte) error {
	var payload struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	if err := json.Unmarshal(body, &payload); err == nil && payload.Error != "" {
		return &AuthServerError{
			Status:  status,
			Message: payload.Error,
			Code:    payload.Code,
			Method:  method,
			Path:    path,
			RawBody: body,
		}
	}
	return &AuthServerError{
		Status:  status,
		Method:  method,
		Path:    path,
		RawBody: body,
	}
}
