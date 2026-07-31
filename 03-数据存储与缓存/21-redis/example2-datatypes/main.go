package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	addr := getenv("REDIS_ADDR", "localhost:6379")
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	defer rdb.Close()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Fatal("连接失败:", err)
	}
	fmt.Println("连接成功")

	// 清理上一轮运行残留，保证输出可复现
	keys := []string{
		"product:1", "views:product:1", "cart:user:1001",
		"recent_orders", "read:1001", "hot_products",
	}
	if err := rdb.Del(ctx, keys...).Err(); err != nil {
		log.Fatal("清理失败:", err)
	}

	// 1. String：商品详情缓存
	fmt.Println("\n=== 1. String：商品详情缓存 ===")
	productDetail := `{"id":1,"name":"iPhone 15","price":6999.00}`
	if err := rdb.Set(ctx, "product:1", productDetail, 10*time.Minute).Err(); err != nil {
		log.Fatal("Set 失败:", err)
	}
	cached, err := rdb.Get(ctx, "product:1").Result()
	if err != nil {
		log.Fatal("Get 失败:", err)
	}
	fmt.Println("商品详情:", cached)

	// 计数器：商品浏览量
	views, err := rdb.Incr(ctx, "views:product:1").Result()
	if err != nil {
		log.Fatal("Incr 失败:", err)
	}
	fmt.Println("浏览量:", views)

	// 2. Hash：购物车
	fmt.Println("\n=== 2. Hash：购物车 ===")
	userID := "user:1001"
	if err := rdb.HSet(ctx, "cart:"+userID, "product:1", 2, "product:2", 1).Err(); err != nil {
		log.Fatal("HSet 失败:", err)
	}
	if err := rdb.HIncrBy(ctx, "cart:"+userID, "product:1", 1).Err(); err != nil { // 加 1 件
		log.Fatal("HIncrBy 失败:", err)
	}
	cart, err := rdb.HGetAll(ctx, "cart:"+userID).Result()
	if err != nil {
		log.Fatal("HGetAll 失败:", err)
	}
	fmt.Println("购物车:", cart)

	// 3. List：最新订单列表
	fmt.Println("\n=== 3. List：最新订单列表 ===")
	orders := []string{"ORD001", "ORD002", "ORD003", "ORD004", "ORD005"}
	for _, o := range orders {
		if err := rdb.LPush(ctx, "recent_orders", o).Err(); err != nil {
			log.Fatal("LPush 失败:", err)
		}
	}
	recent, err := rdb.LRange(ctx, "recent_orders", 0, 2).Result()
	if err != nil {
		log.Fatal("LRange 失败:", err)
	}
	fmt.Println("最近 3 条订单:", recent)

	// 4. Set：用户已读商品
	fmt.Println("\n=== 4. Set：用户已读商品 ===")
	if err := rdb.SAdd(ctx, "read:1001", "product:1", "product:2", "product:3").Err(); err != nil {
		log.Fatal("SAdd 失败:", err)
	}
	read, err := rdb.SMembers(ctx, "read:1001").Result()
	if err != nil {
		log.Fatal("SMembers 失败:", err)
	}
	fmt.Println("已读商品:", read)

	done, err := rdb.SIsMember(ctx, "read:1001", "product:1").Result()
	if err != nil {
		log.Fatal("SIsMember 失败:", err)
	}
	fmt.Println("是否已读 product:1:", done)

	// 5. ZSet：热销排行榜
	fmt.Println("\n=== 5. ZSet：热销排行榜 ===")
	sales := []redis.Z{
		{Score: 10, Member: "product:1"},
		{Score: 5, Member: "product:2"},
		{Score: 20, Member: "product:3"},
	}
	for _, s := range sales {
		if err := rdb.ZIncrBy(ctx, "hot_products", s.Score, s.Member.(string)).Err(); err != nil {
			log.Fatal("ZIncrBy 失败:", err)
		}
	}
	top, err := rdb.ZRevRangeWithScores(ctx, "hot_products", 0, 2).Result()
	if err != nil {
		log.Fatal("ZRevRangeWithScores 失败:", err)
	}
	fmt.Println("热销排行榜:")
	for _, z := range top {
		fmt.Printf("  %s: 销量 %.0f\n", z.Member, z.Score)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
