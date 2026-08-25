package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/go-book/servicediscovery/registry"
)

// 演示服务注册的完整生命周期:
//
//	go run . -role=register   # 注册 user-service@10.0.0.1:8080,TTL=5s
//	go run . -role=list       # 查看当前存活实例
//
// 关键观察:
//  1. register 进程活着时,list 能看到实例(租约被持续续期)
//  2. kill 掉 register 进程后,最多 5 秒,实例从 etcd 里自动消失
//     (进程崩溃来不及主动下线,靠租约 TTL 兜底)
func main() {
	role := flag.String("role", "register", "register / list / crash")
	name := flag.String("name", "user-service", "服务名")
	addr := flag.String("addr", "10.0.0.1:8080", "注册地址")
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

	switch *role {
	case "list":
		infos, err := registry.Discover(ctx, cli, *name)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("服务 %s 存活实例 %d 个:\n", *name, len(infos))
		for _, info := range infos {
			fmt.Printf("  - %s\n", info.Addr)
		}

	case "crash":
		// 注册后立刻 panic,模拟进程崩溃(没机会执行 Deregister)
		if _, err := registry.Register(ctx, cli,
			registry.ServiceInfo{Name: *name, Addr: *addr}, *ttl); err != nil {
			log.Fatal(err)
		}
		fmt.Printf("已注册 %s@%s,现在模拟崩溃(panic),观察 %ds 后实例被摘除\n",
			*name, *addr, *ttl)
		panic("simulated process crash")

	default: // register
		reg, err := registry.Register(ctx, cli,
			registry.ServiceInfo{Name: *name, Addr: *addr}, *ttl)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("已注册 %s@%s,租约保活中(TTL=%ds)。Ctrl+C 优雅下线\n", *name, *addr, *ttl)

		time.Sleep(2 * time.Second)
		fmt.Println("--- 2 秒后主动下线(Deregister,撤销租约)---")
		if err := reg.Deregister(ctx); err != nil {
			log.Fatal(err)
		}
		fmt.Println("已下线,再 list 就看不到这个实例了")
	}
}
