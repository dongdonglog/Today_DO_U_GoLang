package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const maxResponseBytes = 1 << 20

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
	Stream      bool          `json:"stream"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func main() {
	endpoint := env("LLM_ENDPOINT", "http://localhost:11434/v1/chat/completions")
	model := env("LLM_MODEL", "qwen2.5:3b")
	question := "订单取消后，优惠券会自动退回吗？"

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	answer, err := complete(ctx, http.DefaultClient, endpoint, os.Getenv("LLM_API_KEY"), model, question)
	if err != nil {
		fmt.Fprintln(os.Stderr, "模型请求失败:", err)
		os.Exit(1)
	}
	fmt.Println(answer)
}

func complete(ctx context.Context, client *http.Client, endpoint, apiKey, model, question string) (string, error) {
	if strings.TrimSpace(question) == "" {
		return "", errors.New("question is empty")
	}
	if len(question) > 4000 {
		return "", errors.New("question exceeds 4000 bytes")
	}
	if strings.TrimSpace(endpoint) == "" || strings.TrimSpace(model) == "" {
		return "", errors.New("endpoint and model are required")
	}

	payload := chatRequest{
		Model:       model,
		Temperature: 0.2,
		MaxTokens:   600,
		Stream:      false,
		Messages: []chatMessage{
			{Role: "system", Content: "你是企业客服助手。只给出简洁、谨慎的候选答复；信息不足时明确说明需要查询业务规则。"},
			{Role: "user", Content: question},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("model endpoint returned %s", resp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	if len(data) > maxResponseBytes {
		return "", errors.New("model response exceeds 1 MiB")
	}

	var result chatResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if len(result.Choices) == 0 || strings.TrimSpace(result.Choices[0].Message.Content) == "" {
		return "", errors.New("model response contains no answer")
	}
	return result.Choices[0].Message.Content, nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
