package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-book/config/example5-admin-config/config"
	"github.com/go-book/config/example5-admin-config/handler"
	"github.com/go-book/config/example5-admin-config/middleware"
	"github.com/go-book/config/example5-admin-config/store"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting %s v%s [%s]", cfg.App.Name, cfg.App.Version, cfg.App.Env)
	log.Printf("Server: %s", cfg.Server.Addr())
	log.Printf("Database: %s", cfg.Database.MaskedDSN())
	log.Printf("Redis: %s (db=%d)", cfg.Redis.Addr(), cfg.Redis.DB)
	log.Printf("Log: level=%s format=%s", cfg.Log.Level, cfg.Log.Format)

	// 初始化存储
	userStore := store.NewMemoryStore()

	// 初始化处理器
	userHandler := handler.NewUserHandler(userStore)

	// 创建路由
	r := gin.New()
	r.Use(gin.Recovery(), middleware.Logger())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"app":     cfg.App.Name,
			"version": cfg.App.Version,
			"env":     cfg.App.Env,
		})
	})

	// API v1 路由
	v1 := r.Group("/api/v1")
	{
		users := v1.Group("/users")
		{
			users.GET("", userHandler.ListUsers)
			users.GET("/:id", userHandler.GetUser)
			users.POST("", userHandler.CreateUser)
			users.PUT("/:id", userHandler.UpdateUser)
			users.PATCH("/:id", userHandler.UpdateUser)
			users.DELETE("/:id", userHandler.DeleteUser)
		}
	}

	// 创建服务器
	srv := &http.Server{
		Addr:         cfg.Server.Addr(),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// 优雅关闭
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		log.Println("Shutting down server...")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("Server forced to shutdown: %v", err)
		}

		log.Println("Server exited")
	}()

	// 启动服务器
	log.Printf("Server is running on %s", cfg.Server.Addr())
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}
}
