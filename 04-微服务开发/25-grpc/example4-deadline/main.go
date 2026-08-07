package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/go-book/grpc/internal/server"
	invpb "github.com/go-book/grpc/proto/inventory"
)

// deadline / 取消传播：客户端设的超时会通过 HTTP/2 的 RST_STREAM / Trailers
// 传给服务端，服务端 ctx.Done() 会被触发，从而及时停下慢查询。
// 这是微服务“雪崩防护”的第一道关：上游超时了，下游就别再白干活。

func main() {
	addr := flag.String("addr", "localhost:50051", "监听或连接地址")
	role := flag.String("role", "server", "server 或 client")
	slow := flag.Bool("slow", false, "调用时让服务端慢处理")
	timeout := flag.Duration("timeout", 200*time.Millisecond, "客户端超时")
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
	c := invpb.NewInventoryServiceClient(conn)

	// 用 WithTimeout 设客户端 deadline；到点后 ctx 自动取消
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	start := time.Now()
	resp, err := c.GetStock(ctx, &invpb.GetStockReq{SkuId: 1001, Slow: *slow})
	cost := time.Since(start)

	if err != nil {
		st, _ := status.FromError(err)
		// 超时会返回 codes.DeadlineExceeded
		fmt.Printf("耗时=%v code=%s msg=%s\n", cost, st.Code(), st.Message())
		if errors.Is(err, context.DeadlineExceeded) || st.Code() == codes.DeadlineExceeded {
			fmt.Println("=> 触发 deadline，服务端会收到 ctx 取消并停止慢操作")
		}
		return
	}
	fmt.Printf("耗时=%v 库存=%d\n", cost, resp.GetAvailable())
}
