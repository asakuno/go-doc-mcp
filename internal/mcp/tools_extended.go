package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/asakuno/go-doc-mcp/internal/vectorstore"
	"github.com/mark3labs/mcp-go/mcp"
)

// handleSearchWithContext performs semantic search with surrounding context
func (s *DocumentMCPServer) handleSearchWithContext(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	query, ok := arguments["query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("query parameter is required and must be a string"), nil
	}

	limit := 5
	if limitVal, ok := arguments["limit"].(float64); ok {
		limit = int(limitVal)
	}

	// Parse filters
	var filter *vectorstore.SearchFilter
	if fileTypes, ok := arguments["file_types"].([]interface{}); ok && len(fileTypes) > 0 {
		filter = &vectorstore.SearchFilter{}
		for _, ft := range fileTypes {
			if ftStr, ok := ft.(string); ok {
				filter.FileExtensions = append(filter.FileExtensions, ftStr)
			}
		}
	}

	ctx := context.Background()
	results, err := s.vectorStore.SearchWithContext(ctx, query, limit, filter)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to search documents: %v", err)), nil
	}

	if len(results) == 0 {
		return mcp.NewToolResultText("No relevant documents found."), nil
	}

	var resultText strings.Builder
	resultText.WriteString(fmt.Sprintf("Found %d relevant document chunks:\n\n", len(results)))

	for i, result := range results {
		resultText.WriteString(fmt.Sprintf("--- Result %d (Score: %.4f) ---\n", i+1, result.Score))
		resultText.WriteString(fmt.Sprintf("File: %s\n", result.FilePath))
		resultText.WriteString(fmt.Sprintf("Chunk Index: %d\n\n", result.ChunkIndex))

		// Show previous chunk for context
		if result.PreviousChunk != "" {
			resultText.WriteString("⬆️  Previous Context:\n")
			resultText.WriteString(truncateText(result.PreviousChunk, 200))
			resultText.WriteString("\n\n")
		}

		// Show main content
		resultText.WriteString("📄 Main Content:\n")
		resultText.WriteString(result.ChunkText)
		resultText.WriteString("\n\n")

		// Show next chunk for context
		if result.NextChunk != "" {
			resultText.WriteString("⬇️  Next Context:\n")
			resultText.WriteString(truncateText(result.NextChunk, 200))
			resultText.WriteString("\n\n")
		}

		resultText.WriteString("---\n\n")
	}

	return mcp.NewToolResultText(resultText.String()), nil
}

// handleSearchWithFilter performs semantic search with metadata filtering
func (s *DocumentMCPServer) handleSearchWithFilter(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	query, ok := arguments["query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("query parameter is required and must be a string"), nil
	}

	limit := 5
	if limitVal, ok := arguments["limit"].(float64); ok {
		limit = int(limitVal)
	}

	// Build filter
	filter := &vectorstore.SearchFilter{}

	// File type filter
	if fileTypes, ok := arguments["file_types"].([]interface{}); ok && len(fileTypes) > 0 {
		for _, ft := range fileTypes {
			if ftStr, ok := ft.(string); ok {
				filter.FileExtensions = append(filter.FileExtensions, ftStr)
			}
		}
	}

	// Path filter
	if paths, ok := arguments["paths"].([]interface{}); ok && len(paths) > 0 {
		for _, p := range paths {
			if pStr, ok := p.(string); ok {
				filter.FilePaths = append(filter.FilePaths, pStr)
			}
		}
	}

	ctx := context.Background()
	results, err := s.vectorStore.SearchWithFilter(ctx, query, limit, filter)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to search documents: %v", err)), nil
	}

	if len(results) == 0 {
		return mcp.NewToolResultText("No relevant documents found with the given filters."), nil
	}

	var resultText strings.Builder
	resultText.WriteString(fmt.Sprintf("Found %d relevant document chunks", len(results)))
	if len(filter.FileExtensions) > 0 {
		resultText.WriteString(fmt.Sprintf(" (filtered by types: %v)", filter.FileExtensions))
	}
	if len(filter.FilePaths) > 0 {
		resultText.WriteString(fmt.Sprintf(" (filtered by paths: %v)", filter.FilePaths))
	}
	resultText.WriteString(":\n\n")

	for i, result := range results {
		resultText.WriteString(fmt.Sprintf("--- Result %d (Score: %.4f) ---\n", i+1, result.Score))
		resultText.WriteString(fmt.Sprintf("File: %s\n", result.FilePath))
		resultText.WriteString(fmt.Sprintf("Chunk Index: %d\n", result.ChunkIndex))
		resultText.WriteString(fmt.Sprintf("\nContent:\n%s\n\n", result.ChunkText))
	}

	return mcp.NewToolResultText(resultText.String()), nil
}

