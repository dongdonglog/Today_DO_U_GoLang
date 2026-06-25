# 第 6 章 Interface 设计思想

## 场景

你读 Go 标准库源码，发现：
- `io.Reader` 只有一个方法：`Read(p []byte) (n int, err error)`
- `http.Handler` 只有一个方法：`ServeHTTP(ResponseWriter, *Request)`
- `error` 只有一个方法：`Error() string`

为什么 Go 的接口都这么小？
为什么 Go 不需要 `implements` 关键字？
为什么"接受接口，返回结构体"是最佳实践？

本章从设计哲学出发，讲透 Go 接口的精髓。

---

## 6.1 接口基础

> 代码：`example1-basic/main.go`

### 6.1.1 接口定义与实现

```go
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
```

**隐式实现**：不需要 `implements` 关键字，只要实现了接口定义的所有方法，就自动实现了该接口。

```go
var animal Animal = Dog{Name: "Buddy"}
fmt.Println(animal.Speak()) // Woof!
```

**对比 Java/C# 的显式实现**：

```java
// Java
class Dog implements Animal {
    @Override
    public String speak() {
        return "Woof!";
    }
}
```

Go 的隐式实现更灵活：
- 可以为已有类型添加接口实现（只要方法匹配）
- 不需要修改原有代码
- 解耦更彻底

### 6.1.2 接口是契约

接口定义行为，不定义实现。

```go
// Reader 读取接口
type Reader interface {
    Read(p []byte) (n int, err error)
}
```

`Reader` 只定义了"可以读取"这个行为，不关心：
- 从哪里读（文件、网络、内存）
- 怎么读（阻塞、非阻塞）
- 读什么格式（文本、二进制）

**面向接口编程 vs 面向实现编程**：

```go
// 面向接口编程（推荐）
func copyData(dst io.Writer, src io.Reader) error {
    buf := make([]byte, 1024)
    for {
        n, err := src.Read(buf)
        if n > 0 {
            _, err = dst.Write(buf[:n])
            if err != nil {
                return err
            }
        }
        if err != nil {
            if err == io.EOF {
                break
            }
            return err
        }
    }
    return nil
}

// 可以复制任何 Reader 到任何 Writer
copyData(os.Stdout, os.Stdin)        // 标准输入到标准输出
copyData(file, networkConn)          // 文件到网络
copyData(&bytes.Buffer{}, strings.NewReader("hello")) // 字符串到缓冲区
```

### 6.1.3 接口零值

接口零值是 `nil`。

```go
var animal Animal
fmt.Println(animal == nil) // true
```

**nil 接口的陷阱**：

```go
var dog *Dog          // dog 是 nil
var animal Animal = dog // animal 不是 nil！

fmt.Println(dog == nil)    // true
fmt.Println(animal == nil) // false！

// animal 包含：
// - 类型信息：*Dog（不为 nil）
// - 值信息：nil
```

**一张图：接口的底层结构**：

```
┌─────────────────────────────────────┐
│  接口变量 (interface)                │
├─────────────────────────────────────┤
│  type: *Dog (类型信息)               │
│  data: nil (值指针)                  │
└─────────────────────────────────────┘
```

接口包含两部分：
- `type`：具体类型（`*Dog`）
- `data`：具体值（`nil`）

只有当 `type` 和 `data` 都为 `nil` 时，接口才是 `nil`。

### 深入：接口的底层结构

Go 有两种接口：

**空接口 `interface{}`（eface）**：

```go
type eface struct {
    _type *_type         // 类型信息
    data  unsafe.Pointer // 数据指针
}
```

**非空接口（iface）**：

```go
type iface struct {
    tab  *itab          // 接口表
    data unsafe.Pointer // 数据指针
}

type itab struct {
    inter *interfacetype // 接口类型
    _type *_type         // 具体类型
    link  *itab          // 链表
    bad   int32
    inhash int32
    fun   [1]uintptr     // 方法表（可变长）
}
```

**为什么接口调用有性能开销？**

1. 需要通过 `itab` 查找方法地址
2. 无法内联优化
3. 需要类型检查

但在大多数场景下，这个开销可以忽略不计。

---

## 6.2 小接口原则

### 6.2.1 为什么 Go 接口这么小

Go 标准库中，最常见的接口只有 1-2 个方法：

