package main

import (
	"errors"
	"fmt"
)

// ========================================
// 1. 错误包装（Wrapping）
// ========================================

var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
)

// 模拟数据库层
func findUserInDB(id string) (string, error) {
	if id == "999" {
		return "", fmt.Errorf("user %s: %w", id, ErrNotFound)
	}
	return "Alice", nil
}

// 模拟服务层
func getUserFromService(id string) (string, error) {
	user, err := findUserInDB(id)
	if err != nil {
		// 使用 %w 包装错误，保留错误链
		return "", fmt.Errorf("getUserFromService failed: %w", err)
	}
	return user, nil
}

// 模拟处理器层
func handleGetUser(id string) (string, error) {
	user, err := getUserFromService(id)
	if err != nil {
		// 继续包装
		return "", fmt.Errorf("handleGetUser failed: %w", err)
	}
	return user, nil
}

// ========================================
// 2. errors.Is - 检查错误链
// ========================================

func checkWithIs(id string) {
	_, err := handleGetUser(id)
	if err != nil {
		// 检查错误链中是否包含特定错误
		if errors.Is(err, ErrNotFound) {
			fmt.Printf("用户 %s 不存在\n", id)
		} else if errors.Is(err, ErrUnauthorized) {
			fmt.Printf("未授权访问\n")
		} else {
			fmt.Printf("其他错误: %v\n", err)
		}
	}
}

// ========================================
// 3. errors.As - 提取错误类型
// ========================================

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation: %s - %s", e.Field, e.Message)
}

func processInput(input string) error {
	if input == "" {
		return &ValidationError{
			Field:   "input",
			Message: "cannot be empty",
		}
	}
	return nil
}

func handleInput(input string) error {
	err := processInput(input)
	if err != nil {
		// 包装错误
		return fmt.Errorf("handleInput failed: %w", err)
	}
	return nil
}

func checkWithAs(input string) {
	err := handleInput(input)
	if err != nil {
		// 提取特定类型的错误
		var valErr *ValidationError
		if errors.As(err, &valErr) {
			fmt.Printf("验证失败: field=%s, message=%s\n", valErr.Field, valErr.Message)
		} else {
			fmt.Printf("其他错误: %v\n", err)
		}
	}
}

// ========================================
// 4. %w vs %v 的区别
// ========================================

func compareWrapping() {
	fmt.Println("\n=== %w vs %v ===")

	// 使用 %w（可解包）
	err1 := fmt.Errorf("layer1: %w", ErrNotFound)
	err2 := fmt.Errorf("layer2: %w", err1)

	fmt.Printf("使用 %%w:\n")
	fmt.Printf("  errors.Is(err2, ErrNotFound) = %v\n", errors.Is(err2, ErrNotFound))

	// 使用 %v（不可解包）
	err3 := fmt.Errorf("layer1: %v", ErrNotFound)
	err4 := fmt.Errorf("layer2: %v", err3)

	fmt.Printf("使用 %%v:\n")
	fmt.Printf("  errors.Is(err4, ErrNotFound) = %v\n", errors.Is(err4, ErrNotFound))
}

// ========================================
// 5. 多层错误链
// ========================================

func multiLayerChain() {
	fmt.Println("\n=== 多层错误链 ===")

	// 构建错误链
	err := errors.New("root cause")
	err = fmt.Errorf("layer 1: %w", err)
	err = fmt.Errorf("layer 2: %w", err)
	err = fmt.Errorf("layer 3: %w", err)

	fmt.Printf("完整错误: %v\n", err)
	fmt.Printf("解包一层: %v\n", errors.Unwrap(err))
	fmt.Printf("解包两层: %v\n", errors.Unwrap(errors.Unwrap(err)))

	// 检查错误链
	fmt.Printf("\nerrors.Is(err, root): %v\n", errors.Is(err, errors.New("root cause")))
	// 注意：errors.Is 比较的是错误值，不是字符串
}

// ========================================
// 6. 自定义 Unwrap
// ========================================

type MultiError struct {
	Errors []error
}

func (e *MultiError) Error() string {
	return fmt.Sprintf("multiple errors: %d errors occurred", len(e.Errors))
}

// 实现 Unwrap 返回错误切片（Go 1.20+）
func (e *MultiError) Unwrap() []error {
	return e.Errors
}

func multiErrorDemo() {
	fmt.Println("\n=== 多错误解包（Go 1.20+）===")

	err := &MultiError{
		Errors: []error{
			ErrNotFound,
			ErrUnauthorized,
		},
	}

	fmt.Printf("多错误: %v\n", err)
	fmt.Printf("errors.Is(err, ErrNotFound): %v\n", errors.Is(err, ErrNotFound))
	fmt.Printf("errors.Is(err, ErrUnauthorized): %v\n", errors.Is(err, ErrUnauthorized))
}

func main() {
	// ========================================
	// errors.Is 示例
	// ========================================
	fmt.Println("=== errors.Is ===")
	checkWithIs("999")
	checkWithIs("1")

	// ========================================
	// errors.As 示例
	// ========================================
	fmt.Println("\n=== errors.As ===")
	checkWithAs("")
	checkWithAs("valid")

	// ========================================
	// %w vs %v
	// ========================================
	compareWrapping()

	// ========================================
	// 多层错误链
	// ========================================
	multiLayerChain()

	// ========================================
	// 多错误解包
	// ========================================
	multiErrorDemo()
}
