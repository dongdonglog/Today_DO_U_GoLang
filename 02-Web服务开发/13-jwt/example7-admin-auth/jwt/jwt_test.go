package jwt

import (
	"strings"
	"testing"
	"time"

	golangjwt "github.com/golang-jwt/jwt/v5"
)

func newTestManager() *JWTManager {
	return NewJWTManager(strings.Repeat("a", 32), strings.Repeat("b", 32), 15*time.Minute, 7*24*time.Hour)
}

func TestGenerateAndParseAccessToken(t *testing.T) {
	manager := newTestManager()

	token, err := manager.GenerateAccessToken(1, "admin")
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}

	claims, err := manager.ParseAccessToken(token)
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	if claims.UserID != 1 || claims.Role != "admin" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestRejectsUnexpectedSigningMethod(t *testing.T) {
	manager := newTestManager()

	token := golangjwt.NewWithClaims(golangjwt.SigningMethodHS384, AccessClaims{UserID: 1, Role: "admin"})
	tokenString, err := token.SignedString([]byte(strings.Repeat("a", 32)))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := manager.ParseAccessToken(tokenString); err == nil {
		t.Fatalf("expected unexpected signing method error")
	}
}

func TestGenerateRefreshTokenHasID(t *testing.T) {
	manager := newTestManager()

	token, err := manager.GenerateRefreshToken(1)
	if err != nil {
		t.Fatalf("generate refresh token: %v", err)
	}
	if token.TokenID == "" {
		t.Fatalf("expected refresh token id")
	}

	claims, err := manager.ParseRefreshToken(token.Token)
	if err != nil {
		t.Fatalf("parse refresh token: %v", err)
	}
	if claims.ID != token.TokenID {
		t.Fatalf("expected jti %s, got %s", token.TokenID, claims.ID)
	}
}
