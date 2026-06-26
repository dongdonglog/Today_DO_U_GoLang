package service

import "fmt"

// MockUserStore Mock 用户存储
type MockUserStore struct {
	users     map[int]*User
	saveErr   error
	findErr   error
	deleteErr error
}

// NewMockUserStore 创建 Mock 存储
func NewMockUserStore() *MockUserStore {
	return &MockUserStore{
		users: make(map[int]*User),
	}
}

// SetSaveError 设置保存错误
func (m *MockUserStore) SetSaveError(err error) {
	m.saveErr = err
}

// SetFindError 设置查找错误
func (m *MockUserStore) SetFindError(err error) {
	m.findErr = err
}

// SetDeleteError 设置删除错误
func (m *MockUserStore) SetDeleteError(err error) {
	m.deleteErr = err
}

// Save 保存用户
func (m *MockUserStore) Save(user *User) error {
	if m.saveErr != nil {
		return m.saveErr
	}

	if user.ID == 0 {
		user.ID = len(m.users) + 1
	}
	m.users[user.ID] = user
	return nil
}

// FindByID 根据 ID 查找用户
func (m *MockUserStore) FindByID(id int) (*User, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}

	user, ok := m.users[id]
	if !ok {
		return nil, fmt.Errorf("%w: id=%d", ErrUserNotFound, id)
	}
	return user, nil
}

// FindByEmail 根据邮箱查找用户
func (m *MockUserStore) FindByEmail(email string) (*User, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}

	for _, user := range m.users {
		if user.Email == email {
			return user, nil
		}
	}
	return nil, fmt.Errorf("%w: email=%s", ErrUserNotFound, email)
}

// Delete 删除用户
func (m *MockUserStore) Delete(id int) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}

	if _, ok := m.users[id]; !ok {
		return fmt.Errorf("%w: id=%d", ErrUserNotFound, id)
	}
	delete(m.users, id)
	return nil
}
