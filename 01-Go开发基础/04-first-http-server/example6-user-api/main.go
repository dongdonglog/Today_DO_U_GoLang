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
)

func main() {
	store := NewMemoryStore()

	userHandler := NewUserHandler(store)

	mux := http.NewServeMux()
	userHandler.RegisterRoutes(mux)
	RegisterHealthRoute(mux)

	handler := ChainMiddleware(mux,
		recoveryMiddleware,
		loggingMiddleware,
		requestIDMiddleware,
		corsMiddleware,
	)

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		fmt.Println("=========================================")
		fmt.Println("  User Management API")
		fmt.Println("=========================================")
		fmt.Println("Server starting on http://localhost:8080")
		fmt.Println("")
		fmt.Println("Endpoints:")
		fmt.Println("  GET    /health          - Health check")
		fmt.Println("  GET    /api/users       - List all users")
		fmt.Println("  GET    /api/users/{id}  - Get user by ID")
		fmt.Println("  POST   /api/users       - Create user")
		fmt.Println("  PUT    /api/users/{id}  - Update user")
		fmt.Println("  DELETE /api/users/{id}  - Delete user")
		fmt.Println("")
		fmt.Println("Test commands:")
		fmt.Println("  curl http://localhost:8080/health")
		fmt.Println("  curl -X POST http://localhost:8080/api/users \\")
		fmt.Println("    -H 'Content-Type: application/json' \\")
		fmt.Println("    -d '{\"name\":\"Alice\",\"email\":\"alice@example.com\"}'")
		fmt.Println("  curl http://localhost:8080/api/users")
		fmt.Println("  curl http://localhost:8080/api/users/1")
		fmt.Println("=========================================")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutdown signal received, starting graceful shutdown...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}
