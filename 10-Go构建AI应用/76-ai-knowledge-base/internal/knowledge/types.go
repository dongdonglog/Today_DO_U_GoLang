package knowledge

import "time"

type DocumentInput struct {
	CollectionID string
	DocumentKey  string
	Version      string
	Title        string
	SourceURI    string
	Text         string
}

type Chunk struct {
	ID           string
	TenantID     string
	CollectionID string
	DocumentKey  string
	Version      string
	Title        string
	SourceURI    string
	Content      string
	Embedding    []float32
	Distance     float64
	CreatedAt    time.Time
}

type Citation struct {
	ChunkID         string  `json:"chunk_id"`
	DocumentKey     string  `json:"document_key"`
	DocumentVersion string  `json:"document_version"`
	Title           string  `json:"title"`
	SourceURI       string  `json:"source_uri"`
	Distance        float64 `json:"distance"`
}

type QueryResult struct {
	Answer    string     `json:"answer"`
	Citations []Citation `json:"citations"`
}
