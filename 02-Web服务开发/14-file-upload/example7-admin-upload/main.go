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
	"github.com/go-book/file-upload/example7-admin-upload/config"
	"github.com/go-book/file-upload/example7-admin-upload/handler"
	"github.com/go-book/file-upload/example7-admin-upload/middleware"
	"github.com/go-book/file-upload/example7-admin-upload/storage"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化存储
	fileStorage := storage.NewLocalStorage(
		cfg.Upload.StoragePath,
		cfg.Upload.AllowedTypes,
		cfg.Upload.MaxSize,
	)

	// 初始化处理器
	userHandler := handler.NewUserHandler(
		cfg.JWT.AccessSecret,
		cfg.JWT.RefreshSecret,
		cfg.JWT.AccessTokenExpire,
		cfg.JWT.RefreshTokenExpire,
	)
	fileHandler := handler.NewFileHandler(fileStorage)

	r := newRouter(userHandler, fileHandler, cfg.JWT.AccessSecret)

	// 启动服务器
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("Server starting on %s\n", addr)
	fmt.Println("\n测试命令:")
	fmt.Println("\n  # 1. 登录")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/login \\")
	fmt.Println("    -H 'Content-Type: application/json' \\")
	fmt.Println("    -d '{\"username\":\"admin\",\"password\":\"admin123\"}'")
	fmt.Println("\n  # 2. 上传文件")
	fmt.Println("  curl -X POST http://localhost:8080/api/v1/upload \\")
	fmt.Println("    -H 'Authorization: Bearer <access_token>' \\")
	fmt.Println("    -F 'file=@test.jpg'")
	fmt.Println("\n  # 3. 预览文件")
	fmt.Println("  curl http://localhost:8080/api/v1/files/2024/01/15/abc-123.jpg")
	fmt.Println("\n  # 4. 下载文件")
	fmt.Println("  curl http://localhost:8080/api/v1/download/2024/01/15/abc-123.jpg -o downloaded.jpg")
	fmt.Println("\n  # 5. 删除文件（需要 admin 权限）")
	fmt.Println("  curl -X DELETE http://localhost:8080/api/v1/files/2024/01/15/abc-123.jpg \\")
	fmt.Println("    -H 'Authorization: Bearer <access_token>'")

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

func newRouter(userHandler *handler.UserHandler, fileHandler *handler.FileHandler, accessSecret string) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())

	r.POST("/api/v1/login", userHandler.Login)

	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(accessSecret))
	{
		api.GET("/me", userHandler.GetCurrentUser)
		api.POST("/upload", fileHandler.UploadFile)

		admin := api.Group("")
		admin.Use(middleware.RequireRole("admin"))
		{
			admin.DELETE("/files/*filepath", fileHandler.DeleteFile)
		}
	}

	r.GET("/api/v1/files/*filepath", fileHandler.PreviewFile)
	r.GET("/api/v1/download/*filepath", fileHandler.DownloadFile)

	return r
}
