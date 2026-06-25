package service

import "fmt"

// UserService 用户服务
type UserService struct {
	store UserStore // 依赖接口，而不是具体实现
}

// NewUserService 创建用户服务
func NewUserService(store UserStore) *UserService {
	return &UserService{store: store}
}

// CreateUser 创建用户
func (s *UserService) CreateUser(name, email string) (*User, error) {
	// 检查邮箱是否已存在
	existing, _ := s.store.FindByEmail(email)
	if existing != nil {
		return nil, fmt.Errorf("email already exists: %s", email)
	}

	user := &User{
		Name:  name,
		Email: email,
	}

	if err := s.store.Save(user); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	return user, nil
}

// GetUser 获取用户
func (s *UserService) GetUser(id int) (*User, error) {
	user, err := s.store.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return user, nil
}

// DeleteUser 删除用户
func (s *UserService) DeleteUser(id int) error {
	if err := s.store.Delete(id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}
