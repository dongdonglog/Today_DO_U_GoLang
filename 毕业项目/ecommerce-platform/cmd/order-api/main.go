package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-book/graduate/ecommerce-platform/internal/platform"
	"github.com/go-book/graduate/ecommerce-platform/internal/store"
	invpb "github.com/go-book/graduate/ecommerce-platform/proto/inventory"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type api struct {
	store     *store.Store
	redis     *redis.Client
	inventory invpb.InventoryServiceClient
}

type createOrderRequest struct {
	UserID   int64 `json:"user_id" binding:"required"`
	SkuID    int64 `json:"sku_id" binding:"required"`
	Quantity int64 `json:"quantity" binding:"required,min=1,max=99"`
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

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	defer redisClient.Close()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("connect redis: %v", err)
	}

	conn, err := grpc.NewClient(cfg.InventoryTarget, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("connect inventory: %v", err)
	}
	defer conn.Close()

	a := &api{
		store:     db,
		redis:     redisClient,
		inventory: invpb.NewInventoryServiceClient(conn),
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/healthz", a.health)
	r.POST("/v1/orders", a.createOrder)
	r.GET("/v1/orders/:order_no", a.getOrder)

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	go func() {
		log.Printf("order api listening on %s", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve order api: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
}

func (a *api) health(c *gin.Context) {
	ctx := c.Request.Context()
	status := http.StatusOK
	deps := map[string]string{"mysql": "ok", "redis": "ok", "inventory": "ok"}
	if err := a.store.Ping(ctx); err != nil {
		deps["mysql"] = "unavailable"
		status = http.StatusServiceUnavailable
	}
	if err := a.redis.Ping(ctx).Err(); err != nil {
		deps["redis"] = "unavailable"
		status = http.StatusServiceUnavailable
	}
	grpcCtx, cancel := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancel()
	if _, err := a.inventory.GetStock(grpcCtx, &invpb.GetStockReq{SkuId: 1001}); err != nil {
		deps["inventory"] = "unavailable"
		status = http.StatusServiceUnavailable
	}
	c.JSON(status, gin.H{"dependencies": deps})
}

func (a *api) createOrder(c *gin.Context) {
	idem := c.GetHeader("Idempotency-Key")
	if idem == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing Idempotency-Key"})
		return
	}
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if orderNo, err := a.redis.Get(c.Request.Context(), "idem:"+idem).Result(); err == nil {
		if order, err := a.store.GetOrderByNo(c.Request.Context(), orderNo); err == nil {
			c.JSON(http.StatusOK, gin.H{"data": order, "idempotent": true})
			return
		}
	}

	grpcCtx, cancel := context.WithTimeout(c.Request.Context(), 300*time.Millisecond)
	stock, err := a.inventory.GetStock(grpcCtx, &invpb.GetStockReq{SkuId: req.SkuID})
	cancel()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "inventory unavailable"})
		return
	}
	if int64(stock.GetAvailable()) < req.Quantity {
		c.JSON(http.StatusConflict, gin.H{"error": "not enough stock"})
		return
	}

	order, created, err := a.store.CreateOrder(c.Request.Context(), store.CreateOrderInput{
		IdempotencyKey: idem,
		UserID:         req.UserID,
		SkuID:          req.SkuID,
		Quantity:       req.Quantity,
	})
	if err != nil {
		writeStoreError(c, err)
		return
	}
	_ = a.redis.Set(c.Request.Context(), "idem:"+idem, order.OrderNo, 24*time.Hour).Err()
	if created {
		c.JSON(http.StatusCreated, gin.H{"data": order})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": order, "idempotent": true})
}

func (a *api) getOrder(c *gin.Context) {
	order, err := a.store.GetOrderByNo(c.Request.Context(), c.Param("order_no"))
	if err != nil {
		writeStoreError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": order})
}

func writeStoreError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, store.ErrNoStock):
		c.JSON(http.StatusConflict, gin.H{"error": "not enough stock"})
	case errors.Is(err, store.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	case errors.Is(err, store.ErrConflict):
		c.JSON(http.StatusConflict, gin.H{"error": "conflict"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "request failed"})
	}
}
