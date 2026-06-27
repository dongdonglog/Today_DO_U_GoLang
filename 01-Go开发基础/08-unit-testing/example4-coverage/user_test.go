package user

import (
	"errors"
	"fmt"
	"testing"
)

type mockStore struct {
	users     map[int]*User
	saveErr   error
	findErr   error
	deleteErr error
}

func newMockStore() *mockStore {
	return &mockStore{users: make(map[int]*User)}
}

func (m *mockStore) Save(user *User) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	if user.ID == 0 {
		user.ID = len(m.users) + 1
	}
	m.users[user.ID] = user
	return nil
}

func (m *mockStore) FindByID(id int) (*User, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	user, ok := m.users[id]
	if !ok {
		return nil, fmt.Errorf("%w: id=%d", ErrUserNotFound, id)
	}
	return user, nil
}

func (m *mockStore) FindByEmail(email string) (*User, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, fmt.Errorf("%w: email=%s", ErrUserNotFound, email)
}

func (m *mockStore) Delete(id int) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if _, ok := m.users[id]; !ok {
		return fmt.Errorf("%w: id=%d", ErrUserNotFound, id)
	}
	delete(m.users, id)
	return nil
}

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name      string
		inputName string
		inputEmail string
		setup     func(*mockStore)
		wantErr   error
	}{
		{"正常创建", "Alice", "alice@example.com", func(s *mockStore) {}, nil},
		{"姓名为空", "", "alice@example.com", func(s *mockStore) {}, ErrNameEmpty},
		{"邮箱为空", "Alice", "", func(s *mockStore) {}, ErrEmailEmpty},
		{"邮箱格式错误", "Alice", "invalid", func(s *mockStore) {}, ErrEmailInvalid},
		{"保存失败", "Alice", "alice@example.com", func(s *mockStore) { s.saveErr = errors.New("db error") }, errors.New("save user")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockStore()
			tt.setup(store)
			svc := NewUserService(store)

			_, err := svc.CreateUser(tt.inputName, tt.inputEmail)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("Expected error containing %q, got nil", tt.wantErr.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGetUser(t *testing.T) {
	store := newMockStore()
	svc := NewUserService(store)

	created, _ := svc.CreateUser("Alice", "alice@example.com")

	user, err := svc.GetUser(created.ID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}
	if user.Name != "Alice" {
		t.Errorf("Expected name Alice, got %s", user.Name)
	}
}

func TestGetUser_NotFound(t *testing.T) {
	store := newMockStore()
	svc := NewUserService(store)

	_, err := svc.GetUser(999)
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

func TestDeleteUser(t *testing.T) {
	store := newMockStore()
	svc := NewUserService(store)

	created, _ := svc.CreateUser("Alice", "alice@example.com")

	err := svc.DeleteUser(created.ID)
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	_, err = svc.GetUser(created.ID)
	if err == nil {
		t.Error("Expected error after deletion")
	}
}
