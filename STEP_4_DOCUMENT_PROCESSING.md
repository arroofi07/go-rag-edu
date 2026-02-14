# 🔄 STEP 4: Document Processing (Text Extraction, Chunking, Embedding)

> Fitur untuk process dokumen: extract text, chunking, generate embeddings, dan simpan ke database

## 4.1 Install Dependencies

```bash
# PDF processing
go get github.com/unidoc/unipdf/v3@v3.55.0

# Image processing & OCR
go get github.com/otiai10/gosseract/v2@v2.4.1
go get github.com/disintegration/imaging@v1.6.2
```

## 4.2 Create Document Chunk Entity & Repository

**File**: `internal/domain/entity/document_chunk.go`

```go
package entity

import (
	"time"

	"github.com/pgvector/pgvector-go"
)

type ChunkMetadata struct {
	Source     string  `json:"source"` // "text" or "ocr"
	PageNumber int     `json:"pageNumber,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
}

type DocumentChunk struct {
	ID         string          `db:"id" json:"id"`
	DocumentID string          `db:"document_id" json:"documentId"`
	ChunkIndex int             `db:"chunk_index" json:"chunkIndex"`
	Content    string          `db:"content" json:"content"`
	Embedding  pgvector.Vector `db:"embedding" json:"-"`
	Metadata   []byte          `db:"metadata" json:"metadata"` // JSON
	CreatedAt  time.Time       `db:"created_at" json:"createdAt"`
}

type SimilarChunk struct {
	DocumentChunk
	Similarity float64 `db:"similarity" json:"similarity"`
}
```

**File**: `internal/domain/repository/chunk_repository.go`

```go
package repository

import (
	"context"
	"rag-api/internal/domain/entity"

	"github.com/pgvector/pgvector-go"
)

type ChunkRepository interface {
	Create(ctx context.Context, chunk *entity.DocumentChunk) error
	CreateBatch(ctx context.Context, chunks []entity.DocumentChunk) error
	SearchSimilar(ctx context.Context, embedding pgvector.Vector, topK int, threshold float64) ([]entity.SimilarChunk, error)
	DeleteByDocumentID(ctx context.Context, documentID string) error
}
```

**File**: `internal/adapter/repository/postgres/chunk_repository.go`

```go
package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/pgvector/pgvector-go"
	"rag-api/internal/domain/entity"
	"rag-api/internal/domain/repository"
)

type chunkRepository struct {
	db *sqlx.DB
}

func NewChunkRepository(db *sqlx.DB) repository.ChunkRepository {
	return &chunkRepository{db: db}
}

