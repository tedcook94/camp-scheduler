package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"camp-scheduler/internal/api"
	"camp-scheduler/internal/db"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

const claimsKey = "auth_claims"
const loggerKey = "auth_logger"

func Middleware(auth Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		token, found := strings.CutPrefix(header, "Bearer ")
		if !found || token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			return
		}

		claims, err := auth.ValidateToken(c.Request.Context(), token)
		if err != nil {
			if errors.Is(err, ErrInvalidToken) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
				return
			}
			slog.With("error", err).Error("error validating token")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		c.Set(claimsKey, claims)

		log := slog.With("user_id", claims.UserID)
		if claims.CampID != "" {
			log = log.With("camp_id", claims.CampID)
		}
		if claims.ImpersonatedBy != "" {
			log = log.With("impersonated_by", claims.ImpersonatedBy)
		}
		c.Set(loggerKey, log)

		c.Next()
	}
}

// Logger returns the request-scoped logger that the auth middleware
// pre-configured with user context fields (user_id, camp_id, and
// impersonated_by when present). Falls back to slog.Default() for
// unauthenticated routes.
func Logger(c *gin.Context) *slog.Logger {
	v, exists := c.Get(loggerKey)
	if !exists {
		return slog.Default()
	}
	log, _ := v.(*slog.Logger)
	if log == nil {
		return slog.Default()
	}
	return log
}

func GetClaims(c *gin.Context) *Claims {
	v, exists := c.Get(claimsKey)
	if !exists {
		return nil
	}
	claims, _ := v.(*Claims)
	return claims
}

// GetCampID returns the camp ID from the authenticated user's JWT claims.
// If claims are missing (middleware misconfiguration), it aborts with 401
// so the failure is visible rather than silently passing an empty camp ID.
func GetCampID(c *gin.Context) string {
	claims := GetClaims(c)
	if claims == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authentication claims"})
		return ""
	}
	return claims.CampID
}

// RequireCampScope rejects requests from users without a camp association
// (i.e. super-admins). Camp-scoped routes should use this middleware to
// prevent super-admin tokens from reaching handlers that expect a camp ID.
func RequireCampScope() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetClaims(c)
		if claims == nil || claims.CampID == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "camp-scoped access required"})
			return
		}
		c.Next()
	}
}

func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := GetClaims(c)
		if claims == nil || claims.Role != RoleSuperAdmin {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "super-admin access required"})
			return
		}
		c.Next()
	}
}

// RequireCampEnabled rejects mutating requests (anything other than GET/HEAD)
// when the authenticated user's camp is disabled. Read-only access is still
// allowed so users can view their data.
func RequireCampEnabled(queries *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodHead {
			c.Next()
			return
		}

		claims := GetClaims(c)
		if claims == nil || claims.CampID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "camp-scoped authentication required"})
			return
		}

		campID, err := api.ParseUUID(claims.CampID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid camp ID"})
			return
		}

		camp, err := queries.GetCamp(c.Request.Context(), campID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "camp not found"})
				return
			}
			Logger(c).
				With("camp_id", claims.CampID).
				With("error", err).
				Error("error checking camp enabled status")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
			return
		}

		if !camp.CampEnabled {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "camp is disabled — changes are not allowed"})
			return
		}

		c.Next()
	}
}
