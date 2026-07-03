package config

import (
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Log      LogConfig      `mapstructure:"log"`
}

// AppConfig 应用配置
type AppConfig struct {
	Name    string `mapstructure:"name"`
	Env     string `mapstructure:"env"`
	Version string `mapstructure:"version"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// Addr 返回监听地址
func (s *ServerConfig) Addr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	Name         string `mapstructure:"name"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

// DSN 生成数据库连接字符串
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		d.User, d.Password, d.Host, d.Port, d.Name)
}

// MaskedDSN 返回脱敏后的连接字符串，只用于日志展示。
func (d *DatabaseConfig) MaskedDSN() string {
	password := ""
	if d.Password != "" {
		password = "******"
	}
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		d.User, password, d.Host, d.Port, d.Name)
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// Addr 生成 Redis 地址
func (r *RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// Validate 验证配置
func (c *Config) Validate() error {
	if !slices.Contains([]string{"dev", "test", "prod"}, c.App.Env) {
		return fmt.Errorf("invalid app env: %s", c.App.Env)
	}
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.Server.ReadTimeout <= 0 || c.Server.WriteTimeout <= 0 {
		return fmt.Errorf("server timeout must be greater than zero")
	}
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}
	if c.App.Env == "prod" && c.Database.Password == "" {
		return fmt.Errorf("database password is required in prod")
	}
	if c.Database.MaxOpenConns <= 0 || c.Database.MaxIdleConns < 0 {
		return fmt.Errorf("invalid database pool config")
	}
	if c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		return fmt.Errorf("max_idle_conns cannot be greater than max_open_conns")
	}
	if !slices.Contains([]string{"debug", "info", "warn", "error"}, c.Log.Level) {
		return fmt.Errorf("invalid log level: %s", c.Log.Level)
	}
	if !slices.Contains([]string{"json", "text"}, c.Log.Format) {
		return fmt.Errorf("invalid log format: %s", c.Log.Format)
	}
	return nil
}

// Load 加载配置
func Load(paths ...string) (*Config, error) {
	// 获取环境，默认 dev
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	v := viper.New()
	v.SetConfigType("yaml")
	v.SetEnvPrefix("ADMIN_API")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	bindEnv(v)

	// 加载公共配置
	v.SetConfigName("config")
	if len(paths) == 0 {
		paths = []string{"./config", "."}
	}
	for _, path := range paths {
		v.AddConfigPath(path)
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read base config: %w", err)
	}

	// 合并环境配置（覆盖公共配置）
	v.SetConfigName("config." + env)
	if err := v.MergeInConfig(); err != nil {
		log.Printf("No %s config file found, using base config only", env)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &cfg, nil
}

func bindEnv(v *viper.Viper) {
	keys := []string{
		"app.name",
		"app.env",
		"app.version",
		"server.host",
		"server.port",
		"server.read_timeout",
		"server.write_timeout",
		"database.host",
		"database.port",
		"database.user",
		"database.password",
		"database.name",
		"database.max_open_conns",
		"database.max_idle_conns",
		"redis.host",
		"redis.port",
		"redis.password",
		"redis.db",
		"log.level",
		"log.format",
		"log.output",
	}
	for _, key := range keys {
		_ = v.BindEnv(key)
	}
	_ = v.BindEnv("app.env", "APP_ENV")
	_ = v.BindEnv("database.password", "DB_PASSWORD")
	_ = v.BindEnv("redis.password", "REDIS_PASSWORD")
}
