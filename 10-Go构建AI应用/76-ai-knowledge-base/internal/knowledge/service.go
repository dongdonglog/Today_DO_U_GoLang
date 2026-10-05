package knowledge

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	embeddingDimensions = 768
	maxDocumentBytes    = 256 << 10
	maxDocumentChunks   = 100
	chunkRunes          = 800
	chunkOverlap        = 100
	maxMatches          = 5
	maxEvidenceRunes    = 10000
)

type Store interface {
	Ping(context.Context) error
	ReplaceDocument(context.Context, string, string, []Chunk) error
	Search(context.Context, string, string, []float32, int) ([]Chunk, error)
}

type Embedder interface {
	Embed(context.Context, []string) ([][]float32, error)
}

type Answerer interface {
	Generate(context.Context, string, []Chunk) (string, error)
}

type Service struct {
	store       Store
	embedder    Embedder
	answerer    Answerer
	maxDistance float64
}

func NewService(store Store, embedder Embedder, answerer Answerer, maxDistance float64) (*Service, error) {
	if store == nil || embedder == nil {
		return nil, errors.New("store and embedder are required")
	}
	if math.IsNaN(maxDistance) || math.IsInf(maxDistance, 0) || maxDistance < 0 || maxDistance > 2 {
		return nil, errors.New("maximum vector distance must be between 0 and 2")
	}
	return &Service{store: store, embedder: embedder, answerer: answerer, maxDistance: maxDistance}, nil
}

func (s *Service) Ping(ctx context.Context) error { return s.store.Ping(ctx) }

func (s *Service) Ingest(ctx context.Context, tenantID string, input DocumentInput) (int, error) {
	if strings.TrimSpace(tenantID) == "" {
		return 0, errors.New("tenant is required")
	}
	if len(input.Text) == 0 || len(input.Text) > maxDocumentBytes || !utf8.ValidString(input.Text) {
		return 0, fmt.Errorf("document must be valid UTF-8 and between 1 byte and %d bytes", maxDocumentBytes)
	}
	parts := splitText(input.Text, chunkRunes, chunkOverlap)
	if len(parts) == 0 || len(parts) > maxDocumentChunks {
		return 0, fmt.Errorf("document produced %d chunks; limit is %d", len(parts), maxDocumentChunks)
	}
	vectors, err := s.embedder.Embed(ctx, parts)
	if err != nil {
		return 0, fmt.Errorf("embed document: %w", err)
	}
	if len(vectors) != len(parts) {
		return 0, errors.New("embedding service returned a mismatched vector count")
	}

	chunks := make([]Chunk, 0, len(parts))
	for i, part := range parts {
		if len(vectors[i]) != embeddingDimensions {
			return 0, fmt.Errorf("embedding %d has %d dimensions; expected %d", i, len(vectors[i]), embeddingDimensions)
		}
		chunks = append(chunks, Chunk{
			ID:           uuid.NewString(),
			TenantID:     tenantID,
			CollectionID: input.CollectionID,
			DocumentKey:  input.DocumentKey,
			Version:      input.Version,
			Title:        input.Title,
			SourceURI:    input.SourceURI,
			Content:      part,
			Embedding:    vectors[i],
		})
	}
	if err := s.store.ReplaceDocument(ctx, tenantID, input.CollectionID, chunks); err != nil {
		return 0, fmt.Errorf("replace document version: %w", err)
	}
	return len(chunks), nil
}

