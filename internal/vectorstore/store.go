package vectorstore

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/asakuno/go-doc-mcp/internal/document"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
)

type SearchResult struct {
	ChunkText    string
	FilePath     string
	Score        float64
	ChunkIndex   int
	DocumentID   int
	FullDocument string
}

type VectorStore struct {
	pool              *pgxpool.Pool
	embeddingService  *EmbeddingService
	embeddingDim      int
}

func NewVectorStore(ctx context.Context, connString string, embeddingService *EmbeddingService, embeddingDim int) (*VectorStore, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &VectorStore{
		pool:             pool,
		embeddingService: embeddingService,
		embeddingDim:     embeddingDim,
	}, nil
}

// Close closes the database connection pool
func (v *VectorStore) Close() {
	v.pool.Close()
}

// AddDocument adds a document and its chunks to the vector store
func (v *VectorStore) AddDocument(ctx context.Context, doc *document.Document, chunks []*document.Chunk) error {
	tx, err := v.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Insert document
	metadataJSON, err := json.Marshal(doc.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var docID int
	err = tx.QueryRow(ctx,
		"INSERT INTO documents (content, metadata, file_path) VALUES ($1, $2, $3) RETURNING id",
		doc.Content, metadataJSON, doc.FilePath,
	).Scan(&docID)
	if err != nil {
		return fmt.Errorf("failed to insert document: %w", err)
	}

	// Prepare texts for embedding
	texts := make([]string, len(chunks))
	for i, chunk := range chunks {
		texts[i] = chunk.Text
	}

	// Create embeddings
	embeddings, err := v.embeddingService.CreateEmbeddings(ctx, texts)
	if err != nil {
		return fmt.Errorf("failed to create embeddings: %w", err)
	}

	// Insert chunks with embeddings
	batch := &pgx.Batch{}
	for i, chunk := range chunks {
		embedding := pgvector.NewVector(embeddings[i])
		batch.Queue(
			"INSERT INTO embeddings (document_id, embedding, chunk_index, chunk_text) VALUES ($1, $2, $3, $4)",
			docID, embedding, chunk.Index, chunk.Text,
		)
	}

	br := tx.SendBatch(ctx, batch)
	if err := br.Close(); err != nil {
		return fmt.Errorf("failed to insert embeddings: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Search performs similarity search on the vector store
func (v *VectorStore) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	// Create embedding for query
	queryEmbedding, err := v.embeddingService.CreateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to create query embedding: %w", err)
	}

	// Perform similarity search using cosine distance
	rows, err := v.pool.Query(ctx, `
		SELECT
			e.chunk_text,
			d.file_path,
			1 - (e.embedding <=> $1) as score,
			e.chunk_index,
			e.document_id,
			d.content
		FROM embeddings e
		JOIN documents d ON e.document_id = d.id
		ORDER BY e.embedding <=> $1
		LIMIT $2
	`, pgvector.NewVector(queryEmbedding), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query embeddings: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		err := rows.Scan(
			&result.ChunkText,
			&result.FilePath,
			&result.Score,
			&result.ChunkIndex,
			&result.DocumentID,
			&result.FullDocument,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

// DeleteDocument deletes a document and its embeddings
func (v *VectorStore) DeleteDocument(ctx context.Context, filePath string) error {
	_, err := v.pool.Exec(ctx, "DELETE FROM documents WHERE file_path = $1", filePath)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}
	return nil
}

// GetDocumentCount returns the total number of documents
func (v *VectorStore) GetDocumentCount(ctx context.Context) (int, error) {
	var count int
	err := v.pool.QueryRow(ctx, "SELECT COUNT(*) FROM documents").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get document count: %w", err)
	}
	return count, nil
}

// ListDocuments returns a list of all document file paths
func (v *VectorStore) ListDocuments(ctx context.Context) ([]string, error) {
	rows, err := v.pool.Query(ctx, "SELECT file_path FROM documents ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	var filePaths []string
	for rows.Next() {
		var filePath string
		if err := rows.Scan(&filePath); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		filePaths = append(filePaths, filePath)
	}

	return filePaths, nil
}
