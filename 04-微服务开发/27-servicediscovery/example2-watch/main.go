package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/go-book/servicediscovery/registry"
)

// 演示发现端如何动态感知实例上下线:
//
//	go run . -role=watch     # 终端 A:持续监听 user-service 的实例变化
//	go run . -role=register  # 终端 B:注册一个实例 → A 立刻收到"上线"
//	                        # Ctrl+C 掉 B → A 立刻收到"下线"(Deregister 撤销租约)
func main() {
	role := flag.String("role", "watch", "watch / register")
	name := flag.String("name", "order-service", "服务名")
	addr := flag.String("addr", "10.0.1.1:9090", "注册地址")
	ttl := flag.Int64("ttl", 5, "租约 TTL(秒)")
	flag.Parse()

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		log.Fatalf("连接 etcd: %v", err)
	}
	defer cli.Close()
	ctx := context.Background()

	if *role == "register" {
		reg, err := registry.Register(ctx, cli,
			registry.ServiceInfo{Name: *name, Addr: *addr}, *ttl)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("已注册 %s@%s,Ctrl+C 优雅下线\n", *name, *addr)

		// 监听 Ctrl+C,主动 Deregister(撤销租约)而不是等 TTL 自然过期
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		dctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := reg.Deregister(dctx); err != nil {
			log.Printf("主动下线失败(TTL 兜底): %v", err)
		}
		fmt.Println("已优雅下线")
		return
	}

	w, err := registry.NewWatcher(ctx, cli, *name)
	if err != nil {
		log.Fatal(err)
	}
	printAddrs(w.Addrs(), "初始列表")

	for range w.Changed() {
		printAddrs(w.Addrs(), "检测到变化")
	}
}

func printAddrs(addrs []string, tag string) {
	fmt.Printf("[%s] %s 实例(%d):%v\n",
		time.Now().Format("15:04:05"), tag, len(addrs), addrs)
}
