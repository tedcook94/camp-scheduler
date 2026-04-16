package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"time"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

// JWTAuthenticator implements Authenticator using locally-signed JWTs for
// access tokens and opaque, DB-backed tokens for refresh token rotation.
type JWTAuthenticator struct {
	pool            *pgxpool.Pool
	queries         *db.Queries
	signingKey      []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

type JWTConfig struct {
	SigningKey      []byte
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

func NewJWTAuthenticator(pool *pgxpool.Pool, queries *db.Queries, cfg JWTConfig) *JWTAuthenticator {
	return &JWTAuthenticator{
		pool:            pool,
		queries:         queries,
		signingKey:      cfg.SigningKey,
		accessTokenTTL:  cfg.AccessTokenTTL,
		refreshTokenTTL: cfg.RefreshTokenTTL,
	}
}

func (a *JWTAuthenticator) Login(ctx context.Context, username, password string) (string, string, error) {
	user, err := a.queries.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrInvalidCredentials
		}
		return "", "", fmt.Errorf("error looking up user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", ErrInvalidCredentials
	}

	userID := api.UUIDToString(user.ID)
	campID := api.UUIDToString(user.CampID)

	accessToken, err := a.generateAccessToken(userID, campID, user.Username, Role(user.Role), user.TokenVersion, "")
	if err != nil {
		return "", "", fmt.Errorf("error generating access token: %w", err)
	}

	refreshToken, err := a.createRefreshToken(ctx, user.ID, user.TokenVersion, pgtype.UUID{})
	if err != nil {
		return "", "", fmt.Errorf("error creating refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (a *JWTAuthenticator) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	hash := hashToken(refreshToken)

	tx, err := a.pool.Begin(ctx)
	if err != nil {
		return "", "", fmt.Errorf("error starting transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	txQueries := a.queries.WithTx(tx)

	stored, err := txQueries.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrInvalidToken
		}
		return "", "", fmt.Errorf("error looking up refresh token: %w", err)
	}

	if time.Now().After(stored.ExpiresAt.Time) {
		return "", "", ErrInvalidToken
	}

	_, err = txQueries.RevokeRefreshToken(ctx, stored.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrInvalidToken
		}
		return "", "", fmt.Errorf("error revoking refresh token: %w", err)
	}

	user, err := txQueries.GetUserByID(ctx, stored.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrInvalidToken
		}
		return "", "", fmt.Errorf("error looking up user for refresh: %w", err)
	}

	if stored.TokenVersion != user.TokenVersion {
		return "", "", ErrInvalidToken
	}

	userID := api.UUIDToString(user.ID)
	campID := api.UUIDToString(user.CampID)

	impersonatedBy := api.UUIDToString(stored.ImpersonatedBy)

	newAccess, err := a.generateAccessToken(userID, campID, user.Username, Role(user.Role), user.TokenVersion, impersonatedBy)
	if err != nil {
		return "", "", fmt.Errorf("error generating access token: %w", err)
	}

	newRefresh, err := a.createRefreshTokenTx(ctx, txQueries, user.ID, user.TokenVersion, stored.ImpersonatedBy)
	if err != nil {
		return "", "", fmt.Errorf("error creating refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", "", fmt.Errorf("error committing refresh transaction: %w", err)
	}

	return newAccess, newRefresh, nil
}

var ErrCannotImpersonate = errors.New("cannot impersonate a super-admin user")
var ErrUserNotFound = errors.New("user not found")

func (a *JWTAuthenticator) ImpersonateUser(ctx context.Context, targetUserID string, impersonatorUserID string) (string, string, error) {
	uid, err := api.ParseUUID(targetUserID)
	if err != nil {
		return "", "", fmt.Errorf("error parsing target user id: %w", err)
	}

	user, err := a.queries.GetUserByID(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", ErrUserNotFound
		}
		return "", "", fmt.Errorf("error looking up target user: %w", err)
	}

	if Role(user.Role) == RoleSuperAdmin {
		return "", "", ErrCannotImpersonate
	}

	userID := api.UUIDToString(user.ID)
	campID := api.UUIDToString(user.CampID)

	accessToken, err := a.generateAccessToken(userID, campID, user.Username, Role(user.Role), user.TokenVersion, impersonatorUserID)
	if err != nil {
		return "", "", fmt.Errorf("error generating access token: %w", err)
	}

	impersonatorUUID, err := api.ParseUUID(impersonatorUserID)
	if err != nil {
		return "", "", fmt.Errorf("error parsing impersonator user id: %w", err)
	}

	refreshToken, err := a.createRefreshToken(ctx, user.ID, user.TokenVersion, impersonatorUUID)
	if err != nil {
		return "", "", fmt.Errorf("error creating refresh token: %w", err)
	}

	return accessToken, refreshToken, nil
}

func (a *JWTAuthenticator) ValidateToken(ctx context.Context, tokenString string) (*Claims, error) {
	claims, err := a.parseToken(tokenString)
	if err != nil {
		return nil, ErrInvalidToken
	}

	tokenType, _ := claims["type"].(string)
	if tokenType != "access" {
		return nil, ErrInvalidToken
	}

	userID, _ := claims.GetSubject()
	campID, _ := claims["camp_id"].(string)
	username, _ := claims["username"].(string)
	role, _ := claims["role"].(string)

	if userID == "" || username == "" || role == "" {
		return nil, ErrInvalidToken
	}

	if campID == "" && Role(role) != RoleSuperAdmin {
		return nil, ErrInvalidToken
	}

	rawVersion, ok := claims["token_version"].(float64)
	if !ok || rawVersion < 1 || rawVersion > math.MaxInt32 || rawVersion != math.Trunc(rawVersion) {
		return nil, ErrInvalidToken
	}
	tokenVersion := int32(rawVersion)

	uid, err := api.ParseUUID(userID)
	if err != nil {
		return nil, ErrInvalidToken
	}

	currentVersion, err := a.queries.GetUserTokenVersion(ctx, uid)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("error checking token version: %w", err)
	}

	if tokenVersion != currentVersion {
		return nil, ErrInvalidToken
	}

	impersonatedBy, _ := claims["impersonated_by"].(string)

	return &Claims{
		UserID:         userID,
		CampID:         campID,
		Username:       username,
		Role:           Role(role),
		TokenVersion:   currentVersion,
		ImpersonatedBy: impersonatedBy,
	}, nil
}

