package store

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-book/gin-framework/example7-admin-api/model"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

// MemoryStore 内存存储
type MemoryStore struct {
	mu     sync.RWMutex
	users  map[int]*model.User
	nextID int
}

// NewMemoryStore 创建内存存储
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users:  make(map[int]*model.User),
		nextID: 1,
	}
}

// List 获取用户列表
func (s *MemoryStore) List(page, size int) ([]*model.User, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]*model.User, 0, len(s.users))
	for _, u := range s.users {
		all = append(all, u)
	}

	total := len(all)

	// 分页
	start := (page - 1) * size
	if start >= total {
		return []*model.User{}, total
	}
	end := start + size
	if end > total {
		end = total
	}

	return all[start:end], total
}

// GetByID 根据 ID 获取用户
func (s *MemoryStore) GetByID(id int) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[id]
	if !ok {
		return nil, fmt.Errorf("%w: %d", ErrUserNotFound, id)
	}
	return user, nil
}

// GetByEmail 根据邮箱获取用户
func (s *MemoryStore) GetByEmail(email string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, u := range s.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrUserNotFound, email)
}

// Create 创建用户
func (s *MemoryStore) Create(name, email string) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查邮箱是否已存在
	for _, u := range s.users {
		if u.Email == email {
			return nil, fmt.Errorf("%w: %s", ErrUserExists, email)
		}
	}

	now := time.Now()
	user := &model.User{
		ID:        s.nextID,
		Name:      name,
		Email:     email,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.users[s.nextID] = user
	s.nextID++

	return user, nil
}

// Update 更新用户
func (s *MemoryStore) Update(id int, name, email *string) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[id]
	if !ok {
		return nil, fmt.Errorf("%w: %d", ErrUserNotFound, id)
	}

	if name != nil {
		user.Name = *name
	}
	if email != nil {
		// 检查邮箱是否已被其他用户使用
		for _, u := range s.users {
			if u.ID != id && u.Email == *email {
				return nil, fmt.Errorf("%w: %s", ErrUserExists, *email)
			}
		}
		user.Email = *email
	}
	user.UpdatedAt = time.Now()

	return user, nil
}

// Delete 删除用户
func (s *MemoryStore) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[id]; !ok {
		return fmt.Errorf("%w: %d", ErrUserNotFound, id)
	}
	delete(s.users, id)
	return nil
}
