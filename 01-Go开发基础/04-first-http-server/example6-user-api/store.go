package main

import (
	"fmt"
	"sync"
)

type UserStore interface {
	Get(id int) (*User, error)
	List() []*User
	Create(user *User) error
	Update(id int, name, email string) (*User, error)
	Delete(id int) error
}

type MemoryStore struct {
	mu     sync.RWMutex
	users  map[int]User
	nextID int
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		users:  make(map[int]User),
		nextID: 1,
	}
}

func (s *MemoryStore) Get(id int) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[id]
	if !ok {
		return nil, fmt.Errorf("user %d not found", id)
	}
	return &user, nil
}

func (s *MemoryStore) List() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]*User, 0, len(s.users))
	for _, u := range s.users {
		u := u
		list = append(list, &u)
	}
	return list
}

func (s *MemoryStore) Create(user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user.ID = s.nextID
	s.users[s.nextID] = *user
	s.nextID++
	return nil
}

func (s *MemoryStore) Update(id int, name, email string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[id]
	if !ok {
		return nil, fmt.Errorf("user %d not found", id)
	}

	if name != "" {
		user.Name = name
	}
	if email != "" {
		user.Email = email
	}
	s.users[id] = user
	return &user, nil
}

func (s *MemoryStore) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.users[id]; !ok {
		return fmt.Errorf("user %d not found", id)
	}
	delete(s.users, id)
	return nil
}
