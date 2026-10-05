package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-book/graduate/user-center/internal/store"
)

type principal struct {
	User            *store.User
	AccessTokenID   string
	AccessExpiresAt time.Time
}

func (s *Server) requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			fail(c, http.StatusUnauthorized, "missing_token", "missing authorization header")
			c.Abort()
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			fail(c, http.StatusUnauthorized, "bad_token_format", "authorization must use Bearer token")
			c.Abort()
			return
		}
		claims, err := s.jwt.ParseAccessToken(parts[1])
		if err != nil {
			fail(c, http.StatusUnauthorized, "invalid_token", "invalid or expired token")
			c.Abort()
			return
		}
		revoked, err := s.redis.IsAccessTokenRevoked(c.Request.Context(), claims.ID)
		if err != nil {
			fail(c, http.StatusServiceUnavailable, "dependency_unavailable", "token state is unavailable")
			c.Abort()
			return
		}
		if revoked {
			fail(c, http.StatusUnauthorized, "revoked_token", "token has been revoked")
			c.Abort()
			return
		}
		user, err := s.db.GetUserByID(c.Request.Context(), claims.UserID)
		if err != nil {
			fail(c, http.StatusUnauthorized, "invalid_token", "user is not available")
			c.Abort()
			return
		}
		if user.Status != store.StatusActive {
			fail(c, http.StatusForbidden, "user_disabled", "user has been disabled")
			c.Abort()
			return
		}
		c.Set("principal", principal{
			User:            user,
			AccessTokenID:   claims.ID,
			AccessExpiresAt: claims.ExpiresAt.Time,
		})
		c.Next()
	}
}

func requireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, ok := getPrincipal(c)
		if !ok {
			fail(c, http.StatusUnauthorized, "unauthorized", "unauthorized")
			c.Abort()
			return
		}
		if p.User.Role != role {
			fail(c, http.StatusForbidden, "forbidden", "insufficient permissions")
			c.Abort()
			return
		}
		c.Next()
	}
}

func getPrincipal(c *gin.Context) (principal, bool) {
	value, ok := c.Get("principal")
	if !ok {
		return principal{}, false
	}
	p, ok := value.(principal)
	return p, ok
}
