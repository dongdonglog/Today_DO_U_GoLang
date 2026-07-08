package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	authjwt "github.com/go-book/jwt/example7-admin-auth/jwt"
	"github.com/go-book/jwt/example7-admin-auth/response"
	"github.com/go-book/jwt/example7-admin-auth/store"
)

type apiResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type tokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

func setupRouter() http.Handler {
	gin.SetMode(gin.TestMode)
	manager := authjwt.NewJWTManager(strings.Repeat("a", 32), strings.Repeat("b", 32), 15*time.Minute, 7*24*time.Hour)
	return newRouter(store.NewMemoryStore(), manager)
}

func doRequest(t *testing.T, h http.Handler, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func decodeAPIResponse(t *testing.T, w *httptest.ResponseRecorder) apiResponse {
	t.Helper()

	var resp apiResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v, body=%s", err, w.Body.String())
	}
	return resp
}

func login(t *testing.T, h http.Handler) tokenData {
	t.Helper()

	w := doRequest(t, h, http.MethodPost, "/api/v1/login", `{"username":"admin","password":"admin123"}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	resp := decodeAPIResponse(t, w)
	var data tokenData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		t.Fatalf("decode token data: %v", err)
	}
	if data.AccessToken == "" || data.RefreshToken == "" {
		t.Fatalf("expected access and refresh tokens")
	}
	return data
}

func TestProtectedRouteRequiresToken(t *testing.T) {
	h := setupRouter()

	w := doRequest(t, h, http.MethodGet, "/api/v1/users", "", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", w.Code)
	}

	resp := decodeAPIResponse(t, w)
	if resp.Code != response.CodeUnauthorized {
		t.Fatalf("expected unauthorized code, got %d", resp.Code)
	}
}

func TestLoginAndAccessProtectedRoute(t *testing.T) {
	h := setupRouter()
	tokens := login(t, h)

	w := doRequest(t, h, http.MethodGet, "/api/v1/users", "", tokens.AccessToken)
	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestRefreshTokenRotation(t *testing.T) {
	h := setupRouter()
	tokens := login(t, h)

	w := doRequest(t, h, http.MethodPost, "/api/v1/refresh", `{"refresh_token":"`+tokens.RefreshToken+`"}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("expected refresh status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	resp := decodeAPIResponse(t, w)
	var refreshed tokenData
	if err := json.Unmarshal(resp.Data, &refreshed); err != nil {
		t.Fatalf("decode refreshed token data: %v", err)
	}
	if refreshed.RefreshToken == "" || refreshed.RefreshToken == tokens.RefreshToken {
		t.Fatalf("expected rotated refresh token")
	}

	w = doRequest(t, h, http.MethodPost, "/api/v1/refresh", `{"refresh_token":"`+tokens.RefreshToken+`"}`, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected old refresh token rejected, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	h := setupRouter()
	tokens := login(t, h)

	w := doRequest(t, h, http.MethodPost, "/api/v1/logout", `{"refresh_token":"`+tokens.RefreshToken+`"}`, "")
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected logout status 204, got %d, body=%s", w.Code, w.Body.String())
	}

	w = doRequest(t, h, http.MethodPost, "/api/v1/refresh", `{"refresh_token":"`+tokens.RefreshToken+`"}`, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected revoked refresh token rejected, got %d, body=%s", w.Code, w.Body.String())
	}
}
