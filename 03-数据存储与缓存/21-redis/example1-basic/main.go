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

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 5,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	defer rdb.Close()

	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("连接失败: %v\n启动前请执行:\n  docker run --name go-book-redis -p 6379:6379 -d redis:7-alpine", err)
	}
	fmt.Println("连接成功:", pong)

	// 1. Set/Get
	fmt.Println("\n=== 1. Set/Get ===")
	err = rdb.Set(ctx, "greeting", "hello redis", 0).Err()
	if err != nil {
		log.Fatal(err)
	}
	val, err := rdb.Get(ctx, "greeting").Result()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("greeting:", val)

	// 2. 过期时间
	fmt.Println("\n=== 2. 过期时间 ===")
	rdb.Set(ctx, "temp_key", "will expire", 3*time.Second)
	ttl, _ := rdb.TTL(ctx, "temp_key").Result()
	fmt.Println("剩余时间:", ttl)
	time.Sleep(4 * time.Second)
	_, err = rdb.Get(ctx, "temp_key").Result()
	if err == redis.Nil {
		fmt.Println("key 已过期")
	}

	// 3. 计数器
	fmt.Println("\n=== 3. 计数器 ===")
	rdb.Del(ctx, "counter")
	for i := 0; i < 5; i++ {
		val, _ := rdb.Incr(ctx, "counter").Result()
		fmt.Printf("第 %d 次访问\n", val)
	}

	// 4. 批量操作
	fmt.Println("\n=== 4. 批量操作 ===")
	rdb.MSet(ctx, "k1", "v1", "k2", "v2", "k3", "v3")
	vals, _ := rdb.MGet(ctx, "k1", "k2", "k3").Result()
	for _, v := range vals {
		fmt.Println("批量查询:", v)
	}

	// 5. 连接池统计
	fmt.Println("\n=== 5. 连接池统计 ===")
	stats := rdb.PoolStats()
	fmt.Printf("总连接数: %d\n", stats.TotalConns)
	fmt.Printf("空闲连接: %d\n", stats.IdleConns)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