```go
// 1 个方法
type Reader interface { Read(p []byte) (n int, err error) }
type Writer interface { Write(p []byte) (n int, err error) }
type Closer interface { Close() error }
type Stringer interface { String() string }
type Handler interface { ServeHTTP(ResponseWriter, *Request) }
type error interface { Error() string }

// 2 个方法
type ReadWriter interface { Reader; Writer }
type ReadCloser interface { Reader; Closer }
```

**对比 Java 的大接口**：

```java
// Java List 接口：40+ 方法
public interface List<E> {
    int size();
    boolean isEmpty();
    boolean contains(Object o);
    Iterator<E> iterator();
    Object[] toArray();
    boolean add(E e);
    boolean remove(Object o);
    // ... 还有 30+ 方法
}
```

**Rob Pike 的名言**：

> "The bigger the interface, the weaker the abstraction."
> 
> 接口越大，抽象越弱。

### 6.2.2 小接口的优势

1. **容易实现**：实现 1 个方法比实现 20 个方法简单
2. **容易组合**：小接口可以组合成大接口
3. **容易测试**：Mock 1 个方法比 Mock 20 个方法简单
4. **容易理解**：小接口语义清晰

### 6.2.3 标准库的小接口

```go
// io.Reader：可以读取
type Reader interface {
    Read(p []byte) (n int, err error)
}

// io.Writer：可以写入
type Writer interface {
    Write(p []byte) (n int, err error)
}

// io.Closer：可以关闭
type Closer interface {
    Close() error
}

// fmt.Stringer：可以转字符串
type Stringer interface {
    String() string
}

// http.Handler：可以处理 HTTP 请求
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}

// error：表示错误
type error interface {
    Error() string
}
```

### 深入：为什么小接口更好？

**接口隔离原则（ISP）**：

> Clients should not be forced to depend upon interfaces that they do not use.
> 
> 客户端不应该被迫依赖它不使用的接口。

**正交性**：小接口更容易正交组合。

```go
// 正交组合
type ReadWriter interface {
    Reader
    Writer
}

type ReadCloser interface {
    Reader
    Closer
}

type WriteCloser interface {
    Writer
    Closer
}

type ReadWriteCloser interface {
    Reader
    Writer
    Closer
}
```

**组合爆炸问题**：

如果有一个大接口 `BigInterface`（20 个方法），要实现它的子集，需要定义 2^20 = 1,048,576 个接口。

但如果拆分成 20 个小接口，只需要定义 20 个接口，通过组合可以满足大部分需求。

---

## 6.3 接口组合

> 代码：`example2-composition/main.go`

### 6.3.1 组合小接口

```go
// Reader 读取接口
type Reader interface {
    Read(p []byte) (n int, err error)
}

// Writer 写入接口
type Writer interface {
    Write(p []byte) (n int, err error)
}

// Closer 关闭接口
type Closer interface {
    Close() error
}

// ReadWriter 读写接口
type ReadWriter interface {
    Reader
    Writer
}

// ReadCloser 读关闭接口
type ReadCloser interface {
    Reader
    Closer
}

// ReadWriteCloser 读写关闭接口
type ReadWriteCloser interface {
    Reader
    Writer
    Closer
}
```

**组合 vs 继承**：

```go
// Go：组合
type ReadWriter interface {
    Reader
    Writer
}

// Java：继承
interface ReadWriter extends Reader, Writer {
}
```

Go 的组合更灵活：
- 可以选择性组合
- 不需要继承层次结构
- 更容易重构

### 6.3.2 标准库的组合示例

```go
// io 包中的组合
type ReadWriter interface { Reader; Writer }
type ReadCloser interface { Reader; Closer }
type WriteCloser interface { Writer; Closer }
type ReadWriteCloser interface { Reader; Writer; Closer }
type ReadWriteSeeker interface { Reader; Writer; Seeker }
```

### 6.3.3 自定义组合

```go
// Logger 日志接口
type Logger interface {
    Log(msg string)
}

// Flusher 刷新接口
type Flusher interface {
    Flush() error
}

// LogFlusher 日志刷新接口
type LogFlusher interface {
    Logger
    Flusher
}

// ConsoleLogger 实现
type ConsoleLogger struct{}

func (c *ConsoleLogger) Log(msg string) {
    fmt.Printf("[LOG] %s\n", msg)
}

func (c *ConsoleLogger) Flush() error {
    fmt.Println("[FLUSH] Flushing logs")
    return nil
}

// 使用
var lf LogFlusher = &ConsoleLogger{}
lf.Log("test message")
lf.Flush()
```

