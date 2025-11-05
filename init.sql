-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Create documents table
CREATE TABLE IF NOT EXISTS documents (
    id SERIAL PRIMARY KEY,
    content TEXT NOT NULL,
    metadata JSONB,
    file_path TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create embeddings table with vector support
CREATE TABLE IF NOT EXISTS embeddings (
    id SERIAL PRIMARY KEY,
    document_id INTEGER REFERENCES documents(id) ON DELETE CASCADE,
    embedding vector(1536),
    chunk_index INTEGER,
    chunk_text TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index for vector similarity search (HNSW for better performance)
CREATE INDEX IF NOT EXISTS embeddings_embedding_idx ON embeddings
USING hnsw (embedding vector_cosine_ops);

-- Create index for document lookup
CREATE INDEX IF NOT EXISTS embeddings_document_id_idx ON embeddings(document_id);
