package auth

import (
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

		claims, err := auth.ValidateToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
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
