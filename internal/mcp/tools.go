package mcp

import (
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

func (s *DocumentMCPServer) handleSearchDocuments(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	query, limit, err := getQueryAndLimit(arguments)
	if err != nil {
		return wrapToolError("%s", err)
	}

	ctx := newContext()
	results, err := s.vectorStore.Search(ctx, query, limit)
	if err != nil {
		return wrapToolError(ErrFailedToSearch, err)
	}

	if len(results) == 0 {
		return wrapToolSuccess(MsgNoDocumentsFound)
	}

	return wrapToolSuccess(formatSearchResults(results))
}

func (s *DocumentMCPServer) handleIndexDocument(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	absPath, err := getFilePath(arguments)
	if err != nil {
		return wrapToolError("%s", err)
	}

	doc, chunks, err := s.loadAndChunkDocument(absPath)
	if err != nil {
		return wrapToolError("%s", err)
	}

	ctx := newContext()
	if err := s.vectorStore.AddDocument(ctx, doc, chunks); err != nil {
		return wrapToolError(ErrFailedToIndex, err)
	}

	return wrapToolSuccess(MsgDocumentIndexed, absPath, len(chunks))
}

func (s *DocumentMCPServer) handleIndexDirectory(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
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

	return wrapToolSuccess(formatDirectoryIndexResult(successCount, len(docs), totalChunks, errors))
}

func (s *DocumentMCPServer) handleListDocuments(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	ctx := newContext()
	filePaths, err := s.vectorStore.ListDocuments(ctx)
	if err != nil {
		return wrapToolError(ErrFailedToList, err)
	}

	if len(filePaths) == 0 {
		return wrapToolSuccess(MsgNoDocumentsIndexed)
	}

	return wrapToolSuccess(formatDocumentList(filePaths))
}

func (s *DocumentMCPServer) handleDeleteDocument(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	filePath, ok := arguments["file_path"].(string)
	if !ok || filePath == "" {
		return wrapToolError(ErrFilePathRequired)
	}

	ctx := newContext()
	if err := s.vectorStore.DeleteDocument(ctx, filePath); err != nil {
		return wrapToolError(ErrFailedToDelete, err)
	}

	return wrapToolSuccess(MsgDocumentDeleted, filePath)
}

func (s *DocumentMCPServer) handleGetStats(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	ctx := newContext()

	count, err := s.vectorStore.GetDocumentCount(ctx)
	if err != nil {
		return wrapToolError(ErrFailedToGetStats, err)
	}

	statsByExt, err := s.vectorStore.GetStatsByExtension(ctx)
	if err != nil {
		return wrapToolError(ErrFailedToGetStats, err)
	}

	return wrapToolSuccess(formatStats(count, s.config, statsByExt))
}

// Helper formatting functions

func formatDirectoryIndexResult(successCount, totalCount, totalChunks int, errors []string) string {
	var resultText strings.Builder
	resultText.WriteString(fmt.Sprintf("Indexed %d/%d documents (%d chunks total)\n", successCount, totalCount, totalChunks))

	if len(errors) > 0 {
		resultText.WriteString("\nErrors:\n")
		for _, errMsg := range errors {
			resultText.WriteString(fmt.Sprintf("- %s\n", errMsg))
		}
	}

	return resultText.String()
}

func formatDocumentList(filePaths []string) string {
	var resultText strings.Builder
	resultText.WriteString(fmt.Sprintf("Total indexed documents: %d\n\n", len(filePaths)))
	for i, path := range filePaths {
		resultText.WriteString(fmt.Sprintf("%d. %s\n", i+1, path))
	}
	return resultText.String()
}

func formatStats(count int, config *Config, statsByExt map[string]int) string {
	var resultText strings.Builder
	resultText.WriteString("Document Vector Store Statistics\n")
	resultText.WriteString("================================\n\n")
	resultText.WriteString(fmt.Sprintf("Total Documents: %d\n", count))
	resultText.WriteString(fmt.Sprintf("Embedding Provider: %s\n", config.EmbeddingProvider))
	resultText.WriteString(fmt.Sprintf("Embedding Model: %s\n", config.EmbeddingModel))
	resultText.WriteString(fmt.Sprintf("Embedding Dimensions: %d\n", config.EmbeddingDim))
	resultText.WriteString(fmt.Sprintf("Chunk Size: %d characters\n", config.ChunkSize))
	resultText.WriteString(fmt.Sprintf("Chunk Overlap: %d characters\n", config.ChunkOverlap))

	if len(statsByExt) > 0 {
		resultText.WriteString("\nDocuments by File Type:\n")
		for ext, extCount := range statsByExt {
			resultText.WriteString(fmt.Sprintf("  %s: %d\n", ext, extCount))
		}
	}

	return resultText.String()
}
