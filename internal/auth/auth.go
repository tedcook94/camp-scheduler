package auth

import "context"

type Role string

const RoleAdmin Role = "admin"

type Claims struct {
	UserID   string
	CampID   string
	Username string
	Role     Role
}

// Authenticator abstracts authentication so the implementation (local JWT,
// external provider, etc.) can be swapped without changing application code.
type Authenticator interface {
	Login(ctx context.Context, username, password string) (accessToken, refreshToken string, err error)
	Refresh(ctx context.Context, refreshToken string) (accessToken, newRefreshToken string, err error)
	ValidateToken(tokenString string) (*Claims, error)
}
