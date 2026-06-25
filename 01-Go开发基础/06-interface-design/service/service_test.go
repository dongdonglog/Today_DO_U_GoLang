package service

import (
	"fmt"
	"testing"
)

func TestCreateUser(t *testing.T) {
	// 使用 Mock 存储
	store := NewMockUserStore()
	service := NewUserService(store)

	// 测试创建用户
	user, err := service.CreateUser("Alice", "alice@example.com")
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
	store := NewMockUserStore()
	service := NewUserService(store)

	// 创建第一个用户
	_, err := service.CreateUser("Alice", "alice@example.com")
	if err != nil {
		t.Fatalf("First CreateUser failed: %v", err)
	}

	// 尝试创建相同邮箱的用户
	_, err = service.CreateUser("Bob", "alice@example.com")
	if err == nil {
		t.Error("Expected error for duplicate email")
	}
}

func TestCreateUserSaveError(t *testing.T) {
	store := NewMockUserStore()
	store.SetSaveError(fmt.Errorf("database error"))
	service := NewUserService(store)

	_, err := service.CreateUser("Alice", "alice@example.com")
	if err == nil {
		t.Error("Expected error when save fails")
	}
}

func TestGetUser(t *testing.T) {
	store := NewMockUserStore()
	service := NewUserService(store)

	// 先创建用户
	created, _ := service.CreateUser("Alice", "alice@example.com")

	// 获取用户
	user, err := service.GetUser(created.ID)
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
	store := NewMockUserStore()
	service := NewUserService(store)

	_, err := service.GetUser(999)
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

func TestDeleteUser(t *testing.T) {
	store := NewMockUserStore()
	service := NewUserService(store)

	// 先创建用户
	created, _ := service.CreateUser("Alice", "alice@example.com")

	// 删除用户
	err := service.DeleteUser(created.ID)
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	// 验证用户已被删除
	_, err = service.GetUser(created.ID)
	if err == nil {
		t.Error("Expected error after deletion")
	}
}

func TestDeleteUserNotFound(t *testing.T) {
	store := NewMockUserStore()
	service := NewUserService(store)

	err := service.DeleteUser(999)
	if err == nil {
		t.Error("Expected error for non-existent user")
	}
}

func TestWithMemoryStore(t *testing.T) {
	// 使用真实的内存存储
	store := NewMemoryUserStore()
	service := NewUserService(store)

	// 创建用户
	user, err := service.CreateUser("Alice", "alice@example.com")
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// 获取用户
	found, err := service.GetUser(user.ID)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	if found.Name != "Alice" {
		t.Errorf("Expected name Alice, got %s", found.Name)
	}

	// 删除用户
	err = service.DeleteUser(user.ID)
	if err != nil {
		t.Fatalf("DeleteUser failed: %v", err)
	}

	// 验证删除
	_, err = service.GetUser(user.ID)
	if err == nil {
		t.Error("Expected error after deletion")
	}
}
