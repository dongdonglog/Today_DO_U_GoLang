package main

import "fmt"

// 包级变量只能用 var
var globalVar = "global"
var globalInt int // 零值: 0

func main() {
	// ========================================
	// 1. 零值设计
	// ========================================
	fmt.Println("=== 零值设计 ===")

	var i int      // 零值: 0
	var s string   // 零值: ""
	var b bool     // 零值: false
	var p *int     // 零值: nil
	var slice []int // 零值: nil
	var m map[string]int // 零值: nil

	fmt.Printf("int: %d\n", i)
	fmt.Printf("string: %q\n", s)
	fmt.Printf("bool: %v\n", b)
	fmt.Printf("pointer: %v\n", p)
	fmt.Printf("slice: %v\n", slice)
	fmt.Printf("map: %v\n", m)

	// ========================================
	// 2. 短变量声明 :=
	// ========================================
	fmt.Println("\n=== 短变量声明 ===")

	// 短变量声明，类型自动推断
	name := "Alice"
	age := 30
	pi := 3.14159

	fmt.Printf("name: %s, age: %d, pi: %f\n", name, age, pi)

	// 多变量声明
	x, y := 1, 2
	fmt.Printf("x: %d, y: %d\n", x, y)

	// ========================================
	// 3. var vs := 的选择
	// ========================================
	fmt.Println("\n=== var vs := ===")

	// var: 包级变量、需要显式类型、零值初始化
	var count int = 100
	var piFloat float64 = 3.14159265358979

	// :=: 函数内局部变量、类型推断
	message := "hello"
	isActive := true

	fmt.Printf("count: %d, piFloat: %f\n", count, piFloat)
	fmt.Printf("message: %s, isActive: %v\n", message, isActive)

	// ========================================
	// 4. 类型断言
	// ========================================
	fmt.Println("\n=== 类型断言 ===")

	var any interface{} = "hello world"

	// 方式 1: 直接断言（失败会 panic）
	str := any.(string)
	fmt.Printf("直接断言: %s\n", str)

	// 方式 2: 安全断言（推荐）
	if str, ok := any.(string); ok {
		fmt.Printf("安全断言: %s\n", str)
	}

	// 方式 3: 类型开关
	switch v := any.(type) {
	case int:
		fmt.Printf("是 int: %d\n", v)
	case string:
		fmt.Printf("是 string: %s\n", v)
	case bool:
		fmt.Printf("是 bool: %v\n", v)
	default:
		fmt.Printf("未知类型: %T\n", v)
	}

	// ========================================
	// 5. 作用域陷阱（shadowing）
	// ========================================
	fmt.Println("\n=== 作用域陷阱 ===")

	outer := "outer"
	fmt.Printf("外层: %s\n", outer)

	{
		outer := "inner" // 新的变量，遮蔽外层
		fmt.Printf("内层: %s\n", outer)
	}

	fmt.Printf("外层（未变）: %s\n", outer)
}
