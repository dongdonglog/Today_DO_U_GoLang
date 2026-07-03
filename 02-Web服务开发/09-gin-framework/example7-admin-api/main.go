package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-book/gin-framework/example7-admin-api/store"
)

func main() {
	userStore := store.NewMemoryStore()
	r := newRouter(userStore)

	log.Println("Server starting on :8080")
	log.Println("API Endpoints:")
	log.Println("  GET    /health          - Health check")
	log.Println("  GET    /api/v1/users    - List users")
	log.Println("  GET    /api/v1/users/1  - Get user by ID")
	log.Println("  POST   /api/v1/users    - Create user")
	log.Println("  PUT    /api/v1/users/1  - Update user")
	log.Println("  DELETE /api/v1/users/1  - Delete user")
	log.Println("")
	log.Println("Test commands:")
	log.Println("  curl http://localhost:8080/health")
	log.Println("  curl -X POST http://localhost:8080/api/v1/users \\")
	log.Println("    -H 'Content-Type: application/json' \\")
	log.Println("    -d '{\"name\":\"Alice\",\"email\":\"alice@example.com\"}'")
	log.Println("  curl http://localhost:8080/api/v1/users")

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}
	log.Println("server stopped")
}
