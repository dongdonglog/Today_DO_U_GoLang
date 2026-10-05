package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-book/graduate/user-center/internal/auth"
	"github.com/go-book/graduate/user-center/internal/store"
)

type registerRequest struct {
	Email       string `json:"email" binding:"required,email"`
	DisplayName string `json:"display_name" binding:"required,min=2,max=80"`
	Password    string `json:"password" binding:"required,min=8,max=128"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (s *Server) register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		fail(c, http.StatusInternalServerError, "internal_error", "failed to hash password")
		return
	}
	user, err := s.db.CreateUser(c.Request.Context(), strings.ToLower(req.Email), req.DisplayName, passwordHash, store.RoleUser)
	if err != nil {
		failByError(c, err)
		return
	}
	created(c, user)
}

func (s *Server) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	email := strings.ToLower(req.Email)
	ip := c.ClientIP()
	failures, err := s.redis.LoginFailures(c.Request.Context(), email, ip)
	if err != nil {
		fail(c, http.StatusServiceUnavailable, "dependency_unavailable", "login state is unavailable")
		return
	}
	if failures >= s.cfg.LoginFailLimit {
		fail(c, http.StatusTooManyRequests, "too_many_attempts", "too many login attempts")
		return
	}

	user, err := s.db.GetUserByEmail(c.Request.Context(), email)
	if err != nil || !auth.CheckPassword(user.PasswordHash, req.Password) || user.Status != store.StatusActive {
		_ = s.redis.RecordLoginFailure(c.Request.Context(), email, ip, 15*time.Minute)
		fail(c, http.StatusUnauthorized, "invalid_credentials", "invalid email or password")
		return
	}
	_ = s.redis.ClearLoginFailures(c.Request.Context(), email, ip)
	s.issueTokens(c, user)
}

func (s *Server) refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	claims, err := s.jwt.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		fail(c, http.StatusUnauthorized, "invalid_token", "invalid or expired refresh token")
		return
	}
	user, err := s.db.GetUserByID(c.Request.Context(), claims.UserID)
	if err != nil || user.Status != store.StatusActive {
		fail(c, http.StatusUnauthorized, "invalid_token", "invalid or expired refresh token")
		return
	}
	accessToken, err := s.jwt.NewAccessToken(user.ID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "internal_error", "failed to issue token")
		return
	}
	refreshToken, err := s.jwt.NewRefreshToken(user.ID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "internal_error", "failed to issue token")
		return
	}
	err = s.db.RotateRefreshToken(
		c.Request.Context(),
		claims.ID,
		user.ID,
		auth.HashToken(req.RefreshToken),
		refreshToken.TokenID,
		refreshToken.TokenHash,
		refreshToken.ExpiresAt,
	)
	if err != nil {
		failByError(c, err)
		return
	}
	ok(c, gin.H{
		"access_token":          accessToken.Value,
		"access_token_expires":  accessToken.ExpiresAt,
		"refresh_token":         refreshToken.Value,
		"refresh_token_expires": refreshToken.ExpiresAt,
	})
}

func (s *Server) logout(c *gin.Context) {
	p, _ := getPrincipal(c)
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, http.StatusBadRequest, "bad_request", "invalid request body")
		return
	}
	claims, err := s.jwt.ParseRefreshToken(req.RefreshToken)
	if err != nil || claims.UserID != p.User.ID {
		fail(c, http.StatusUnauthorized, "invalid_token", "invalid or expired refresh token")
		return
	}
	if err := s.db.RevokeRefreshToken(c.Request.Context(), claims.ID, p.User.ID, auth.HashToken(req.RefreshToken)); err != nil {
		failByError(c, err)
		return
	}
	if err := s.redis.RevokeAccessToken(c.Request.Context(), p.AccessTokenID, time.Until(p.AccessExpiresAt)); err != nil {
		fail(c, http.StatusServiceUnavailable, "dependency_unavailable", "token state is unavailable")
		return
	}
	noContent(c)
}

func (s *Server) me(c *gin.Context) {
	p, _ := getPrincipal(c)
	ok(c, p.User)
}

func (s *Server) listUsers(c *gin.Context) {
	users, err := s.db.ListUsers(c.Request.Context())
	if err != nil {
		failByError(c, err)
		return
	}
	ok(c, users)
}

func (s *Server) disableUser(c *gin.Context) {
	s.setUserStatus(c, store.StatusDisabled)
}

func (s *Server) enableUser(c *gin.Context) {
	s.setUserStatus(c, store.StatusActive)
}

func (s *Server) setUserStatus(c *gin.Context, status string) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		fail(c, http.StatusBadRequest, "bad_request", "invalid user id")
		return
	}
	user, err := s.db.SetUserStatus(c.Request.Context(), id, status)
	if err != nil {
		failByError(c, err)
		return
	}
	ok(c, user)
}

func (s *Server) issueTokens(c *gin.Context, user *store.User) {
	accessToken, err := s.jwt.NewAccessToken(user.ID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "internal_error", "failed to issue token")
		return
	}
	refreshToken, err := s.jwt.NewRefreshToken(user.ID)
	if err != nil {
		fail(c, http.StatusInternalServerError, "internal_error", "failed to issue token")
		return
	}
	if err := s.db.SaveRefreshToken(c.Request.Context(), refreshToken.TokenID, user.ID, refreshToken.TokenHash, refreshToken.ExpiresAt); err != nil {
		fail(c, http.StatusInternalServerError, "internal_error", "failed to save token")
		return
	}
	ok(c, gin.H{
		"access_token":          accessToken.Value,
		"access_token_expires":  accessToken.ExpiresAt,
		"refresh_token":         refreshToken.Value,
		"refresh_token_expires": refreshToken.ExpiresAt,
		"user":                  user,
	})
}
