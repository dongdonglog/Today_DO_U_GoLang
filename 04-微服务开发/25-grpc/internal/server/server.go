package server

import (
	"net"

	"google.golang.org/grpc"

	invpb "github.com/go-book/grpc/proto/inventory"
	orderpb "github.com/go-book/grpc/proto/order"
	userpb "github.com/go-book/grpc/proto/user"
)

// NewAllServer 创建一个注册了全部三个 service 的 *grpc.Server。
// 各 example 复用它，通过命令行参数选择启动哪个演示。
func NewAllServer(opts ...grpc.ServerOption) *grpc.Server {
	s := grpc.NewServer(opts...)
	userpb.RegisterUserServiceServer(s, &UserServer{})
	orderpb.RegisterOrderServiceServer(s, &OrderServer{})
	invpb.RegisterInventoryServiceServer(s, &InventoryServer{})
	return s
}

// Serve 在 addr 上监听并启动 s，阻塞直到出错。
func Serve(s *grpc.Server, addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.Serve(lis)
}
