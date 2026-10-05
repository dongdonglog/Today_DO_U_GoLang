CREATE EXTENSION IF NOT EXISTS vector;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kb_app') THEN
        CREATE ROLE kb_app LOGIN PASSWORD 'kb-local-only' NOSUPERUSER NOBYPASSRLS;
    END IF;
END
$$;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'kb_reader') THEN
        CREATE ROLE kb_reader LOGIN PASSWORD 'kb-reader-local-only' NOSUPERUSER NOBYPASSRLS;
    END IF;
END
$$;

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

ALTER TABLE knowledge_chunks ENABLE ROW LEVEL SECURITY;
ALTER TABLE knowledge_chunks FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation ON knowledge_chunks;
CREATE POLICY tenant_isolation ON knowledge_chunks
    USING (tenant_id = current_setting('app.tenant_id', TRUE))
    WITH CHECK (tenant_id = current_setting('app.tenant_id', TRUE));

CREATE INDEX IF NOT EXISTS knowledge_chunks_scope_idx
    ON knowledge_chunks (tenant_id, collection_id, active);

CREATE INDEX IF NOT EXISTS knowledge_chunks_embedding_idx
    ON knowledge_chunks USING hnsw (embedding vector_cosine_ops)
    WHERE active = TRUE;

GRANT SELECT, INSERT, UPDATE ON knowledge_chunks TO kb_app;
GRANT USAGE ON SCHEMA public TO kb_app, kb_reader;
GRANT SELECT ON knowledge_chunks TO kb_reader;
