package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	return fmt.Errorf("auth-server %s %s returned %d: %s", method, path, resp.StatusCode, string(respBody))
}
