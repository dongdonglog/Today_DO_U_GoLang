package main

import (
	"time"

	"go.uber.org/zap"
)

func main() {
	// 创建生产环境 logger（JSON 格式）
	logger, _ := zap.NewProduction()
	defer logger.Sync() // 确保缓冲区刷新

	// 基本日志
	logger.Info("server starting",
		zap.String("host", "0.0.0.0"),
		zap.Int("port", 8080),
	)

	// 不同级别的日志
	logger.Debug("debug message", zap.String("detail", "this is debug"))
	logger.Info("info message", zap.String("detail", "this is info"))
	logger.Warn("warn message", zap.String("detail", "this is warn"))
	logger.Error("error message", zap.String("detail", "this is error"))

	// 结构化字段
	userID := 123
	userName := "Alice"
	email := "alice@example.com"

	logger.Info("user created",
		zap.Int("user_id", userID),
		zap.String("name", userName),
		zap.String("email", email),
		zap.Time("created_at", time.Now()),
	)

	// 错误日志
	err := &CustomError{Code: 404, Message: "user not found"}
	logger.Error("database query failed",
		zap.Int("user_id", 999),
		zap.Error(err),
	)

	// 使用 Named logger 创建子 logger
	dbLogger := logger.Named("database")
	dbLogger.Info("connection established",
		zap.String("host", "127.0.0.1"),
		zap.Int("port", 3306),
	)

	// 使用 With 添加公共字段
	requestLogger := logger.With(
		zap.String("request_id", "abc-123"),
		zap.String("client_ip", "192.168.1.1"),
	)
	requestLogger.Info("request received",
		zap.String("method", "POST"),
		zap.String("path", "/api/v1/users"),
	)
}

// CustomError 自定义错误
type CustomError struct {
	Code    int
	Message string
}

func (e *CustomError) Error() string {
	return e.Message
}
