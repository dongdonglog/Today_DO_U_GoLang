package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const principalKey = "knowledge_principal"

type Claims struct {
	TenantID string   `json:"tenant_id"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

type Principal struct {
	TenantID string
	Roles    map[string]struct{}
}

func IssueToken(secret []byte, issuer, tenantID string, roles []string, ttl time.Duration) (string, error) {
	if len(secret) < 32 || strings.TrimSpace(issuer) == "" || strings.TrimSpace(tenantID) == "" {
		return "", errors.New("secret, issuer and tenant are required")
	}
	if ttl <= 0 || ttl > 24*time.Hour {
		return "", errors.New("token lifetime must be between 0 and 24 hours")
	}
	now := time.Now()
	claims := Claims{
		TenantID: tenantID,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   tenantID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

func Middleware(secret []byte, issuer string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := strings.TrimSpace(c.GetHeader("Authorization"))
		kind, tokenString, ok := strings.Cut(header, " ")
		if !ok || !strings.EqualFold(kind, "Bearer") || strings.TrimSpace(tokenString) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "bearer token required"})
			return
		}

		claims := new(Claims)
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}
			return secret, nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer(issuer), jwt.WithExpirationRequired())
		if err != nil || token == nil || !token.Valid || strings.TrimSpace(claims.TenantID) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		roles := make(map[string]struct{}, len(claims.Roles))
		for _, role := range claims.Roles {
			if role = strings.TrimSpace(role); role != "" {
				roles[role] = struct{}{}
			}
		}
		c.Set(principalKey, Principal{TenantID: claims.TenantID, Roles: roles})
		c.Next()
	}
}

func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := GetPrincipal(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		if _, allowed := principal.Roles[role]; !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
			return
		}
		c.Next()
	}
}

func RequireAnyRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := GetPrincipal(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}
		for _, role := range roles {
			if _, allowed := principal.Roles[role]; allowed {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient role"})
		return
	}
}

func GetPrincipal(c *gin.Context) (Principal, bool) {
	value, ok := c.Get(principalKey)
	if !ok {
		return Principal{}, false
	}
	principal, ok := value.(Principal)
	return principal, ok
}
