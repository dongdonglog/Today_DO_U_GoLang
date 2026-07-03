package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-book/gin-framework/example7-admin-api/response"
	"github.com/go-book/gin-framework/example7-admin-api/store"
)

type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return newRouter(store.NewMemoryStore())
}

func request(t *testing.T, r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
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

func TestCreateAndGetUser(t *testing.T) {
	r := setupTestRouter()

	w := request(t, r, http.MethodPost, "/api/v1/users", `{"name":"Alice","email":"alice@example.com"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	resp := decodeResponse(t, w)
	if resp.Code != response.CodeSuccess {
		t.Fatalf("expected code 0, got %d", resp.Code)
	}

	w = request(t, r, http.MethodGet, "/api/v1/users/1", "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestCreateUserValidationError(t *testing.T) {
	r := setupTestRouter()

	w := request(t, r, http.MethodPost, "/api/v1/users", `{"name":"A","email":"bad-email"}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", w.Code)
	}

	resp := decodeResponse(t, w)
	if resp.Code != response.CodeInvalidParam {
		t.Fatalf("expected invalid parameter code, got %d", resp.Code)
	}
}

func TestDuplicateEmailReturnsConflict(t *testing.T) {
	r := setupTestRouter()

	request(t, r, http.MethodPost, "/api/v1/users", `{"name":"Alice","email":"alice@example.com"}`)
	w := request(t, r, http.MethodPost, "/api/v1/users", `{"name":"Alice2","email":"alice@example.com"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d, body=%s", w.Code, w.Body.String())
	}

	resp := decodeResponse(t, w)
	if resp.Code != response.CodeUserExists {
		t.Fatalf("expected user exists code, got %d", resp.Code)
	}
}

func TestGetMissingUserReturnsNotFound(t *testing.T) {
	r := setupTestRouter()

	w := request(t, r, http.MethodGet, "/api/v1/users/999", "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}

	resp := decodeResponse(t, w)
	if resp.Code != response.CodeUserNotFound {
		t.Fatalf("expected user not found code, got %d", resp.Code)
	}
}
