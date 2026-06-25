package main

import "fmt"

// ========================================
// 1. 小接口定义
// ========================================

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

// Stringer 字符串接口
type Stringer interface {
	String() string
}

// ========================================
// 2. 接口组合
// ========================================

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

// WriteCloser 写关闭接口
type WriteCloser interface {
	Writer
	Closer
}

// ReadWriteCloser 读写关闭接口
type ReadWriteCloser interface {
	Reader
	Writer
	Closer
}

// ========================================
// 3. 实现组合接口
// ========================================

// File 文件
type File struct {
	Name   string
	closed bool
}

func (f *File) Read(p []byte) (n int, err error) {
	if f.closed {
		return 0, fmt.Errorf("file closed")
	}
	fmt.Printf("Reading from %s\n", f.Name)
	return len(p), nil
}

func (f *File) Write(p []byte) (n int, err error) {
	if f.closed {
		return 0, fmt.Errorf("file closed")
	}
	fmt.Printf("Writing to %s: %s\n", f.Name, string(p))
	return len(p), nil
}

func (f *File) Close() error {
	if f.closed {
		return fmt.Errorf("already closed")
	}
	f.closed = true
	fmt.Printf("Closing %s\n", f.Name)
	return nil
}

// ========================================
// 4. 使用组合接口
// ========================================

func copyData(dst Writer, src Reader) error {
	buf := make([]byte, 1024)
	n, err := src.Read(buf)
	if err != nil {
		return err
	}
	_, err = dst.Write(buf[:n])
	return err
}

func closeAndLog(c Closer) {
	if err := c.Close(); err != nil {
		fmt.Printf("Error closing: %v\n", err)
	}
}

// ========================================
// 5. 自定义组合
// ========================================

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

// ConsoleLogger 控制台日志
type ConsoleLogger struct{}

func (c *ConsoleLogger) Log(msg string) {
	fmt.Printf("[LOG] %s\n", msg)
}

func (c *ConsoleLogger) Flush() error {
	fmt.Println("[FLUSH] Flushing logs")
	return nil
}

func main() {
	// 接口组合
	fmt.Println("=== 接口组合 ===")

	file := &File{Name: "test.txt"}

	// 作为 Reader 使用
	var r Reader = file
	r.Read([]byte("data"))

	// 作为 Writer 使用
	var w Writer = file
	w.Write([]byte("hello"))

	// 作为 ReadWriter 使用
	var rw ReadWriter = file
	rw.Read([]byte("data"))
	rw.Write([]byte("world"))

	// 作为 ReadWriteCloser 使用
	var rwc ReadWriteCloser = file
	rwc.Read([]byte("data"))
	rwc.Write([]byte("test"))
	rwc.Close()

	// 函数参数
	fmt.Println("\n=== 函数参数 ===")
	src := &File{Name: "source.txt"}
	dst := &File{Name: "dest.txt"}
	copyData(dst, src)

	// 自定义组合
	fmt.Println("\n=== 自定义组合 ===")
	logger := &ConsoleLogger{}
	var lf LogFlusher = logger
	lf.Log("test message")
	lf.Flush()
}
