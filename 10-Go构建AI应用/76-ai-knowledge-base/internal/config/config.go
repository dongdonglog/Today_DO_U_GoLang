package config

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ListenAddr     string
	DatabaseURL    string
	MCPDatabaseURL string
	JWTSecret      string
	JWTIssuer      string
	MaxInFlight    int
	OllamaBaseURL  string
	EmbeddingModel string
	ChatModel      string
	OllamaAPIKey   string
	MaxDistance    float64
	MCPTenantID    string
	MCPCollection  string
}

func Load() (Config, error) {
	maxInFlight, err := integer("KB_MAX_IN_FLIGHT", 8)
	if err != nil || maxInFlight < 1 || maxInFlight > 256 {
		return Config{}, errors.New("KB_MAX_IN_FLIGHT must be between 1 and 256")
	}
	maxDistance, err := decimal("RAG_MAX_DISTANCE", 0.55)
	if err != nil || maxDistance < 0 || maxDistance > 2 {
		return Config{}, errors.New("RAG_MAX_DISTANCE must be between 0 and 2")
	}

	return Config{
		ListenAddr:     value("KB_LISTEN_ADDR", ":8080"),
		DatabaseURL:    firstValue("KB_DATABASE_URL", "DATABASE_URL"),
		MCPDatabaseURL: value("MCP_DATABASE_URL", "postgres://kb_reader:kb-reader-local-only@localhost:5432/knowledge?sslmode=disable"),
		JWTSecret:      value("KB_JWT_SECRET", ""),
		JWTIssuer:      value("KB_JWT_ISSUER", "go-book-knowledge-local"),
		MaxInFlight:    maxInFlight,
		OllamaBaseURL:  value("OLLAMA_BASE_URL", "http://localhost:11434"),
		EmbeddingModel: value("OLLAMA_EMBED_MODEL", "nomic-embed-text"),
		ChatModel:      value("OLLAMA_CHAT_MODEL", "qwen2.5:3b"),
		OllamaAPIKey:   value("OLLAMA_API_KEY", "ollama"),
		MaxDistance:    maxDistance,
		MCPTenantID:    firstValueWithDefault("MCP_TENANT_ID", "TENANT_ID", "demo-tenant"),
		MCPCollection:  firstValueWithDefault("MCP_COLLECTION_ID", "COLLECTION_ID", "support-policy"),
	}, nil
}

func (c Config) ValidateAPI() error {
	if err := c.ValidateKnowledge(); err != nil {
		return err
	}
	if len(c.JWTSecret) < 32 {
		return errors.New("KB_JWT_SECRET must contain at least 32 bytes")
	}
	if strings.TrimSpace(c.JWTIssuer) == "" || strings.TrimSpace(c.ChatModel) == "" {
		return errors.New("KB_JWT_ISSUER and OLLAMA_CHAT_MODEL are required")
	}
	return nil
}

func (c Config) ValidateKnowledge() error {
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return errors.New("KB_DATABASE_URL or DATABASE_URL is required")
	}
	if strings.TrimSpace(c.OllamaBaseURL) == "" || strings.TrimSpace(c.EmbeddingModel) == "" {
		return errors.New("OLLAMA_BASE_URL and OLLAMA_EMBED_MODEL are required")
	}
	return nil
}

func (c Config) ValidateMCP() error {
	if strings.TrimSpace(c.MCPDatabaseURL) == "" || strings.TrimSpace(c.OllamaBaseURL) == "" || strings.TrimSpace(c.EmbeddingModel) == "" {
		return errors.New("MCP_DATABASE_URL, OLLAMA_BASE_URL and OLLAMA_EMBED_MODEL are required")
	}
	if strings.TrimSpace(c.MCPTenantID) == "" || strings.TrimSpace(c.MCPCollection) == "" {
		return errors.New("MCP_TENANT_ID and MCP_COLLECTION_ID are required")
	}
	return nil
}

func LoadDotEnv(path string) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 64*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, raw, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); exists {
			continue
		}
		value := strings.TrimSpace(raw)
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s: %w", key, err)
		}
	}
	return scanner.Err()
}

func value(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func firstValue(keys ...string) string {
	for _, key := range keys {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return ""
}

func firstValueWithDefault(first, second, fallback string) string {
	if result := firstValue(first, second); result != "" {
		return result
	}
	return fallback
}

func integer(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	return strconv.Atoi(value)
}

func decimal(key string, fallback float64) (float64, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	return strconv.ParseFloat(value, 64)
}
