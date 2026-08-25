package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	etcdresolver "go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	userpb "github.com/go-book/grpc/proto/user"
	"github.com/go-book/servicediscovery/registry"
)

// userServer 是最小化的用户服务实现(第25章的简化版)
type userServer struct {
	userpb.UnimplementedUserServiceServer
	addr string // 本实例的地址,回显在响应里,方便观察轮询
}

func (s *userServer) GetUser(ctx context.Context, req *userpb.GetUserReq) (*userpb.User, error) {
	return &userpb.User{Id: req.GetId(), Username: "alice", Email: s.addr, Status: 1}, nil
}

// 演示 etcd + gRPC 的端到端整合:
//
//	go run . -role=server -addr=:50061 -name=user-service   # 终端 A:实例 1
//	go run . -role=server -addr=:50062 -name=user-service   # 终端 B:实例 2
//	go run . -role=client                                   # 终端 C:round_robin 轮询两个实例
//
// 客户端不写死地址,只写 "etcd:///user-service":
// gRPC 通过 etcd resolver 拿到存活实例列表,并用 round_robin 在它们之间轮询。
// kill 掉一个实例,租约到期后客户端自动摘除,无需重启。
func main() {
	role := flag.String("role", "client", "server / client")
	name := flag.String("name", "user-service", "服务名")
	addr := flag.String("addr", ":50061", "服务监听地址")
	ttl := flag.Int64("ttl", 5, "注册 TTL(秒)")
	flag.Parse()

	if *role == "server" {
		runServer(*name, *addr, *ttl)
		return
	}
	runClient(*name)
}

func runServer(name, addr string, ttl int64) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()
	ctx := context.Background()

	host := fmt.Sprintf("localhost%s", addr) // 本机演示;真实环境是 Pod IP
	reg, err := registry.Register(ctx, cli,
		registry.ServiceInfo{Name: name, Addr: host}, ttl)
	if err != nil {
		log.Fatal(err)
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s 实例启动 %s(已注册到 etcd,Ctrl+C 下线)\n", name, addr)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		dctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = reg.Deregister(dctx)
		os.Exit(0)
	}()

	s := grpc.NewServer()
	userpb.RegisterUserServiceServer(s, &userServer{addr: host})
	log.Fatal(s.Serve(lis))
}

func runClient(serviceName string) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 3 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close()

	// 1. 构造 etcd resolver,注册为 "etcd" scheme
	builder, err := etcdresolver.NewBuilder(cli)
	if err != nil {
		log.Fatal(err)
	}

	// 2. 连接时用 etcd:///<service-name>;负载均衡用 round_robin
	conn, err := grpc.NewClient(
		"etcd:///"+serviceName,
		grpc.WithResolvers(builder),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig":[{"round_robin":{}}]}`),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	c := userpb.NewUserServiceClient(conn)

	for i := 1; i <= 6; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		resp, err := c.GetUser(ctx, &userpb.GetUserReq{Id: int64(i)})
		cancel()
		if err != nil {
			log.Printf("调用失败: %v", err)
			continue
		}
		fmt.Printf("第 %d 次调用 -> 用户=%s(实例 %s)\n", i, resp.GetUsername(), resp.GetEmail())
		time.Sleep(200 * time.Millisecond)
	}
}
