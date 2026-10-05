package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

// 计数器服务:演示容器编排下的应用如何连接同网络内的依赖容器。
//
// 主机名直接写 compose 服务名(redis),由 Docker 内置 DNS 解析——
// 这是容器互联最基本的方式,也是 K8s Service 的前身思路。
//
// 访问 /count 每次自增并返回;/healthz 探测 Redis 连通性(供 compose healthcheck)。

func main() {
	// scratch 镜像里没有任何工具(wget/curl 都没有),healthcheck 没法调外部命令。
	// 惯用解法:让应用自己支持 "-healthcheck" 子命令,Dockerfile 的 HEALTHCHECK 调用它。
	if len(os.Args) > 1 && os.Args[1] == "-healthcheck" {
		resp, err := http.Get("http://localhost:" + portOf(getenv("LISTEN_ADDR", ":8080")) + "/healthz")
		if err != nil || resp.StatusCode != http.StatusOK {
			os.Exit(1)
		}
		os.Exit(0)
	}

	redisAddr := getenv("REDIS_ADDR", "redis:6379") // 服务名当主机名
	rdb := redis.NewClient(&redis.Options{Addr: redisAddr})
	ctx := context.Background()

	// 容器启动时依赖可能还没就绪:重试等待而不是直接 Fatal(compose 的 depends_on condition 解决这个问题)
	for i := 0; i < 30; i++ {
		if err := rdb.Ping(ctx).Err(); err == nil {
			log.Printf("redis 已就绪: %s", redisAddr)
			break
		}
		if i == 29 {
			log.Fatal("redis 始终连不上")
		}
		time.Sleep(time.Second)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		if err := rdb.Ping(ctx).Err(); err != nil {
			http.Error(w, "redis unreachable", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/count", func(w http.ResponseWriter, _ *http.Request) {
		n, err := rdb.Incr(ctx, "demo:counter").Result()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"count": n, "redis": redisAddr})
	})

	addr := getenv("LISTEN_ADDR", ":8080")
	log.Printf("listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// portOf 从 ":8080" 形式的地址里取端口
func portOf(addr string) string {
	if i := len(addr) - 1; i >= 0 {
		for j := i; j >= 0; j-- {
			if addr[j] == ':' {
				return addr[j+1:]
			}
		}
	}
	return addr
}
