package auth

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid or expired token")

// JWKSAuthenticator validates JWTs issued by the BetterAuth auth-server using
// EdDSA (Ed25519) signatures. Public keys are fetched from the JWKS endpoint
// and cached; on a kid miss the cache is refreshed automatically.
type JWKSAuthenticator struct {
	jwksURL string
	client  *http.Client

	mu        sync.RWMutex
	keys      map[string]ed25519.PublicKey
	fetchedAt time.Time
}

type JWKSConfig struct {
	JWKSURL string
	Timeout time.Duration
}

func NewJWKSAuthenticator(cfg JWKSConfig) *JWKSAuthenticator {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &JWKSAuthenticator{
		jwksURL: cfg.JWKSURL,
		client:  &http.Client{Timeout: timeout},
		keys:    map[string]ed25519.PublicKey{},
	}
}

func (a *JWKSAuthenticator) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{"EdDSA"}))
	token, err := parser.Parse(tokenString, a.keyFunc(ctx))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	c, err := claimsFromMap(claims)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	return c, nil
}

func (a *JWKSAuthenticator) keyFunc(ctx context.Context) jwt.Keyfunc {
	return func(token *jwt.Token) (any, error) {
		kid, _ := token.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("token missing kid header")
		}

		if key, ok := a.lookupKey(kid); ok {
			return key, nil
		}

		// Cache miss: refresh and try again. BetterAuth supports key rotation
		// so a kid we haven't seen is expected, not an error.
		if err := a.refresh(ctx); err != nil {
			return nil, fmt.Errorf("error refreshing JWKS: %w", err)
		}
		if key, ok := a.lookupKey(kid); ok {
			return key, nil
		}
		return nil, fmt.Errorf("no key found for kid %s", kid)
	}
}

func (a *JWKSAuthenticator) lookupKey(kid string) (ed25519.PublicKey, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	key, ok := a.keys[kid]
	return key, ok
}

// Warmup populates the JWKS cache; call once at startup so the first request
// doesn't pay the fetch cost. Failures are logged but not fatal — the cache
// will be filled on first ValidateToken.
func (a *JWKSAuthenticator) Warmup(ctx context.Context) {
	if err := a.refresh(ctx); err != nil {
		slog.With("error", err).Warn("error warming up JWKS cache")
	}
}

func (a *JWKSAuthenticator) refresh(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Coalesce concurrent refreshes: if another goroutine fetched in the
	// last 2 seconds, skip.
	if time.Since(a.fetchedAt) < 2*time.Second && len(a.keys) > 0 {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("error building JWKS request: %w", err)
	}
	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("error fetching JWKS: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("JWKS endpoint returned %d: %s", resp.StatusCode, string(body))
	}

	var doc struct {
		Keys []struct {
			Kty string `json:"kty"`
			Crv string `json:"crv"`
			Kid string `json:"kid"`
			Alg string `json:"alg"`
			X   string `json:"x"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return fmt.Errorf("error decoding JWKS: %w", err)
	}

	keys := map[string]ed25519.PublicKey{}
	for _, k := range doc.Keys {
		if k.Kty != "OKP" || k.Crv != "Ed25519" {
			continue
		}
		raw, err := base64.RawURLEncoding.DecodeString(k.X)
		if err != nil {
			slog.With("kid", k.Kid).With("error", err).Warn("error decoding JWKS key")
			continue
		}
		if len(raw) != ed25519.PublicKeySize {
			slog.With("kid", k.Kid).With("size", len(raw)).Warn("unexpected Ed25519 key size in JWKS")
			continue
		}
		keys[k.Kid] = ed25519.PublicKey(raw)
	}
	a.keys = keys
	a.fetchedAt = time.Now()
	return nil
}

func claimsFromMap(m jwt.MapClaims) (*Claims, error) {
	sub, _ := m.GetSubject()
	if sub == "" {
		return nil, errors.New("missing sub claim")
	}
	role, _ := m["role"].(string)
	if role == "" {
		role = string(RoleUser)
	}
	username, _ := m["username"].(string)
	email, _ := m["email"].(string)
	campID, _ := m["camp_id"].(string)
	orgRole, _ := m["org_role"].(string)
	impersonatedBy, _ := m["impersonated_by"].(string)
	firstName, _ := m["first_name"].(string)
	lastName, _ := m["last_name"].(string)

	return &Claims{
		UserID:         sub,
		CampID:         campID,
		Username:       username,
		Email:          email,
		Role:           Role(role),
		OrgRole:        orgRole,
		ImpersonatedBy: impersonatedBy,
		FirstName:      firstName,
		LastName:       lastName,
	}, nil
}
