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
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from graceful shutdown server")
	})

	mux.HandleFunc("GET /slow", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Slow request started from %s", r.RemoteAddr)
		time.Sleep(5 * time.Second)
		log.Printf("Slow request completed")
		fmt.Fprintf(w, "Slow response after 5 seconds")
	})

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		fmt.Println("Server starting on http://localhost:8080")
		fmt.Println("Endpoints:")
		fmt.Println("  GET /      - Fast response")
		fmt.Println("  GET /slow  - 5 second delay")
		fmt.Println("\nPress Ctrl+C to gracefully shutdown")
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
