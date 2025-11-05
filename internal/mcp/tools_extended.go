package mcp

import (
	"fmt"
	"strings"

	"github.com/asakuno/go-doc-mcp/internal/vectorstore"
	"github.com/mark3labs/mcp-go/mcp"
)

// handleSearchWithContext performs semantic search with surrounding context
func (s *DocumentMCPServer) handleSearchWithContext(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	query, limit, err := getQueryAndLimit(arguments)
	if err != nil {
		return wrapToolError("%s", err)
	}

	filter := parseFilterFromArguments(arguments)

	ctx := newContext()
	results, err := s.vectorStore.SearchWithContext(ctx, query, limit, filter)
	if err != nil {
		return wrapToolError(ErrFailedToSearch, err)
	}

	if len(results) == 0 {
		return wrapToolSuccess(MsgNoDocumentsFound)
	}

	return wrapToolSuccess(formatSearchResultsWithContext(results))
}

// handleSearchWithFilter performs semantic search with metadata filtering
func (s *DocumentMCPServer) handleSearchWithFilter(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	query, limit, err := getQueryAndLimit(arguments)
	if err != nil {
		return wrapToolError("%s", err)
	}

	filter := parseFilterFromArguments(arguments)

	ctx := newContext()
	results, err := s.vectorStore.SearchWithFilter(ctx, query, limit, filter)
	if err != nil {
		return wrapToolError(ErrFailedToSearch, err)
	}

	if len(results) == 0 {
		return wrapToolSuccess("No relevant documents found with the given filters.")
	}

	return wrapToolSuccess(formatFilteredSearchResults(results, filter))
}

// handleReindexDocument re-indexes a document (useful when content has changed)
func (s *DocumentMCPServer) handleReindexDocument(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	absPath, err := getFilePath(arguments)
	if err != nil {
		return wrapToolError("%s", err)
	}

	doc, chunks, err := s.loadAndChunkDocument(absPath)
	if err != nil {
		return wrapToolError("%s", err)
	}

	ctx := newContext()
	if err := s.vectorStore.ReindexDocument(ctx, doc, chunks); err != nil {
		return wrapToolError(ErrFailedToReindex, err)
	}

	return wrapToolSuccess(MsgDocumentReindexed, absPath, len(chunks))
}

// handleIndexDirectoryIncremental indexes a directory with incremental updates (skips unchanged files)
func (s *DocumentMCPServer) handleIndexDirectoryIncremental(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	absPath, err := getDirectoryPath(arguments)
	if err != nil {
		return wrapToolError("%s", err)
	}

	docs, err := s.loader.LoadDirectory(absPath)
	if err != nil {
		return wrapToolError(ErrFailedToLoad, err)
	}

	if len(docs) == 0 {
		return wrapToolSuccess(MsgNoDocumentsInDir)
	}

	ctx := newContext()
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

	return wrapToolSuccess(formatIncrementalIndexResult(len(docs), addedCount, updatedCount, skippedCount, totalChunks, errors))
}

// handleGetDocumentInfo returns detailed information about a document
func (s *DocumentMCPServer) handleGetDocumentInfo(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	filePath, ok := arguments["file_path"].(string)
	if !ok || filePath == "" {
		return wrapToolError(ErrFilePathRequired)
	}

	ctx := newContext()
	info, err := s.vectorStore.GetDocumentInfo(ctx, filePath)
	if err != nil {
		return wrapToolError(ErrFailedToGetInfo, err)
	}

	return wrapToolSuccess(formatDocumentInfo(info))
}

// Helper formatting functions for extended tools

func formatFilteredSearchResults(results []vectorstore.SearchResult, filter *vectorstore.SearchFilter) string {
	var resultText strings.Builder
	resultText.WriteString(fmt.Sprintf("Found %d relevant document chunks", len(results)))

	if filter != nil {
		if len(filter.FileExtensions) > 0 {
			resultText.WriteString(fmt.Sprintf(" (filtered by types: %v)", filter.FileExtensions))
		}
		if len(filter.FilePaths) > 0 {
			resultText.WriteString(fmt.Sprintf(" (filtered by paths: %v)", filter.FilePaths))
		}
	}
	resultText.WriteString(":\n\n")

	for i, result := range results {
		resultText.WriteString(fmt.Sprintf("--- Result %d (Score: %.4f) ---\n", i+1, result.Score))
		resultText.WriteString(fmt.Sprintf("File: %s\n", result.FilePath))
		resultText.WriteString(fmt.Sprintf("Chunk Index: %d\n", result.ChunkIndex))
		resultText.WriteString(fmt.Sprintf("\nContent:\n%s\n\n", result.ChunkText))
	}

	return resultText.String()
}

func formatIncrementalIndexResult(totalDocs, addedCount, updatedCount, skippedCount, totalChunks int, errors []string) string {
	var resultText strings.Builder
	resultText.WriteString(fmt.Sprintf("Processed %d documents:\n", totalDocs))
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

	return resultText.String()
}

func formatDocumentInfo(info map[string]interface{}) string {
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

	return resultText.String()
}
