package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

const dimensions = 768

type Client struct {
	endpoint string
	model    string
	client   *http.Client
}

type request struct {
	Model    string   `json:"model"`
	Input    []string `json:"input"`
	Truncate bool     `json:"truncate"`
}

type response struct {
	Embeddings [][]float32 `json:"embeddings"`
}

func NewClient(baseURL, model string) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || strings.TrimSpace(model) == "" {
		return nil, errors.New("embedding URL and model are required")
	}
	return &Client{
		endpoint: baseURL + "/api/embed",
		model:    model,
		client:   &http.Client{Timeout: 45 * time.Second},
	}, nil
}

func (c *Client) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 || len(texts) > 100 {
		return nil, errors.New("embedding batch must contain between 1 and 100 texts")
	}
	body, err := json.Marshal(request{Model: c.model, Input: texts, Truncate: false})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("Ollama embedding returned %s: %s", resp.Status, strings.TrimSpace(string(message)))
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 2<<20 {
		return nil, errors.New("embedding response exceeds size limit")
	}
	var result response
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	if len(result.Embeddings) != len(texts) {
		return nil, fmt.Errorf("Ollama returned %d embeddings for %d inputs", len(result.Embeddings), len(texts))
	}
	for i, vector := range result.Embeddings {
		if len(vector) != dimensions {
			return nil, fmt.Errorf("embedding %d has %d dimensions; expected %d", i, len(vector), dimensions)
		}
		for _, value := range vector {
			if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
				return nil, fmt.Errorf("embedding %d contains a non-finite value", i)
			}
		}
	}
	return result.Embeddings, nil
}
