CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE invoice_embeddings (
    invoice_id UUID PRIMARY KEY REFERENCES invoices(id) ON DELETE CASCADE,
    embedding  vector(1536) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX ON invoice_embeddings
    USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);
