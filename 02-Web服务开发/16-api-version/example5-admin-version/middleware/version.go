package middleware

import (
	"log"

	"github.com/gin-gonic/gin"
)

// DeprecationMiddleware 废弃中间件
func DeprecationMiddleware(version, sunsetDate string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Deprecation", "true")
		c.Header("Sunset", sunsetDate)
		c.Header("Link", `<https://api.example.com/api/v2/users>; rel="successor-version"`)
		c.Header("Warning", `299 - "This API version is deprecated"`)
		log.Printf("deprecated_api_used version=%s method=%s path=%s client_ip=%s user_agent=%q",
			version,
			c.Request.Method,
			c.FullPath(),
			c.ClientIP(),
			c.Request.UserAgent(),
		)
		c.Next()
	}
}
