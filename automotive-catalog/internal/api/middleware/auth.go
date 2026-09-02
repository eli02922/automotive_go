package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	claimsKey = "claims"
)

type Claims struct {
	UserID string
	Email  string
	Roles  []string
}

// BearerAuth validates a JWT bearer token (stub — swap for real JWT library).
func BearerAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing authorization header"})
			return
		}

		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}

		claims, err := parseJWT(parts[1], jwtSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		c.Set(claimsKey, claims)
		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		v, exists := c.Get(claimsKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		claims := v.(*Claims)
		for _, required := range roles {
			for _, r := range claims.Roles {
				if r == required {
					c.Next()
					return
				}
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
	}
}

func GetClaims(c *gin.Context) *Claims {
	v, _ := c.Get(claimsKey)
	if v == nil {
		return &Claims{}
	}
	return v.(*Claims)
}

// parseJWT is a stub; replace with github.com/golang-jwt/jwt/v5 in production.
func parseJWT(token, secret string) (*Claims, error) {
	// Validate expiry, signature, etc. via JWT library.
	_ = time.Now()
	return &Claims{UserID: "system", Email: "system@example.com", Roles: []string{"admin"}}, nil
}
