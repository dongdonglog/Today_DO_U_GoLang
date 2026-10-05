package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

// 一个最小的 Go HTTP 服务,专门用来演示 Docker 多阶段构建。
// 刻意只用标准库:编译产物是纯静态二进制,能塞进 scratch 空镜像。

type buildInfo struct {
	GoVersion string `json:"go_version"`
	BuildTime string `json:"build_time"`
	Stage     string `json:"stage"`
}

var startTime = time.Now()

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/info", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(buildInfo{
			GoVersion: os.Getenv("GO_VERSION"),
			BuildTime: os.Getenv("BUILD_TIME"),
			Stage:     "runtime",
		})
	})

	mux.HandleFunc("/uptime", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(time.Since(startTime).Round(time.Second).String()))
	})

	addr := ":8080"
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
