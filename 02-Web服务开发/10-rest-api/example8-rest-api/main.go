package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-book/rest-api/example8-rest-api/store"
)

func main() {
	userStore := store.NewMemoryStore()
	r := newRouter(userStore)

	log.Println("Server starting on :8080")
	log.Println("")
	log.Println("REST API Endpoints:")
	log.Println("  GET    /api/v1/users         - List users (pagination, filter, sort)")
	log.Println("  GET    /api/v1/users/:id     - Get user by ID")
	log.Println("  POST   /api/v1/users         - Create user")
	log.Println("  PUT    /api/v1/users/:id     - Full update user")
	log.Println("  PATCH  /api/v1/users/:id     - Partial update user")
	log.Println("  DELETE /api/v1/users/:id     - Delete user")
	log.Println("")
	log.Println("Test commands:")
	log.Println("  # Create user (201 Created)")
	log.Println("  curl -X POST http://localhost:8080/api/v1/users \\")
	log.Println("    -H 'Content-Type: application/json' \\")
	log.Println("    -d '{\"name\":\"Alice\",\"email\":\"alice@example.com\"}'")
	log.Println("")
	log.Println("  # List users with pagination (200 OK)")
	log.Println("  curl 'http://localhost:8080/api/v1/users?page=1&size=10'")
	log.Println("")
	log.Println("  # Get user by ID (200 OK)")
	log.Println("  curl http://localhost:8080/api/v1/users/1")
	log.Println("")
	log.Println("  # Partial update (200 OK)")
	log.Println("  curl -X PATCH http://localhost:8080/api/v1/users/1 \\")
	log.Println("    -H 'Content-Type: application/json' \\")
	log.Println("    -d '{\"name\":\"Alice Updated\"}'")
	log.Println("")
	log.Println("  # Delete user (204 No Content)")
	log.Println("  curl -X DELETE http://localhost:8080/api/v1/users/1")
	log.Println("")
	log.Println("  # Validation error (400 Bad Request)")
	log.Println("  curl -X POST http://localhost:8080/api/v1/users \\")
	log.Println("    -H 'Content-Type: application/json' \\")
	log.Println("    -d '{\"name\":\"A\"}'")
	log.Println("")
	log.Println("  # Not found (404 Not Found)")
	log.Println("  curl http://localhost:8080/api/v1/users/999")

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
