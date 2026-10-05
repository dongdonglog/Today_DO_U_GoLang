package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	maxRequestBytes  = 8 << 10
	maxResponseBytes = 2 << 20
)

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model     string         `json:"model"`
	Messages  []message      `json:"messages"`
	Stream    bool           `json:"stream"`
	KeepAlive string         `json:"keep_alive,omitempty"`
	Options   map[string]any `json:"options,omitempty"`
}

type chatResponse struct {
	Model   string  `json:"model"`
	Message message `json:"message"`
	Done    bool    `json:"done"`
	Error   string  `json:"error,omitempty"`
}

func main() {
	question := flag.String("question", "", "question to send to Ollama")
	flag.Parse()
	if strings.TrimSpace(*question) == "" || len(*question) > 4000 {
		fatal("question is required and must be at most 4000 bytes", 2)
	}

	baseURL := strings.TrimRight(env("OLLAMA_BASE_URL", "http://localhost:11434"), "/")
	model := env("OLLAMA_MODEL", "qwen2.5:3b")
	endpoint, err := url.JoinPath(baseURL, "/api/chat")
	if err != nil {
		fatal("invalid OLLAMA_BASE_URL", 2)
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		fatal("OLLAMA_BASE_URL must be an http or https URL", 2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 90 * time.Second}
	answer, err := chat(ctx, client, endpoint, model, *question)
	if err != nil {
		fatal("Ollama request failed: "+err.Error(), 1)
	}
	fmt.Println(answer)
}

func chat(ctx context.Context, client *http.Client, endpoint, model, question string) (string, error) {
	payload, err := json.Marshal(chatRequest{
		Model: model,
		Messages: []message{
			{Role: "system", Content: "你是客服知识助手。回答应简洁；不确定时说明需要查阅正式规则，不要承诺退款或执行订单操作。"},
			{Role: "user", Content: question},
		},
		Stream:    false,
		KeepAlive: "5m",
		Options:   map[string]any{"temperature": 0.2, "num_predict": 600},
	})
	if err != nil {
		return "", err
	}
	if len(payload) > maxRequestBytes {
		return "", errors.New("request exceeds 8 KiB")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("endpoint returned %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return "", err
	}
	if len(body) > maxResponseBytes {
		return "", errors.New("response exceeds 2 MiB")
	}
	var result chatResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if result.Error != "" {
		return "", errors.New(result.Error)
	}
	if !result.Done || strings.TrimSpace(result.Message.Content) == "" {
		return "", errors.New("Ollama response contains no completed answer")
	}
	return strings.TrimSpace(result.Message.Content), nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func fatal(message string, code int) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(code)
}