---

## 6.4 接口最佳实践

### 6.4.1 接受接口，返回结构体

```go
// 推荐：返回结构体
func NewUserStore() *MemoryStore {
    return &MemoryStore{}
}

// 不推荐：返回接口
func NewUserStore() UserStore {
    return &MemoryStore{}
}
```

**为什么？**

1. **灵活性**：调用方可以决定是否需要接口
2. **可扩展性**：可以访问结构体的特定方法
3. **性能**：避免接口开销

```go
// 调用方决定是否需要接口
store := NewUserStore() // 返回 *MemoryStore

// 需要接口时，显式转换
var userStore UserStore = store

// 不需要接口时，直接使用
store.Save(user)
store.FindByID(1)
```

### 6.4.2 在需要时定义接口

```go
// 反模式：提前定义接口
type UserStore interface {
    Save(user *User) error
    FindByID(id int) (*User, error)
}

type MemoryStore struct {
    // ...
}

// 推荐：当有两个以上实现时再抽象
type MemoryStore struct {
    // ...
}

// 当需要 MySQLStore 时，再定义接口
type UserStore interface {
    Save(user *User) error
    FindByID(id int) (*User, error)
}
```

**"Don't design interfaces, discover them"**

> 不要设计接口，发现接口。

接口应该从具体实现中抽象出来，而不是提前设计。

### 6.4.3 接口由使用方定义

```go
// 库提供结构体
package database

type MySQLStore struct {
    // ...
}

func (m *MySQLStore) Save(user *User) error {
    // ...
}

// 使用方定义接口
package service

type UserStore interface {
    Save(user *User) error
}

type UserService struct {
    store UserStore // 依赖接口
}

func NewUserService(store UserStore) *UserService {
    return &UserService{store: store}
}
```

**为什么这样更好？**

1. **解耦**：库不需要知道使用方需要什么
2. **灵活**：使用方可以定义自己需要的接口
3. **可测试**：容易 Mock

### 6.4.4 避免在接口中使用指针接收者

**方法集规则**：

| 类型 | 方法集 |
|------|--------|
| `T` | 值接收者方法 |
| `*T` | 值接收者 + 指针接收者方法 |

```go
type Animal interface {
    Speak() string
}

type Dog struct {
    Name string
}

// 值接收者
func (d Dog) Speak() string {
    return "Woof!"
}

// 指针接收者
func (d *Dog) SetName(name string) {
    d.Name = name
}

var dog Dog
var animal Animal = dog  // OK：Dog 实现了 Animal

var pDog *Dog = &Dog{}
var pAnimal Animal = pDog  // OK：*Dog 也实现了 Animal
```

**最佳实践**：

- 接口方法尽量用值接收者
- 如果必须修改状态，用指针接收者
- 记住方法集规则

---

## 6.5 常见接口模式

> 代码：`example3-patterns/main.go`

### 6.5.1 Reader/Writer 模式

```go
// 读取并转大写
type UpperCaseReader struct {
    r io.Reader
}

func (u *UpperCaseReader) Read(p []byte) (n int, err error) {
    n, err = u.r.Read(p)
    for i := 0; i < n; i++ {
        if p[i] >= 'a' && p[i] <= 'z' {
            p[i] -= 32 // 转大写
        }
    }
    return
}

// 使用
src := strings.NewReader("hello world")
upper := &UpperCaseReader{r: src}

buf := make([]byte, 1024)
n, _ := upper.Read(buf)
fmt.Printf("Upper: %s\n", buf[:n]) // HELLO WORLD
```

**管道与过滤器模式**：

```go
// 可以组合多个 Reader/Writer
src := strings.NewReader("hello")
upper := &UpperCaseReader{r: src}
lower := &LowerCaseReader{r: upper}
// ...
```

### 6.5.2 Handler 模式

```go
// Handler 处理器接口
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}

// HandlerFunc 函数适配器
type HandlerFunc func(ResponseWriter, *Request)

func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
    f(w, r)
}

// 中间件
func LoggingMiddleware(next Handler) Handler {
    return HandlerFunc(func(w ResponseWriter, r *Request) {
        fmt.Printf("[LOG] %s %s\n", r.Method, r.Path)
        next.ServeHTTP(w, r)
    })
}

func AuthMiddleware(next Handler) Handler {
    return HandlerFunc(func(w ResponseWriter, r *Request) {
        fmt.Println("[AUTH] Checking authentication")
        next.ServeHTTP(w, r)
    })
}

// 使用
handler := HandlerFunc(func(w ResponseWriter, r *Request) {
    w.Write([]byte("Hello, World!"))
})

handler = LoggingMiddleware(handler)
handler = AuthMiddleware(handler)
```

