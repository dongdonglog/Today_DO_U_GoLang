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
	v1 "github.com/go-book/api-version/example5-admin-version/handler/v1"
	v2 "github.com/go-book/api-version/example5-admin-version/handler/v2"
	"github.com/go-book/api-version/example5-admin-version/middleware"
	"github.com/go-book/api-version/example5-admin-version/service"
)

const sunsetV1 = "Thu, 01 Apr 2027 00:00:00 GMT"

func main() {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// 初始化服务
	userService := service.NewUserService()

	// 初始化处理器
	v1Handler := v1.NewUserHandler(userService)
	v2Handler := v2.NewUserHandler(userService)

	// v1 路由组（已废弃）
	v1Group := r.Group("/api/v1")
	v1Group.Use(middleware.DeprecationMiddleware("v1", sunsetV1))
	{
		v1Group.GET("/users", v1Handler.ListUsers)
		v1Group.POST("/users", v1Handler.CreateUser)
	}

	// v2 路由组（当前版本）
	v2Group := r.Group("/api/v2")
	{
		v2Group.GET("/users", v2Handler.ListUsers)
		v2Group.POST("/users", v2Handler.CreateUser)
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

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

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
