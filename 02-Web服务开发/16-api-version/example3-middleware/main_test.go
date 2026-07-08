package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	userService := NewUserService()
	r.Use(VersionMiddleware())
	r.GET("/api/:version/users", func(c *gin.Context) {
		listUsers(c, userService)
	})
	r.GET("/api/users", func(c *gin.Context) {
		listUsers(c, userService)
	})
	return r
}

func TestVersionFromURL(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v2/users", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"phone"`) {
		t.Fatalf("expected v2 response to include phone, body=%s", rec.Body.String())
	}
}

func TestVersionFromHeader(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("X-API-Version", "v2")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"phone"`) {
		t.Fatalf("expected v2 response to include phone, body=%s", rec.Body.String())
	}
}

func TestUnsupportedVersion(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/v9/users", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}
