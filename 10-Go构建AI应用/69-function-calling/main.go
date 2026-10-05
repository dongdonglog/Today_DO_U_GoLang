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
	"regexp"
	"strings"
	"time"
)

const (
	maxRounds             = 3
	maxToolCalls          = 2
	maxModelResponseBytes = 1 << 20
	maxOrderResponseBytes = 128 << 10
)

var orderIDPattern = regexp.MustCompile(`^ORD-[0-9]{6,16}$`)

type message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
}

type toolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type toolDefinition struct {
	Type     string `json:"type"`
	Function struct {
		Name        string         `json:"name"`
		Description string         `json:"description"`
		Parameters  map[string]any `json:"parameters"`
	} `json:"function"`
}

type chatRequest struct {
	Model       string           `json:"model"`
	Messages    []message        `json:"messages"`
	Tools       []toolDefinition `json:"tools"`
	ToolChoice  string           `json:"tool_choice"`
	Temperature float64          `json:"temperature"`
	MaxTokens   int              `json:"max_tokens"`
	Stream      bool             `json:"stream"`
}

type chatResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
}

type orderStatus struct {
	OrderID   string `json:"order_id"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
}

func main() {
	question := flag.String("question", "", "customer question")
	flag.Parse()
	if strings.TrimSpace(*question) == "" || len(*question) > 2000 {
		fmt.Fprintln(os.Stderr, "question is required and must be at most 2000 bytes")
		os.Exit(2)
	}

	endpoint := env("LLM_ENDPOINT", "http://localhost:11434/v1/chat/completions")
	model := env("LLM_MODEL", "qwen2.5:3b")
	orderEndpoint := env("ORDER_SERVICE_URL", "http://localhost:8081/internal/orders")
	callerID := strings.TrimSpace(os.Getenv("CALLER_USER_ID"))
	orderToken := os.Getenv("ORDER_SERVICE_TOKEN")
	apiKey := os.Getenv("LLM_API_KEY")
	client := &http.Client{Timeout: 90 * time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	answer, err := run(ctx, client, endpoint, apiKey, model, orderEndpoint, orderToken, callerID, *question)
	if err != nil {
		fmt.Fprintln(os.Stderr, "assistant request failed:", err)
		os.Exit(1)
	}
	fmt.Println(answer)
}

func run(ctx context.Context, client *http.Client, endpoint, apiKey, model, orderEndpoint, orderToken, callerID, question string) (string, error) {
	messages := []message{
		{Role: "system", Content: "你是客服助手。需要订单状态时调用 get_order_status。只能转述订单服务返回的状态，不承诺退款或执行订单操作。资料不足时说明需要人工核实。"},
		{Role: "user", Content: question},
	}
	tools := []toolDefinition{orderStatusTool()}
	callsMade := 0
	for round := 0; round < maxRounds; round++ {
		resp, err := askModel(ctx, client, endpoint, apiKey, model, messages, tools)
		if err != nil {
			return "", err
		}
		if len(resp.Choices) == 0 {
			return "", errors.New("model returned no choices")
		}
		assistant := resp.Choices[0].Message
		messages = append(messages, assistant)
		if len(assistant.ToolCalls) == 0 {
			if answer := strings.TrimSpace(assistant.Content); answer != "" {
				return answer, nil
			}
			return "", errors.New("model returned neither text nor tool calls")
		}
		if callsMade+len(assistant.ToolCalls) > maxToolCalls {
			return "", errors.New("tool call limit reached")
		}
		for _, call := range assistant.ToolCalls {
			if call.ID == "" {
				return "", errors.New("tool call has no id")
			}
			result := executeTool(ctx, client, call, orderEndpoint, orderToken, callerID)
			encoded, err := json.Marshal(result)
			if err != nil {
				return "", fmt.Errorf("encode tool result: %w", err)
			}
			messages = append(messages, message{Role: "tool", ToolCallID: call.ID, Content: string(encoded)})
			callsMade++
		}
	}
	return "", errors.New("model did not produce a final answer within the round limit")
}

func askModel(ctx context.Context, client *http.Client, endpoint, apiKey, model string, messages []message, tools []toolDefinition) (chatResponse, error) {
	payload := chatRequest{
		Model: model, Messages: messages, Tools: tools, ToolChoice: "auto",
		Temperature: 0, MaxTokens: 500, Stream: false,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return chatResponse{}, fmt.Errorf("encode model request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return chatResponse{}, fmt.Errorf("create model request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return chatResponse{}, fmt.Errorf("call model endpoint: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return chatResponse{}, fmt.Errorf("model endpoint returned %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxModelResponseBytes+1))
	if err != nil {
		return chatResponse{}, fmt.Errorf("read model response: %w", err)
	}
	if len(data) > maxModelResponseBytes {
		return chatResponse{}, errors.New("model response exceeds 1 MiB")
	}
	var result chatResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return chatResponse{}, fmt.Errorf("decode model response: %w", err)
	}
	return result, nil
}

func orderStatusTool() toolDefinition {
	tool := toolDefinition{Type: "function"}
	tool.Function.Name = "get_order_status"
	tool.Function.Description = "读取当前已认证客服有权访问的单个订单状态。只读，不执行退款或取消。"
	tool.Function.Parameters = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"order_id": map[string]any{"type": "string", "description": "订单编号，例如 ORD-104288"},
		},
		"required":             []string{"order_id"},
		"additionalProperties": false,
	}
	return tool
}

func executeTool(ctx context.Context, client *http.Client, call toolCall, endpoint, serviceToken, callerID string) map[string]string {
	if call.Type != "function" || call.Function.Name != "get_order_status" {
		return map[string]string{"error": "tool is not allowed"}
	}
	var args struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil || !orderIDPattern.MatchString(args.OrderID) {
		return map[string]string{"error": "invalid order_id"}
	}
	if callerID == "" {
		return map[string]string{"error": "authenticated caller is required"}
	}
	parsed, err := url.Parse(strings.TrimRight(endpoint, "/") + "/" + url.PathEscape(args.OrderID))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return map[string]string{"error": "order service is unavailable"}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return map[string]string{"error": "order service is unavailable"}
	}
	req.Header.Set("X-Caller-User", callerID)
	if serviceToken != "" {
		req.Header.Set("Authorization", "Bearer "+serviceToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return map[string]string{"error": "order service is unavailable"}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return map[string]string{"error": "order status lookup failed"}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxOrderResponseBytes+1))
	if err != nil || len(data) > maxOrderResponseBytes {
		return map[string]string{"error": "order service returned an invalid response"}
	}
	var status orderStatus
	if err := json.Unmarshal(data, &status); err != nil || status.OrderID != args.OrderID || status.Status == "" {
		return map[string]string{"error": "order service returned an invalid response"}
	}
	return map[string]string{"order_id": status.OrderID, "status": status.Status, "updated_at": status.UpdatedAt}
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
