package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/go-book/grpc/internal/server"
	userpb "github.com/go-book/grpc/proto/user"
)

// 一元（unary）RPC：一次请求一次响应，最接近普通的函数调用。
// 一个 server 可以注册多个 service。
func main() {
	addr := flag.String("addr", "localhost:50051", "监听或连接地址")
	role := flag.String("role", "server", "server 或 client")
	flag.Parse()

	if *role == "server" {
		runServer(*addr)
		return
	}
	runClient(*addr)
}

func runServer(addr string) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("监听失败: %v", err)
	}
	// 明文传输仅用于本地演示；生产必须用 TLS / mTLS（见后文）
	s := grpc.NewServer()
	userpb.RegisterUserServiceServer(s, &server.UserServer{})
	log.Printf("gRPC server 监听 %s", addr)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Serve 失败: %v", err)
	}
}

func runClient(addr string) {
	// grpc.NewClient 是当前推荐的连接构造方式（grpc.Dial 已废弃）。
	// 默认开启异步重连、连接就绪探测；本地演示用 insecure 凭证。
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()

	c := userpb.NewUserServiceClient(conn)

	// 每次 RPC 都应带超时：超时由客户端说了算，服务端会收到 ctx 取消
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := c.GetUser(ctx, &userpb.GetUserReq{Id: 1})
	if err != nil {
		log.Fatalf("GetUser 失败: %v", err)
	}
	fmt.Printf("用户: id=%d username=%s email=%s status=%d\n",
		resp.GetId(), resp.GetUsername(), resp.GetEmail(), resp.GetStatus())
}
