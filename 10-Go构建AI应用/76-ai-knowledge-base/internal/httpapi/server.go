package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"example.com/go-book/76-ai-knowledge-base/internal/auth"
	"example.com/go-book/76-ai-knowledge-base/internal/config"
	"example.com/go-book/76-ai-knowledge-base/internal/knowledge"
	"github.com/gin-gonic/gin"
)

const maxJSONBytes = 300 << 10

var collectionPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,63}$`)

type API struct {
	service *knowledge.Service
	limit   chan struct{}
}

type importRequest struct {
	CollectionID string `json:"collection_id"`
	DocumentKey  string `json:"document_key"`
	Version      string `json:"version"`
	Title        string `json:"title"`
	SourceURI    string `json:"source_uri"`
	Text         string `json:"text"`
}

type queryRequest struct {
	CollectionID string `json:"collection_id"`
	Question     string `json:"question"`
}

func New(service *knowledge.Service, cfg config.Config) *gin.Engine {
	api := &API{service: service, limit: make(chan struct{}, cfg.MaxInFlight)}
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	router.GET("/readyz", api.ready)

	knowledgeRoutes := router.Group("/v1/knowledge")
	knowledgeRoutes.Use(auth.Middleware([]byte(cfg.JWTSecret), cfg.JWTIssuer))
	knowledgeRoutes.POST("/documents", auth.RequireRole("knowledge_admin"), api.withLimit, api.importDocument)
	knowledgeRoutes.POST("/query", auth.RequireAnyRole("knowledge_reader", "knowledge_admin"), api.withLimit, api.query)
	return router
}

func (a *API) ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()
	if err := a.service.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}

func (a *API) withLimit(c *gin.Context) {
	select {
	case a.limit <- struct{}{}:
		defer func() { <-a.limit }()
		c.Next()
	case <-c.Request.Context().Done():
		c.Abort()
	default:
		c.Header("Retry-After", "1")
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "server is busy"})
	}
}

func (a *API) importDocument(c *gin.Context) {
	var request importRequest
	if !decodeJSON(c, &request) {
		return
	}
	request.CollectionID = strings.TrimSpace(request.CollectionID)
	request.DocumentKey = strings.TrimSpace(request.DocumentKey)
	request.Version = strings.TrimSpace(request.Version)
	request.Title = strings.TrimSpace(request.Title)
	request.SourceURI = strings.TrimSpace(request.SourceURI)
	if !collectionPattern.MatchString(request.CollectionID) || !collectionPattern.MatchString(request.DocumentKey) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "collection_id and document_key must use letters, digits, dot, underscore or hyphen"})
		return
	}
	if request.Version == "" || runeCount(request.Version) > 128 || request.Title == "" || runeCount(request.Title) > 200 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "version and title are required and must be within their length limits"})
		return
	}
	if len(request.SourceURI) > 2048 || !validSourceURI(request.SourceURI) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source_uri must be an absolute https, s3, internal or support URI"})
		return
	}
	if len(request.Text) == 0 || len(request.Text) > 256<<10 || !utf8.ValidString(request.Text) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "text must be valid UTF-8 and at most 256 KiB"})
		return
	}
	principal, ok := auth.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Minute)
	defer cancel()
	count, err := a.service.Ingest(ctx, principal.TenantID, knowledge.DocumentInput{
		CollectionID: request.CollectionID,
		DocumentKey:  request.DocumentKey,
		Version:      request.Version,
		Title:        request.Title,
		SourceURI:    request.SourceURI,
		Text:         request.Text,
	})
	if err != nil {
		slog.Error("knowledge document import failed", "tenant_id", principal.TenantID, "document_key", request.DocumentKey, "error", err)
		writeServiceError(c, ctx)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"document_key": request.DocumentKey, "version": request.Version, "chunks": count})
}

func (a *API) query(c *gin.Context) {
	var request queryRequest
	if !decodeJSON(c, &request) {
		return
	}
	request.CollectionID = strings.TrimSpace(request.CollectionID)
	request.Question = strings.TrimSpace(request.Question)
	if !collectionPattern.MatchString(request.CollectionID) || request.Question == "" || len(request.Question) > 2000 || !utf8.ValidString(request.Question) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a valid collection_id and a question of at most 2000 bytes are required"})
		return
	}
	principal, ok := auth.GetPrincipal(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 90*time.Second)
	defer cancel()
	result, err := a.service.Ask(ctx, principal.TenantID, request.CollectionID, request.Question)
	if err != nil {
		slog.Error("knowledge query failed", "tenant_id", principal.TenantID, "error", err)
		writeServiceError(c, ctx)
		return
	}
	c.JSON(http.StatusOK, result)
}

func decodeJSON(c *gin.Context, target any) bool {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxJSONBytes)
	if err := c.ShouldBindJSON(target); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON request"})
		return false
	}
	return true
}

func validSourceURI(value string) bool {
	parsed, err := url.Parse(value)
	if err != nil || !parsed.IsAbs() || parsed.User != nil {
		return false
	}
	switch parsed.Scheme {
	case "https":
		return parsed.Host != ""
	case "s3", "internal", "support":
		return parsed.Host != "" || parsed.Opaque != ""
	default:
		return false
	}
}

func runeCount(value string) int { return utf8.RuneCountInString(value) }

func writeServiceError(c *gin.Context, requestContext context.Context) {
	if errors.Is(requestContext.Err(), context.Canceled) {
		c.Abort()
		return
	}
	if errors.Is(requestContext.Err(), context.DeadlineExceeded) {
		c.JSON(http.StatusGatewayTimeout, gin.H{"error": "knowledge request timed out"})
		return
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "knowledge service is temporarily unavailable"})
}
