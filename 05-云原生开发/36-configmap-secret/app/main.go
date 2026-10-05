package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// 配置注入演示服务:展示 ConfigMap/Secret 的两种注入方式在应用侧的样子。
//
//   - 环境变量注入:GREETING 来自 ConfigMap,DB_PASSWORD 来自 Secret
//   - 文件挂载注入:/etc/app/config.yaml 来自 ConfigMap(卷挂载,可热更新)
//
// /env   返回环境变量注入的配置
// /file  返回文件挂载的配置原文
// /mask  返回敏感信息的脱敏视图(演示 Secret 存在,但不外泄)
func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/env", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"greeting":     os.Getenv("GREETING"),
			"db_host":      os.Getenv("DB_HOST"),
			"log_level":    os.Getenv("LOG_LEVEL"),
			"has_password": fmt.Sprintf("%v", os.Getenv("DB_PASSWORD") != ""),
		})
	})

	mux.HandleFunc("/file", func(w http.ResponseWriter, _ *http.Request) {
		b, err := os.ReadFile("/etc/app/config.yaml")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(b)
	})

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	log.Printf("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
