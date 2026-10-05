package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-book/graduate/ecommerce-platform/internal/platform"
	"github.com/go-book/graduate/ecommerce-platform/internal/store"
	invpb "github.com/go-book/graduate/ecommerce-platform/proto/inventory"
	"google.golang.org/grpc"
)

type inventoryServer struct {
	invpb.UnimplementedInventoryServiceServer
	store *store.Store
}

func (s *inventoryServer) GetStock(ctx context.Context, req *invpb.GetStockReq) (*invpb.Stock, error) {
	if req.GetSlow() {
		select {
		case <-time.After(500 * time.Millisecond):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	stock, err := s.store.GetStock(ctx, req.GetSkuId())
	if err != nil {
		return nil, err
	}
	return &invpb.Stock{SkuId: req.GetSkuId(), Available: int32(stock)}, nil
}

func main() {
	cfg := platform.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := store.Open(ctx, cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("connect mysql: %v", err)
	}
	defer db.Close()

	listener, err := net.Listen("tcp", cfg.InventoryAddr)
	if err != nil {
		log.Fatalf("listen inventory: %v", err)
	}
	server := grpc.NewServer()
	invpb.RegisterInventoryServiceServer(server, &inventoryServer{store: db})

	go func() {
		log.Printf("inventory grpc listening on %s", cfg.InventoryAddr)
		if err := server.Serve(listener); err != nil {
			log.Fatalf("serve inventory: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	server.GracefulStop()
}
