package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

var (
	users   = make(map[int]User)
	nextID  = 1
	usersMu sync.RWMutex
)

func writeJSON(w http.ResponseWriter, status int, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	usersMu.RLock()
	defer usersMu.RUnlock()

	userList := make([]User, 0, len(users))
	for _, u := range users {
		userList = append(userList, u)
	}

	writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    userList,
	})
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid user id",
		})
		return
	}

	usersMu.RLock()
	defer usersMu.RUnlock()

	user, ok := users[id]
	if !ok {
		writeJSON(w, http.StatusNotFound, Response{
			Code:    404,
			Message: "user not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    user,
	})
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, Response{
			Code:    400,
			Message: "invalid request body",
		})
		return
	}

	if input.Name == "" || input.Email == "" {
		writeJSON(w, http.StatusBadRequest, Response{
			Code:    400,
			Message: "name and email are required",
		})
		return
	}

	usersMu.Lock()
	user := User{
		ID:    nextID,
		Name:  input.Name,
		Email: input.Email,
	}
	users[nextID] = user
	nextID++
	usersMu.Unlock()

	writeJSON(w, http.StatusCreated, Response{
		Code:    0,
		Message: "user created",
		Data:    user,
	})
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /users", getUsersHandler)
	mux.HandleFunc("GET /users/{id}", getUserHandler)
	mux.HandleFunc("POST /users", createUserHandler)

	fmt.Println("Server starting on http://localhost:8080")
	fmt.Println("API Endpoints:")
	fmt.Println("  GET    /users       - List all users")
	fmt.Println("  GET    /users/{id}  - Get user by ID")
	fmt.Println("  POST   /users       - Create user (JSON body)")

	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
