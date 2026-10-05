package store

import (
	"errors"
	"time"
)

var (
	ErrConflict     = errors.New("resource conflict")
	ErrInvalidToken = errors.New("invalid token")
	ErrNotFound     = errors.New("resource not found")
)

const (
	RoleAdmin = "admin"
	RoleUser  = "user"

	StatusActive   = "active"
	StatusDisabled = "disabled"
)

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	DisplayName  string    `json:"display_name"`
	Role         string    `json:"role"`
	Status       string    `json:"status"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
