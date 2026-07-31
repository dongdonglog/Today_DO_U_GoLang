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

	// 1. Pipeline：批量发送，减少网络开销
	fmt.Println("\n=== 1. Pipeline ===")
	pipe := rdb.Pipeline()

	// 批量设置 100 个 key
	start := time.Now()
	for i := 0; i < 100; i++ {
		pipe.Set(ctx, fmt.Sprintf("pipeline:key:%d", i), i, 0)
	}
	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Fatal("Pipeline 失败:", err)
	}
	fmt.Printf("Pipeline 批量设置 100 个 key 耗时: %v\n", time.Since(start))

	// 批量获取
	pipe = rdb.Pipeline()
	for i := 0; i < 100; i++ {
		pipe.Get(ctx, fmt.Sprintf("pipeline:key:%d", i))
	}
	cmds, err := pipe.Exec(ctx)
	if err != nil {
		log.Fatal("Pipeline 获取失败:", err)
	}
	fmt.Printf("Pipeline 批量获取 100 个 key 耗时: %v\n", time.Since(start))
	fmt.Printf("返回 %d 个结果\n", len(cmds))

	// 对比：逐条设置
	fmt.Println("\n=== 2. 逐条设置（对比）===")
	start = time.Now()
	for i := 0; i < 100; i++ {
		rdb.Set(ctx, fmt.Sprintf("normal:key:%d", i), i, 0)
	}
	fmt.Printf("逐条设置 100 个 key 耗时: %v\n", time.Since(start))

	// 2. 事务 Pipeline
	fmt.Println("\n=== 3. 事务 Pipeline ===")
	rdb.Del(ctx, "counter_a", "counter_b")
	tx := rdb.TxPipeline()
	tx.Incr(ctx, "counter_a")
	tx.Incr(ctx, "counter_a")
	tx.Incr(ctx, "counter_b")
	_, err = tx.Exec(ctx)
	if err != nil {
		log.Fatal("事务失败:", err)
	}

	a, _ := rdb.Get(ctx, "counter_a").Int()
	b, _ := rdb.Get(ctx, "counter_b").Int()
	fmt.Printf("counter_a: %d, counter_b: %d\n", a, b)
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
