package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/asakuno/go-doc-mcp/internal/document"
	"github.com/asakuno/go-doc-mcp/internal/vectorstore"
	"github.com/mark3labs/mcp-go/mcp"
)

// getQueryAndLimit extracts query and limit from arguments
func getQueryAndLimit(arguments map[string]interface{}) (query string, limit int, err error) {
	q, ok := arguments["query"].(string)
	if !ok || q == "" {
		return "", 0, fmt.Errorf(ErrQueryRequired)
	}

	limit = DefaultSearchLimit
	if limitVal, ok := arguments["limit"].(float64); ok {
		limit = int(limitVal)
	}

	return q, limit, nil
}

// getFilePath extracts and validates file path from arguments
func getFilePath(arguments map[string]interface{}) (string, error) {
	filePath, ok := arguments["file_path"].(string)
	if !ok || filePath == "" {
		return "", fmt.Errorf(ErrFilePathRequired)
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf(ErrInvalidPath, err)
	}

	return absPath, nil
}

// getDirectoryPath extracts and validates directory path from arguments
func getDirectoryPath(arguments map[string]interface{}) (string, error) {
	dirPath, ok := arguments["directory_path"].(string)
	if !ok || dirPath == "" {
		return "", fmt.Errorf(ErrDirectoryPathRequired)
	}

	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return "", fmt.Errorf(ErrInvalidPath, err)
	}

	return absPath, nil
}

// parseFilterFromArguments extracts filter parameters
func parseFilterFromArguments(arguments map[string]interface{}) *vectorstore.SearchFilter {
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

	// Return nil if no filters were set
	if len(filter.FileExtensions) == 0 && len(filter.FilePaths) == 0 {
		return nil
	}

	return filter
}

// loadAndChunkDocument is a helper to load a file and chunk it
func (s *DocumentMCPServer) loadAndChunkDocument(filePath string) (*document.Document, []*document.Chunk, error) {
	doc, err := s.loader.LoadFile(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf(ErrFailedToLoad, err)
	}

	chunks := s.chunker.ChunkDocument(doc)
	return doc, chunks, nil
}

// formatSearchResults formats search results into readable text
func formatSearchResults(results []vectorstore.SearchResult) string {
	var resultText strings.Builder
	resultText.WriteString(fmt.Sprintf("Found %d relevant document chunks:\n\n", len(results)))

	for i, result := range results {
		resultText.WriteString(fmt.Sprintf("--- Result %d (Score: %.4f) ---\n", i+1, result.Score))
		resultText.WriteString(fmt.Sprintf("File: %s\n", result.FilePath))
		resultText.WriteString(fmt.Sprintf("Chunk Index: %d\n", result.ChunkIndex))
		resultText.WriteString(fmt.Sprintf("\nContent:\n%s\n\n", result.ChunkText))
	}

	return resultText.String()
}

// formatSearchResultsWithContext formats search results with context
func formatSearchResultsWithContext(results []vectorstore.SearchResult) string {
	var resultText strings.Builder
	resultText.WriteString(fmt.Sprintf("Found %d relevant document chunks:\n\n", len(results)))

	for i, result := range results {
		resultText.WriteString(fmt.Sprintf("--- Result %d (Score: %.4f) ---\n", i+1, result.Score))
		resultText.WriteString(fmt.Sprintf("File: %s\n", result.FilePath))
		resultText.WriteString(fmt.Sprintf("Chunk Index: %d\n\n", result.ChunkIndex))

		// Show previous chunk for context
		if result.PreviousChunk != "" {
			resultText.WriteString("⬆️  Previous Context:\n")
			resultText.WriteString(truncateText(result.PreviousChunk, DefaultContextTruncate))
			resultText.WriteString("\n\n")
		}

		// Show main content
		resultText.WriteString("📄 Main Content:\n")
		resultText.WriteString(result.ChunkText)
		resultText.WriteString("\n\n")

		// Show next chunk for context
		if result.NextChunk != "" {
			resultText.WriteString("⬇️  Next Context:\n")
			resultText.WriteString(truncateText(result.NextChunk, DefaultContextTruncate))
			resultText.WriteString("\n\n")
		}

		resultText.WriteString("---\n\n")
	}

	return resultText.String()
}

// truncateText truncates text to maxLen characters
func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}

// newContext creates a background context (helper for consistency)
func newContext() context.Context {
	return context.Background()
}

// wrapToolError wraps an error message in a tool result
func wrapToolError(format string, args ...interface{}) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError(fmt.Sprintf(format, args...)), nil
}

// wrapToolSuccess wraps a success message in a tool result
func wrapToolSuccess(format string, args ...interface{}) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(fmt.Sprintf(format, args...)), nil
}
