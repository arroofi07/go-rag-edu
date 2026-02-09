package entity

import (
	"time"

	"github.com/pgvector/pgvector-go"
)

type ChunkMetadata struct {
	Source         string  `json:"source"` // "text" or "ocr"
	Confidence     float64 `json:"confidence,omitempty"`
	PageNumber     int     `json:"pageNumber,omitempty"`
	ImageIndex     int     `json:"imageIndex,omitempty"`
	ProcessingTime int64   `json:"processingTime,omitempty"`
}

type DocumentChunk struct {
	ID         string          `db:"id" json:"id"`
	DocumentID string          `db:"document_id" json:"documentId"`
	ChunkIndex int             `db:"chunk_index" json:"chunkIndex"`
	Content    string          `db:"content" json:"content"`
	Embedding  pgvector.Vector `db:"embedding" json:"-"`
	Metadata   ChunkMetadata   `db:"metadata" json:"metadata"`
	CreatedAt  time.Time       `db:"created_at" json:"createdAt"`
}

type SimiliarChunk struct {
	DocumentChunk
	Similarity float64 `db:"similarity" json:"similarity"`
}
