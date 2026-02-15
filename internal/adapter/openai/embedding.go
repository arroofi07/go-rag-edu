package openai

import (
	"context"

	"github.com/pgvector/pgvector-go"
	openai "github.com/sashabaranov/go-openai"
)

type EmbeddingClient struct {
	client *openai.Client
	model  string
}

// NewEmbeddingClient creates a new OpenAI embedding client
func NewEmbeddingClient(apiKey, model string) *EmbeddingClient {
	return &EmbeddingClient{
		client: openai.NewClient(apiKey),
		model:  model,
	}
}

// Generate Embedding
func (c *EmbeddingClient) GenerateEmbedding(ctx context.Context, text string) (pgvector.Vector, error) {
	resp, err := c.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel(c.model),
	})
	if err != nil {
		return pgvector.Vector{}, err
	}

	if len(resp.Data) == 0 {
		return pgvector.Vector{}, nil
	}

	// convert []float32 to pgvector.Vector
	embedding := make([]float32, len(resp.Data[0].Embedding))
	for i, v := range resp.Data[0].Embedding {
		embedding[i] = v
	}

	return pgvector.NewVector(embedding), nil

}

// generate batch embedding
func (c * EmbeddingClient) GenerateBatchEmbeddings(ctx context.Context,  texts []string) ([]pgvector.Vector, error){
	resp, err := c.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: texts,
		Model: openai.EmbeddingModel(c.model),
	})

	if err != nil {
		return nil, err
	}

	vectors := make([]pgvector.Vector, len(resp.Data))
	for i, data := range resp.Data {
		embedding := make([]float32, len(data.Embedding))
		for j, v := range data.Embedding {
			embedding[j] = v
		}
		vectors[i] = pgvector.NewVector(embedding)
	}

	return vectors, nil

}

