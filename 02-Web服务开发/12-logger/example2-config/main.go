package main

import (
	"log"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogConfig 日志配置
type LogConfig struct {
	Level       string `mapstructure:"level"`
	Format      string `mapstructure:"format"`
	Development bool   `mapstructure:"development"`
}

// NewLogger 根据配置创建 logger
func NewLogger(cfg LogConfig) (*zap.Logger, error) {
	// 解析日志级别
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
		level = zapcore.InfoLevel
	}

	// 选择编码器
	var encoder zapcore.Encoder
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	if cfg.Format == "console" || cfg.Development {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// 创建 core
	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(zapcore.Lock(zapcore.AddSync(nil))), // 占位，实际应该用 stdout
		level,
	)

	// 构建 logger
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return logger, nil
}

// NewDevelopmentLogger 创建开发环境 logger
func NewDevelopmentLogger() (*zap.Logger, error) {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	return config.Build()
}

// NewProductionLogger 创建生产环境 logger
func NewProductionLogger() (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	return config.Build()
}

func main() {
	// 方式 1：使用预定义配置
	devLogger, _ := NewDevelopmentLogger()
	defer devLogger.Sync()

	devLogger.Info("development logger",
		zap.String("env", "dev"),
		zap.Int("port", 8080),
	)

	prodLogger, _ := NewProductionLogger()
	defer prodLogger.Sync()

	prodLogger.Info("production logger",
		zap.String("env", "prod"),
		zap.Int("port", 8080),
	)

	// 方式 2：使用自定义配置
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("No config file found, using default")
	}

	var logCfg LogConfig
	if err := viper.UnmarshalKey("log", &logCfg); err != nil {
		logCfg = LogConfig{
			Level:  "info",
			Format: "json",
		}
	}

	customLogger, _ := NewLogger(logCfg)
	defer customLogger.Sync()

	customLogger.Info("custom logger",
		zap.String("level", logCfg.Level),
		zap.String("format", logCfg.Format),
	)
}
