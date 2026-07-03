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
	"github.com/go-book/logger/example5-admin-logger/config"
	"github.com/go-book/logger/example5-admin-logger/handler"
	"github.com/go-book/logger/example5-admin-logger/logger"
	"github.com/go-book/logger/example5-admin-logger/middleware"
	"github.com/go-book/logger/example5-admin-logger/store"
	"go.uber.org/zap"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 创建 logger
	appLogger := logger.MustNewLogger(cfg.Log)
	defer appLogger.Sync()

	// 记录启动信息
	appLogger.Info("application starting",
		zap.String("app", cfg.App.Name),
		zap.String("version", cfg.App.Version),
		zap.String("env", cfg.App.Env),
		zap.String("addr", cfg.Server.Addr()),
	)

	// 初始化存储
	userStore := store.NewMemoryStore()
	appLogger.Info("store initialized", zap.String("type", "memory"))

	// 初始化处理器
	userHandler := handler.NewUserHandler(userStore, appLogger)

	// 创建路由
	r := gin.New()

	// 中间件顺序很重要
	r.Use(middleware.RecoveryMiddleware(appLogger))  // 1. 错误恢复
	r.Use(middleware.RequestIDMiddleware())           // 2. 请求 ID
	r.Use(middleware.LoggerMiddleware(appLogger))     // 3. 请求日志

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		requestID := c.GetString("request_id")
		appLogger.Info("health check",
			zap.String("request_id", requestID),
		)
		c.JSON(http.StatusOK, gin.H{
			"status":     "ok",
			"app":        cfg.App.Name,
			"version":    cfg.App.Version,
			"env":        cfg.App.Env,
			"request_id": requestID,
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
		sig := <-quit

		appLogger.Info("shutdown signal received",
			zap.String("signal", sig.String()),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			appLogger.Error("server forced to shutdown",
				zap.Error(err),
			)
		}

		appLogger.Info("server exited gracefully")
	}()

	// 启动服务器
	fmt.Printf("Server is running on %s\n", cfg.Server.Addr())
	fmt.Println("\nTest commands:")
	fmt.Println("  curl http://localhost:8080/health")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/users \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"name\":\"Alice\",\"email\":\"alice@example.com\"}'")
	fmt.Println("  curl http://localhost:8080/api/v1/users")
	fmt.Println("  curl http://localhost:8080/api/v1/users/1")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		appLogger.Fatal("server failed to start",
			zap.Error(err),
		)
	}
}
