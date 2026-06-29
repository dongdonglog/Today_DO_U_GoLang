package user

import (
	"errors"
	"fmt"
	"strings"
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

func newTestUser() *User {
	return &User{Name: "TestUser", Email: "test@example.com"}
}

func TestCreateUser_Success(t *testing.T) {
	store := newMockStore()
	svc := NewUserService(store)

	user, err := svc.CreateUser("Alice", "alice@example.com")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.Name != "Alice" {
		t.Errorf("Expected name Alice, got %s", user.Name)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("Expected email alice@example.com, got %s", user.Email)
	}
}

func TestCreateUser_ValidationErrors(t *testing.T) {
	tests := []struct {
		name       string
		inputName  string
		inputEmail string
		wantErr    error
	}{
		{"姓名为空", "", "alice@example.com", ErrNameEmpty},
		{"邮箱为空", "Alice", "", ErrEmailEmpty},
		{"邮箱格式错误", "Alice", "invalid-email", ErrEmailInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockStore()
			svc := NewUserService(store)

			_, err := svc.CreateUser(tt.inputName, tt.inputEmail)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Expected error %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestCreateUser_StoreErrors(t *testing.T) {
	tests := []struct {
		name            string
		setup           func(*mockStore)
		wantErrContains string
	}{
		{
			name: "查找邮箱失败",
			setup: func(s *mockStore) {
				s.findErr = errors.New("database connection error")
			},
			wantErrContains: "check email",
		},
		{
			name: "保存失败",
			setup: func(s *mockStore) {
				s.saveErr = errors.New("database write error")
			},
			wantErrContains: "save user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockStore()
			tt.setup(store)
			svc := NewUserService(store)

			_, err := svc.CreateUser("Alice", "alice@example.com")
			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErrContains) {
				t.Errorf("Expected error containing %q, got %q", tt.wantErrContains, err.Error())
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
	if user.ID != created.ID {
		t.Errorf("Expected ID %d, got %d", created.ID, user.ID)
	}
}

func TestGetUser_InvalidID(t *testing.T) {
	store := newMockStore()
	svc := NewUserService(store)

	_, err := svc.GetUser(0)
	if err == nil {
		t.Error("Expected error for ID 0")
	}

	_, err = svc.GetUser(-1)
	if err == nil {
		t.Error("Expected error for negative ID")
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

func TestDeleteUser_InvalidID(t *testing.T) {
	store := newMockStore()
	svc := NewUserService(store)

	err := svc.DeleteUser(0)
	if err == nil {
		t.Error("Expected error for ID 0")
	}
}
