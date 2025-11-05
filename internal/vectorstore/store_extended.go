package vectorstore

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/asakuno/go-doc-mcp/internal/document"
	"github.com/jackc/pgx/v5"
	"github.com/pgvector/pgvector-go"
)

// DocumentExists checks if a document already exists and whether it has changed
func (v *VectorStore) DocumentExists(ctx context.Context, filePath, fileHash string) (exists bool, changed bool, err error) {
	var existingHash string
	err = v.pool.QueryRow(ctx,
		"SELECT file_hash FROM documents WHERE file_path = $1",
		filePath,
	).Scan(&existingHash)

	if err == pgx.ErrNoRows {
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("failed to check document existence: %w", err)
	}

	// Document exists, check if hash changed
	return true, existingHash != fileHash, nil
}

// AddOrUpdateDocument adds a document if it doesn't exist, or updates it if the content has changed
func (v *VectorStore) AddOrUpdateDocument(ctx context.Context, doc *document.Document, chunks []*document.Chunk) (added bool, updated bool, skipped bool, err error) {
	exists, changed, err := v.DocumentExists(ctx, doc.FilePath, doc.FileHash)
	if err != nil {
		return false, false, false, err
	}

	if exists && !changed {
		// Document exists and hasn't changed, skip
		return false, false, true, nil
	}

	if exists && changed {
		// Document exists but has changed, reindex
		if err := v.ReindexDocument(ctx, doc, chunks); err != nil {
			return false, false, false, err
		}
		return false, true, false, nil
	}

	// Document doesn't exist, add it
	if err := v.AddDocument(ctx, doc, chunks); err != nil {
		return false, false, false, err
	}
	return true, false, false, nil
}

// ReindexDocument deletes and re-adds a document
func (v *VectorStore) ReindexDocument(ctx context.Context, doc *document.Document, chunks []*document.Chunk) error {
	tx, err := v.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Delete existing document (cascades to embeddings)
	_, err = tx.Exec(ctx, "DELETE FROM documents WHERE file_path = $1", doc.FilePath)
	if err != nil {
		return fmt.Errorf("failed to delete existing document: %w", err)
	}

	// Insert updated document
	metadataJSON, err := json.Marshal(doc.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var docID int
	err = tx.QueryRow(ctx,
		"INSERT INTO documents (content, metadata, file_path, file_hash, file_size, file_modified_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, NOW()) RETURNING id",
		doc.Content, metadataJSON, doc.FilePath, doc.FileHash, doc.FileSize, doc.FileModifiedAt,
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

// SearchWithFilter performs similarity search with metadata filtering
func (v *VectorStore) SearchWithFilter(ctx context.Context, query string, limit int, filter *SearchFilter) ([]SearchResult, error) {
	// Create embedding for query
	queryEmbedding, err := v.embeddingService.CreateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to create query embedding: %w", err)
	}

	// Build WHERE clause for filtering
	whereClause := ""
	args := []interface{}{pgvector.NewVector(queryEmbedding)}
	argIndex := 2

	if filter != nil {
		var conditions []string

		if len(filter.FileExtensions) > 0 {
			extConditions := make([]string, len(filter.FileExtensions))
			for i, ext := range filter.FileExtensions {
				extConditions[i] = fmt.Sprintf("d.file_path LIKE $%d", argIndex)
				args = append(args, "%"+ext)
				argIndex++
			}
			conditions = append(conditions, "("+strings.Join(extConditions, " OR ")+")")
		}

		if len(filter.FilePaths) > 0 {
			pathConditions := make([]string, len(filter.FilePaths))
			for i, path := range filter.FilePaths {
				pathConditions[i] = fmt.Sprintf("d.file_path LIKE $%d", argIndex)
				args = append(args, path+"%")
				argIndex++
			}
			conditions = append(conditions, "("+strings.Join(pathConditions, " OR ")+")")
		}

		if len(conditions) > 0 {
			whereClause = "WHERE " + strings.Join(conditions, " AND ")
		}
	}

	query_sql := fmt.Sprintf(`
		SELECT
			e.chunk_text,
			d.file_path,
			1 - (e.embedding <=> $1) as score,
			e.chunk_index,
			e.document_id,
			d.content
		FROM embeddings e
		JOIN documents d ON e.document_id = d.id
		%s
		ORDER BY e.embedding <=> $1
		LIMIT $%d
	`, whereClause, argIndex)

	args = append(args, limit)

	rows, err := v.pool.Query(ctx, query_sql, args...)
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

// SearchWithContext performs similarity search and includes surrounding chunks for context
func (v *VectorStore) SearchWithContext(ctx context.Context, query string, limit int, filter *SearchFilter) ([]SearchResult, error) {
	// First get the basic search results
	var results []SearchResult
	var err error

	if filter != nil {
		results, err = v.SearchWithFilter(ctx, query, limit, filter)
	} else {
		results, err = v.Search(ctx, query, limit)
	}

	if err != nil {
		return nil, err
	}

	// For each result, fetch previous and next chunks
	for i := range results {
		// Get previous chunk
		if results[i].ChunkIndex > 0 {
			var prevChunk string
			err := v.pool.QueryRow(ctx, `
				SELECT chunk_text
				FROM embeddings
				WHERE document_id = $1 AND chunk_index = $2
			`, results[i].DocumentID, results[i].ChunkIndex-1).Scan(&prevChunk)
			if err == nil {
				results[i].PreviousChunk = prevChunk
			}
		}

		// Get next chunk
		var nextChunk string
		err := v.pool.QueryRow(ctx, `
			SELECT chunk_text
			FROM embeddings
			WHERE document_id = $1 AND chunk_index = $2
		`, results[i].DocumentID, results[i].ChunkIndex+1).Scan(&nextChunk)
		if err == nil {
			results[i].NextChunk = nextChunk
		}
	}

	return results, nil
}

// GetDocumentInfo returns information about a specific document
func (v *VectorStore) GetDocumentInfo(ctx context.Context, filePath string) (map[string]interface{}, error) {
	var (
		fileHash       string
		fileSize       int64
		fileModifiedAt string
		createdAt      string
		updatedAt      string
		chunkCount     int
	)

	err := v.pool.QueryRow(ctx, `
		SELECT
			d.file_hash,
			d.file_size,
			d.file_modified_at,
			d.created_at,
			d.updated_at,
			COUNT(e.id) as chunk_count
		FROM documents d
		LEFT JOIN embeddings e ON d.id = e.document_id
		WHERE d.file_path = $1
		GROUP BY d.id
	`, filePath).Scan(&fileHash, &fileSize, &fileModifiedAt, &createdAt, &updatedAt, &chunkCount)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("document not found: %s", filePath)
		}
		return nil, fmt.Errorf("failed to get document info: %w", err)
	}

	info := map[string]interface{}{
		"file_path":        filePath,
		"file_hash":        fileHash,
		"file_size":        fileSize,
		"file_modified_at": fileModifiedAt,
		"created_at":       createdAt,
		"updated_at":       updatedAt,
		"chunk_count":      chunkCount,
		"extension":        filepath.Ext(filePath),
	}

	return info, nil
}

// GetStatsByExtension returns document statistics grouped by file extension
func (v *VectorStore) GetStatsByExtension(ctx context.Context) (map[string]int, error) {
	rows, err := v.pool.Query(ctx, `
		SELECT
			substring(file_path from '\.([^.]*)$') as extension,
			COUNT(*) as count
		FROM documents
		GROUP BY extension
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var ext string
		var count int
		if err := rows.Scan(&ext, &count); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		if ext != "" {
			stats["."+ext] = count
		}
	}

	return stats, nil
}
