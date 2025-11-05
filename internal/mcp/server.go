package mcp

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/asakuno/go-doc-mcp/internal/document"
	"github.com/asakuno/go-doc-mcp/internal/vectorstore"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type Config struct {
	PostgresHost      string
	PostgresPort      string
	PostgresUser      string
	PostgresPassword  string
	PostgresDB        string
	EmbeddingProvider string
	OpenAIAPIKey      string
	OllamaURL         string
	DocumentPath      string
	ChunkSize         int
	ChunkOverlap      int
	EmbeddingModel    string
	EmbeddingDim      int
}

type DocumentMCPServer struct {
	config       *Config
	vectorStore  *vectorstore.VectorStore
	loader       *document.Loader
	chunker      *document.Chunker
	mcpServer    *server.MCPServer
}

func LoadConfig() *Config {
	chunkSize, _ := strconv.Atoi(getEnv("CHUNK_SIZE", "1000"))
	chunkOverlap, _ := strconv.Atoi(getEnv("CHUNK_OVERLAP", "200"))
	embeddingDim, _ := strconv.Atoi(getEnv("EMBEDDING_DIMENSIONS", "1536"))

	// Default to Ollama for free embeddings
	provider := getEnv("EMBEDDING_PROVIDER", "ollama")
	model := getEnv("EMBEDDING_MODEL", "nomic-embed-text")
	ollamaURL := getEnv("OLLAMA_URL", "http://localhost:11434")

	// Override defaults for OpenAI
	if provider == "openai" {
		model = getEnv("EMBEDDING_MODEL", "text-embedding-3-small")
	}

	return &Config{
		PostgresHost:      getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:      getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:      getEnv("POSTGRES_USER", "postgres"),
		PostgresPassword:  getEnv("POSTGRES_PASSWORD", "postgres"),
		PostgresDB:        getEnv("POSTGRES_DB", "vectordb"),
		EmbeddingProvider: provider,
		OpenAIAPIKey:      getEnv("OPENAI_API_KEY", ""),
		OllamaURL:         ollamaURL,
		DocumentPath:      getEnv("DOCUMENT_PATH", "./docs"),
		ChunkSize:         chunkSize,
		ChunkOverlap:      chunkOverlap,
		EmbeddingModel:    model,
		EmbeddingDim:      embeddingDim,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func NewDocumentMCPServer(ctx context.Context, config *Config) (*DocumentMCPServer, error) {
	// Initialize embedding service
	var provider vectorstore.EmbeddingProvider
	if config.EmbeddingProvider == "openai" {
		provider = vectorstore.ProviderOpenAI
	} else {
		provider = vectorstore.ProviderOllama
	}
	embeddingService := vectorstore.NewEmbeddingService(provider, config.OpenAIAPIKey, config.EmbeddingModel, config.OllamaURL)

	// Initialize vector store
	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		config.PostgresUser,
		config.PostgresPassword,
		config.PostgresHost,
		config.PostgresPort,
		config.PostgresDB,
	)

	vectorStore, err := vectorstore.NewVectorStore(ctx, connString, embeddingService, config.EmbeddingDim)
	if err != nil {
		return nil, fmt.Errorf("failed to create vector store: %w", err)
	}

	// Initialize document loader and chunker
	loader := document.NewLoader()
	chunker := document.NewChunker(config.ChunkSize, config.ChunkOverlap)

	// Create MCP server
	mcpServer := server.NewMCPServer(
		"Go Document MCP Server",
		"1.0.0",
	)

	s := &DocumentMCPServer{
		config:      config,
		vectorStore: vectorStore,
		loader:      loader,
		chunker:     chunker,
		mcpServer:   mcpServer,
	}

	// Register tools
	s.registerTools()

	return s, nil
}

func (s *DocumentMCPServer) registerTools() {
	// Search documents tool
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "search_documents",
		Description: "Search through indexed documents using vector similarity search. Returns relevant document chunks based on semantic similarity to the query.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search query to find relevant documents",
				},
				"limit": map[string]interface{}{
					"type":        "number",
					"description": "Maximum number of results to return (default: 5)",
					"default":     5,
				},
			},
			Required: []string{"query"},
		},
	}, s.handleSearchDocuments)

	// Index document tool
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "index_document",
		Description: "Index a single document file into the vector database. The document will be split into chunks and embedded for semantic search.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the document file to index",
				},
			},
			Required: []string{"file_path"},
		},
	}, s.handleIndexDocument)

	// Index directory tool
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "index_directory",
		Description: "Index all supported documents in a directory recursively. Supported formats: .md, .txt, .go, .js, .ts, .py, .java, .c, .cpp, .h, .hpp",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"directory_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the directory to index",
				},
			},
			Required: []string{"directory_path"},
		},
	}, s.handleIndexDirectory)

	// List documents tool
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "list_documents",
		Description: "List all indexed documents in the vector database",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}, s.handleListDocuments)

	// Delete document tool
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "delete_document",
		Description: "Delete a document from the vector database by file path",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"file_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the document file to delete",
				},
			},
			Required: []string{"file_path"},
		},
	}, s.handleDeleteDocument)

	// Get stats tool
	s.mcpServer.AddTool(mcp.Tool{
		Name:        "get_stats",
		Description: "Get statistics about the indexed documents",
		InputSchema: mcp.ToolInputSchema{
			Type:       "object",
			Properties: map[string]interface{}{},
		},
	}, s.handleGetStats)
}

func (s *DocumentMCPServer) Start() error {
	log.Println("Starting Go Document MCP Server...")
	return server.ServeStdio(s.mcpServer)
}

func (s *DocumentMCPServer) Close() {
	s.vectorStore.Close()
}
