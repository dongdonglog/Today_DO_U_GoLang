package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

// version 由 Dockerfile 构建期注入:go build -ldflags "-X main.version=v2"
var version string

// 部署单元演示服务:支持多版本构建(v1/v2),用于滚动更新实验。
// /info 返回版本号,滚动更新时用 curl 看响应在 v1/v2 之间渐进切换。
func main() {
	if version == "" {
		version = "dev"
	}

	start := time.Now()
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/info", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"version": version,
			"uptime":  time.Since(start).Round(time.Second).String(),
		})
	})

	log.Printf("listening on :8080 (%s)", version)
	log.Fatal(http.ListenAndServe(":8080", mux))
}
