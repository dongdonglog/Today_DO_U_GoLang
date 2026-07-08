package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
)

func main() {
	// 设置配置文件名（不带扩展名）
	viper.SetConfigName("config")
	// 设置配置文件类型
	viper.SetConfigType("yaml")
	// 添加配置文件搜索路径
	viper.AddConfigPath(".")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/etc/myapp/")

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %v", err)
	}

	// 打印实际使用的配置文件
	fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())

	// 读取配置项
	fmt.Println("\n--- 读取配置项 ---")
	fmt.Printf("App Name: %s\n", viper.GetString("app.name"))
	fmt.Printf("Server Port: %d\n", viper.GetInt("server.port"))
	fmt.Printf("Database Host: %s\n", viper.GetString("database.host"))
	fmt.Printf("Database Port: %d\n", viper.GetInt("database.port"))
	fmt.Printf("Log Level: %s\n", viper.GetString("log.level"))

	// 配置优先级演示
	fmt.Println("\n--- 配置优先级 ---")
	fmt.Println("优先级：命令行参数 > 环境变量 > 配置文件 > 默认值")

	// 设置默认值
	viper.SetDefault("app.version", "1.0.0")
	fmt.Printf("Default Version: %s\n", viper.GetString("app.version"))

	// 环境变量绑定
	viper.SetEnvPrefix("MYAPP")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	_ = viper.BindEnv("app.env", "APP_ENV")
	_ = viper.BindEnv("database.password", "DB_PASSWORD")
	_ = viper.BindEnv("server.port")
	// 如果设置了环境变量 APP_ENV，会覆盖配置文件中的值
	fmt.Printf("App Env: %s\n", viper.GetString("app.env"))

	// 检查配置项是否存在
	if viper.IsSet("database.password") {
		fmt.Println("Database password is set")
	} else {
		fmt.Println("Database password is not set")
	}

	fmt.Println("\n--- 安全提示 ---")
	fmt.Println("配置已加载，生产日志不要打印包含 password、token、secret 的完整配置。")
}
