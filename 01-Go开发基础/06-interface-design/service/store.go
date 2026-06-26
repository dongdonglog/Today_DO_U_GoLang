package service

import "errors"

var ErrUserNotFound = errors.New("user not found")

// UserStore 用户存储接口
// 由使用方（service 包）定义，而不是由实现方定义
type UserStore interface {
	Save(user *User) error
	FindByID(id int) (*User, error)
	FindByEmail(email string) (*User, error)
	Delete(id int) error
}
