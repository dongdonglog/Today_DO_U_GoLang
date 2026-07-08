package store

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-book/jwt/example7-admin-auth/model"
	"golang.org/x/crypto/bcrypt"
)

type MemoryStore struct {
	mu            sync.RWMutex
	users         map[string]*model.User
	refreshTokens map[string]*RefreshTokenRecord
	nextID        int
}

type RefreshTokenRecord struct {
	TokenID   string
	UserID    int
	ExpiresAt time.Time
	Revoked   bool
}

func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		users:         make(map[string]*model.User),
		refreshTokens: make(map[string]*RefreshTokenRecord),
		nextID:        1,
	}

	// 初始化一些测试用户
	store.CreateUser("admin", "admin123", "admin", "admin@example.com")
	store.CreateUser("alice", "123456", "user", "alice@example.com")
	store.CreateUser("guest", "guest123", "guest", "guest@example.com")

	return store
}

func (s *MemoryStore) CreateUser(username, password, role, email string) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; exists {
		return nil, fmt.Errorf("username already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user := &model.User{
		ID:        s.nextID,
		Username:  username,
		Password:  string(hashedPassword),
		Role:      role,
		Email:     email,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.users[username] = user
	s.nextID++

	return user, nil
}

func (s *MemoryStore) GetUserByUsername(username string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[username]
	if !exists {
		return nil, fmt.Errorf("user not found")
	}

	return user, nil
}

func (s *MemoryStore) GetUserByID(id int) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		if user.ID == id {
			return user, nil
		}
	}

	return nil, fmt.Errorf("user not found")
}

func (s *MemoryStore) ListUsers() []*model.User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*model.User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}

	return users
}

func (s *MemoryStore) DeleteUser(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for username, user := range s.users {
		if user.ID == id {
			delete(s.users, username)
			return nil
		}
	}

	return fmt.Errorf("user not found")
}

func (s *MemoryStore) VerifyPassword(username, password string) error {
	user, err := s.GetUserByUsername(username)
	if err != nil {
		return err
	}

	return bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
}

func (s *MemoryStore) SaveRefreshToken(tokenID string, userID int, expiresAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.refreshTokens[tokenID] = &RefreshTokenRecord{
		TokenID:   tokenID,
		UserID:    userID,
		ExpiresAt: expiresAt,
	}
}

func (s *MemoryStore) RotateRefreshToken(oldTokenID string, userID int, newTokenID string, newExpiresAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldToken, ok := s.refreshTokens[oldTokenID]
	if !ok {
		return fmt.Errorf("refresh token not found")
	}
	if oldToken.Revoked {
		return fmt.Errorf("refresh token revoked")
	}
	if oldToken.UserID != userID {
		return fmt.Errorf("refresh token user mismatch")
	}
	if time.Now().After(oldToken.ExpiresAt) {
		return fmt.Errorf("refresh token expired")
	}

	oldToken.Revoked = true
	s.refreshTokens[newTokenID] = &RefreshTokenRecord{
		TokenID:   newTokenID,
		UserID:    userID,
		ExpiresAt: newExpiresAt,
	}
	return nil
}

func (s *MemoryStore) RevokeRefreshToken(tokenID string, userID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, ok := s.refreshTokens[tokenID]
	if !ok {
		return fmt.Errorf("refresh token not found")
	}
	if token.UserID != userID {
		return fmt.Errorf("refresh token user mismatch")
	}
	token.Revoked = true
	return nil
}
