package server

import (
	"context"
	"time"

	invpb "github.com/go-book/grpc/proto/inventory"
	orderpb "github.com/go-book/grpc/proto/order"
	userpb "github.com/go-book/grpc/proto/user"
)

// UserServer 实现 user.UserService
type UserServer struct {
	userpb.UnimplementedUserServiceServer
}

func (s *UserServer) GetUser(ctx context.Context, req *userpb.GetUserReq) (*userpb.User, error) {
	return &userpb.User{
		Id:       req.GetId(),
		Username: "alice",
		Email:    "alice@example.com",
		Status:   1,
	}, nil
}

// OrderServer 实现 order.OrderService
type OrderServer struct {
	orderpb.UnimplementedOrderServiceServer
}

func (s *OrderServer) ListOrders(req *orderpb.ListOrdersReq, stream orderpb.OrderService_ListOrdersServer) error {
	orders := []*orderpb.Order{
		{Id: 1001, OrderNo: "NO-20241001-001", UserId: req.GetUserId(), Amount: 19900, Status: "paid"},
		{Id: 1002, OrderNo: "NO-20241002-002", UserId: req.GetUserId(), Amount: 25900, Status: "shipped"},
		{Id: 1003, OrderNo: "NO-20241003-003", UserId: req.GetUserId(), Amount: 6900, Status: "created"},
	}
	for _, o := range orders {
		// 每条消息 Send 一次；真实场景里可以边查边发，不必一次性把结果攒进内存
		if err := stream.Send(o); err != nil {
			return err
		}
		time.Sleep(10 * time.Millisecond)
	}
	return nil
}

// InventoryServer 实现 inventory.InventoryService
type InventoryServer struct {
	invpb.UnimplementedInventoryServiceServer
}

func (s *InventoryServer) GetStock(ctx context.Context, req *invpb.GetStockReq) (*invpb.Stock, error) {
	if req.GetSlow() {
		// 故意慢处理，用来演示客户端 deadline 触发
		select {
		case <-time.After(500 * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return &invpb.Stock{SkuId: req.GetSkuId(), Available: 42}, nil
}
