package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/go-book/grpc/internal/server"
	userpb "github.com/go-book/grpc/proto/user"
)

// 拦截器（interceptor）相当于 gRPC 版的 HTTP 中间件：
// 在真正的 handler 前后统一做鉴权、日志、指标、recover。
// 这里演示最常用的一元拦截器：鉴权 + 访问日志。

const tokenKey = "authorization"

// authUnaryServerInterceptor 校验 metadata 里的 token
func authUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok || len(md.Get(tokenKey)) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing token")
		}
		token := md.Get(tokenKey)[0]
		if token != "Bearer valid-token" {
			return nil, status.Errorf(codes.PermissionDenied, "invalid token")
		}
		// 把解析出的用户身份塞进 ctx，供下游 handler 使用
		ctx = context.WithValue(ctx, ctxUserKey{}, "alice")
		return handler(ctx, req)
	}
}

type ctxUserKey struct{}

// loggingUnaryServerInterceptor 打印方法耗时与状态码
func loggingUnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		log.Printf("method=%s code=%s duration=%s", info.FullMethod, status.Code(err), time.Since(start))
		return resp, err
	}
}

func main() {
	addr := flag.String("addr", "localhost:50051", "监听或连接地址")
	role := flag.String("role", "server", "server 或 client")
	token := flag.String("token", "Bearer valid-token", "客户端携带的 token")
	flag.Parse()

	if *role == "server" {
		s := server.NewAllServer(
			grpc.ChainUnaryInterceptor(
				loggingUnaryServerInterceptor(),
				authUnaryServerInterceptor(),
			),
		)
		log.Printf("gRPC server(带拦截器) 监听 %s", *addr)
		log.Fatal(server.Serve(s, *addr))
	}

	conn, err := grpc.NewClient(*addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()
	c := userpb.NewUserServiceClient(conn)

	call := func(tok string) {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		// 通过 metadata 透传认证信息（gRPC 版的 header）
		md := metadata.Pairs(tokenKey, tok)
		ctx = metadata.NewOutgoingContext(ctx, md)

		resp, err := c.GetUser(ctx, &userpb.GetUserReq{Id: 1})
		if err != nil {
			st, _ := status.FromError(err)
			fmt.Printf("token=%q -> code=%s msg=%s\n", tok, st.Code(), st.Message())
			return
		}
		fmt.Printf("token=%q -> 用户=%s\n", tok, resp.GetUsername())
	}

	call(*token)
	call("Bearer wrong-token")
	call("") // 不带 token
}
