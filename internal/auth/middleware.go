package auth

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const claimsKey = "auth_claims"

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
		c.Next()
	}
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
