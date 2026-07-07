package main

import (
	"fmt"
	"log"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// 密钥
var jwtSecret = []byte("your-secret-key")

// 自定义 Claims
type UserClaims struct {
	UserID int    `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// 生成 Token
func GenerateToken(userID int, role string) (string, error) {
	claims := UserClaims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "my-app",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// 解析 Token
func ParseToken(tokenString string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*UserClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func main() {
	// 生成 Token
	userID := 1
	role := "admin"
	tokenString, err := GenerateToken(userID, role)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("生成的 Token:\n%s\n\n", tokenString)

	// 解析 Token
	claims, err := ParseToken(tokenString)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("解析结果:\n")
	fmt.Printf("  UserID: %d\n", claims.UserID)
	fmt.Printf("  Role: %s\n", claims.Role)
	fmt.Printf("  Issuer: %s\n", claims.Issuer)
	fmt.Printf("  ExpiresAt: %v\n", claims.ExpiresAt.Time)

	// 验证过期 Token
	fmt.Println("\n--- 测试过期 Token ---")
	expiredClaims := UserClaims{
		UserID: 2,
		Role:   "user",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // 已过期
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredTokenString, _ := expiredToken.SignedString(jwtSecret)

	_, err = ParseToken(expiredTokenString)
	if err != nil {
		fmt.Printf("过期 Token 验证失败（预期行为）: %v\n", err)
	}

	// 验证无效签名
	fmt.Println("\n--- 测试无效签名 ---")
	_, err = ParseToken(tokenString + "invalid")
	if err != nil {
		fmt.Printf("无效签名验证失败（预期行为）: %v\n", err)
	}
}
