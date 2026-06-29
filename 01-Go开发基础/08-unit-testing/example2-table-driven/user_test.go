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

func TestCreateUser_TableDriven(t *testing.T) {
	tests := []struct {
		name            string
		inputName       string
		inputEmail      string
		setup           func(*mockStore)
		wantErrIs       error
		wantErrContains string
		wantName        string
		wantEmail       string
	}{
		{
			name:       "正常创建",
			inputName:  "Alice",
			inputEmail: "alice@example.com",
			setup:      func(s *mockStore) {},
			wantName:   "Alice",
			wantEmail:  "alice@example.com",
		},
		{
			name:       "姓名为空",
			inputName:  "",
			inputEmail: "bob@example.com",
			setup:      func(s *mockStore) {},
			wantErrIs:  ErrNameEmpty,
		},
		{
			name:       "邮箱为空",
			inputName:  "Bob",
			inputEmail: "",
			setup:      func(s *mockStore) {},
			wantErrIs:  ErrEmailEmpty,
		},
		{
			name:       "邮箱格式错误",
			inputName:  "Bob",
			inputEmail: "invalid-email",
			setup:      func(s *mockStore) {},
			wantErrIs:  ErrEmailInvalid,
		},
		{
			name:       "邮箱已存在",
			inputName:  "Bob",
			inputEmail: "alice@example.com",
			setup: func(s *mockStore) {
				s.Save(&User{Name: "Alice", Email: "alice@example.com"})
			},
			wantErrContains: "email already exists",
		},
		{
			name:       "保存失败",
			inputName:  "Alice",
			inputEmail: "alice@example.com",
			setup: func(s *mockStore) {
				s.saveErr = errors.New("database error")
			},
			wantErrContains: "save user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockStore()
			tt.setup(store)
			svc := NewUserService(store)

			user, err := svc.CreateUser(tt.inputName, tt.inputEmail)

			if tt.wantErrIs != nil {
				if err == nil {
					t.Fatalf("Expected error %v, got nil", tt.wantErrIs)
				}
				if !errors.Is(err, tt.wantErrIs) {
					t.Errorf("Expected error %v, got %v", tt.wantErrIs, err)
				}
				return
			}
			if tt.wantErrContains != "" {
				if err == nil {
					t.Fatalf("Expected error containing %q, got nil", tt.wantErrContains)
				}
				if !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Errorf("Expected error containing %q, got %q", tt.wantErrContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if user.Name != tt.wantName {
				t.Errorf("Expected name %q, got %q", tt.wantName, user.Name)
			}
			if user.Email != tt.wantEmail {
				t.Errorf("Expected email %q, got %q", tt.wantEmail, user.Email)
			}
		})
	}
}

func TestGetUser_TableDriven(t *testing.T) {
	tests := []struct {
		name     string
		id       int
		setup    func(*mockStore)
		wantErr  bool
		wantName string
	}{
		{
			name: "正常获取",
			id:   1,
			setup: func(s *mockStore) {
				s.Save(&User{ID: 1, Name: "Alice", Email: "alice@example.com"})
			},
			wantErr:  false,
			wantName: "Alice",
		},
		{
			name:    "用户不存在",
			id:      999,
			setup:   func(s *mockStore) {},
			wantErr: true,
		},
		{
			name:    "无效 ID",
			id:      0,
			setup:   func(s *mockStore) {},
			wantErr: true,
		},
		{
			name:    "负数 ID",
			id:      -1,
			setup:   func(s *mockStore) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newMockStore()
			tt.setup(store)
			svc := NewUserService(store)

			user, err := svc.GetUser(tt.id)

			if tt.wantErr {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if user.Name != tt.wantName {
				t.Errorf("Expected name %q, got %q", tt.wantName, user.Name)
			}
		})
	}
}
