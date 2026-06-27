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
	store := newMockStore()
	svc := NewUserService(store)

	user, err := svc.CreateUser("Alice", "alice@example.com")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	if user.ID == 0 {
		t.Error("User ID should not be 0")
	}
	if user.Name != "Alice" {
		t.Errorf("Expected name Alice, got %s", user.Name)
	}
	if user.Email != "alice@example.com" {
		t.Errorf("Expected email alice@example.com, got %s", user.Email)
	}
}

func TestCreateUserDuplicateEmail(t *testing.T) {
	store := newMockStore()
	svc := NewUserService(store)

	_, err := svc.CreateUser("Alice", "alice@example.com")
	if err != nil {
		t.Fatalf("First CreateUser failed: %v", err)
	}

	_, err = svc.CreateUser("Bob", "alice@example.com")
	if err == nil {
		t.Error("Expected error for duplicate email")
	}
}

func TestCreateUserSaveError(t *testing.T) {
	store := newMockStore()
	store.saveErr = errors.New("database error")
	svc := NewUserService(store)

	_, err := svc.CreateUser("Alice", "alice@example.com")
	if err == nil {
		t.Error("Expected error when save fails")
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
	if user.Name != "Alice" {
		t.Errorf("Expected name Alice, got %s", user.Name)
	}
}

func TestGetUserNotFound(t *testing.T) {
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
