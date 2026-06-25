package main

import "fmt"

// ========================================
// 1. 空接口基础
// ========================================

func emptyInterfaceBasics() {
	fmt.Println("=== 空接口基础 ===")

	// 空接口可以存储任何值
	var any interface{}

	any = 42
	fmt.Printf("int: %v (type: %T)\n", any, any)

	any = "hello"
	fmt.Printf("string: %v (type: %T)\n", any, any)

	any = []int{1, 2, 3}
	fmt.Printf("slice: %v (type: %T)\n", any, any)

	any = map[string]int{"a": 1}
	fmt.Printf("map: %v (type: %T)\n", any, any)
}

// ========================================
// 2. 类型断言
// ========================================

func typeAssertion() {
	fmt.Println("\n=== 类型断言 ===")

	var any interface{} = "hello"

	// 不安全的类型断言
	str := any.(string)
	fmt.Printf("string: %s\n", str)

	// 安全的类型断言
	if num, ok := any.(int); ok {
		fmt.Printf("int: %d\n", num)
	} else {
		fmt.Println("Not an int")
	}

	// 类型开关
	switch v := any.(type) {
	case int:
		fmt.Printf("int: %d\n", v)
	case string:
		fmt.Printf("string: %s\n", v)
	case bool:
		fmt.Printf("bool: %v\n", v)
	default:
		fmt.Printf("unknown: %T\n", v)
	}
}

// ========================================
// 3. fmt.Println 的实现
// ========================================

func fmtPrintlnImplementation() {
	fmt.Println("\n=== fmt.Println 实现 ===")

	// fmt.Println 使用 interface{} 作为参数
	// 可以接受任何类型
	fmt.Println(42)
	fmt.Println("hello")
	fmt.Println([]int{1, 2, 3})
	fmt.Println(map[string]int{"a": 1})
}

// ========================================
// 4. 泛型 vs 空接口
// ========================================

// 使用空接口
func maxEmpty(a, b interface{}) interface{} {
	// 需要类型断言
	aInt, aOk := a.(int)
	bInt, bOk := b.(int)
	if !aOk || !bOk {
		return nil
	}
	if aInt > bInt {
		return aInt
	}
	return bInt
}

// 使用泛型（Go 1.18+）
func maxGeneric[T int | float64](a, b T) T {
	if a > b {
		return a
	}
	return b
}

func genericVsEmpty() {
	fmt.Println("\n=== 泛型 vs 空接口 ===")

	// 空接口版本
	result := maxEmpty(10, 20)
	fmt.Printf("maxEmpty: %v (type: %T)\n", result, result)

	// 泛型版本
	resultInt := maxGeneric(10, 20)
	fmt.Printf("maxGeneric(int): %v (type: %T)\n", resultInt, resultInt)

	resultFloat := maxGeneric(10.5, 20.3)
	fmt.Printf("maxGeneric(float): %v (type: %T)\n", resultFloat, resultFloat)
}

// ========================================
// 5. 空接口的陷阱
// ========================================

func emptyInterfacePitfalls() {
	fmt.Println("\n=== 空接口的陷阱 ===")

	// 陷阱 1：性能开销
	// 每次使用空接口都会发生装箱/拆箱
	var any interface{}
	any = 42 // 装箱：int -> interface{}
	num := any.(int) // 拆箱：interface{} -> int
	fmt.Printf("num: %d\n", num)

	// 陷阱 2：类型安全
	// 编译时无法检查类型
	any = "hello"
	// result := any.(int) // 运行时 panic

	// 陷阱 3：无法使用 == 比较
	// 切片、map、函数不能比较
	any = []int{1, 2, 3}
	// fmt.Println(any == any) // panic
}

// ========================================
// 6. 空接口的底层结构
// ========================================

func emptyInterfaceStructure() {
	fmt.Println("\n=== 空接口的底层结构 ===")

	// 空接口（eface）的结构：
	// type eface struct {
	//     _type *_type  // 类型信息
	//     data  unsafe.Pointer  // 数据指针
	// }

	var any interface{} = 42
	fmt.Printf("value: %v\n", any)
	fmt.Printf("type: %T\n", any)

	// 空接口零值
	var nilAny interface{}
	fmt.Printf("nil interface: %v\n", nilAny == nil)
}

func main() {
	emptyInterfaceBasics()
	typeAssertion()
	fmtPrintlnImplementation()
	genericVsEmpty()
	emptyInterfacePitfalls()
	emptyInterfaceStructure()
}
