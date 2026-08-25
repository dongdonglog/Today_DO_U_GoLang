package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// 限流:保护自己不被突发流量打垮。
//
//	go run .                # 启动,内置压测
//
// 演示两种限流:
//   1. 全局限流:整个服务每秒最多 N 个请求(x/time/rate 令牌桶)
//   2. 按 key 限流:每个用户独立配额(常见于 API 网关)
//
// x/time/rate 是官方维护的令牌桶实现:Limiter 允许"预支"令牌(burst),
// Wait 阻塞等待 / Allow 直接拒绝,按场景选择。

func main() {
	r := gin.Default()

	// 全局限流:10 QPS,桶容量 20(允许瞬时突发 20 个)
	globalLimiter := rate.NewLimiter(rate.Limit(10), 20)

	// 按 key 限流:每个用户 2 QPS,容量 3
	users := &keyedLimiter{limits: map[string]*rate.Limiter{}}
	r.Use(func(c *gin.Context) {
		user := c.GetHeader("X-User-ID")
		if user == "" {
			user = "anonymous"
		}
		if !users.allow(user) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests,
				gin.H{"code": 429, "message": "rate limited (per-user)"})
			return
		}
		c.Next()
	})

	r.GET("/api/orders", func(c *gin.Context) {
		if !globalLimiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests,
				gin.H{"code": 429, "message": "rate limited (global)"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": []string{"order-1", "order-2"}})
	})

	go func() {
		if err := r.Run(":8080"); err != nil {
			log.Fatal(err)
		}
	}()
	time.Sleep(300 * time.Millisecond)

	bench()
}

type keyedLimiter struct {
	mu     sync.Mutex
	limits map[string]*rate.Limiter
}

func (k *keyedLimiter) allow(key string) bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	l, ok := k.limits[key]
	if !ok {
		l = rate.NewLimiter(2, 3)
		k.limits[key] = l
	}
	return l.Allow()
}

// bench 用 30 并发打接口,统计成功/限流分布
func bench() {
	client := &http.Client{Timeout: 2 * time.Second}
	var (
		ok200, tooMany int
		mu             sync.Mutex
		wg             sync.WaitGroup
	)

	fire := func(userID string) {
		defer wg.Done()
		req, _ := http.NewRequest("GET", "http://localhost:8080/api/orders", nil)
		req.Header.Set("X-User-ID", userID)
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()
		mu.Lock()
		defer mu.Unlock()
		switch resp.StatusCode {
		case 200:
			ok200++
		case 429:
			tooMany++
		}
	}

	fmt.Println("=== 单用户连发 12 次(per-user 配额 2QPS/桶3)===")
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go fire("user-A")
	}
	wg.Wait()
	fmt.Printf("200=%d 429=%d\n", ok200, tooMany)

	ok200, tooMany = 0, 0
	fmt.Println("\n=== 三个用户各发 6 次 ===")
	for _, u := range []string{"u1", "u2", "u3"} {
		for i := 0; i < 6; i++ {
			wg.Add(1)
			go fire(u)
		}
	}
	wg.Wait()
	fmt.Printf("200=%d 429=%d(全局 10QPS 也可能拦)\n", ok200, tooMany)

	// 展示 Limiter 的阻塞用法:Wait 会等到令牌可用而不是拒绝
	l := rate.NewLimiter(rate.Limit(5), 1)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	start := time.Now()
	for i := 0; i < 5; i++ {
		if err := l.Wait(ctx); err != nil {
			break // 超时
		}
	}
	fmt.Printf("\nWait 方式匀速发 5 个请求(5QPS):耗时 %v\n",
		time.Since(start).Round(100*time.Millisecond))
	_ = json.Marshal // 占位避免未使用告警
}
