package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	v1 "github.com/go-book/api-version/example5-admin-version/handler/v1"
	v2 "github.com/go-book/api-version/example5-admin-version/handler/v2"
	"github.com/go-book/api-version/example5-admin-version/middleware"
	"github.com/go-book/api-version/example5-admin-version/service"
)

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	userService := service.NewUserService()
	v1Handler := v1.NewUserHandler(userService)
	v2Handler := v2.NewUserHandler(userService)

	v1Group := r.Group("/api/v1")
	v1Group.Use(middleware.DeprecationMiddleware("v1", sunsetV1))
	{
		v1Group.GET("/users", v1Handler.ListUsers)
		v1Group.POST("/users", v1Handler.CreateUser)
	}

	v2Group := r.Group("/api/v2")
	{
		v2Group.GET("/users", v2Handler.ListUsers)
		v2Group.POST("/users", v2Handler.CreateUser)
	}
	return r
}

func TestV1CreateUserDoesNotRequirePhone(t *testing.T) {
	r := newTestRouter()
	body := strings.NewReader(`{"name":"Charlie","email":"charlie@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d, body=%s", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Deprecation") != "true" {
		t.Fatal("expected v1 response to include Deprecation header")
	}
}

func TestV2CreateUserRequiresPhone(t *testing.T) {
	r := newTestRouter()
	body := strings.NewReader(`{"name":"Charlie","email":"charlie@example.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/users", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", rec.Code, rec.Body.String())
	}
}
