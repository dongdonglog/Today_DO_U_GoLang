package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-book/jwt/example7-admin-auth/config"
	"github.com/go-book/jwt/example7-admin-auth/handler"
	"github.com/go-book/jwt/example7-admin-auth/jwt"
	"github.com/go-book/jwt/example7-admin-auth/middleware"
	"github.com/go-book/jwt/example7-admin-auth/store"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化存储
	userStore := store.NewMemoryStore()

	// 初始化 JWT 管理器
	jwtManager := jwt.NewJWTManager(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessTokenExpire,
		cfg.JWT.RefreshTokenExpire,
	)

	r := newRouter(userStore, jwtManager)

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("Server starting on %s\n", addr)
	fmt.Println("\n测试命令:")
	fmt.Println("\n  # 1. 登录")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/login \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"username\":\"admin\",\"password\":\"admin123\"}'")
	fmt.Println("\n  # 2. 获取当前用户信息")
	fmt.Println("  curl http://localhost:8080/api/v1/me \\")
	fmt.Println("    -H 'Authorization: Bearer <access_token>'")
	fmt.Println("\n  # 3. 获取用户列表")
	fmt.Println("  curl http://localhost:8080/api/v1/users \\")
	fmt.Println("    -H 'Authorization: Bearer <access_token>'")
	fmt.Println("\n  # 4. 创建用户（需要 admin 权限）")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/users \\")
	fmt.Println("    -H 'Authorization: Bearer <access_token>' \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"username\":\"newuser\",\"password\":\"password123\",\"role\":\"user\",\"email\":\"newuser@example.com\"}'")
	fmt.Println("\n  # 5. 删除用户（需要 admin 权限）")
	fmt.Println("  curl -X DELETE http://localhost:8080/api/v1/users/2 \\")
	fmt.Println("    -H 'Authorization: Bearer <access_token>'")
	fmt.Println("\n  # 6. 刷新 Token")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/refresh \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"refresh_token\":\"<refresh_token>\"}'")
	fmt.Println("\n  # 7. 退出登录")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/logout \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"refresh_token\":\"<refresh_token>\"}'")

	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to shutdown server: %v", err)
	}
	log.Println("Server stopped")
}

func newRouter(userStore *store.MemoryStore, jwtManager *jwt.JWTManager) *gin.Engine {
	authHandler := handler.NewAuthHandler(userStore, jwtManager)
	userHandler := handler.NewUserHandler(userStore)

	r := gin.New()
	r.Use(gin.Recovery())

	r.POST("/api/v1/login", authHandler.Login)
	r.POST("/api/v1/refresh", authHandler.Refresh)
	r.POST("/api/v1/logout", authHandler.Logout)

	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(jwtManager))
	{
		api.GET("/me", userHandler.GetCurrentUser)
		api.GET("/users", userHandler.ListUsers)
		api.GET("/users/:id", userHandler.GetUser)

		admin := api.Group("")
		admin.Use(middleware.RequireRole("admin"))
		{
			admin.POST("/users", userHandler.CreateUser)
			admin.DELETE("/users/:id", userHandler.DeleteUser)
		}
	}

	return r
}
