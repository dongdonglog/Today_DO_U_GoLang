package httpapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-book/graduate/user-center/internal/auth"
	"github.com/go-book/graduate/user-center/internal/config"
	"github.com/go-book/graduate/user-center/internal/store"
)

type Server struct {
	cfg   config.Config
	db    *store.MySQLStore
	redis *store.RedisStore
	jwt   *auth.Manager
}

func NewRouter(cfg config.Config, db *store.MySQLStore, redis *store.RedisStore, jwtManager *auth.Manager) *gin.Engine {
	server := &Server{cfg: cfg, db: db, redis: redis, jwt: jwtManager}
	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/healthz", server.health)

	v1 := r.Group("/v1")
	v1.POST("/auth/register", server.register)
	v1.POST("/auth/login", server.login)
	v1.POST("/auth/refresh", server.refresh)

	protected := v1.Group("")
	protected.Use(server.requireAuth())
	protected.POST("/auth/logout", server.logout)
	protected.GET("/me", server.me)

	admin := protected.Group("/admin")
	admin.Use(requireRole(store.RoleAdmin))
	admin.GET("/users", server.listUsers)
	admin.POST("/users/:id/disable", server.disableUser)
	admin.POST("/users/:id/enable", server.enableUser)

	return r
}

func (s *Server) health(c *gin.Context) {
	ctx := c.Request.Context()
	deps := map[string]string{"mysql": "ok", "redis": "ok"}
	status := http.StatusOK
	if err := s.db.Ping(ctx); err != nil {
		deps["mysql"] = "unavailable"
		status = http.StatusServiceUnavailable
	}
	if err := s.redis.Ping(ctx); err != nil {
		deps["redis"] = "unavailable"
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{"status": statusText(status), "dependencies": deps})
}

func statusText(status int) string {
	if status == http.StatusOK {
		return "ok"
	}
	return "degraded"
}
