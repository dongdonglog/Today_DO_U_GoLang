package main

import (
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
	maxSteps     = 4
	maxToolCalls = 3
	maxToolBytes = 128 << 10
)

var orderIDPattern = regexp.MustCompile(`^ORD-[0-9]{6,16}$`)

type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

type Decision struct {
	Text string
	Call *ToolCall
}

type Turn struct {
	Question   string
	ToolName   string
	ToolInput  string
	ToolOutput string
}

type Model interface {
	Next(context.Context, string, []Turn) (Decision, error)
}

type Tool interface {
	Name() string
	Run(context.Context, string) (string, error)
}

type Agent struct {
	Model       Model
	Tools       map[string]Tool
	MaxSteps    int
	MaxCalls    int
	StepTimeout time.Duration
}

func (a Agent) Run(ctx context.Context, question string) (string, error) {
	if a.Model == nil || len(a.Tools) == 0 {
		return "", errors.New("agent requires a model and at least one tool")
	}
	if a.MaxSteps < 1 || a.MaxCalls < 1 || a.StepTimeout <= 0 {
		return "", errors.New("agent budgets must be positive")
	}
	if strings.TrimSpace(question) == "" || len(question) > 2000 {
		return "", errors.New("question is required and must be at most 2000 bytes")
	}

	var history []Turn
	seen := make(map[string]bool)
	toolCalls := 0
	for step := 0; step < a.MaxSteps; step++ {
		stepCtx, cancel := context.WithTimeout(ctx, a.StepTimeout)
		decision, err := a.Model.Next(stepCtx, question, history)
		cancel()
		if err != nil {
			return "", fmt.Errorf("model step %d: %w", step+1, err)
		}
		if decision.Call == nil {
			if answer := strings.TrimSpace(decision.Text); answer != "" {
				return answer, nil
			}
			return "", errors.New("model returned neither a final answer nor a tool call")
		}
		call := *decision.Call
		tool, ok := a.Tools[call.Name]
		if !ok || tool.Name() != call.Name {
			return "", fmt.Errorf("tool %q is not allowed", call.Name)
		}
		key := call.Name + "\x00" + call.Arguments
		if seen[key] {
			return "", fmt.Errorf("repeated tool call %q", call.Name)
		}
		seen[key] = true
		toolCalls++
		if toolCalls > a.MaxCalls {
			return "", errors.New("tool call budget reached")
		}
		toolCtx, cancel := context.WithTimeout(ctx, a.StepTimeout)
		output, err := tool.Run(toolCtx, call.Arguments)
		cancel()
		if err != nil {
			output = `{"error":"tool failed; ask an operator to verify"}`
		}
		if len(output) > maxToolBytes {
			return "", fmt.Errorf("tool %q returned too much data", call.Name)
		}
		history = append(history, Turn{
			Question: question, ToolName: call.Name,
			ToolInput: call.Arguments, ToolOutput: output,
		})
	}
	return "", errors.New("agent step budget reached without a final answer")
}

type ReplayModel struct {
	step int
}

func (m *ReplayModel) Next(_ context.Context, question string, history []Turn) (Decision, error) {
	m.step++
	switch m.step {
	case 1:
		orderID := orderIDPattern.FindString(question)
		if orderID == "" {
			return Decision{Text: "请提供订单编号，客服再为你查询。"}, nil
		}
		args, _ := json.Marshal(map[string]string{"order_id": orderID})
		return Decision{Call: &ToolCall{ID: "replay-1", Name: "get_order_status", Arguments: string(args)}}, nil
	case 2:
		if len(history) != 1 || history[0].ToolName != "get_order_status" {
			return Decision{}, errors.New("replay expected one order lookup")
		}
		var result struct {
			OrderID string `json:"order_id"`
			Status  string `json:"status"`
		}
		if err := json.Unmarshal([]byte(history[0].ToolOutput), &result); err != nil || result.Status == "" {
			return Decision{Text: "订单状态暂时无法确认，请转人工核实。"}, nil
		}
		return Decision{Text: fmt.Sprintf("订单 %s 当前状态为：%s。", result.OrderID, result.Status)}, nil
	default:
		return Decision{}, errors.New("replay has no further decisions")
	}
}

type OrderStatusTool struct {
	Client  *http.Client
	BaseURL string
	Token   string
	Caller  string
}

func (t OrderStatusTool) Name() string { return "get_order_status" }

func (t OrderStatusTool) Run(ctx context.Context, raw string) (string, error) {
	var args struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal([]byte(raw), &args); err != nil || !orderIDPattern.MatchString(args.OrderID) {
		return "", errors.New("invalid order_id")
	}
	if t.Caller == "" {
		return "", errors.New("authenticated caller is required")
	}
	u, err := url.Parse(strings.TrimRight(t.BaseURL, "/") + "/" + url.PathEscape(args.OrderID))
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", errors.New("invalid order service URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Caller-User", t.Caller)
	if t.Token != "" {
		req.Header.Set("Authorization", "Bearer "+t.Token)
	}
	resp, err := t.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("order service returned %s", resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxToolBytes+1))
	if err != nil || len(body) > maxToolBytes {
		return "", errors.New("invalid order service response")
	}
	var result struct {
		OrderID string `json:"order_id"`
		Status  string `json:"status"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.OrderID != args.OrderID || result.Status == "" {
		return "", errors.New("invalid order status")
	}
	minimal, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return string(minimal), nil
}

func main() {
	question := flag.String("question", "", "customer question")
	flag.Parse()
	client := &http.Client{Timeout: 5 * time.Second}
	tool := OrderStatusTool{
		Client:  client,
		BaseURL: env("ORDER_SERVICE_URL", "http://localhost:8081/internal/orders"),
		Token:   os.Getenv("ORDER_SERVICE_TOKEN"),
		Caller:  strings.TrimSpace(os.Getenv("CALLER_USER_ID")),
	}
	agent := Agent{
		Model:    &ReplayModel{},
		Tools:    map[string]Tool{tool.Name(): tool},
		MaxSteps: maxSteps, MaxCalls: maxToolCalls, StepTimeout: 5 * time.Second,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	answer, err := agent.Run(ctx, *question)
	if err != nil {
		fmt.Fprintln(os.Stderr, "agent failed:", err)
		os.Exit(1)
	}
	fmt.Println(answer)
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
