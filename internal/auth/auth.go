package auth

import "context"

type Role string

const (
	RoleUser       Role = "user"
	RoleSuperAdmin Role = "super_admin"
)

// Claims is the request-scoped view of an authenticated caller. Fields are
// derived from the JWT issued by the auth-server (BetterAuth + jwt plugin).
//
// CampID corresponds to the active organization (BetterAuth org id == camps.id).
// OrgRole is the caller's role within that organization (owner/admin/member).
// Role is the global role on the user (user/super_admin).
type Claims struct {
	UserID         string
	CampID         string
	Username       string
	Email          string
	Role           Role
	OrgRole        string
	ImpersonatedBy string
	FirstName      string
	LastName       string
}

// Authenticator validates a bearer access token and returns the associated
// claims. The Go server is a token consumer only: token issuance, login,
// refresh, and impersonation all happen in the auth-server.
type Authenticator interface {
	ValidateToken(ctx context.Context, tokenString string) (*Claims, error)
}
