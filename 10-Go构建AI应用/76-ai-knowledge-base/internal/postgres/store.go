package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"example.com/go-book/76-ai-knowledge-base/internal/knowledge"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	poolConfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database URL: %w", err)
	}
	poolConfig.MaxConns = 20
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create database pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }

func (s *Store) ReplaceDocument(ctx context.Context, tenantID, collectionID string, chunks []knowledge.Chunk) error {
	if len(chunks) == 0 {
		return errors.New("cannot activate a document without chunks")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := setTenant(ctx, tx, tenantID); err != nil {
		return err
	}
	lockKey := strings.Join([]string{tenantID, collectionID, chunks[0].DocumentKey}, "\x1f")
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, lockKey); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE knowledge_chunks
		SET active = FALSE
		WHERE tenant_id = $1 AND collection_id = $2 AND document_key = $3`,
		tenantID, collectionID, chunks[0].DocumentKey); err != nil {
		return err
	}
	for index, chunk := range chunks {
		if chunk.TenantID != tenantID || chunk.CollectionID != collectionID || len(chunk.Embedding) != 768 {
			return fmt.Errorf("chunk %d has inconsistent scope or embedding dimensions", index)
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO knowledge_chunks
			  (tenant_id, collection_id, chunk_id, document_key, document_version,
			   title, source_uri, content, embedding, active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::vector, TRUE)`,
			chunk.TenantID, chunk.CollectionID, chunk.ID, chunk.DocumentKey, chunk.Version,
			chunk.Title, chunk.SourceURI, chunk.Content, vectorLiteral(chunk.Embedding))
		if err != nil {
			return fmt.Errorf("insert chunk %d: %w", index+1, err)
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) Search(ctx context.Context, tenantID, collectionID string, vector []float32, limit int) ([]knowledge.Chunk, error) {
	if len(vector) != 768 || limit < 1 || limit > 20 {
		return nil, errors.New("invalid vector search dimensions or limit")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if err := setTenant(ctx, tx, tenantID); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `
		SELECT chunk_id, tenant_id, collection_id, document_key, document_version,
		       title, source_uri, content, embedding <=> $3::vector AS distance, created_at
		FROM knowledge_chunks
		WHERE tenant_id = $1 AND collection_id = $2 AND active = TRUE
		ORDER BY embedding <=> $3::vector
		LIMIT $4`, tenantID, collectionID, vectorLiteral(vector), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	chunks := make([]knowledge.Chunk, 0, limit)
	for rows.Next() {
		var chunk knowledge.Chunk
		if err := rows.Scan(&chunk.ID, &chunk.TenantID, &chunk.CollectionID, &chunk.DocumentKey,
			&chunk.Version, &chunk.Title, &chunk.SourceURI, &chunk.Content, &chunk.Distance, &chunk.CreatedAt); err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return chunks, nil
}

func setTenant(ctx context.Context, tx pgx.Tx, tenantID string) error {
	_, err := tx.Exec(ctx, `SELECT set_config('app.tenant_id', $1, TRUE)`, tenantID)
	return err
}

func vectorLiteral(vector []float32) string {
	parts := make([]string, len(vector))
	for i, value := range vector {
		parts[i] = strconv.FormatFloat(float64(value), 'f', -1, 32)
	}
	return "[" + strings.Join(parts, ",") + "]"
}
