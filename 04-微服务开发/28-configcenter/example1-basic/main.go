package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/go-book/configcenter/configstore"
)

// AppConfig 是服务的全部动态配置。放 etcd 的 value 里(JSON 格式)。
type AppConfig struct {
	DBDSN     string `json:"db_dsn"`
	RateLimit int    `json:"rate_limit"` // 每秒允许的请求数
	TimeoutMS int    `json:"timeout_ms"` // 下游超时
	LogLevel  string `json:"log_level"`  // debug/info/warn/error
}

// validate 启动与热更新共用一套校验:坏配置无论何时进来都进不了内存。
func validate(cfg *AppConfig) error {
	if cfg.RateLimit <= 0 || cfg.RateLimit > 100000 {
		return fmt.Errorf("rate_limit 必须在 (0,100000],当前 %d", cfg.RateLimit)
	}
	if cfg.TimeoutMS < 10 || cfg.TimeoutMS > 60000 {
		return fmt.Errorf("timeout_ms 必须在 [10,60000],当前 %d", cfg.TimeoutMS)
	}
	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("非法 log_level %q", cfg.LogLevel)
	}
	if cfg.DBDSN == "" {
		return fmt.Errorf("db_dsn 不能为空")
	}
	return nil
}

func main() {
	role := flag.String("role", "app", "app / set")
	flag.Parse()

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()
	ctx := context.Background()

	const key = "demo/app-config"

	if *role == "set" {
		// 运维/发布系统通过 etcdctl 或此命令改配置
		raw, _ := json.MarshalIndent(AppConfig{
			DBDSN:     "root:root@tcp(localhost:3306)/go_book",
			RateLimit: 500,
			TimeoutMS: 200,
			LogLevel:  "info",
		}, "", "  ")
		if _, err := cli.Put(ctx, key, string(raw)); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("已写入配置到 %s:\n%s\n", key, raw)
		return
	}

	// 应用侧:defaults 兜底(etcd 没有配置时用本地默认并回写)
	defaults := &AppConfig{
		DBDSN:     "root:root@tcp(localhost:3306)/go_book",
		RateLimit: 100,
		TimeoutMS: 500,
		LogLevel:  "info",
	}

	store, err := configstore.New(ctx, cli, key, defaults, validate)
	if err != nil {
		log.Fatalf("初始化配置失败: %v", err)
	}

	cfg := store.Get()
	fmt.Println("启动完成,当前生效配置:")
	printCfg(cfg)
}

func printCfg(c *AppConfig) {
	fmt.Printf("  db_dsn=%s\n  rate_limit=%d\n  timeout_ms=%d\n  log_level=%s\n",
		c.DBDSN, c.RateLimit, c.TimeoutMS, c.LogLevel)
}
