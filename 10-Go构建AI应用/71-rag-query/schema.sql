CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS knowledge_chunks (
    tenant_id TEXT NOT NULL,
    collection_id TEXT NOT NULL,
    chunk_id TEXT PRIMARY KEY,
    document_key TEXT NOT NULL,
    document_version TEXT NOT NULL,
    title TEXT NOT NULL,
    source_uri TEXT NOT NULL,
    content TEXT NOT NULL,
    embedding VECTOR(768) NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS knowledge_chunks_scope_idx
    ON knowledge_chunks (tenant_id, collection_id, active);

CREATE INDEX IF NOT EXISTS knowledge_chunks_embedding_idx
    ON knowledge_chunks USING hnsw (embedding vector_cosine_ops)
    WHERE active = TRUE;
