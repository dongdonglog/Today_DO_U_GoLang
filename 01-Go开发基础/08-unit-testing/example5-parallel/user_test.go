package user

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

type mockStore struct {
	mu        sync.Mutex
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
	m.mu.Lock()
	defer m.mu.Unlock()
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
	m.mu.Lock()
	defer m.mu.Unlock()
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
	m.mu.Lock()
	defer m.mu.Unlock()
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
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[id]; !ok {
		return fmt.Errorf("%w: id=%d", ErrUserNotFound, id)
	}
	delete(m.users, id)
	return nil
}

func TestCreateUser_Parallel(t *testing.T) {
	tests := []struct {
		name       string
		inputName  string
		inputEmail string
		wantErr    error
	}{
		{"正常创建_Alice", "Alice", "alice@example.com", nil},
		{"正常创建_Bob", "Bob", "bob@example.com", nil},
		{"姓名为空", "", "charlie@example.com", ErrNameEmpty},
		{"邮箱格式错误", "Dave", "invalid-email", ErrEmailInvalid},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := newMockStore()
			svc := NewUserService(store)

			_, err := svc.CreateUser(tt.inputName, tt.inputEmail)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGetUser_Parallel(t *testing.T) {
	store := newMockStore()
	svc := NewUserService(store)

	for i := 1; i <= 5; i++ {
		svc.CreateUser(fmt.Sprintf("User%d", i), fmt.Sprintf("user%d@example.com", i))
	}

	for i := 1; i <= 5; i++ {
		i := i
		t.Run(fmt.Sprintf("GetUser_%d", i), func(t *testing.T) {
			t.Parallel()

			user, err := svc.GetUser(i)
			if err != nil {
				t.Fatalf("GetUser(%d) failed: %v", i, err)
			}
			expectedName := fmt.Sprintf("User%d", i)
			if user.Name != expectedName {
				t.Errorf("Expected name %q, got %q", expectedName, user.Name)
			}
		})
	}
}