### 6.5.3 Stringer 模式

```go
// Point 点
type Point struct {
    X, Y int
}

// String 实现 fmt.Stringer
func (p Point) String() string {
    return fmt.Sprintf("(%d, %d)", p.X, p.Y)
}

// 使用
p := Point{X: 10, Y: 20}
fmt.Printf("Point: %v\n", p) // Point: (10, 20)
```

### 6.5.4 Sort 模式

```go
// Sortable 可排序接口
type Sortable interface {
    Len() int
    Less(i, j int) bool
    Swap(i, j int)
}

// IntSlice 整数切片
type IntSlice []int

func (s IntSlice) Len() int           { return len(s) }
func (s IntSlice) Less(i, j int) bool { return s[i] < s[j] }
func (s IntSlice) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }

// 排序
func Sort(data Sortable) {
    // 排序算法
}

// 使用
nums := IntSlice{5, 2, 8, 1, 9}
Sort(nums)
```

---

## 6.6 空接口 interface{}

> 代码：`example4-empty/main.go`

### 6.6.1 空接口的用途

空接口可以存储任何值。

```go
var any interface{}

any = 42
any = "hello"
any = []int{1, 2, 3}
any = map[string]int{"a": 1}
```

**使用场景**：

```go
// fmt.Println 使用 interface{}
func Println(a ...interface{}) (n int, err error)

// container/list 使用 interface{}
type Element struct {
    Value interface{}
    // ...
}
```

### 6.6.2 类型断言

```go
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
```

### 6.6.3 泛型 vs 空接口

```go
// 使用空接口
func maxEmpty(a, b interface{}) interface{} {
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
```

**对比**：

| 特性 | 空接口 | 泛型 |
|------|--------|------|
| 类型安全 | 运行时检查 | 编译时检查 |
| 性能 | 装箱/拆箱开销 | 零开销 |
| 代码简洁 | 需要类型断言 | 直接使用 |
| 灵活性 | 可以存储任何类型 | 受类型约束限制 |

**什么时候用泛型，什么时候用 interface{}？**

- **用泛型**：类型明确，需要类型安全
- **用 interface{}**： truly 任意类型（如 `fmt.Println`）

### 深入：空接口的底层结构

```go
// 空接口（eface）
type eface struct {
    _type *_type         // 类型信息
    data  unsafe.Pointer // 数据指针
}
```

**为什么空接口调用有开销？**

1. 需要类型检查
2. 需要装箱/拆箱
3. 无法内联优化

---

## 6.7 接口与多态

### 6.7.1 Go 的多态实现

```go
type Animal interface {
    Speak() string
}

type Dog struct{ Name string }
func (d Dog) Speak() string { return "Woof!" }

type Cat struct{ Name string }
func (c Cat) Speak() string { return "Meow!" }

func makeAnimalSpeak(animal Animal) {
    fmt.Println(animal.Speak())
}

// 多态
makeAnimalSpeak(Dog{Name: "Buddy"})   // Woof!
makeAnimalSpeak(Cat{Name: "Whiskers"}) // Meow!
```

**对比 Java 的继承多态**：

```java
// Java
abstract class Animal {
    abstract String speak();
}

class Dog extends Animal {
    @Override
    String speak() { return "Woof!"; }
}
```

Go 的多态更灵活：
- 不需要继承层次
- 可以为已有类型添加接口实现
- 组合优于继承

### 6.7.2 多态的实际应用

**策略模式**：

```go
type SortStrategy interface {
    Sort(data []int)
}

type QuickSort struct{}
func (q QuickSort) Sort(data []int) { /* 快速排序 */ }

type MergeSort struct{}
func (m MergeSort) Sort(data []int) { /* 归并排序 */ }

type Sorter struct {
    strategy SortStrategy
}

func (s *Sorter) Sort(data []int) {
    s.strategy.Sort(data)
}

// 使用
sorter := &Sorter{strategy: QuickSort{}}
sorter.Sort(data)
```

**插件系统**：

