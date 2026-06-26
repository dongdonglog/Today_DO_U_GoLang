package service

import (
	"fmt"
	"sync"
)

// MemoryUserStore 内存用户存储
type MemoryUserStore struct {
	mu     sync.RWMutex
	users  map[int]*User
	nextID int
}

// NewMemoryUserStore 创建内存存储
func NewMemoryUserStore() *MemoryUserStore {
	return &MemoryUserStore{
		users:  make(map[int]*User),
		nextID: 1,
	}
}

// Save 保存用户
func (m *MemoryUserStore) Save(user *User) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if user.ID == 0 {
		user.ID = m.nextID
		m.nextID++
	}

	m.users[user.ID] = user
	return nil
}

// FindByID 根据 ID 查找用户
func (m *MemoryUserStore) FindByID(id int) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	user, ok := m.users[id]
	if !ok {
		return nil, fmt.Errorf("%w: id=%d", ErrUserNotFound, id)
	}
	return user, nil
}

// FindByEmail 根据邮箱查找用户
func (m *MemoryUserStore) FindByEmail(email string) (*User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, fmt.Errorf("%w: email=%s", ErrUserNotFound, email)
}

// Delete 删除用户
func (m *MemoryUserStore) Delete(id int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.users[id]; !ok {
		return fmt.Errorf("%w: id=%d", ErrUserNotFound, id)
	}
	delete(m.users, id)
	return nil
}
