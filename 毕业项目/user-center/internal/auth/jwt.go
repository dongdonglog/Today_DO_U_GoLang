package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type AccessClaims struct {
	UserID int64  `json:"user_id"`
	Kind   string `json:"kind"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	UserID int64  `json:"user_id"`
	Kind   string `json:"kind"`
	jwt.RegisteredClaims
}

type Token struct {
	Value     string
	TokenID   string
	TokenHash string
	ExpiresAt time.Time
}

type Manager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

func NewManager(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

func (m *Manager) NewAccessToken(userID int64) (*Token, error) {
	tokenID, err := randomID()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	expiresAt := now.Add(m.accessTTL)
	claims := AccessClaims{
		UserID: userID,
		Kind:   "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			Subject:   fmt.Sprintf("%d", userID),
			Issuer:    "go-book-user-center",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	value, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.accessSecret)
	if err != nil {
		return nil, err
	}
	return &Token{Value: value, TokenID: tokenID, ExpiresAt: expiresAt}, nil
}

func (m *Manager) NewRefreshToken(userID int64) (*Token, error) {
	tokenID, err := randomID()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	expiresAt := now.Add(m.refreshTTL)
	claims := RefreshClaims{
		UserID: userID,
		Kind:   "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        tokenID,
			Subject:   fmt.Sprintf("%d", userID),
			Issuer:    "go-book-user-center",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	value, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.refreshSecret)
	if err != nil {
		return nil, err
	}
	return &Token{Value: value, TokenID: tokenID, TokenHash: HashToken(value), ExpiresAt: expiresAt}, nil
}

func (m *Manager) ParseAccessToken(value string) (*AccessClaims, error) {
	token, err := jwt.ParseWithClaims(value, &AccessClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return m.accessSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*AccessClaims)
	if !ok || !token.Valid || claims.Kind != "access" || claims.ExpiresAt == nil || claims.ID == "" {
		return nil, fmt.Errorf("invalid access token")
	}
	return claims, nil
}

func (m *Manager) ParseRefreshToken(value string) (*RefreshClaims, error) {
	token, err := jwt.ParseWithClaims(value, &RefreshClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return m.refreshSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*RefreshClaims)
	if !ok || !token.Valid || claims.Kind != "refresh" || claims.ExpiresAt == nil || claims.ID == "" {
		return nil, fmt.Errorf("invalid refresh token")
	}
	return claims, nil
}

func HashToken(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func randomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