func (r *chunkRepository) Create(ctx context.Context, chunk *entity.DocumentChunk) error {
	chunk.ID = uuid.New().String()
	chunk.CreatedAt = time.Now()

	query := `
		INSERT INTO document_chunks (id, document_id, chunk_index, content, embedding, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query,
		chunk.ID,
		chunk.DocumentID,
		chunk.ChunkIndex,
		chunk.Content,
		chunk.Embedding,
		chunk.Metadata,
		chunk.CreatedAt,
	)

	return err
}

func (r *chunkRepository) CreateBatch(ctx context.Context, chunks []entity.DocumentChunk) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO document_chunks (id, document_id, chunk_index, content, embedding, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	for i := range chunks {
		chunks[i].ID = uuid.New().String()
		chunks[i].CreatedAt = time.Now()

		_, err := tx.ExecContext(ctx, query,
			chunks[i].ID,
			chunks[i].DocumentID,
			chunks[i].ChunkIndex,
			chunks[i].Content,
			chunks[i].Embedding,
			chunks[i].Metadata,
			chunks[i].CreatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *chunkRepository) SearchSimilar(
	ctx context.Context,
	embedding pgvector.Vector,
	topK int,
	threshold float64,
) ([]entity.SimilarChunk, error) {
	query := `
		SELECT
			dc.id,
			dc.document_id,
			dc.chunk_index,
			dc.content,
			dc.metadata,
			dc.created_at,
			1 - (dc.embedding <=> $1) as similarity
		FROM document_chunks dc
		INNER JOIN documents d ON dc.document_id = d.id
		WHERE d.status = 'COMPLETED'
			AND (1 - (dc.embedding <=> $1)) >= $2
		ORDER BY dc.embedding <=> $1
		LIMIT $3
	`

	rows, err := r.db.QueryContext(ctx, query, embedding, threshold, topK)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chunks []entity.SimilarChunk
	for rows.Next() {
		var chunk entity.SimilarChunk
		err := rows.Scan(
			&chunk.ID,
			&chunk.DocumentID,
			&chunk.ChunkIndex,
			&chunk.Content,
			&chunk.Metadata,
			&chunk.CreatedAt,
			&chunk.Similarity,
		)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, chunk)
	}

	return chunks, rows.Err()
}

func (r *chunkRepository) DeleteByDocumentID(ctx context.Context, documentID string) error {
	query := `DELETE FROM document_chunks WHERE document_id = $1`
	_, err := r.db.ExecContext(ctx, query, documentID)
	return err
}
```

## 4.3 Create OpenAI Adapter

**File**: `internal/adapter/openai/embedding.go`

```go
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

func NewEmbeddingClient(apiKey, model string) *EmbeddingClient {
	return &EmbeddingClient{
		client: openai.NewClient(apiKey),
		model:  model,
	}
}

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

	// Convert []float32 to pgvector.Vector
	embedding := make([]float32, len(resp.Data[0].Embedding))
	for i, v := range resp.Data[0].Embedding {
		embedding[i] = v
	}

	return pgvector.NewVector(embedding), nil
}

func (c *EmbeddingClient) GenerateBatchEmbeddings(ctx context.Context, texts []string) ([]pgvector.Vector, error) {
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
```

## 4.4 Create Text Extractor Service

**File**: `internal/usecase/document/text_extractor.go`

```go
package document

import (
	"bytes"
	"fmt"

	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
)

type TextExtractor struct{}

func NewTextExtractor() *TextExtractor {
	return &TextExtractor{}
}

func (te *TextExtractor) ExtractFromPDF(data []byte) (string, error) {
	reader := bytes.NewReader(data)
	pdfReader, err := model.NewPdfReader(reader)
	if err != nil {
		return "", fmt.Errorf("failed to create PDF reader: %w", err)
	}

	numPages, err := pdfReader.GetNumPages()
	if err != nil {
		return "", fmt.Errorf("failed to get number of pages: %w", err)
	}

	var fullText string
	for i := 1; i <= numPages; i++ {
		page, err := pdfReader.GetPage(i)
		if err != nil {
			continue
		}

		ex, err := extractor.New(page)
		if err != nil {
			continue
		}

		text, err := ex.ExtractText()
		if err != nil {
			continue
		}

		fullText += text + "\n"
	}

	return fullText, nil
}
```

## 4.5 Create Chunker Service

**File**: `internal/usecase/document/chunker.go`

```go
package document

import (
	"strings"
	"unicode"
)

type Chunker struct {
	chunkSize    int
	chunkOverlap int
}

func NewChunker(chunkSize, chunkOverlap int) *Chunker {
	return &Chunker{
		chunkSize:    chunkSize,
		chunkOverlap: chunkOverlap,
	}
}

func (c *Chunker) ChunkText(text string) []string {
	// Clean text
	text = strings.TrimSpace(text)
	text = cleanText(text)

	if len(text) == 0 {
		return []string{}
	}

	var chunks []string
	start := 0

	for start < len(text) {
		end := start + c.chunkSize
		if end > len(text) {
			end = len(text)
		}

		// Try to break at sentence boundary
		if end < len(text) {
			for i := end; i > start+c.chunkSize/2; i-- {
				if text[i] == '.' || text[i] == '!' || text[i] == '?' || text[i] == '\n' {
					end = i + 1
					break
				}
			}
		}

		chunk := strings.TrimSpace(text[start:end])
		if len(chunk) > 0 {
			chunks = append(chunks, chunk)
		}

		// Move start position with overlap
		start = end - c.chunkOverlap
		if start < 0 {
			start = 0
		}
	}

	return chunks
}

func cleanText(text string) string {
	// Remove excessive whitespace
	var result strings.Builder
	prevSpace := false

	for _, r := range text {
		if unicode.IsSpace(r) {
			if !prevSpace {
				result.WriteRune(' ')
				prevSpace = true
			}
		} else {
			result.WriteRune(r)
			prevSpace = false
		}
	}

	return result.String()
}
```

## 4.6 Update Document Usecase

**Update File**: `internal/usecase/document/document_usecase.go`

```go
package document

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"rag-api/internal/domain/entity"
	"rag-api/internal/domain/repository"
)

type EmbeddingService interface {
	GenerateBatchEmbeddings(ctx context.Context, texts []string) ([]interface{}, error)
}

type DocumentUsecase struct {
	docRepo      repository.DocumentRepository
	chunkRepo    repository.ChunkRepository
	embedder     EmbeddingService
	extractor    *TextExtractor
	chunker      *Chunker
}

func NewDocumentUsecase(
	docRepo repository.DocumentRepository,
	chunkRepo repository.ChunkRepository,
	embedder EmbeddingService,
	chunkSize, chunkOverlap int,
) *DocumentUsecase {
	return &DocumentUsecase{
		docRepo:   docRepo,
		chunkRepo: chunkRepo,
		embedder:  embedder,
		extractor: NewTextExtractor(),
		chunker:   NewChunker(chunkSize, chunkOverlap),
	}
}

func (uc *DocumentUsecase) UploadDocument(
	ctx context.Context,
	userID string,
	filename string,
	fileData []byte,
	mimeType string,
	visibility entity.DocumentVisibility,
) (*entity.Document, error) {
	// Create document record
	doc := &entity.Document{
		UserID:       userID,
		Filename:     fmt.Sprintf("%d-%s", time.Now().Unix(), filename),
		OriginalName: filename,
		FileSize:     int64(len(fileData)),
		MimeType:     mimeType,
		Status:       entity.StatusProcessing,
		Visibility:   visibility,
		TotalChunks:  0,
	}

	if err := uc.docRepo.Create(ctx, doc); err != nil {
		return nil, err
	}

	// Process document in background
	go func() {
		if err := uc.ProcessDocument(context.Background(), doc.ID, fileData, mimeType); err != nil {
			log.Printf("Error processing document %s: %v", doc.ID, err)
			uc.docRepo.UpdateStatus(context.Background(), doc.ID, entity.StatusFailed)
		}
	}()

	return doc, nil
}

func (uc *DocumentUsecase) ProcessDocument(
	ctx context.Context,
	documentID string,
	fileData []byte,
	mimeType string,
) error {
	// 1. Extract text
	var text string
	var err error

	if mimeType == "application/pdf" {
		text, err = uc.extractor.ExtractFromPDF(fileData)
		if err != nil {
			return fmt.Errorf("failed to extract text: %w", err)
		}
	} else {
		return fmt.Errorf("unsupported file type: %s", mimeType)
	}

	if len(text) == 0 {
		return fmt.Errorf("no text extracted from document")
	}

	// 2. Chunk text
	textChunks := uc.chunker.ChunkText(text)
	if len(textChunks) == 0 {
		return fmt.Errorf("no chunks generated")
	}

	// 3. Generate embeddings
	embeddings, err := uc.embedder.GenerateBatchEmbeddings(ctx, textChunks)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// 4. Create chunks with embeddings
	var chunks []entity.DocumentChunk
	for i, content := range textChunks {
		metadata, _ := json.Marshal(entity.ChunkMetadata{
			Source:     "text",
			PageNumber: i/10 + 1, // Rough estimate
		})

		chunks = append(chunks, entity.DocumentChunk{
			DocumentID: documentID,
			ChunkIndex: i,
			Content:    content,
			Embedding:  embeddings[i].(interface{ Slice() []float32 }),
			Metadata:   metadata,
		})
	}

	// 5. Save chunks to database
	if err := uc.chunkRepo.CreateBatch(ctx, chunks); err != nil {
		return fmt.Errorf("failed to save chunks: %w", err)
	}

	// 6. Update document status
	if err := uc.docRepo.UpdateTotalChunks(ctx, documentID, len(chunks)); err != nil {
		return err
	}

	if err := uc.docRepo.UpdateStatus(ctx, documentID, entity.StatusCompleted); err != nil {
		return err
	}

	log.Printf("✅ Document %s processed successfully: %d chunks", documentID, len(chunks))
	return nil
}

// Keep existing methods (ListDocuments, GetDocument, DeleteDocument)
func (uc *DocumentUsecase) ListDocuments(
	ctx context.Context,
	userID string,
	page, limit int,
) ([]entity.Document, int, error) {
	return uc.docRepo.List(ctx, userID, page, limit)
}

func (uc *DocumentUsecase) GetDocument(
	ctx context.Context,
	documentID, userID string,
) (*entity.Document, error) {
	return uc.docRepo.FindByIDAndUserID(ctx, documentID, userID)
}

func (uc *DocumentUsecase) DeleteDocument(
	ctx context.Context,
	documentID, userID string,
) error {
	doc, err := uc.docRepo.FindByIDAndUserID(ctx, documentID, userID)
	if err != nil {
		return err
	}
	if doc == nil {
		return fmt.Errorf("document not found")
	}

	// Delete chunks first
	if err := uc.chunkRepo.DeleteByDocumentID(ctx, documentID); err != nil {
		return err
	}

	return uc.docRepo.Delete(ctx, documentID)
}
```

## 4.7 Update Main Application

**Update File**: `cmd/api/main.go`

```go
package main

import (
	"fmt"
	"log"

	"rag-api/internal/adapter/openai"
	"rag-api/internal/adapter/repository/postgres"
	"rag-api/internal/delivery/http/handler"
	"rag-api/internal/delivery/http/middleware"
	"rag-api/internal/usecase/auth"
	"rag-api/internal/usecase/document"
	"rag-api/pkg/config"
	"rag-api/pkg/database"
	_ "rag-api/docs"
	"github.com/gofiber/fiber/v2"
	fiberSwagger "github.com/swaggo/fiber-swagger"
)

// @title           RAG API
// @version         1.0
// @description     API documentation for the RAG (Retrieval-Augmented Generation) service
// @host            localhost:8080
// @BasePath        /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.
func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("✅ Connected to database")

	// Initialize OpenAI clients
	embeddingClient := openai.NewEmbeddingClient(cfg.OpenAIKey, cfg.OpenAIEmbeddingModel)

	// Initialize repositories
	userRepo := postgres.NewUserRepository(db)
	docRepo := postgres.NewDocumentRepository(db)
	chunkRepo := postgres.NewChunkRepository(db)

	// Initialize usecases
	authUsecase := auth.NewAuthUsecase(userRepo, cfg.JWTSecret, cfg.JWTExpiration)
	docUsecase := document.NewDocumentUsecase(
		docRepo,
		chunkRepo,
		embeddingClient,
		cfg.ChunkSize,
		cfg.ChunkOverlap,
	)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authUsecase)
	docHandler := handler.NewDocumentHandler(docUsecase)

	// Setup Fiber app
	app := fiber.New()

	// Swagger route
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	// Public Routes
	api := app.Group("/api")
	api.Post("/auth/register", authHandler.Register)
	api.Post("/auth/login", authHandler.Login)

	// Protected Routes
	protected := api.Group("", middleware.JWTAuth(cfg.JWTSecret))
	protected.Get("/auth/me", authHandler.Me)

	// Document routes
	protected.Post("/documents/upload", docHandler.Upload)
	protected.Get("/documents", docHandler.List)
	protected.Get("/documents/:id", docHandler.GetByID)
	protected.Delete("/documents/:id", docHandler.Delete)

	// Start server
	log.Printf("🚀 Server starting on port %d", cfg.Port)
	log.Printf("📚 Swagger UI: http://localhost:%d/swagger/index.html", cfg.Port)
	if err := app.Listen(fmt.Sprintf(":%d", cfg.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

## 4.8 Test Document Processing

**Test: Upload PDF dan Tunggu Processing**

```bash
# Upload document
curl -X POST http://localhost:8080/api/documents/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@test.pdf" \
  -F "visibility=PRIVATE"

# Response akan dapat document ID
# {
#   "id": "abc-123",
#   "filename": "1234567890-test.pdf",
#   "status": "PROCESSING",
#   "message": "Document uploaded successfully. Processing in background."
# }

# Tunggu beberapa detik, lalu check status
curl -X GET http://localhost:8080/api/documents/abc-123 \
  -H "Authorization: Bearer $TOKEN"

# Status akan berubah menjadi "COMPLETED" dan totalChunks > 0
# {
#   "id": "abc-123",
#   "filename": "1234567890-test.pdf",
#   "status": "COMPLETED",
#   "totalChunks": 25,
#   ...
# }
```

**Check Database**:
```sql
-- Check chunks
SELECT COUNT(*) FROM document_chunks WHERE document_id = 'abc-123';

-- Check embeddings
SELECT chunk_index, LEFT(content, 50) as preview
FROM document_chunks
WHERE document_id = 'abc-123'
ORDER BY chunk_index
LIMIT 5;
```

### ✅ Checklist STEP 4

- [ ] Chunk entity & repository created
- [ ] OpenAI embedding client implemented
- [ ] Text extractor service created
- [ ] Chunker service created
- [ ] Document usecase updated dengan processing logic
- [ ] Upload PDF berhasil
- [ ] Document status berubah ke "COMPLETED"
- [ ] Chunks tersimpan di database
- [ ] Embeddings ter-generate dengan benar

---

**STEP 4 selesai! Lanjut ke STEP 5 untuk RAG Query.**
