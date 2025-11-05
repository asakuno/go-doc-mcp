-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Create documents table
CREATE TABLE IF NOT EXISTS documents (
    id SERIAL PRIMARY KEY,
    content TEXT NOT NULL,
    metadata JSONB,
    file_path TEXT UNIQUE NOT NULL,
    file_hash TEXT,
    file_size BIGINT,
    file_modified_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create embeddings table with vector support
-- Default dimension is 768 for Ollama's nomic-embed-text
-- Change to 1536 if using OpenAI's text-embedding-3-small
CREATE TABLE IF NOT EXISTS embeddings (
    id SERIAL PRIMARY KEY,
    document_id INTEGER REFERENCES documents(id) ON DELETE CASCADE,
    embedding vector(768),
    chunk_index INTEGER,
    chunk_text TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index for vector similarity search (HNSW for better performance)
CREATE INDEX IF NOT EXISTS embeddings_embedding_idx ON embeddings
USING hnsw (embedding vector_cosine_ops);

-- Create index for document lookup
CREATE INDEX IF NOT EXISTS embeddings_document_id_idx ON embeddings(document_id);

-- Create index for file_path lookup (for duplicate check)
CREATE INDEX IF NOT EXISTS documents_file_path_idx ON documents(file_path);

-- Create index for file_hash lookup (for change detection)
CREATE INDEX IF NOT EXISTS documents_file_hash_idx ON documents(file_hash);
