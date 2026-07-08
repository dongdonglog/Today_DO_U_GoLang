package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Server ServerConfig `mapstructure:"server"`
	JWT    JWTConfig    `mapstructure:"jwt"`
	Upload UploadConfig `mapstructure:"upload"`
	Log    LogConfig    `mapstructure:"log"`
}

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type JWTConfig struct {
	AccessSecret       string        `mapstructure:"access_secret"`
	RefreshSecret      string        `mapstructure:"refresh_secret"`
	AccessTokenExpire  time.Duration `mapstructure:"access_token_expire"`
	RefreshTokenExpire time.Duration `mapstructure:"refresh_token_expire"`
}

type UploadConfig struct {
	MaxSize      int64    `mapstructure:"max_size"`
	AllowedTypes []string `mapstructure:"allowed_types"`
	StoragePath  string   `mapstructure:"storage_path"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

func Load(paths ...string) (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	if len(paths) == 0 {
		paths = []string{"./config", "."}
	}
	for _, path := range paths {
		v.AddConfigPath(path)
	}
	v.SetEnvPrefix("ADMIN_UPLOAD")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	bindEnv(v)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.JWT.AccessSecret == "" || c.JWT.RefreshSecret == "" {
		return fmt.Errorf("jwt access and refresh secrets are required")
	}
	if c.JWT.AccessSecret == c.JWT.RefreshSecret {
		return fmt.Errorf("jwt access secret and refresh secret must be different")
	}
	if isWeakSecret(c.JWT.AccessSecret) || isWeakSecret(c.JWT.RefreshSecret) {
		return fmt.Errorf("jwt secret is too weak")
	}
	if c.JWT.AccessTokenExpire <= 0 || c.JWT.RefreshTokenExpire <= 0 {
		return fmt.Errorf("jwt token expiration must be greater than zero")
	}
	if c.Upload.MaxSize <= 0 {
		return fmt.Errorf("upload max_size must be greater than zero")
	}
	if len(c.Upload.AllowedTypes) == 0 {
		return fmt.Errorf("upload allowed_types is required")
	}
	if c.Upload.StoragePath == "" {
		return fmt.Errorf("upload storage_path is required")
	}
	return nil
}

func isWeakSecret(secret string) bool {
	weakSecrets := map[string]bool{
		"secret":                               true,
		"your-secret-key":                      true,
		"your-secret-key-change-in-production": true,
	}
	return len(secret) < 32 || weakSecrets[secret]
}

func bindEnv(v *viper.Viper) {
	_ = v.BindEnv("jwt.access_secret", "JWT_ACCESS_SECRET")
	_ = v.BindEnv("jwt.refresh_secret", "JWT_REFRESH_SECRET")
	_ = v.BindEnv("jwt.access_token_expire")
	_ = v.BindEnv("jwt.refresh_token_expire")
	_ = v.BindEnv("server.host")
	_ = v.BindEnv("server.port")
	_ = v.BindEnv("upload.max_size")
	_ = v.BindEnv("upload.storage_path")
}
