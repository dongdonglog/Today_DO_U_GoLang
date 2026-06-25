package main

import (
	"fmt"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Welcome to the API\n")
		fmt.Fprintf(w, "Try: GET /users/{id}\n")
	})

	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		fmt.Fprintf(w, "Getting user: %s\n", id)
	})

	mux.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Creating user\n")
	})

	mux.HandleFunc("PUT /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		fmt.Fprintf(w, "Updating user: %s\n", id)
	})

	mux.HandleFunc("DELETE /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		fmt.Fprintf(w, "Deleting user: %s\n", id)
	})

	mux.HandleFunc("GET /files/{path...}", func(w http.ResponseWriter, r *http.Request) {
		path := r.PathValue("path")
		fmt.Fprintf(w, "Getting file: %s\n", path)
	})

	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println("Routes:")
	fmt.Println("  GET  /")
	fmt.Println("  GET  /users/{id}")
	fmt.Println("  POST /users")
	fmt.Println("  PUT  /users/{id}")
	fmt.Println("  DELETE /users/{id}")
	fmt.Println("  GET  /files/{path...}")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
