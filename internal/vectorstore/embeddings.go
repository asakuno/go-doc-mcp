package vectorstore

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sashabaranov/go-openai"
)

type EmbeddingProvider string

const (
	ProviderOpenAI EmbeddingProvider = "openai"
	ProviderOllama EmbeddingProvider = "ollama"
)

type EmbeddingService struct {
	provider    EmbeddingProvider
	openaiClient *openai.Client
	ollamaURL   string
	model       string
}

func NewEmbeddingService(provider EmbeddingProvider, apiKey, model, ollamaURL string) *EmbeddingService {
	service := &EmbeddingService{
		provider:  provider,
		model:     model,
		ollamaURL: ollamaURL,
	}

	if provider == ProviderOpenAI {
		service.openaiClient = openai.NewClient(apiKey)
	}

	return service
}

// CreateEmbedding creates an embedding vector for a single text
func (e *EmbeddingService) CreateEmbedding(ctx context.Context, text string) ([]float32, error) {
	switch e.provider {
	case ProviderOpenAI:
		return e.createOpenAIEmbedding(ctx, text)
	case ProviderOllama:
		return e.createOllamaEmbedding(ctx, text)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", e.provider)
	}
}

// CreateEmbeddings creates embeddings for multiple texts in batch
func (e *EmbeddingService) CreateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	switch e.provider {
	case ProviderOpenAI:
		return e.createOpenAIEmbeddings(ctx, texts)
	case ProviderOllama:
		return e.createOllamaEmbeddings(ctx, texts)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", e.provider)
	}
}

// OpenAI implementation
func (e *EmbeddingService) createOpenAIEmbedding(ctx context.Context, text string) ([]float32, error) {
	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel(e.model),
	}

	resp, err := e.openaiClient.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenAI embedding: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding returned from OpenAI")
	}

	return resp.Data[0].Embedding, nil
}

func (e *EmbeddingService) createOpenAIEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
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

		resp, err := e.openaiClient.CreateEmbeddings(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("failed to create OpenAI embeddings for batch: %w", err)
		}

		for _, data := range resp.Data {
			allEmbeddings = append(allEmbeddings, data.Embedding)
		}
	}

	return allEmbeddings, nil
}

// Ollama implementation
type ollamaEmbedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ollamaEmbedResponse struct {
	Embedding []float64 `json:"embedding"`
}

func (e *EmbeddingService) createOllamaEmbedding(ctx context.Context, text string) ([]float32, error) {
	reqBody := ollamaEmbedRequest{
		Model:  e.model,
		Prompt: text,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", e.ollamaURL+"/api/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama API error (status %d): %s", resp.StatusCode, string(body))
	}

	var ollamaResp ollamaEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode Ollama response: %w", err)
	}

	// Convert float64 to float32
	embedding := make([]float32, len(ollamaResp.Embedding))
	for i, v := range ollamaResp.Embedding {
		embedding[i] = float32(v)
	}

	return embedding, nil
}

func (e *EmbeddingService) createOllamaEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	var allEmbeddings [][]float32

	// Ollama doesn't support batch requests, so we process one by one
	for _, text := range texts {
		embedding, err := e.createOllamaEmbedding(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to create Ollama embedding: %w", err)
		}
		allEmbeddings = append(allEmbeddings, embedding)
	}

	return allEmbeddings, nil
}
