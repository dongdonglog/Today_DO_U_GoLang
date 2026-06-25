package main

import "fmt"

// ========================================
// 1. 指针基础
// ========================================

func pointerBasics() {
	fmt.Println("=== 指针基础 ===")

	i := 42
	p := &i // p 是指向 i 的指针

	fmt.Printf("i 的值: %d\n", i)
	fmt.Printf("i 的地址: %p\n", &i)
	fmt.Printf("p 的值（地址）: %p\n", p)
	fmt.Printf("p 指向的值: %d\n", *p)

	// 修改指针指向的值
	*p = 100
	fmt.Printf("修改后 i 的值: %d\n", i)
}

// ========================================
// 2. 为什么 Go 保留指针？
// ========================================

// 大结构体
type LargeStruct struct {
	data [1000]int
}

// 值传递：拷贝整个结构体（慢）
func processByValue(s LargeStruct) {
	// 拷贝了 1000 * 8 = 8000 字节
	fmt.Printf("值传递: %d\n", s.data[0])
}

// 指针传递：只拷贝 8 字节（快）
func processByPointer(s *LargeStruct) {
	// 只拷贝了指针（8 字节）
	fmt.Printf("指针传递: %d\n", s.data[0])
}

func whyPointer() {
	fmt.Println("\n=== 为什么 Go 保留指针？===")

	large := LargeStruct{}
	large.data[0] = 42

	// 值传递（慢）
	processByValue(large)

	// 指针传递（快）
	processByPointer(&large)
}

// ========================================
// 3. 共享数据
// ========================================

func sharedData() {
	fmt.Println("\n=== 共享数据 ===")

	// 多个变量指向同一数据
	data := []int{1, 2, 3}
	ref1 := &data
	ref2 := &data

	// 通过 ref1 修改
	(*ref1)[0] = 99

	// ref2 也能看到修改
	fmt.Printf("data: %v\n", data)
	fmt.Printf("ref1: %v\n", *ref1)
	fmt.Printf("ref2: %v\n", *ref2)
}

// ========================================
// 4. 方法接收者
// ========================================

type Counter struct {
	value int
}

// 值接收者：不会修改原始对象
func (c Counter) GetValue() int {
	return c.value
}

// 值接收者：修改的是副本
func (c Counter) IncrementByValue() {
	c.value++ // 修改副本，原始对象不变
}

// 指针接收者：会修改原始对象
func (c *Counter) IncrementByPointer() {
	c.value++ // 修改原始对象
}

func methodReceiver() {
	fmt.Println("\n=== 方法接收者 ===")

	c := &Counter{value: 0}

	c.IncrementByValue()
	fmt.Printf("值接收者 Increment 后: %d\n", c.value) // 仍然是 0

	c.IncrementByPointer()
	fmt.Printf("指针接收者 Increment 后: %d\n", c.value) // 变成 1
}

// ========================================
// 5. 什么时候用指针？
// ========================================

func whenToUsePointer() {
	fmt.Println("\n=== 什么时候用指针？===")

	// 1. 需要修改原始值
	fmt.Println("1. 需要修改原始值:")
	x := 10
	p := &x
	*p = 20
	fmt.Printf("  x = %d\n", x)

	// 2. 大结构体（避免拷贝）
	fmt.Println("2. 大结构体（避免拷贝）:")
	_ = LargeStruct{} // 示例：创建大结构体
	fmt.Printf("  LargeStruct 大小: %d 字节\n", 1000*8)
	fmt.Printf("  指针大小: %d 字节\n", 8)

	// 3. 方法需要修改接收者
	fmt.Println("3. 方法需要修改接收者:")
	c := &Counter{value: 0}
	c.IncrementByPointer()
	fmt.Printf("  修改后: %d\n", c.value)

	// 4. 实现接口（方法集）
	fmt.Println("4. 实现接口（方法集）:")
	fmt.Println("  *T 的方法集包含 T 的所有方法")
	fmt.Println("  T 的方法集不包含 *T 的指针方法")

	// 5. 避免 slice/map 的 nil 问题
	fmt.Println("5. 避免 nil 问题:")
	var m map[string]int
	fmt.Printf("  nil map: %v\n", m)
	// m["key"] = 1 // panic!

	m = make(map[string]int)
	m["key"] = 1
	fmt.Printf("  初始化后: %v\n", m)
}

// ========================================
// 6. 指针的指针
// ========================================

func pointerToPointer() {
	fmt.Println("\n=== 指针的指针 ===")

	i := 42
	p := &i
	pp := &p

	fmt.Printf("i = %d\n", i)
	fmt.Printf("*p = %d\n", *p)
	fmt.Printf("**pp = %d\n", **pp)

	// 修改
	**pp = 100
	fmt.Printf("修改后 i = %d\n", i)
}

// ========================================
// 7. new 函数
// ========================================

func newFunction() {
	fmt.Println("\n=== new 函数 ===")

	// new 分配内存并返回指针
	p1 := new(int)
	fmt.Printf("new(int): %p, 值: %d\n", p1, *p1)

	*p1 = 42
	fmt.Printf("赋值后: %d\n", *p1)

	// 对比 make
	s1 := make([]int, 5) // 初始化内部数据结构，返回 slice（不是指针）
	fmt.Printf("make([]int): %v\n", s1)

	// new 用于结构体
	type User struct {
		Name string
		Age  int
	}

	u1 := new(User) // 等价于 &User{}
	fmt.Printf("new(User): %+v\n", u1)

	u2 := &User{} // 更常用的写法
	fmt.Printf("&User{}: %+v\n", u2)
}

// ========================================
// 8. 实战：链表节点
// ========================================

type Node struct {
	value int
	next  *Node
}

func linkedList() {
	fmt.Println("\n=== 实战：链表节点 ===")

	// 创建链表
	head := &Node{value: 1}
	second := &Node{value: 2}
	third := &Node{value: 3}

	head.next = second
	second.next = third

	// 遍历链表
	fmt.Print("链表: ")
	current := head
	for current != nil {
		fmt.Printf("%d -> ", current.value)
		current = current.next
	}
	fmt.Println("nil")
}

func main() {
	pointerBasics()
	whyPointer()
	sharedData()
	methodReceiver()
	whenToUsePointer()
	pointerToPointer()
	newFunction()
	linkedList()
}
