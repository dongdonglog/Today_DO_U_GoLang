package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/go-book/grpc/internal/server"
	orderpb "github.com/go-book/grpc/proto/order"
)

// 服务端流式 RPC：客户端发一个请求，服务端返回多条消息。
// 适合“一个请求对应一批结果”的场景，比如分页拉取、实时增量。
func main() {
	addr := flag.String("addr", "localhost:50051", "监听或连接地址")
	role := flag.String("role", "server", "server 或 client")
	userID := flag.Int64("user", 1, "要查询的用户 ID")
	flag.Parse()

	if *role == "server" {
		log.Printf("gRPC server 监听 %s", *addr)
		log.Fatal(server.Serve(server.NewAllServer(), *addr))
	}

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()

	c := orderpb.NewOrderServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stream, err := c.ListOrders(ctx, &orderpb.ListOrdersReq{UserId: *userID})
	if err != nil {
		log.Fatalf("ListOrders 失败: %v", err)
	}

	var (
		total int64
		count int
	)
	for {
		o, err := stream.Recv()
		if err == io.EOF {
			break // 服务端关闭流，读取结束
		}
		if err != nil {
			log.Fatalf("Recv 失败: %v", err)
		}
		fmt.Printf("收到订单: %s  金额=%d分  状态=%s\n", o.GetOrderNo(), o.GetAmount(), o.GetStatus())
		total += o.GetAmount()
		count++
	}
	fmt.Printf("=== 共 %d 个订单，合计 %d 分\n", count, total)
}
