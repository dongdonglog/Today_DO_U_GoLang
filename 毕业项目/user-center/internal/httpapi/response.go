package httpapi

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-book/graduate/user-center/internal/store"
)

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"data": data})
}

func created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, gin.H{"data": data})
}

func noContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func fail(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": apiError{Code: code, Message: message}})
}

func failByError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, store.ErrConflict):
		fail(c, http.StatusConflict, "conflict", "resource already exists")
	case errors.Is(err, store.ErrNotFound):
		fail(c, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, store.ErrInvalidToken):
		fail(c, http.StatusUnauthorized, "invalid_token", "invalid or expired token")
	default:
		fail(c, http.StatusInternalServerError, "internal_error", "request failed")
	}
}
