package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/asakuno/go-doc-mcp/internal/mcp"
)

func main() {
	// Load configuration
	config := mcp.LoadConfig()

	// Validate required configuration
	if config.EmbeddingProvider == "openai" && config.OpenAIAPIKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required when using openai provider")
	}

	// Create context that cancels on interrupt
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received interrupt signal, shutting down...")
		cancel()
	}()

	// Initialize and start MCP server
	server, err := mcp.NewDocumentMCPServer(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}
	defer server.Close()

	log.Printf("Go Document MCP Server starting...")
	log.Printf("PostgreSQL: %s:%s/%s", config.PostgresHost, config.PostgresPort, config.PostgresDB)
	log.Printf("Embedding Provider: %s", config.EmbeddingProvider)
	log.Printf("Embedding Model: %s (dim: %d)", config.EmbeddingModel, config.EmbeddingDim)
	log.Printf("Chunk Size: %d, Overlap: %d", config.ChunkSize, config.ChunkOverlap)

	if err := server.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
