package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type UserHandler struct {
	store UserStore
}

func NewUserHandler(store UserStore) *UserHandler {
	return &UserHandler{store: store}
}

func writeJSON(w http.ResponseWriter, status int, resp Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users := h.store.List()
	writeJSON(w, http.StatusOK, NewSuccessResponse(users))
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, NewErrorResponse(400, "invalid user id"))
		return
	}

	user, err := h.store.Get(id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, NewErrorResponse(404, err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, NewSuccessResponse(user))
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, NewErrorResponse(400, "invalid request body"))
		return
	}

	if req.Name == "" || req.Email == "" {
		writeJSON(w, http.StatusBadRequest, NewErrorResponse(400, "name and email are required"))
		return
	}

	user := &User{Name: req.Name, Email: req.Email}
	if err := h.store.Create(user); err != nil {
		writeJSON(w, http.StatusInternalServerError, NewErrorResponse(500, err.Error()))
		return
	}

	writeJSON(w, http.StatusCreated, NewSuccessResponse(user))
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, NewErrorResponse(400, "invalid user id"))
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, NewErrorResponse(400, "invalid request body"))
		return
	}

	var name, email string
	if req.Name != nil {
		name = *req.Name
	}
	if req.Email != nil {
		email = *req.Email
	}

	user, err := h.store.Update(id, name, email)
	if err != nil {
		writeJSON(w, http.StatusNotFound, NewErrorResponse(404, err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, NewSuccessResponse(user))
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, NewErrorResponse(400, "invalid user id"))
		return
	}

	if err := h.store.Delete(id); err != nil {
		writeJSON(w, http.StatusNotFound, NewErrorResponse(404, err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, NewSuccessResponse(nil))
}

func (h *UserHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/users", h.ListUsers)
	mux.HandleFunc("GET /api/users/{id}", h.GetUser)
	mux.HandleFunc("POST /api/users", h.CreateUser)
	mux.HandleFunc("PUT /api/users/{id}", h.UpdateUser)
	mux.HandleFunc("DELETE /api/users/{id}", h.DeleteUser)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, NewSuccessResponse(map[string]string{
		"status": "ok",
	}))
}

func RegisterHealthRoute(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", healthHandler)
}

func init() {
	fmt.Println("User API module loaded")
}
