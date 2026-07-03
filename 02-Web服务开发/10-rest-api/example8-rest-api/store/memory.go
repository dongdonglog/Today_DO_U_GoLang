package store

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-book/rest-api/example8-rest-api/model"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)

// MemoryStore 内存存储
type MemoryStore struct {
	mu              sync.RWMutex
	users           map[int]*model.User
	idempotencyKeys map[string]int
	nextID          int
}

// NewMemoryStore 创建内存存储
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users:           make(map[int]*model.User),
		idempotencyKeys: make(map[string]int),
		nextID:          1,
	}
}

// List 获取用户列表（支持分页、过滤、排序）
func (s *MemoryStore) List(page, size int, nameFilter, sortBy, order string) ([]*model.User, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 过滤
	filtered := make([]*model.User, 0)
	for _, u := range s.users {
		if nameFilter != "" && !strings.Contains(strings.ToLower(u.Name), strings.ToLower(nameFilter)) {
			continue
		}
		filtered = append(filtered, u)
	}

	// 排序
	if sortBy != "" {
		sort.Slice(filtered, func(i, j int) bool {
			var less bool
			switch sortBy {
			case "name":
				less = filtered[i].Name < filtered[j].Name
			case "email":
				less = filtered[i].Email < filtered[j].Email
			case "created_at":
				less = filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
			default:
				less = filtered[i].ID < filtered[j].ID
			}
			if order == "desc" {
				return !less
			}
			return less
		})
	}

	total := len(filtered)

	// 分页
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}
	start := (page - 1) * size
	if start >= total {
		return []*model.User{}, total
	}
	end := start + size
	if end > total {
		end = total
	}

	return filtered[start:end], total
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
	return s.CreateWithIdempotencyKey("", name, email)
}

// CreateWithIdempotencyKey 创建用户，同一个幂等键重复提交时返回第一次创建的资源。
func (s *MemoryStore) CreateWithIdempotencyKey(key, name, email string) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if key != "" {
		if userID, ok := s.idempotencyKeys[key]; ok {
			return s.users[userID], nil
		}
	}

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
	if key != "" {
		s.idempotencyKeys[key] = s.nextID
	}
	s.nextID++

	return user, nil
}

// Update 全量更新用户
func (s *MemoryStore) Update(id int, name, email string) (*model.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[id]
	if !ok {
		return nil, fmt.Errorf("%w: %d", ErrUserNotFound, id)
	}

	// 检查邮箱是否已被其他用户使用
	if email != user.Email {
		for _, u := range s.users {
			if u.ID != id && u.Email == email {
				return nil, fmt.Errorf("%w: %s", ErrUserExists, email)
			}
		}
	}

	user.Name = name
	user.Email = email
	user.UpdatedAt = time.Now()

	return user, nil
}

// Patch 部分更新用户
func (s *MemoryStore) Patch(id int, name, email *string) (*model.User, error) {
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
		if *email != user.Email {
			for _, u := range s.users {
				if u.ID != id && u.Email == *email {
					return nil, fmt.Errorf("%w: %s", ErrUserExists, *email)
				}
			}
			user.Email = *email
		}
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
