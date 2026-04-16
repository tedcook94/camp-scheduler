package auth

import "context"

type Role string

const (
	RoleAdmin      Role = "admin"
	RoleSuperAdmin Role = "super_admin"
)

type Claims struct {
	UserID         string
	CampID         string
	Username       string
	Role           Role
	TokenVersion   int32
	ImpersonatedBy string
}

// Authenticator abstracts authentication so the implementation (local JWT,
// external provider, etc.) can be swapped without changing application code.
type Authenticator interface {
	Login(ctx context.Context, username, password string) (accessToken, refreshToken string, err error)
	Refresh(ctx context.Context, refreshToken string) (accessToken, newRefreshToken string, err error)
	ValidateToken(ctx context.Context, tokenString string) (*Claims, error)
	ImpersonateUser(ctx context.Context, targetUserID string, impersonatorUserID string) (accessToken, refreshToken string, err error)
}