func (s *Service) Search(ctx context.Context, tenantID, collectionID, query string) ([]Chunk, error) {
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(collectionID) == "" {
		return nil, errors.New("tenant and collection are required")
	}
	if strings.TrimSpace(query) == "" || len(query) > 2000 || !utf8.ValidString(query) {
		return nil, errors.New("query must be valid UTF-8 and between 1 and 2000 bytes")
	}
	vectors, err := s.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(vectors) != 1 || len(vectors[0]) != embeddingDimensions {
		return nil, errors.New("embedding service returned an invalid query vector")
	}
	chunks, err := s.store.Search(ctx, tenantID, collectionID, vectors[0], maxMatches)
	if err != nil {
		return nil, fmt.Errorf("search knowledge: %w", err)
	}
	filtered := chunks[:0]
	for _, chunk := range chunks {
		if math.IsNaN(chunk.Distance) || math.IsInf(chunk.Distance, 0) || chunk.Distance > s.maxDistance {
			continue
		}
		filtered = append(filtered, chunk)
	}
	return filtered, nil
}

func (s *Service) Ask(ctx context.Context, tenantID, collectionID, question string) (QueryResult, error) {
	chunks, err := s.Search(ctx, tenantID, collectionID, question)
	if err != nil {
		return QueryResult{}, err
	}
	if len(chunks) == 0 {
		return QueryResult{
			Answer:    "没有找到足够相关的已审核资料，请转人工核实。",
			Citations: []Citation{},
		}, nil
	}
	if s.answerer == nil {
		return QueryResult{}, errors.New("answer generator is unavailable")
	}
	if evidenceRunes(chunks) > maxEvidenceRunes {
		chunks = trimEvidence(chunks, maxEvidenceRunes)
	}
	answer, err := s.answerer.Generate(ctx, question, chunks)
	if err != nil {
		return QueryResult{}, fmt.Errorf("generate answer: %w", err)
	}
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return QueryResult{}, errors.New("answer model returned empty text")
	}
	citations := make([]Citation, 0, len(chunks))
	for _, chunk := range chunks {
		citations = append(citations, Citation{
			ChunkID:         chunk.ID,
			DocumentKey:     chunk.DocumentKey,
			DocumentVersion: chunk.Version,
			Title:           chunk.Title,
			SourceURI:       chunk.SourceURI,
			Distance:        chunk.Distance,
		})
	}
	return QueryResult{Answer: answer, Citations: citations}, nil
}

func splitText(input string, maxRunes, overlap int) []string {
	input = strings.TrimSpace(strings.ReplaceAll(input, "\r\n", "\n"))
	if input == "" || maxRunes < 1 || overlap < 0 || overlap >= maxRunes {
		return nil
	}
	paragraphs := strings.Split(input, "\n\n")
	var result []string
	var current strings.Builder
	currentRunes := 0
	flush := func() {
		if current.Len() > 0 {
			result = append(result, strings.TrimSpace(current.String()))
			current.Reset()
			currentRunes = 0
		}
	}
	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}
		runes := []rune(paragraph)
		if len(runes) > maxRunes {
			flush()
			for start := 0; start < len(runes); {
				end := start + maxRunes
				if end > len(runes) {
					end = len(runes)
				}
				result = append(result, strings.TrimSpace(string(runes[start:end])))
				if end == len(runes) {
					break
				}
				start = end - overlap
			}
			continue
		}
		separator := 0
		if current.Len() > 0 {
			separator = 2
		}
		if currentRunes+separator+len(runes) > maxRunes {
			flush()
		}
		if current.Len() > 0 {
			current.WriteString("\n\n")
			currentRunes += 2
		}
		current.WriteString(paragraph)
		currentRunes += len(runes)
	}
	flush()
	return result
}

func evidenceRunes(chunks []Chunk) int {
	total := 0
	for _, chunk := range chunks {
		total += utf8.RuneCountInString(chunk.Content)
	}
	return total
}

func trimEvidence(chunks []Chunk, limit int) []Chunk {
	result := make([]Chunk, 0, len(chunks))
	used := 0
	for _, chunk := range chunks {
		remaining := limit - used
		if remaining <= 0 {
			break
		}
		runes := []rune(chunk.Content)
		if len(runes) > remaining {
			chunk.Content = string(runes[:remaining])
		}
		used += utf8.RuneCountInString(chunk.Content)
		result = append(result, chunk)
	}
	return result
}
