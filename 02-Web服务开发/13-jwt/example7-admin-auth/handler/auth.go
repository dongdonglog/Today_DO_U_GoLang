package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/go-book/jwt/example7-admin-auth/jwt"
	"github.com/go-book/jwt/example7-admin-auth/model"
	"github.com/go-book/jwt/example7-admin-auth/response"
	"github.com/go-book/jwt/example7-admin-auth/store"
)

type AuthHandler struct {
	store      *store.MemoryStore
	jwtManager *jwt.JWTManager
}

func NewAuthHandler(store *store.MemoryStore, jwtManager *jwt.JWTManager) *AuthHandler {
	return &AuthHandler{
		store:      store,
		jwtManager: jwtManager,
	}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	err := h.store.VerifyPassword(req.Username, req.Password)
	if err != nil {
		response.Unauthorized(c, "invalid credentials")
		return
	}

	user, err := h.store.GetUserByUsername(req.Username)
	if err != nil {
		response.InternalError(c, "failed to get user")
		return
	}

	accessToken, err := h.jwtManager.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		response.InternalError(c, "failed to generate access token")
		return
	}

	refreshToken, err := h.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		response.InternalError(c, "failed to generate refresh token")
		return
	}
	h.store.SaveRefreshToken(refreshToken.TokenID, user.ID, refreshToken.ExpiresAt)

	response.Success(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken.Token,
		"user":          user,
	})
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req model.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	claims, err := h.jwtManager.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		response.Unauthorized(c, "invalid or expired refresh token")
		return
	}

	user, err := h.store.GetUserByID(claims.UserID)
	if err != nil {
		response.Unauthorized(c, "user not found")
		return
	}

	accessToken, err := h.jwtManager.GenerateAccessToken(user.ID, user.Role)
	if err != nil {
		response.InternalError(c, "failed to generate access token")
		return
	}

	refreshToken, err := h.jwtManager.GenerateRefreshToken(user.ID)
	if err != nil {
		response.InternalError(c, "failed to generate refresh token")
		return
	}

	if err := h.store.RotateRefreshToken(claims.ID, user.ID, refreshToken.TokenID, refreshToken.ExpiresAt); err != nil {
		response.Unauthorized(c, "invalid or expired refresh token")
		return
	}

	response.Success(c, gin.H{
		"access_token":  accessToken,
		"refresh_token": refreshToken.Token,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req model.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err)
		return
	}

	claims, err := h.jwtManager.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		response.Unauthorized(c, "invalid or expired refresh token")
		return
	}

	if err := h.store.RevokeRefreshToken(claims.ID, claims.UserID); err != nil {
		response.Unauthorized(c, "invalid or expired refresh token")
		return
	}

	response.NoContent(c)
}
