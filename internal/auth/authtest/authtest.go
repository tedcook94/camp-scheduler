package authtest

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"time"

	"camp-scheduler/internal/auth"

	"github.com/golang-jwt/jwt/v5"
)

// Authenticator is an in-memory test-only Authenticator. It mints and
// validates EdDSA-signed JWTs with the same claim shape as the production
// auth-server, removing the need for a running auth-server in tests.
type Authenticator struct {
	priv ed25519.PrivateKey
	pub  ed25519.PublicKey
}

func New() *Authenticator {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic("authtest: generating ed25519 keypair: " + err.Error())
	}
	return &Authenticator{priv: priv, pub: pub}
}

// Issue mints a token for the given claims, with a 1-hour lifetime.
func (a *Authenticator) Issue(c auth.Claims) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":             c.UserID,
		"username":        c.Username,
		"email":           c.Email,
		"role":            string(c.Role),
		"camp_id":         c.CampID,
		"org_role":        c.OrgRole,
		"impersonated_by": c.ImpersonatedBy,
		"first_name":      c.FirstName,
		"last_name":       c.LastName,
		"iat":             now.Unix(),
		"exp":             now.Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	token.Header["kid"] = "test"
	return token.SignedString(a.priv)
}

func (a *Authenticator) ValidateToken(_ context.Context, tokenString string) (*auth.Claims, error) {
	parser := jwt.NewParser(jwt.WithValidMethods([]string{"EdDSA"}))
	token, err := parser.Parse(tokenString, func(_ *jwt.Token) (any, error) {
		return a.pub, nil
	})
	if err != nil {
		return nil, errors.Join(auth.ErrInvalidToken, err)
	}
	m, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, auth.ErrInvalidToken
	}
	sub, _ := m.GetSubject()
	role, _ := m["role"].(string)
	if role == "" {
		role = string(auth.RoleUser)
	}
	username, _ := m["username"].(string)
	email, _ := m["email"].(string)
	campID, _ := m["camp_id"].(string)
	orgRole, _ := m["org_role"].(string)
	impersonatedBy, _ := m["impersonated_by"].(string)
	firstName, _ := m["first_name"].(string)
	lastName, _ := m["last_name"].(string)
	return &auth.Claims{
		UserID:         sub,
		CampID:         campID,
		Username:       username,
		Email:          email,
		Role:           auth.Role(role),
		OrgRole:        orgRole,
		ImpersonatedBy: impersonatedBy,
		FirstName:      firstName,
		LastName:       lastName,
	}, nil
}
