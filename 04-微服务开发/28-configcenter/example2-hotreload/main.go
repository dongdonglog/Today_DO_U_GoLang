package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/go-book/configcenter/configstore"
)

// 热更新演示:限流阈值放在 etcd 里,改配置立即生效,不用重启服务。
//
//	终端 A: go run . -role=app        # 启动应用,持续按当前限流值"接流量"
//	终端 B: etcdctl put demo/app-config '<新 JSON>'   # 改配置
//	观察终端 A:[config] 已热更新 + 请求处理日志里的限流值变化
type AppConfig struct {
	RateLimit int `json:"rate_limit"`
	TimeoutMS int `json:"timeout_ms"`
}

func main() {
	role := flagRole()
	if role == "set" {
		setConfig()
		return
	}
	runApp()
}

var roleFlag string

func flagRole() string {
	if len(os.Args) > 2 && os.Args[1] == "-role" {
		roleFlag = os.Args[2]
	}
	return roleFlag
}

func setConfig() {
	cli, err := clientv3.New(clientv3.Config{Endpoints: []string{"localhost:2379"}, DialTimeout: 3 * time.Second})
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	cfg := `{"rate_limit": 800, "timeout_ms": 100}`
	if _, err := cli.Put(context.Background(), "demo/rl-config", cfg); err != nil {
		log.Fatal(err)
	}
	fmt.Println("已把 rate_limit 改为 800")
}

func runApp() {
	cli, err := clientv3.New(clientv3.Config{Endpoints: []string{"localhost:2379"}, DialTimeout: 3 * time.Second})
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	defaults := &AppConfig{RateLimit: 100, TimeoutMS: 500}
	store, err := configstore.New(context.Background(), cli, "demo/rl-config", defaults,
		func(c *AppConfig) error {
			if c.RateLimit <= 0 || c.RateLimit > 100000 {
				return fmt.Errorf("rate_limit 非法: %d", c.RateLimit)
			}
			return nil
		})
	if err != nil {
		log.Fatal(err)
	}

	// OnChange 回调:配置变更时做联动动作(真实场景是重建连接池、调整信号量等)
	store.OnChange(func(old, new *AppConfig) {
		fmt.Printf(">>> 配置变更: rate_limit %d -> %d\n", old.RateLimit, new.RateLimit)
	})

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	// 模拟业务循环:每秒打印一次当前生效的限流值
	tick := time.NewTicker(time.Second)
	defer tick.Stop()

	for {
		select {
		case <-sig:
			fmt.Println("退出")
			return
		case <-tick.C:
			cfg := store.Get() // 无锁快照读:拿到的永远是完整的某一代配置
			fmt.Printf("处理请求中... 当前限流=%d 超时=%dms\n", cfg.RateLimit, cfg.TimeoutMS)
			// 真实场景:用 atomic 值驱动限流器(第31章)、超时控制等
			var _ atomic.Value // 占位说明:高频路径同样可以无锁读
		}
	}
}
