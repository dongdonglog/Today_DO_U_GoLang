package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-book/openapi/example6-admin-docs/docs"
	"github.com/go-book/openapi/example6-admin-docs/handler"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Admin API
// @version 1.0
// @description 后台管理系统 API 文档
// @description 这是一个完整的后台管理系统 API，包含用户管理、文件管理、认证等功能
// @host localhost:8080
// @BasePath /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Bearer {token}

func main() {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// 初始化处理器
	userHandler := handler.NewUserHandler()
	fileHandler := handler.NewFileHandler()
	authHandler := handler.NewAuthHandler()

	// 公开接口
	r.POST("/api/v1/login", authHandler.Login)

	// 需要认证的接口
	api := r.Group("/api/v1")
	api.Use(authHandler.AuthMiddleware())
	{
		// 用户管理
		api.GET("/me", userHandler.GetCurrentUser)
		api.GET("/users", userHandler.ListUsers)
		api.POST("/users", userHandler.CreateUser)
		api.PUT("/users/:id", userHandler.UpdateUser)
		api.DELETE("/users/:id", userHandler.DeleteUser)

		// 文件管理
		api.POST("/upload", fileHandler.UploadFile)
		api.GET("/files/:filename", fileHandler.DownloadFile)
		api.DELETE("/files/:filename", fileHandler.DeleteFile)
	}

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	if swaggerEnabled() {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	addr := getenv("HTTP_ADDR", ":8080")
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("admin api listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server listen failed: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}
	log.Println("server stopped")
}

func swaggerEnabled() bool {
	if os.Getenv("SWAGGER_ENABLED") == "true" {
		return true
	}
	if os.Getenv("APP_ENV") == "prod" {
		return false
	}
	return os.Getenv("SWAGGER_ENABLED") != "false"
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