func (a *JWTAuthenticator) generateAccessToken(userID, campID, username string, role Role, tokenVersion int32, impersonatedBy string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":           userID,
		"camp_id":       campID,
		"username":      username,
		"role":          string(role),
		"token_version": tokenVersion,
		"type":          "access",
		"iat":           now.Unix(),
		"exp":           now.Add(a.accessTokenTTL).Unix(),
	}

	if impersonatedBy != "" {
		claims["impersonated_by"] = impersonatedBy
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(a.signingKey)
}

func (a *JWTAuthenticator) createRefreshToken(ctx context.Context, userID pgtype.UUID, tokenVersion int32, impersonatedBy pgtype.UUID) (string, error) {
	return a.createRefreshTokenTx(ctx, a.queries, userID, tokenVersion, impersonatedBy)
}

func (a *JWTAuthenticator) createRefreshTokenTx(ctx context.Context, q *db.Queries, userID pgtype.UUID, tokenVersion int32, impersonatedBy pgtype.UUID) (string, error) {
	raw, err := generateOpaqueToken()
	if err != nil {
		return "", fmt.Errorf("error generating opaque token: %w", err)
	}

	hash := hashToken(raw)

	_, err = q.CreateRefreshToken(ctx, db.CreateRefreshTokenParams{
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(a.refreshTokenTTL),
			Valid: true,
		},
		TokenVersion:   tokenVersion,
		ImpersonatedBy: impersonatedBy,
	})
	if err != nil {
		return "", fmt.Errorf("error storing refresh token: %w", err)
	}

	return raw, nil
}

func (a *JWTAuthenticator) parseToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.signingKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func generateOpaqueToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
