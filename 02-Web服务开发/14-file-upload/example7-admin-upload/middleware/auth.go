package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-book/file-upload/example7-admin-upload/response"
	"github.com/golang-jwt/jwt/v5"
)

type AccessClaims struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func AuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			response.Unauthorized(c, "missing authorization header")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "invalid authorization format")
			c.Abort()
			return
		}

		token, err := jwt.ParseWithClaims(parts[1], &AccessClaims{}, func(token *jwt.Token) (interface{}, error) {
			if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))

		if err != nil {
			response.Unauthorized(c, "invalid or expired token")
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(*AccessClaims); ok && token.Valid {
			c.Set("user_id", claims.UserID)
			c.Set("role", claims.Role)
			c.Next()
			return
		}

		response.Unauthorized(c, "invalid token")
		c.Abort()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			response.Unauthorized(c, "unauthorized")
			c.Abort()
			return
		}

		role := userRole.(string)
		for _, r := range roles {
			if role == r {
				c.Next()
				return
			}
		}

		response.Forbidden(c, "forbidden: insufficient permissions")
		c.Abort()
	}
}