// handleReindexDocument re-indexes a document (useful when content has changed)
func (s *DocumentMCPServer) handleReindexDocument(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	filePath, ok := arguments["file_path"].(string)
	if !ok || filePath == "" {
		return mcp.NewToolResultError("file_path parameter is required and must be a string"), nil
	}

	// Make path absolute
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid file path: %v", err)), nil
	}

	// Load document
	doc, err := s.loader.LoadFile(absPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to load document: %v", err)), nil
	}

	// Chunk document
	chunks := s.chunker.ChunkDocument(doc)

	// Reindex
	ctx := context.Background()
	if err := s.vectorStore.ReindexDocument(ctx, doc, chunks); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to reindex document: %v", err)), nil
	}

	return mcp.NewToolResultText(
		fmt.Sprintf("Successfully reindexed document: %s (%d chunks)", absPath, len(chunks)),
	), nil
}

// handleIndexDirectoryIncremental indexes a directory with incremental updates (skips unchanged files)
func (s *DocumentMCPServer) handleIndexDirectoryIncremental(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	dirPath, ok := arguments["directory_path"].(string)
	if !ok || dirPath == "" {
		return mcp.NewToolResultError("directory_path parameter is required and must be a string"), nil
	}

	// Make path absolute
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid directory path: %v", err)), nil
	}

	// Load all documents
	docs, err := s.loader.LoadDirectory(absPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to load documents: %v", err)), nil
	}

	if len(docs) == 0 {
		return mcp.NewToolResultText("No supported documents found in the directory."), nil
	}

	// Process each document with incremental update
	ctx := context.Background()
	addedCount := 0
	updatedCount := 0
	skippedCount := 0
	totalChunks := 0
	var errors []string

	for _, doc := range docs {
		chunks := s.chunker.ChunkDocument(doc)
		added, updated, skipped, err := s.vectorStore.AddOrUpdateDocument(ctx, doc, chunks)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", doc.FilePath, err))
			continue
		}

		if added {
			addedCount++
			totalChunks += len(chunks)
		} else if updated {
			updatedCount++
			totalChunks += len(chunks)
		} else if skipped {
			skippedCount++
		}
	}

	var resultText strings.Builder
	resultText.WriteString(fmt.Sprintf("Processed %d documents:\n", len(docs)))
	resultText.WriteString(fmt.Sprintf("- Added: %d\n", addedCount))
	resultText.WriteString(fmt.Sprintf("- Updated: %d\n", updatedCount))
	resultText.WriteString(fmt.Sprintf("- Skipped (unchanged): %d\n", skippedCount))
	resultText.WriteString(fmt.Sprintf("- Total chunks: %d\n", totalChunks))

	if len(errors) > 0 {
		resultText.WriteString("\nErrors:\n")
		for _, errMsg := range errors {
			resultText.WriteString(fmt.Sprintf("- %s\n", errMsg))
		}
	}

	return mcp.NewToolResultText(resultText.String()), nil
}

// handleGetDocumentInfo returns detailed information about a document
func (s *DocumentMCPServer) handleGetDocumentInfo(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	filePath, ok := arguments["file_path"].(string)
	if !ok || filePath == "" {
		return mcp.NewToolResultError("file_path parameter is required and must be a string"), nil
	}

	ctx := context.Background()
	info, err := s.vectorStore.GetDocumentInfo(ctx, filePath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get document info: %v", err)), nil
	}

	var resultText strings.Builder
	resultText.WriteString("Document Information\n")
	resultText.WriteString("===================\n\n")
	resultText.WriteString(fmt.Sprintf("File Path: %v\n", info["file_path"]))
	resultText.WriteString(fmt.Sprintf("Extension: %v\n", info["extension"]))
	resultText.WriteString(fmt.Sprintf("File Size: %v bytes\n", info["file_size"]))
	resultText.WriteString(fmt.Sprintf("File Hash: %v\n", info["file_hash"]))
	resultText.WriteString(fmt.Sprintf("Chunks: %v\n", info["chunk_count"]))
	resultText.WriteString(fmt.Sprintf("Modified: %v\n", info["file_modified_at"]))
	resultText.WriteString(fmt.Sprintf("Indexed: %v\n", info["created_at"]))
	resultText.WriteString(fmt.Sprintf("Updated: %v\n", info["updated_at"]))

	return mcp.NewToolResultText(resultText.String()), nil
}

// Helper function to truncate text
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
