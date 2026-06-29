package user

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrEmailEmpty   = errors.New("email is required")
	ErrNameEmpty    = errors.New("name is required")
	ErrEmailInvalid = errors.New("email is invalid")
	ErrUserNotFound = errors.New("user not found")
)

type User struct {
	ID    int
	Name  string
	Email string
}

type UserStore interface {
	Save(user *User) error
	FindByID(id int) (*User, error)
	FindByEmail(email string) (*User, error)
	Delete(id int) error
}

type UserService struct {
	store UserStore
}

func NewUserService(store UserStore) *UserService {
	return &UserService{store: store}
}

func (s *UserService) CreateUser(name, email string) (*User, error) {
	if name == "" {
		return nil, ErrNameEmpty
	}
	if email == "" {
		return nil, ErrEmailEmpty
	}
	if !strings.Contains(email, "@") {
		return nil, ErrEmailInvalid
	}

	existing, err := s.store.FindByEmail(email)
	if err != nil && !errors.Is(err, ErrUserNotFound) {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("email already exists: %s", email)
	}

	user := &User{Name: name, Email: email}
	if err := s.store.Save(user); err != nil {
		return nil, fmt.Errorf("save user: %w", err)
	}
	return user, nil
}

func (s *UserService) GetUser(id int) (*User, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid user id: %d", id)
	}
	user, err := s.store.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	return user, nil
}

func (s *UserService) DeleteUser(id int) error {
	if id <= 0 {
		return fmt.Errorf("invalid user id: %d", id)
	}
	if err := s.store.Delete(id); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}
