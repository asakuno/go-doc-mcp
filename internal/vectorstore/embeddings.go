package vectorstore

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

type EmbeddingService struct {
	client *openai.Client
	model  string
}

func NewEmbeddingService(apiKey, model string) *EmbeddingService {
	return &EmbeddingService{
		client: openai.NewClient(apiKey),
		model:  model,
	}
}

// CreateEmbedding creates an embedding vector for a single text
func (e *EmbeddingService) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel(e.model),
	}

	resp, err := e.client.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}

	return resp.Data[0].Embedding, nil
}

// CreateEmbeddings creates embeddings for multiple texts in batch
func (e *EmbeddingService) CreateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// OpenAI has a limit on batch size, so we process in chunks
	const batchSize = 100
	var allEmbeddings [][]float32

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		req := openai.EmbeddingRequest{
			Input: batch,
			Model: openai.EmbeddingModel(e.model),
		}

		resp, err := e.client.CreateEmbeddings(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to create embeddings for batch: %w", err)
		}

		for _, data := range resp.Data {
			allEmbeddings = append(allEmbeddings, data.Embedding)
		}
	}

	return allEmbeddings, nil
}