```go
type Plugin interface {
    Name() string
    Execute() error
}

var plugins []Plugin

func RegisterPlugin(p Plugin) {
    plugins = append(plugins, p)
}

func RunPlugins() error {
    for _, p := range plugins {
        if err := p.Execute(); err != nil {
            return err
        }
    }
    return nil
}
```

### 6.7.3 多态的性能

**接口调用的开销**：

1. 需要通过 `itab` 查找方法地址
2. 无法内联优化
3. 需要类型检查

**性能对比**：

```go
// 直接调用
func directCall(d Dog) {
    d.Speak()
}

// 接口调用
func interfaceCall(a Animal) {
    a.Speak()
}
```

接口调用比直接调用慢约 2-3 倍，但在大多数场景下可以忽略。

**什么时候需要关注性能？**

- 高频调用（每秒百万次）
- 性能敏感的核心路径
- 基准测试发现瓶颈

---

## 6.8 接口反模式

> 代码：`example5-anti/main.go`

### 6.8.1 大接口

```go
// 反模式：20+ 方法的大接口
type UserServiceBad interface {
    CreateUser(name, email string) error
    UpdateUser(id int, name, email string) error
    DeleteUser(id int) error
    GetUser(id int) (map[string]interface{}, error)
    ListUsers() ([]map[string]interface{}, error)
    ValidateUser(user map[string]interface{}) error
    HashPassword(password string) string
    CheckPassword(password, hash string) bool
    GenerateToken(user map[string]interface{}) string
    ValidateToken(token string) (map[string]interface{}, error)
    SendEmail(to, subject, body string) error
    SendSMS(to, message string) error
    LogActivity(userID int, action string) error
    GetStatistics() (map[string]interface{}, error)
    // ... 还有更多方法
}
```

**为什么大接口不好？**

1. **难以实现**：实现 20 个方法比实现 2 个方法困难
2. **难以测试**：Mock 20 个方法比 Mock 2 个方法困难
3. **难以理解**：接口语义不清晰
4. **难以重构**：修改一个方法可能影响所有实现

**如何拆分？**

```go
// 推荐：拆分成小接口
type UserCreator interface {
    CreateUser(name, email string) error
}

type UserReader interface {
    GetUser(id int) (map[string]interface{}, error)
    ListUsers() ([]map[string]interface{}, error)
}

type UserUpdater interface {
    UpdateUser(id int, name, email string) error
}

type UserDeleter interface {
    DeleteUser(id int) error
}
```

### 6.8.2 过度抽象

```go
// 反模式：只有一个实现也定义接口
type UserRepository interface {
    Save(user map[string]interface{}) error
    FindByID(id int) (map[string]interface{}, error)
}

type MemoryUserRepository struct {
    users map[int]map[string]interface{}
}

// ... 实现接口方法
```

**什么时候不需要接口？**

- 只有一个实现
- 不需要 Mock
- 不需要多态

**YAGNI 原则**：

> You Aren't Gonna Need It
> 
> 你不会需要它。

不要提前抽象，当真正需要时再定义接口。

### 6.8.3 接口污染

```go
// 反模式：为了"面向接口"而定义接口
type Logger interface {
    Log(msg string)
}

type ConsoleLogger struct{}

func (c *ConsoleLogger) Log(msg string) {
    fmt.Println(msg)
}

// 使用接口
func processWithInterface(logger Logger) {
    logger.Log("processing")
}

// 推荐：直接使用结构体
func processDirect(logger *ConsoleLogger) {
    logger.Log("processing")
}
```

**接口应该是被发现的，不是被设计的**。

### 6.8.4 返回接口

```go
// 反模式：返回接口
func NewDataStore() DataStore {
    return &MemoryStore{}
}

// 推荐：返回结构体
func NewDataStore() *MemoryStore {
    return &MemoryStore{}
}
```

**为什么？**

- 调用方可以决定是否需要接口
- 可以访问结构体的特定方法
- 避免接口开销

### 6.8.5 接口嵌套过深

```go
// 反模式：嵌套过深
type BaseReader interface {
    Read(p []byte) (n int, err error)
}

type AdvancedReader interface {
    BaseReader
    ReadAt(p []byte, off int64) (n int, err error)
}

type SuperReader interface {
    AdvancedReader
    Seek(offset int64, whence int) (int64, error)
}

type UltraReader interface {
    SuperReader
    ReadDir(n int) ([]string, error)
}

// 推荐：扁平化
type SimpleReader interface {
    Read(p []byte) (n int, err error)
}

type SimpleReaderAt interface {
    ReadAt(p []byte, off int64) (n int, err error)
}

type SimpleSeeker interface {
    Seek(offset int64, whence int) (int64, error)
}
```

