package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// RevocationChecker enforces the auth-server's per-user `revokedAfter`
// timestamp on every authenticated request. Bumping that column on the
// auth-server (e.g. when a user is removed from a camp) causes JWTs minted
// before the bump to be rejected within one cache TTL — the JWT analog of
// the old hand-rolled token_version pattern.
//
// Failure mode is fail-open with stale cache: if the auth-server is
// unreachable but we have a previous answer for the user, we keep using it.
// This trades a small additional revocation window for Go-side availability
// during auth-server restarts.
type RevocationChecker struct {
	baseURL  string // auth-server internal API base, e.g. http://localhost:9101/internal
	secret   string
	cacheTTL time.Duration
	client   *http.Client

	mu    sync.RWMutex
	cache map[string]revocationEntry
}

type revocationEntry struct {
	revokedAfter time.Time
	fetchedAt    time.Time
	notFound     bool
}

type RevocationConfig struct {
	BaseURL  string
	Secret   string
	CacheTTL time.Duration
	Timeout  time.Duration
}

func NewRevocationChecker(cfg RevocationConfig) *RevocationChecker {
	ttl := cfg.CacheTTL
	if ttl == 0 {
		ttl = 30 * time.Second
	}
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &RevocationChecker{
		baseURL:  cfg.BaseURL,
		secret:   cfg.Secret,
		cacheTTL: ttl,
		client:   &http.Client{Timeout: timeout},
		cache:    map[string]revocationEntry{},
	}
}

// Check rejects the JWT if the user has been revoked since `issuedAt`.
// A nil error means the JWT is still valid w.r.t. revocation.
func (r *RevocationChecker) Check(ctx context.Context, userID string, issuedAt time.Time) error {
	if userID == "" {
		return ErrInvalidToken
	}

	entry, ok := r.lookup(userID)
	if !ok || time.Since(entry.fetchedAt) > r.cacheTTL {
		fresh, err := r.fetch(ctx, userID)
		if err != nil {
			if !ok {
				// No prior entry to fall back to — fail closed.
				return fmt.Errorf("error checking revocation for %s: %w", userID, err)
			}
			// Fail-open: keep using the stale entry. Log so operators notice
			// auth-server flakes.
			slog.
				With("user_id", userID).
				With("error", err).
				Warn("revocation check failed; using stale cache")
		} else {
			r.store(userID, fresh)
			entry = fresh
		}
	}

	if entry.notFound {
		return ErrInvalidToken
	}
	// Revocation is inclusive: a JWT issued at exactly the revoke timestamp
	// is rejected. The next mint will be strictly later and will pass.
	if !issuedAt.After(entry.revokedAfter) {
		return ErrInvalidToken
	}
	return nil
}

func (r *RevocationChecker) lookup(userID string) (revocationEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	e, ok := r.cache[userID]
	return e, ok
}

func (r *RevocationChecker) store(userID string, entry revocationEntry) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache[userID] = entry
}

func (r *RevocationChecker) fetch(ctx context.Context, userID string) (revocationEntry, error) {
	url := fmt.Sprintf("%s/users/%s/revoked-after", r.baseURL, userID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return revocationEntry{}, fmt.Errorf("error building revocation request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.secret)

	resp, err := r.client.Do(req)
	if err != nil {
		return revocationEntry{}, fmt.Errorf("error calling revocation endpoint: %w", err)
	}
	defer resp.Body.Close()

	now := time.Now()
	if resp.StatusCode == http.StatusNotFound {
		return revocationEntry{notFound: true, fetchedAt: now}, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return revocationEntry{}, fmt.Errorf("revocation endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		RevokedAfter *time.Time `json:"revokedAfter"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return revocationEntry{}, fmt.Errorf("error decoding revocation response: %w", err)
	}
	var ts time.Time
	if payload.RevokedAfter != nil {
		ts = *payload.RevokedAfter
	}
	return revocationEntry{revokedAfter: ts, fetchedAt: now}, nil
}
