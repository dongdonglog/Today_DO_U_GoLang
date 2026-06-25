package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type contextKey string

const userIDKey contextKey = "userID"

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		log.Printf("[%s] %s %s took %v",
			r.RemoteAddr,
			r.Method,
			r.URL.Path,
			time.Since(start),
		)
	})
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if token == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "missing authorization token",
			})
			return
		}

		if token != "Bearer secret-token" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "invalid token",
			})
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, "user-123")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC recovered: %v", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "internal server error",
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func chainMiddleware(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

func protectedHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(string)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "access granted",
		"userID":  userID,
	})
}

func panicHandler(w http.ResponseWriter, r *http.Request) {
	panic("something went wrong!")
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /public", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "public endpoint")
	})

	protected := chainMiddleware(
		http.HandlerFunc(protectedHandler),
		authMiddleware,
	)
	mux.Handle("GET /protected", protected)

	panicRoute := chainMiddleware(
		http.HandlerFunc(panicHandler),
		recoveryMiddleware,
	)
	mux.Handle("GET /panic", panicRoute)

	handler := chainMiddleware(mux, loggingMiddleware)

	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println("Endpoints:")
	fmt.Println("  GET /public    - No auth required")
	fmt.Println("  GET /protected - Requires: Authorization: Bearer secret-token")
	fmt.Println("  GET /panic     - Will panic (but recovered)")

	if err := http.ListenAndServe(":8080", handler); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
