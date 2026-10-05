package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	ListenAddr     string
	DatabaseDSN    string
	RedisAddr      string
	RedisPassword  string
	RedisDB        int
	AccessSecret   string
	RefreshSecret  string
	AccessTTL      time.Duration
	RefreshTTL     time.Duration
	LoginFailLimit int
	AdminEmail     string
	AdminPassword  string
}

func Load() (Config, error) {
	cfg := Config{
		ListenAddr:     getenv("UC_LISTEN_ADDR", ":8080"),
		DatabaseDSN:    getenv("UC_DATABASE_DSN", "user_center:user-center-local@tcp(127.0.0.1:3306)/user_center?parseTime=true&loc=Local"),
		RedisAddr:      getenv("UC_REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword:  os.Getenv("UC_REDIS_PASSWORD"),
		RedisDB:        getenvInt("UC_REDIS_DB", 0),
		AccessSecret:   os.Getenv("UC_ACCESS_SECRET"),
		RefreshSecret:  os.Getenv("UC_REFRESH_SECRET"),
		AccessTTL:      getenvDuration("UC_ACCESS_TTL", 15*time.Minute),
		RefreshTTL:     getenvDuration("UC_REFRESH_TTL", 7*24*time.Hour),
		LoginFailLimit: getenvInt("UC_LOGIN_FAIL_LIMIT", 5),
		AdminEmail:     getenv("UC_BOOTSTRAP_ADMIN_EMAIL", "admin@example.com"),
		AdminPassword:  os.Getenv("UC_BOOTSTRAP_ADMIN_PASSWORD"),
	}
	return cfg, cfg.Validate()
}

func (c Config) Validate() error {
	if c.DatabaseDSN == "" {
		return fmt.Errorf("UC_DATABASE_DSN is required")
	}
	if c.RedisAddr == "" {
		return fmt.Errorf("UC_REDIS_ADDR is required")
	}
	if len(c.AccessSecret) < 32 || len(c.RefreshSecret) < 32 {
		return fmt.Errorf("UC_ACCESS_SECRET and UC_REFRESH_SECRET must be at least 32 bytes")
	}
	if c.AccessSecret == c.RefreshSecret {
		return fmt.Errorf("UC_ACCESS_SECRET and UC_REFRESH_SECRET must be different")
	}
	if c.AccessTTL <= 0 || c.RefreshTTL <= 0 || c.AccessTTL >= c.RefreshTTL {
		return fmt.Errorf("token TTL configuration is invalid")
	}
	if c.LoginFailLimit <= 0 {
		return fmt.Errorf("UC_LOGIN_FAIL_LIMIT must be greater than 0")
	}
	if c.AdminPassword != "" && len(c.AdminPassword) < 8 {
		return fmt.Errorf("UC_BOOTSTRAP_ADMIN_PASSWORD must be at least 8 characters")
	}
	return nil
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getenvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
