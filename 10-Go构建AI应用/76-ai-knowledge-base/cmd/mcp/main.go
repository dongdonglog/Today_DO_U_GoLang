package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"example.com/go-book/76-ai-knowledge-base/internal/config"
	"example.com/go-book/76-ai-knowledge-base/internal/embedding"
	"example.com/go-book/76-ai-knowledge-base/internal/knowledge"
	"example.com/go-book/76-ai-knowledge-base/internal/postgres"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const maxQueryBytes = 2000

type SearchInput struct {
	Query string `json:"query" jsonschema:"Question about an approved support policy"`
}

type SearchOutput struct {
	Matches []PolicyMatch `json:"matches" jsonschema:"Matching approved policy passages"`
}

type PolicyMatch struct {
	ChunkID  string  `json:"chunk_id" jsonschema:"Stable identifier for this passage"`
	Title    string  `json:"title" jsonschema:"Policy document title"`
	Source   string  `json:"source_uri" jsonschema:"Source URI for verification"`
	Content  string  `json:"content" jsonschema:"Matched passage text"`
	Distance float64 `json:"distance" jsonschema:"Vector distance; smaller is closer"`
}

type policySearch struct {
	service      *knowledge.Service
	tenantID     string
	collectionID string
}

func main() {
	log.SetOutput(os.Stderr)
	if err := run(); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
}

func run() error {
	if err := config.LoadDotEnv(".env"); err != nil {
		return fmt.Errorf("load .env: %w", err)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := cfg.ValidateMCP(); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startup, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	store, err := postgres.Open(startup, cfg.MCPDatabaseURL)
	if err != nil {
		return err
	}
	defer store.Close()
	embedder, err := embedding.NewClient(cfg.OllamaBaseURL, cfg.EmbeddingModel)
	if err != nil {
		return err
	}
	service, err := knowledge.NewService(store, embedder, nil, cfg.MaxDistance)
	if err != nil {
		return err
	}
	search := &policySearch{service: service, tenantID: cfg.MCPTenantID, collectionID: cfg.MCPCollection}
	server := mcp.NewServer(&mcp.Implementation{Name: "support-knowledge", Version: "1.0.0"}, nil)
	openWorld := false
	mcp.AddTool(server, &mcp.Tool{
		Name:        "search_support_knowledge",
		Title:       "Search approved support knowledge",
		Description: "Read approved passages from the configured tenant and collection. Does not change orders or issue refunds.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: &openWorld},
	}, search.handle)
	return server.Run(ctx, &mcp.StdioTransport{})
}

func (s *policySearch) handle(ctx context.Context, _ *mcp.CallToolRequest, input SearchInput) (*mcp.CallToolResult, SearchOutput, error) {
	query := strings.TrimSpace(input.Query)
	if query == "" || len(query) > maxQueryBytes || !utf8.ValidString(query) {
		return nil, SearchOutput{}, errors.New("query must be valid UTF-8 and between 1 and 2000 bytes")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	matches, err := s.service.Search(ctx, s.tenantID, s.collectionID, query)
	if err != nil {
		log.Printf("knowledge search failed: %v", err)
		return nil, SearchOutput{}, errors.New("knowledge search is temporarily unavailable")
	}
	result := SearchOutput{Matches: make([]PolicyMatch, 0, len(matches))}
	for _, match := range matches {
		content := match.Content
		if runes := []rune(content); len(runes) > 1600 {
			content = string(runes[:1600])
		}
		result.Matches = append(result.Matches, PolicyMatch{
			ChunkID: match.ID, Title: match.Title, Source: match.SourceURI,
			Content: content, Distance: match.Distance,
		})
	}
	return nil, result, nil
}
