package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-book/rest-api/example8-rest-api/response"
	"github.com/go-book/rest-api/example8-rest-api/store"
)

type apiResponse struct {
	Code       int             `json:"code"`
	Message    string          `json:"message"`
	Data       json.RawMessage `json:"data,omitempty"`
	Pagination struct {
		Page  int `json:"page"`
		Size  int `json:"size"`
		Total int `json:"total"`
	} `json:"pagination,omitempty"`
}

type userResponse struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return newRouter(store.NewMemoryStore())
}

func request(t *testing.T, r http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeResponse(t *testing.T, w *httptest.ResponseRecorder) apiResponse {
	t.Helper()

	var resp apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v, body=%s", err, w.Body.String())
	}
	return resp
}

func decodeUser(t *testing.T, data json.RawMessage) userResponse {
	t.Helper()

	var user userResponse
	if err := json.Unmarshal(data, &user); err != nil {
		t.Fatalf("decode user: %v, data=%s", err, string(data))
	}
	return user
}

func TestCreateUserReturnsCreated(t *testing.T) {
	r := setupTestRouter()

	w := request(t, r, http.MethodPost, "/api/v1/users", `{"name":"Alice","email":"alice@example.com"}`, nil)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d, body=%s", w.Code, w.Body.String())
	}

	resp := decodeResponse(t, w)
	if resp.Code != response.CodeSuccess || resp.Message != "created" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestCreateUserWithIdempotencyKeyDoesNotDuplicate(t *testing.T) {
	r := setupTestRouter()
	headers := map[string]string{"Idempotency-Key": "create-user-alice"}
	body := `{"name":"Alice","email":"alice@example.com"}`

	request(t, r, http.MethodPost, "/api/v1/users", body, headers)
	request(t, r, http.MethodPost, "/api/v1/users", body, headers)

	w := request(t, r, http.MethodGet, "/api/v1/users?page=1&size=10", "", nil)
	resp := decodeResponse(t, w)
	if resp.Pagination.Total != 1 {
		t.Fatalf("expected one user after repeated idempotent create, got %d", resp.Pagination.Total)
	}
}

func TestDuplicateEmailReturnsConflict(t *testing.T) {
	r := setupTestRouter()

	request(t, r, http.MethodPost, "/api/v1/users", `{"name":"Alice","email":"alice@example.com"}`, nil)
	w := request(t, r, http.MethodPost, "/api/v1/users", `{"name":"Alice2","email":"alice@example.com"}`, nil)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d, body=%s", w.Code, w.Body.String())
	}

	resp := decodeResponse(t, w)
	if resp.Code != response.CodeUserExists {
		t.Fatalf("expected user exists code, got %d", resp.Code)
	}
}

func TestPutReplacesUser(t *testing.T) {
	r := setupTestRouter()

	request(t, r, http.MethodPost, "/api/v1/users", `{"name":"Alice","email":"alice@example.com"}`, nil)
	w := request(t, r, http.MethodPut, "/api/v1/users/1", `{"name":"Bob","email":"bob@example.com"}`, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	user := decodeUser(t, decodeResponse(t, w).Data)
	if user.Name != "Bob" || user.Email != "bob@example.com" {
		t.Fatalf("expected full replacement, got %+v", user)
	}
}

func TestValidationErrorReturnsBadRequest(t *testing.T) {
	r := setupTestRouter()

	w := request(t, r, http.MethodPost, "/api/v1/users", `{"name":"A"}`, nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	resp := decodeResponse(t, w)
	if resp.Code != response.CodeInvalidParam {
		t.Fatalf("expected invalid parameter code, got %d", resp.Code)
	}
}

func TestDeleteReturnsNoContent(t *testing.T) {
	r := setupTestRouter()

	request(t, r, http.MethodPost, "/api/v1/users", `{"name":"Alice","email":"alice@example.com"}`, nil)
	w := request(t, r, http.MethodDelete, "/api/v1/users/1", "", nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Fatalf("expected empty body for 204, got %s", w.Body.String())
	}
}
