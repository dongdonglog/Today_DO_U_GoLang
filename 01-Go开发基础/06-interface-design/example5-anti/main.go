package main

import "fmt"

// ========================================
// 反模式 1：大接口
// ========================================

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

// ========================================
// 反模式 2：过度抽象
// ========================================

// 反模式：只有一个实现也定义接口
type UserRepository interface {
	Save(user map[string]interface{}) error
	FindByID(id int) (map[string]interface{}, error)
}

type MemoryUserRepository struct {
	users map[int]map[string]interface{}
}

func (m *MemoryUserRepository) Save(user map[string]interface{}) error {
	id := user["id"].(int)
	m.users[id] = user
	return nil
}

func (m *MemoryUserRepository) FindByID(id int) (map[string]interface{}, error) {
	user, ok := m.users[id]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// 推荐：直接使用结构体
type SimpleUserRepository struct {
	users map[int]map[string]interface{}
}

func (s *SimpleUserRepository) Save(user map[string]interface{}) error {
	id := user["id"].(int)
	s.users[id] = user
	return nil
}

func (s *SimpleUserRepository) FindByID(id int) (map[string]interface{}, error) {
	user, ok := s.users[id]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// ========================================
// 反模式 3：接口污染
// ========================================

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

// ========================================
// 反模式 4：返回接口
// ========================================

type DataStore interface {
	Get(key string) (string, error)
	Set(key, value string) error
}

type MemoryStore struct {
	data map[string]string
}

func (m *MemoryStore) Get(key string) (string, error) {
	value, ok := m.data[key]
	if !ok {
		return "", fmt.Errorf("key not found")
	}
	return value, nil
}

func (m *MemoryStore) Set(key, value string) error {
	m.data[key] = value
	return nil
}

// 反模式：返回接口
func NewDataStoreBad() DataStore {
	return &MemoryStore{data: make(map[string]string)}
}

// 推荐：返回结构体
func NewDataStoreGood() *MemoryStore {
	return &MemoryStore{data: make(map[string]string)}
}

// ========================================
// 反模式 5：接口嵌套过深
// ========================================

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

// ========================================
// 正确使用示例
// ========================================

// 正确：在需要时定义接口
type StringProcessor interface {
	Process(s string) string
}

type UpperCaseProcessor struct{}

func (u *UpperCaseProcessor) Process(s string) string {
	result := ""
	for _, c := range s {
		if c >= 'a' && c <= 'z' {
			result += string(c - 32)
		} else {
			result += string(c)
		}
	}
	return result
}

// 正确：接受接口，返回结构体
func NewProcessor() *UpperCaseProcessor {
	return &UpperCaseProcessor{}
}

func processWithInterface2(p StringProcessor, s string) string {
	return p.Process(s)
}

func main() {
	fmt.Println("=== 反模式演示 ===")

	// 反模式 1：大接口
	fmt.Println("\n反模式 1：大接口")
	fmt.Println("UserServiceBad 有 20+ 方法，难以实现和测试")
	fmt.Println("推荐拆分成 UserCreator、UserReader 等小接口")

	// 反模式 2：过度抽象
	fmt.Println("\n反模式 2：过度抽象")
	repo := &MemoryUserRepository{users: make(map[int]map[string]interface{})}
	repo.Save(map[string]interface{}{"id": 1, "name": "Alice"})
	user, _ := repo.FindByID(1)
	fmt.Printf("User: %v\n", user)
	fmt.Println("只有一个实现时，不需要定义接口")

	// 反模式 3：接口污染
	fmt.Println("\n反模式 3：接口污染")
	logger := &ConsoleLogger{}
	processWithInterface(logger)
	processDirect(logger)
	fmt.Println("只有一个实现时，直接使用结构体")

	// 反模式 4：返回接口
	fmt.Println("\n反模式 4：返回接口")
	store1 := NewDataStoreBad()
	store1.Set("key", "value")
	val, _ := store1.Get("key")
	fmt.Printf("Value: %s\n", val)

	store2 := NewDataStoreGood()
	store2.Set("key", "value")
	val, _ = store2.Get("key")
	fmt.Printf("Value: %s\n", val)
	fmt.Println("推荐返回结构体，让调用方决定是否需要接口")

	// 正确使用
	fmt.Println("\n正确使用示例")
	processor := NewProcessor()
	result := processWithInterface2(processor, "hello")
	fmt.Printf("Result: %s\n", result)
}
