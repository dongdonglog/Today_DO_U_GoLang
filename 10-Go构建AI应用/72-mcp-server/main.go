package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	embeddingDimensions = 768
	maxResponseBytes    = 2 << 20
	maxQueryBytes       = 2000
	maxMatches          = 5
)

type embeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

type SearchInput struct {
	Query string `json:"query" jsonschema:"Question about an approved support policy"`
}

type SearchOutput struct {
	Matches []PolicyMatch `json:"matches" jsonschema:"Matching approved policy passages"`
}

type PolicyMatch struct {
	ChunkID  string  `json:"chunk_id" jsonschema:"Stable identifier for this passage"`
	Title    string  `json:"title" jsonschema:"Policy document title"`
	Source   string  `json:"source" jsonschema:"Source URI for verification"`
	Content  string  `json:"content" jsonschema:"Matched passage text"`
	Distance float64 `json:"distance" jsonschema:"Vector distance; smaller is closer"`
}

type serviceConfig struct {
	tenantID          string
	collectionID      string
	embeddingEndpoint string
	embeddingModel    string
	apiKey            string
	maxDistance       float64
}

type policySearch struct {
	pool   *pgxpool.Pool
	client *http.Client
	cfg    serviceConfig
}

func main() {
	log.SetOutput(os.Stderr)
	cfg := serviceConfig{
		tenantID:          requiredEnv("TENANT_ID"),
		collectionID:      requiredEnv("COLLECTION_ID"),
		embeddingEndpoint: env("EMBEDDING_ENDPOINT", "http://localhost:11434/v1/embeddings"),
		embeddingModel:    env("EMBEDDING_MODEL", "nomic-embed-text"),
		apiKey:            os.Getenv("LLM_API_KEY"),
		maxDistance:       envFloat("RAG_MAX_DISTANCE", 0.55),
	}
	databaseURL := requiredEnv("DATABASE_URL")
	if cfg.tenantID == "" || cfg.collectionID == "" || databaseURL == "" {
		log.Fatal("DATABASE_URL, TENANT_ID and COLLECTION_ID are required")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatal("connect database: ", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatal("ping database: ", err)
	}

	search := &policySearch{
		pool: pool, client: &http.Client{Timeout: 10 * time.Second}, cfg: cfg,
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "support-policy", Version: "1.0.0"}, nil)
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_support_policy",
		Title:       "Search approved support policies",
		Description: "Read approved policy passages from the configured tenant and collection. Does not change orders or issue refunds.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld},
	}, search.handle)

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal("MCP server stopped: ", err)
	}
}

func (s *policySearch) handle(ctx context.Context, _ *mcp.CallToolRequest, input SearchInput) (*mcp.CallToolResult, SearchOutput, error) {
	query := strings.TrimSpace(input.Query)
	if query == "" || len(query) > maxQueryBytes {
		return nil, SearchOutput{}, errors.New("query must be between 1 and 2000 bytes")
	}
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	vector, err := s.embed(ctx, query)
	if err != nil {
		log.Printf("embedding query failed: %v", err)
		return nil, SearchOutput{}, errors.New("policy search is temporarily unavailable")
	}
	rows, err := s.pool.Query(ctx, `
		SELECT chunk_id, title, source_uri, content,
		       embedding <=> $3::vector AS distance
		FROM knowledge_chunks
		WHERE tenant_id = $1 AND collection_id = $2 AND active = TRUE
		ORDER BY embedding <=> $3::vector
		LIMIT $4`, s.cfg.tenantID, s.cfg.collectionID, vectorLiteral(vector), maxMatches)
	if err != nil {
		log.Printf("policy query failed: %v", err)
		return nil, SearchOutput{}, errors.New("policy search is temporarily unavailable")
	}
	defer rows.Close()

	result := SearchOutput{Matches: make([]PolicyMatch, 0, maxMatches)}
	for rows.Next() {
		var item PolicyMatch
		if err := rows.Scan(&item.ChunkID, &item.Title, &item.Source, &item.Content, &item.Distance); err != nil {
			log.Printf("scan policy result failed: %v", err)
			return nil, SearchOutput{}, errors.New("policy search is temporarily unavailable")
		}
		if math.IsNaN(item.Distance) || item.Distance > s.cfg.maxDistance {
			continue
		}
		if runes := []rune(item.Content); len(runes) > 1600 {
			item.Content = string(runes[:1600])
		}
		result.Matches = append(result.Matches, item)
	}
	if err := rows.Err(); err != nil {
		log.Printf("iterate policy results failed: %v", err)
		return nil, SearchOutput{}, errors.New("policy search is temporarily unavailable")
	}
	return nil, result, nil
}

func (s *policySearch) embed(ctx context.Context, query string) ([]float32, error) {
	body, err := json.Marshal(embeddingRequest{Model: s.cfg.embeddingModel, Input: query})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.embeddingEndpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.cfg.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.cfg.apiKey)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embedding endpoint returned %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxResponseBytes {
		return nil, errors.New("embedding response exceeds size limit")
	}
	var result embeddingResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	if len(result.Data) == 0 || len(result.Data[0].Embedding) != embeddingDimensions {
		return nil, fmt.Errorf("embedding model must return %d dimensions", embeddingDimensions)
	}
	for _, value := range result.Data[0].Embedding {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return nil, errors.New("embedding contains a non-finite value")
		}
	}
	return result.Data[0].Embedding, nil
}

func vectorLiteral(vector []float32) string {
	parts := make([]string, len(vector))
	for i, value := range vector {
		parts[i] = strconv.FormatFloat(float64(value), 'f', -1, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func requiredEnv(key string) string { return strings.TrimSpace(os.Getenv(key)) }

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envFloat(key string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed < 0 || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
		log.Fatalf("%s must be a non-negative number", key)
	}
	return parsed
}