---

## 6.9 实战：设计一个可测试的服务

> 代码：`service/`

### 需求

- 用户服务：CRUD
- 存储层：可替换（内存/MySQL/Redis）
- 可测试：Mock 存储层

### 设计

```go
// 1. 定义存储接口（使用方定义）
type UserStore interface {
    Save(user *User) error
    FindByID(id int) (*User, error)
    FindByEmail(email string) (*User, error)
    Delete(id int) error
}

// 2. 实现内存存储
type MemoryUserStore struct {
    users  map[int]*User
    nextID int
}

func (m *MemoryUserStore) Save(user *User) error {
    if user.ID == 0 {
        user.ID = m.nextID
        m.nextID++
    }
    m.users[user.ID] = user
    return nil
}

// ... 其他方法

// 3. 服务层依赖接口
type UserService struct {
    store UserStore // 依赖接口，而不是具体实现
}

func NewUserService(store UserStore) *UserService {
    return &UserService{store: store}
}

func (s *UserService) CreateUser(name, email string) (*User, error) {
    // 检查邮箱是否已存在
    existing, _ := s.store.FindByEmail(email)
    if existing != nil {
        return nil, fmt.Errorf("email already exists: %s", email)
    }

    user := &User{Name: name, Email: email}
    if err := s.store.Save(user); err != nil {
        return nil, fmt.Errorf("failed to save user: %w", err)
    }

    return user, nil
}

// 4. 测试时 Mock 接口
type MockUserStore struct {
    users   map[int]*User
    saveErr error
}

func (m *MockUserStore) Save(user *User) error {
    if m.saveErr != nil {
        return m.saveErr
    }
    m.users[user.ID] = user
    return nil
}

// ... 其他方法

// 5. 测试
func TestCreateUser(t *testing.T) {
    store := NewMockUserStore()
    service := NewUserService(store)

    user, err := service.CreateUser("Alice", "alice@example.com")
    if err != nil {
        t.Fatalf("CreateUser failed: %v", err)
    }

    if user.Name != "Alice" {
        t.Errorf("Expected name Alice, got %s", user.Name)
    }
}
```

### 代码结构

```
service/
├── model.go          # 数据模型
├── store.go          # 存储接口
├── memory_store.go   # 内存实现
├── service.go        # 业务逻辑
├── mock_store.go     # Mock 实现
└── service_test.go   # 测试
```

### 运行测试

```bash
cd service
go test -v
```

---

## 6.10 源码分析：http.Handler

### 为什么 http.Handler 只有一个方法？

```go
type Handler interface {
    ServeHTTP(ResponseWriter, *Request)
}
```

**最小接口原则**：

- 只定义"可以处理 HTTP 请求"这个行为
- 不关心具体实现（路由、中间件、业务逻辑）
- 容易实现、容易组合、容易测试

### HandlerFunc 适配器

```go
type HandlerFunc func(ResponseWriter, *Request)

func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
    f(w, r)
}
```

**让函数实现接口**：

```go
// 可以直接用函数作为 Handler
http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello, World!")
})
```

**适配器模式**：

```go
// HandlerFunc 是适配器
// 将函数转换为 Handler 接口
handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "Hello!")
})

// handler 实现了 Handler 接口
var h http.Handler = handler
```

### 一张图：Handler 调用链

```
HTTP Request
    ↓
┌─────────────────────────────────────┐
│  Middleware 1 (Logging)             │
│  ┌─────────────────────────────────┐│
│  │  Middleware 2 (Auth)            ││
│  │  ┌─────────────────────────────┐││
│  │  │  Middleware 3 (CORS)        │││
│  │  │  ┌─────────────────────────┐│││
│  │  │  │  Handler                ││││
│  │  │  │  ServeHTTP(w, r)        ││││
│  │  │  └─────────────────────────┘│││
│  │  └─────────────────────────────┘││
│  └─────────────────────────────────┘│
└─────────────────────────────────────┘
    ↓
HTTP Response
```

---

## 6.11 最佳实践总结

