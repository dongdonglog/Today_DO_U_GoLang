package main

import (
	"errors"
	"fmt"
)

// ========================================
// 1. 哨兵错误（Sentinel Errors）
// ========================================

// 全局哨兵错误
var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrConflict     = errors.New("conflict")
)

// ========================================
// 2. 自定义错误类型
// ========================================

// ValidationError 验证错误
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed: field=%s, message=%s", e.Field, e.Message)
}

// 实现 Unwrap 方法（Go 1.13+）
func (e *ValidationError) Unwrap() error {
	return ErrValidation
}

// 基础验证错误
var ErrValidation = errors.New("validation error")

// ========================================
// 3. 带上下文的错误类型
// ========================================

// NotFoundError 404 错误
type NotFoundError struct {
	Resource string
	ID       string
	Err      error
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: id=%s", e.Resource, e.ID)
}

func (e *NotFoundError) Unwrap() error {
	return e.Err
}

// ========================================
// 4. 业务错误类型
// ========================================

// BusinessError 业务错误
type BusinessError struct {
	Code    int    // 业务错误码
	Message string // 用户友好的错误信息
	Err     error  // 原始错误
}

func (e *BusinessError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *BusinessError) Unwrap() error {
	return e.Err
}

// ========================================
// 5. 使用示例
// ========================================

func getUser(id string) (*User, error) {
	if id == "" {
		return nil, &ValidationError{
			Field:   "id",
			Message: "user id is required",
		}
	}

	// 模拟数据库查询
	if id == "999" {
		return nil, &NotFoundError{
			Resource: "user",
			ID:       id,
			Err:      ErrNotFound,
		}
	}

	return &User{ID: id, Name: "Alice"}, nil
}

type User struct {
	ID   string
	Name string
}

func main() {
	// ========================================
	// 1. 哨兵错误判断
	// ========================================
	fmt.Println("=== 哨兵错误 ===")

	err := checkPermission(false)
	if errors.Is(err, ErrUnauthorized) {
		fmt.Println("未授权访问")
	}
	if errors.Is(err, ErrForbidden) {
		fmt.Println("禁止访问")
	}

	// ========================================
	// 2. 自定义错误类型
	// ========================================
	fmt.Println("\n=== 自定义错误类型 ===")

	_, err = getUser("")
	if err != nil {
		var valErr *ValidationError
		if errors.As(err, &valErr) {
			fmt.Printf("验证错误: field=%s, message=%s\n", valErr.Field, valErr.Message)
		}
		fmt.Printf("完整错误: %v\n", err)
	}

	// ========================================
	// 3. NotFoundError
	// ========================================
	fmt.Println("\n=== NotFoundError ===")

	_, err = getUser("999")
	if err != nil {
		var notFound *NotFoundError
		if errors.As(err, &notFound) {
			fmt.Printf("资源: %s, ID: %s\n", notFound.Resource, notFound.ID)
		}
		fmt.Printf("完整错误: %v\n", err)
	}

	// ========================================
	// 4. 业务错误
	// ========================================
	fmt.Println("\n=== 业务错误 ===")

	err = processOrder("order-123")
	if err != nil {
		var bizErr *BusinessError
		if errors.As(err, &bizErr) {
			fmt.Printf("错误码: %d\n", bizErr.Code)
			fmt.Printf("用户提示: %s\n", bizErr.Message)
		}
		fmt.Printf("完整错误: %v\n", err)
	}

	// ========================================
	// 5. 错误链
	// ========================================
	fmt.Println("\n=== 错误链 ===")

	_, err = getUser("999")
	if err != nil {
		// 检查是否包含 ErrNotFound
		if errors.Is(err, ErrNotFound) {
			fmt.Println("错误链中包含 ErrNotFound")
		}

		// 检查是否包含 ErrValidation
		if !errors.Is(err, ErrValidation) {
			fmt.Println("错误链中不包含 ErrValidation")
		}
	}
}

func checkPermission(hasPermission bool) error {
	if !hasPermission {
		return fmt.Errorf("access denied: %w", ErrUnauthorized)
	}
	return nil
}

func processOrder(orderID string) error {
	// 模拟库存不足
	err := checkStock(orderID)
	if err != nil {
		return &BusinessError{
			Code:    1001,
			Message: "库存不足，请稍后重试",
			Err:     err,
		}
	}
	return nil
}

func checkStock(orderID string) error {
	return errors.New("stock is 0")
}
