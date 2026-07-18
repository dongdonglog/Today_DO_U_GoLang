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
	"github.com/go-book/mysql/example2-admin-mysql/handler"
	"github.com/go-book/mysql/example2-admin-mysql/store"
)

func main() {
	// 连接 MySQL
	// 格式：user:password@tcp(host:port)/dbname
	dsn := getenv("MYSQL_DSN", "root:root@tcp(localhost:3306)/admin_db?charset=utf8mb4&parseTime=True&loc=Local")
	mysqlStore, err := store.NewMySQLStore(dsn)
	if err != nil {
		log.Fatal("Failed to connect to MySQL:", err)
	}
	defer mysqlStore.Close()

	// 初始化处理器
	userHandler := handler.NewUserHandler(mysqlStore)

	// 创建路由
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
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
			users.DELETE("/:id", userHandler.DeleteUser)
		}
	}

	addr := getenv("HTTP_ADDR", ":8080")
	log.Printf("Server starting on %s", addr)
	log.Println("")
	log.Println("MySQL connected successfully!")
	log.Println("")
	log.Println("Test commands:")
	log.Println("  # 查询用户列表")
	log.Println("  curl http://localhost:8080/api/v1/users")
	log.Println("")
	log.Println("  # 查询用户列表（分页）")
	log.Println("  curl 'http://localhost:8080/api/v1/users?page=1&size=10'")
	log.Println("")
	log.Println("  # 创建用户")
	log.Println("  curl -X POST http://localhost:8080/api/v1/users \\")
	log.Println("    -H 'Content-Type: application/json' \\")
	log.Println("    -d '{\"username\":\"charlie\",\"email\":\"charlie@example.com\",\"password\":\"ChangeMe_123\",\"phone\":\"13700137000\"}'")
	log.Println("")
	log.Println("  # 查询用户详情")
	log.Println("  curl http://localhost:8080/api/v1/users/1")
	log.Println("")
	log.Println("  # 更新用户")
	log.Println("  curl -X PUT http://localhost:8080/api/v1/users/1 \\")
	log.Println("    -H 'Content-Type: application/json' \\")
	log.Println("    -d '{\"phone\":\"13900139000\"}'")
	log.Println("")
	log.Println("  # 删除用户")
	log.Println("  curl -X DELETE http://localhost:8080/api/v1/users/1")

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
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Failed to shutdown server: %v", err)
	}
	log.Println("Server stopped")
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
