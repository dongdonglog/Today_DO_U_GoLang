package main

import "fmt"

// ========================================
// 1. 接口定义
// ========================================

// Animal 动物接口
type Animal interface {
	Speak() string
}

// Dog 狗
type Dog struct {
	Name string
}

// Speak 实现 Animal 接口
func (d Dog) Speak() string {
	return "Woof!"
}

// Cat 猫
type Cat struct {
	Name string
}

// Speak 实现 Animal 接口
func (c Cat) Speak() string {
	return "Meow!"
}

// Bird 鸟
type Bird struct {
	Name string
}

// Speak 实现 Animal 接口
func (b Bird) Speak() string {
	return "Tweet!"
}

// ========================================
// 2. 隐式实现
// ========================================

// 不需要 implements 关键字
// 只要实现了 Speak() string 方法，就自动实现了 Animal 接口

// ========================================
// 3. 接口使用
// ========================================

func makeAnimalSpeak(animal Animal) {
	fmt.Printf("%s says %s\n", animal, animal.Speak())
}

// ========================================
// 4. 接口零值
// ========================================

func interfaceZeroValue() {
	fmt.Println("\n=== 接口零值 ===")

	var animal Animal
	fmt.Printf("animal == nil: %v\n", animal == nil)

	// 接口零值的陷阱
	var dog *Dog
	animal = dog // dog 是 nil，但 animal 不是 nil
	fmt.Printf("animal == nil: %v\n", animal == nil)
	// fmt.Println(animal.Speak()) // panic: nil pointer dereference
}

// ========================================
// 5. 多态
// ========================================

func polymorphism() {
	fmt.Println("\n=== 多态 ===")

	animals := []Animal{
		Dog{Name: "Buddy"},
		Cat{Name: "Whiskers"},
		Bird{Name: "Tweety"},
	}

	for _, animal := range animals {
		fmt.Println(animal.Speak())
	}
}

// ========================================
// 6. 接口断言
// ========================================

func interfaceAssertion() {
	fmt.Println("\n=== 接口断言 ===")

	var animal Animal = Dog{Name: "Buddy"}

	// 类型断言
	if dog, ok := animal.(Dog); ok {
		fmt.Printf("It's a dog: %s\n", dog.Name)
	}

	// 类型开关
	switch v := animal.(type) {
	case Dog:
		fmt.Printf("Dog: %s\n", v.Name)
	case Cat:
		fmt.Printf("Cat: %s\n", v.Name)
	case Bird:
		fmt.Printf("Bird: %s\n", v.Name)
	default:
		fmt.Printf("Unknown animal\n")
	}
}

func main() {
	// 基本使用
	fmt.Println("=== 基本使用 ===")
	dog := Dog{Name: "Buddy"}
	cat := Cat{Name: "Whiskers"}
	bird := Bird{Name: "Tweety"}

	fmt.Println(dog.Speak())
	fmt.Println(cat.Speak())
	fmt.Println(bird.Speak())

	// 接口作为参数
	fmt.Println("\n=== 接口作为参数 ===")
	makeAnimalSpeak(dog)
	makeAnimalSpeak(cat)
	makeAnimalSpeak(bird)

	// 接口零值
	interfaceZeroValue()

	// 多态
	polymorphism()

	// 接口断言
	interfaceAssertion()
}
