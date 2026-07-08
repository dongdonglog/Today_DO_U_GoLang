package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	App AppConfig `mapstructure:"app"`
	Log LogConfig `mapstructure:"log"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

// ConfigManager 配置管理器
type ConfigManager struct {
	mu  sync.RWMutex
	cfg Config
}

// Get 获取配置（线程安全）
func (cm *ConfigManager) Get() Config {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.cfg
}

// Update 更新配置（线程安全）
func (cm *ConfigManager) Update(cfg Config) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.cfg = cfg
}

func (c *Config) Validate() error {
	if c.Log.Level == "" {
		return fmt.Errorf("log.level is required")
	}
	return nil
}

func main() {
	// 加载初始配置
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	var initialCfg Config
	if err := viper.Unmarshal(&initialCfg); err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
	}
	if err := initialCfg.Validate(); err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	// 创建配置管理器
	cm := &ConfigManager{cfg: initialCfg}

	// 启动配置热更新监听
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Printf("Config file changed: %s", e.Name)

		var newCfg Config
		if err := viper.Unmarshal(&newCfg); err != nil {
			log.Printf("Failed to reload config: %v", err)
			return
		}
		if err := newCfg.Validate(); err != nil {
			log.Printf("Ignore invalid config: %v", err)
			return
		}

		cm.Update(newCfg)
		log.Printf("Config reloaded: log.level=%s", newCfg.Log.Level)
	})

	fmt.Println("Config hot-reload demo")
	fmt.Printf("Initial config: log.level=%s\n", cm.Get().Log.Level)
	fmt.Println("Try modifying config.yaml, changes will be detected automatically")
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println()

	// 模拟业务逻辑定期读取配置
	go func() {
		for {
			cfg := cm.Get()
			log.Printf("[Business] Running with log.level=%s", cfg.Log.Level)
			time.Sleep(3 * time.Second)
		}
	}()

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	fmt.Println("\nShutting down...")
}
