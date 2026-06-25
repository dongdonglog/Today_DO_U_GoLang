package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestServer() (*http.ServeMux, UserStore) {
	store := NewMemoryStore()
	handler := NewUserHandler(store)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	RegisterHealthRoute(mux)
	return mux, store
}

func TestHealthCheck(t *testing.T) {
	mux, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
}

func TestCreateUser(t *testing.T) {
	mux, _ := setupTestServer()

	body := bytes.NewBufferString(`{"name":"Alice","email":"alice@example.com"}`)
	req := httptest.NewRequest("POST", "/api/users", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != 0 {
		t.Errorf("expected code 0, got %d", resp.Code)
	}

	user, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}
	if user["name"] != "Alice" {
		t.Errorf("expected name Alice, got %v", user["name"])
	}
}

func TestCreateUserMissingFields(t *testing.T) {
	mux, _ := setupTestServer()

	body := bytes.NewBufferString(`{"name":"Alice"}`)
	req := httptest.NewRequest("POST", "/api/users", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestGetUser(t *testing.T) {
	mux, store := setupTestServer()

	store.Create(&User{Name: "Bob", Email: "bob@example.com"})

	req := httptest.NewRequest("GET", "/api/users/1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	user, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("expected data to be a map")
	}
	if user["name"] != "Bob" {
		t.Errorf("expected name Bob, got %v", user["name"])
	}
}

func TestGetUserNotFound(t *testing.T) {
	mux, _ := setupTestServer()

	req := httptest.NewRequest("GET", "/api/users/999", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestListUsers(t *testing.T) {
	mux, store := setupTestServer()

	store.Create(&User{Name: "Alice", Email: "alice@example.com"})
	store.Create(&User{Name: "Bob", Email: "bob@example.com"})

	req := httptest.NewRequest("GET", "/api/users", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp Response
	json.NewDecoder(w.Body).Decode(&resp)
	users, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatal("expected data to be a list")
	}
	if len(users) != 2 {
		t.Errorf("expected 2 users, got %d", len(users))
	}
}

func TestDeleteUser(t *testing.T) {
	mux, store := setupTestServer()

	store.Create(&User{Name: "Alice", Email: "alice@example.com"})

	req := httptest.NewRequest("DELETE", "/api/users/1", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	_, err := store.Get(1)
	if err == nil {
		t.Error("expected user to be deleted")
	}
}

func TestDeleteUserNotFound(t *testing.T) {
	mux, _ := setupTestServer()

	req := httptest.NewRequest("DELETE", "/api/users/999", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
