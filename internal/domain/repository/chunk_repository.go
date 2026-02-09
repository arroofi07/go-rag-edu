package repository

import (
	"context"
	"internal/domain/entity"
)

type ChunkRepository interface {
	CreateChunk(ctx context.Context, chunk *entity.DocumentChunk) error
	SearchSimilar(ctx context.Context, params SimilaritySearchParams) ([]entity.SimilarChunk, error)
	DeleteChunk(ctx context.Context, documentID string) error
}
