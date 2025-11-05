package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *DocumentMCPServer) handleSearchDocuments(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	query, ok := arguments["query"].(string)
	if !ok || query == "" {
		return mcp.NewToolResultError("query parameter is required and must be a string"), nil
	}

	limit := 5
	if limitVal, ok := arguments["limit"].(float64); ok {
		limit = int(limitVal)
	}

	ctx := context.Background()
	results, err := s.vectorStore.Search(ctx, query, limit)
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
		resultText.WriteString(fmt.Sprintf("Chunk Index: %d\n", result.ChunkIndex))
		resultText.WriteString(fmt.Sprintf("\nContent:\n%s\n\n", result.ChunkText))
	}

	return mcp.NewToolResultText(resultText.String()), nil
}

func (s *DocumentMCPServer) handleIndexDocument(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
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

	// Add to vector store
	ctx := context.Background()
	if err := s.vectorStore.AddDocument(ctx, doc, chunks); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to index document: %v", err)), nil
	}

	return mcp.NewToolResultText(
		fmt.Sprintf("Successfully indexed document: %s (%d chunks created)", absPath, len(chunks)),
	), nil
}

func (s *DocumentMCPServer) handleIndexDirectory(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
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

	// Process each document
	ctx := context.Background()
	totalChunks := 0
	successCount := 0
	var errors []string

	for _, doc := range docs {
		chunks := s.chunker.ChunkDocument(doc)
		if err := s.vectorStore.AddDocument(ctx, doc, chunks); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", doc.FilePath, err))
			continue
		}
		totalChunks += len(chunks)
		successCount++
	}

	var resultText strings.Builder
	resultText.WriteString(fmt.Sprintf("Indexed %d/%d documents (%d chunks total)\n", successCount, len(docs), totalChunks))

	if len(errors) > 0 {
		resultText.WriteString("\nErrors:\n")
		for _, errMsg := range errors {
			resultText.WriteString(fmt.Sprintf("- %s\n", errMsg))
		}
	}

	return mcp.NewToolResultText(resultText.String()), nil
}

func (s *DocumentMCPServer) handleListDocuments(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	ctx := context.Background()
	filePaths, err := s.vectorStore.ListDocuments(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list documents: %v", err)), nil
	}

	if len(filePaths) == 0 {
		return mcp.NewToolResultText("No documents indexed yet."), nil
	}

	var resultText strings.Builder
	resultText.WriteString(fmt.Sprintf("Total indexed documents: %d\n\n", len(filePaths)))
	for i, path := range filePaths {
		resultText.WriteString(fmt.Sprintf("%d. %s\n", i+1, path))
	}

	return mcp.NewToolResultText(resultText.String()), nil
}

func (s *DocumentMCPServer) handleDeleteDocument(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	filePath, ok := arguments["file_path"].(string)
	if !ok || filePath == "" {
		return mcp.NewToolResultError("file_path parameter is required and must be a string"), nil
	}

	ctx := context.Background()
	if err := s.vectorStore.DeleteDocument(ctx, filePath); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to delete document: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Successfully deleted document: %s", filePath)), nil
}

func (s *DocumentMCPServer) handleGetStats(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	ctx := context.Background()
	count, err := s.vectorStore.GetDocumentCount(ctx)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get stats: %v", err)), nil
	}

	var resultText strings.Builder
	resultText.WriteString("Document Vector Store Statistics\n")
	resultText.WriteString("================================\n\n")
	resultText.WriteString(fmt.Sprintf("Total Documents: %d\n", count))
	resultText.WriteString(fmt.Sprintf("Chunk Size: %d characters\n", s.config.ChunkSize))
	resultText.WriteString(fmt.Sprintf("Chunk Overlap: %d characters\n", s.config.ChunkOverlap))
	resultText.WriteString(fmt.Sprintf("Embedding Model: %s\n", s.config.EmbeddingModel))
	resultText.WriteString(fmt.Sprintf("Embedding Dimensions: %d\n", s.config.EmbeddingDim))

	return mcp.NewToolResultText(resultText.String()), nil
}
