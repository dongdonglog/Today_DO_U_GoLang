package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Log      LogConfig      `mapstructure:"log"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
}

type ServerConfig struct {
	Port int `mapstructure:"port"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

func Load() (*Config, error) {
	// 获取环境，默认 dev
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}

	// 加载公共配置
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read base config: %w", err)
	}

	// 合并环境配置（覆盖公共配置）
	viper.SetConfigName("config." + env)
	if err := viper.MergeInConfig(); err != nil {
		log.Printf("No %s config file found, using base config only", env)
	}

	// 环境变量覆盖
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	_ = viper.BindEnv("app.env", "APP_ENV")
	_ = viper.BindEnv("database.password", "DB_PASSWORD")
	_ = viper.BindEnv("server.port", "SERVER_PORT")

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}

func main() {
	cfg, err := Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	fmt.Println()
	fmt.Printf("App Name: %s\n", cfg.App.Name)
	fmt.Printf("App Env: %s\n", cfg.App.Env)
	fmt.Printf("Server Port: %d\n", cfg.Server.Port)
	fmt.Printf("Database: %s:%d/%s (user=%s)\n",
		cfg.Database.Host, cfg.Database.Port, cfg.Database.Name, cfg.Database.User)
	fmt.Printf("Log Level: %s\n", cfg.Log.Level)
}
