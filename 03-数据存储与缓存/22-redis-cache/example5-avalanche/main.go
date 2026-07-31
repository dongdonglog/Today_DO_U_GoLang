// example5-avalanche 演示缓存雪崩的防御：TTL 随机抖动。
//
// 缓存雪崩：大批 key 在同一时刻集体过期，导致数据库瞬时被回源流量打爆。
// 最常见诱因是预热时给一批 key 设了相同的 TTL。
package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// jitterTTL 在 base 上叠加 0~10% 的随机抖动，把过期时间打散，
// 避免大批 key 同时过期。
func jitterTTL(base time.Duration) time.Duration {
	jitter := time.Duration(rand.Int63n(int64(base) / 10))
	return base + jitter
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	rdb := redis.NewClient(&redis.Options{Addr: getenv("REDIS_ADDR", "localhost:6379")})
	defer rdb.Close()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("连接 Redis 失败: %v\n启动前请执行:\n  docker run --name go-book-redis -p 6379:6379 -d redis:7-alpine", err)
	}

	base := 60 * time.Minute

	// --- 反例：固定 TTL，1000 个 key 的过期时间完全相同 ---
	fmt.Println("=== 固定 TTL（雪崩隐患）===")
	pipe := rdb.Pipeline()
	for i := 0; i < 1000; i++ {
		pipe.Set(ctx, fmt.Sprintf("warm:fixed:%d", i), "v", base)
	}
	_, _ = pipe.Exec(ctx)
	printTTLSpread(ctx, rdb, "warm:fixed:")

	// --- 正例：TTL 加随机抖动，过期时间被打散 ---
	fmt.Println("\n=== 抖动 TTL（削平回源尖刺）===")
	pipe = rdb.Pipeline()
	for i := 0; i < 1000; i++ {
		pipe.Set(ctx, fmt.Sprintf("warm:jitter:%d", i), "v", jitterTTL(base))
	}
	_, _ = pipe.Exec(ctx)
	printTTLSpread(ctx, rdb, "warm:jitter:")

	// 清理
	for _, prefix := range []string{"warm:fixed:", "warm:jitter:"} {
		for i := 0; i < 1000; i++ {
			rdb.Del(ctx, fmt.Sprintf("%s%d", prefix, i))
		}
	}
}

// printTTLSpread 采样若干 key 的剩余 TTL，观察它们是否集中在同一秒。
func printTTLSpread(ctx context.Context, rdb *redis.Client, prefix string) {
	min, max := time.Duration(1<<62), time.Duration(0)
	for i := 0; i < 1000; i++ {
		ttl, err := rdb.TTL(ctx, fmt.Sprintf("%s%d", prefix, i)).Result()
		if err != nil {
			continue
		}
		if ttl < min {
			min = ttl
		}
		if ttl > max {
			max = ttl
		}
	}
	fmt.Printf("1000 个 key 的过期时间跨度: %v ~ %v（跨度 %v）\n",
		min.Round(time.Second), max.Round(time.Second), (max - min).Round(time.Second))
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
