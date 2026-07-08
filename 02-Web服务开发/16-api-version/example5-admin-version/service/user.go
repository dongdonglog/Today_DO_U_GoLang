package service

import (
	"sync"

	"github.com/go-book/api-version/example5-admin-version/model"
)

// UserService 用户服务
type UserService struct {
	mu    sync.RWMutex
	users []*model.User
}

// NewUserService 创建用户服务
func NewUserService() *UserService {
	return &UserService{
		users: []*model.User{
			{ID: 1, Name: "Alice", Email: "alice@example.com", Phone: "13800138000"},
			{ID: 2, Name: "Bob", Email: "bob@example.com", Phone: "13900139000"},
		},
	}
}

// List 获取用户列表
func (s *UserService) List() []*model.User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*model.User, len(s.users))
	for i, user := range s.users {
		copied := *user
		users[i] = &copied
	}
	return users
}

// Create 创建用户
func (s *UserService) Create(req *model.CreateUserInput) *model.User {
	s.mu.Lock()
	defer s.mu.Unlock()

	user := &model.User{
		ID:    len(s.users) + 1,
		Name:  req.Name,
		Email: req.Email,
		Phone: req.Phone,
	}
	s.users = append(s.users, user)

	copied := *user
	return &copied
}