1. **接口要小**：1-2 个方法最好
2. **在需要时定义**：不要提前抽象
3. **接受接口，返回结构体**：灵活性和性能兼顾
4. **接口由使用方定义**：解耦更彻底
5. **用组合代替继承**：更灵活、更易重构
6. **空接口谨慎使用**：泛型优先

---

## 6.12 排障

### nil 接口陷阱

```go
var dog *Dog
var animal Animal = dog

fmt.Println(dog == nil)    // true
fmt.Println(animal == nil) // false！
```

**原因**：接口包含类型信息和值信息，只有两者都为 nil 时，接口才是 nil。

**解决**：

```go
// 检查接口的值是否为 nil
if animal != nil {
    // 还需要检查具体值
    if dog, ok := animal.(*Dog); ok && dog != nil {
        // 安全使用
    }
}
```

### 接口调用 panic

```go
var animal Animal
fmt.Println(animal.Speak()) // panic: nil pointer dereference
```

**原因**：接口为 nil，调用方法会 panic。

**解决**：

```go
if animal != nil {
    fmt.Println(animal.Speak())
}
```

### 方法集不匹配

```go
type Animal interface {
    Speak() string
}

type Dog struct{}

func (d *Dog) Speak() string { // 指针接收者
    return "Woof!"
}

var dog Dog
var animal Animal = dog // 编译错误：Dog 没有实现 Animal
```

**原因**：`Dog` 的方法集不包含指针接收者方法。

**解决**：

```go
// 方式 1：使用指针
var animal Animal = &dog

// 方式 2：改为值接收者
func (d Dog) Speak() string {
    return "Woof!"
}
```

### 接口比较的坑

```go
var a interface{} = []int{1, 2, 3}
var b interface{} = []int{1, 2, 3}

fmt.Println(a == b) // panic: 切片不能比较
```

**原因**：切片、map、函数不能用 `==` 比较。

**解决**：

```go
// 使用 reflect.DeepEqual
fmt.Println(reflect.DeepEqual(a, b)) // true
```

---

## 6.13 面试题

**Q1：Go 接口和 Java 接口的区别？**

A：
- Go：隐式实现，不需要 `implements`
- Java：显式实现，需要 `implements`
- Go：可以为已有类型添加接口实现
- Java：必须提前声明实现关系
- Go：接口更灵活，更容易解耦

**Q2：为什么 Go 不需要 implements？**

A：
- 隐式实现更灵活
- 可以为已有类型添加接口实现
- 不需要修改原有代码
- 解耦更彻底

**Q3：接口的底层结构是什么？**

A：
- 空接口（eface）：`_type` + `data`
- 非空接口（iface）：`itab` + `data`
- `itab` 包含接口类型、具体类型、方法表

**Q4：为什么"接受接口，返回结构体"？**

A：
- 调用方可以决定是否需要接口
- 可以访问结构体的特定方法
- 避免接口开销
- 更灵活

**Q5：interface{} 和泛型的区别？**

A：
- `interface{}`：运行时类型检查，有装箱/拆箱开销
- 泛型：编译时类型检查，零开销
- `interface{}`：可以存储任何类型
- 泛型：受类型约束限制

**Q6：接口调用的性能开销？**

A：
- 需要通过 `itab` 查找方法地址
- 无法内联优化
- 需要类型检查
- 比直接调用慢约 2-3 倍
- 但在大多数场景下可以忽略

---

## 6.14 小结

本章从设计哲学出发，讲透了 Go 接口的精髓：

1. **接口基础**：隐式实现、接口是契约、接口零值
2. **小接口原则**：1-2 个方法最好，容易实现、组合、测试
3. **接口组合**：组合小接口，而不是继承大接口
4. **最佳实践**：接受接口返回结构体、在需要时定义、使用方定义接口
5. **常见模式**：Reader/Writer、Handler、Stringer、Sort
6. **空接口**：可以存储任何值，但谨慎使用
7. **多态**：通过接口实现，比继承更灵活
8. **反模式**：大接口、过度抽象、接口污染
9. **实战**：设计可测试的服务
10. **源码分析**：http.Handler 的设计思想

**核心思想**：

> "Accept interfaces, return structs."
> 
> 接受接口，返回结构体。

> "The bigger the interface, the weaker the abstraction."
> 
> 接口越大，抽象越弱。

> "Don't design interfaces, discover them."
> 
> 不要设计接口，发现接口。

下一章我们将学习 Context 使用规范，掌握 Go 并发编程的核心工具。
