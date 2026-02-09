# Guide Migrasi NestJS RAG ke Golang dengan Clean Architecture

## Ringkasan Project

Aplikasi ini adalah sistem **RAG (Retrieval-Augmented Generation)** untuk pendidikan yang memiliki fitur:
- Upload dan proses dokumen (PDF, gambar) dengan OCR
- Vector search menggunakan PostgreSQL + pgvector
- Chat AI berbasis dokumen dengan OpenAI GPT-4o-mini
- Autentikasi JWT dengan role-based access (Student, Teacher, Admin)
- Background job processing untuk ekstraksi teks dan chunking

## Tech Stack Golang yang Direkomendasikan

| NestJS | Golang Equivalent |
|--------|-------------------|
| Express/NestJS | **Gin** (HTTP framework) |
| Prisma | **sqlx** + **pgvector-go** |
| TypeORM | **GORM** (alternatif) |
| Passport JWT | **golang-jwt/jwt** |
| Bull (Redis queue) | **Asynq** |
| OpenAI SDK | **sashabaranov/go-openai** |
| Tesseract.js | **gosseract** |
| PDF-Parse | **unipdf** atau **pdfcpu** |
5g| Sharp | **imaging** atau **bimg** |
| Class-validator | **go-playground/validator** |

## Struktur Folder Clean Architecture

```
be-rag-go/
├── cmd/
│   ├── api/                    # HTTP server entry point
│   │   └── main.go
│   └── worker/                 # Background worker entry point
│       └── main.go
├── internal/
│   ├── domain/                 # Business entities & interfaces
│   │   ├── entity/
│   │   │   ├── user.go
│   │   │   ├── document.go
│   │   │   ├── document_chunk.go
│   │   │   ├── conversation.go
│   │   │   └── message.go
│   │   └── repository/         # Repository interfaces
│   │       ├── user_repository.go
│   │       ├── document_repository.go
│   │       ├── chunk_repository.go
│   │       ├── conversation_repository.go
│   │       └── message_repository.go
│   ├── usecase/                # Business logic
│   │   ├── auth/
│   │   │   └── auth_usecase.go
│   │   ├── user/
│   │   │   └── user_usecase.go
│   │   ├── document/
│   │   │   ├── document_usecase.go
│   │   │   ├── text_extractor.go
│   │   │   ├── ocr_service.go
│   │   │   └── chunker.go
│   │   ├── chat/
│   │   │   ├── chat_usecase.go
│   │   │   └── rag_service.go
│   │   └── interface.go        # Usecase interfaces
│   ├── adapter/                # Infrastructure adapters
│   │   ├── repository/         # Database implementations
│   │   │   ├── postgres/
│   │   │   │   ├── user_repository.go
│   │   │   │   ├── document_repository.go
│   │   │   │   ├── chunk_repository.go
│   │   │   │   ├── conversation_repository.go
│   │   │   │   └── message_repository.go
│   │   │   └── migrations/
│   │   │       └── 001_init.sql
│   │   ├── openai/             # OpenAI client adapter
│   │   │   ├── embedding.go
│   │   │   └── chat.go
│   │   ├── storage/            # File storage
│   │   │   └── local.go
│   │   └── queue/              # Job queue
│   │       └── asynq.go
│   └── delivery/               # Delivery mechanisms
│       ├── http/               # HTTP handlers
│       │   ├── handler/
│       │   │   ├── auth_handler.go
│       │   │   ├── user_handler.go
│       │   │   ├── document_handler.go
│       │   │   └── chat_handler.go
│       │   ├── middleware/
│       │   │   ├── auth.go
│       │   │   ├── cors.go
│       │   │   └── error.go
│       │   ├── dto/
│       │   │   ├── auth_dto.go
│       │   │   ├── document_dto.go
│       │   │   └── chat_dto.go
│       │   └── router.go
│       └── worker/             # Background job handlers
│           ├── document_processor.go
│           └── task.go
├── pkg/                        # Shared packages
│   ├── config/
│   │   └── config.go
│   ├── database/
│   │   └── postgres.go
│   ├── logger/
│   │   └── logger.go
│   ├── jwt/
│   │   └── jwt.go
│   ├── password/
│   │   └── bcrypt.go
│   └── validator/
│       └── validator.go
├── scripts/
│   └── setup_pgvector.sql
├── docker-compose.yml
├── Dockerfile
├── go.mod
├── go.sum
├── .env.example
└── README.md
```

## Dependencies yang Dibutuhkan (go.mod)

```go
module github.com/yourusername/be-rag-go

go 1.22

require (
    github.com/gin-gonic/gin v1.10.0
    github.com/jmoiron/sqlx v1.3.5
    github.com/lib/pq v1.10.9
    github.com/pgvector/pgvector-go v0.1.1
    github.com/golang-jwt/jwt/v5 v5.2.0
    github.com/joho/godotenv v1.5.1
    github.com/go-playground/validator/v10 v10.19.0
    golang.org/x/crypto v0.21.0
    github.com/sashabaranov/go-openai v1.20.0
    github.com/hibiken/asynq v0.24.1
    github.com/otiai10/gosseract/v2 v2.4.1
    github.com/unidoc/unipdf/v3 v3.55.0
    github.com/disintegration/imaging v1.6.2
    github.com/google/uuid v1.6.0
    github.com/rs/zerolog v1.32.0
)
```

## File-file Kritis dengan Code Lengkap

### 1. Domain Entities

**internal/domain/entity/user.go**
```go
package entity

import "time"

type UserRole string

const (
    RoleStudent UserRole = "STUDENT"
    RoleTeacher UserRole = "TEACHER"
    RoleAdmin   UserRole = "ADMIN"
)

type User struct {
    ID        string    `db:"id" json:"id"`
    Email     string    `db:"email" json:"email"`
    Password  string    `db:"password" json:"-"`
    Name      string    `db:"name" json:"name"`
    Major     string    `db:"major" json:"major"`
    Role      UserRole  `db:"role" json:"role"`
    CreatedAt time.Time `db:"created_at" json:"createdAt"`
    UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}
```

**internal/domain/entity/document.go**
```go
package entity

import "time"

type DocumentStatus string
type DocumentVisibility string

const (
    StatusProcessing DocumentStatus = "PROCESSING"
    StatusCompleted  DocumentStatus = "COMPLETED"
    StatusFailed     DocumentStatus = "FAILED"

    VisibilityPublic  DocumentVisibility = "PUBLIC"
    VisibilityPrivate DocumentVisibility = "PRIVATE"
)

type Document struct {
    ID           string             `db:"id" json:"id"`
    UserID       string             `db:"user_id" json:"userId"`
    Filename     string             `db:"filename" json:"filename"`
    OriginalName string             `db:"original_name" json:"originalName"`
    FileSize     int64              `db:"file_size" json:"fileSize"`
    MimeType     string             `db:"mime_type" json:"mimeType"`
    Status       DocumentStatus     `db:"status" json:"status"`
    TotalChunks  int                `db:"total_chunks" json:"totalChunks"`
    Visibility   DocumentVisibility `db:"visibility" json:"visibility"`
    CreatedAt    time.Time          `db:"created_at" json:"createdAt"`
    UpdatedAt    time.Time          `db:"updated_at" json:"updatedAt"`
}
```

**internal/domain/entity/document_chunk.go**
```go
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
    ID         string              `db:"id" json:"id"`
    DocumentID string              `db:"document_id" json:"documentId"`
    ChunkIndex int                 `db:"chunk_index" json:"chunkIndex"`
    Content    string              `db:"content" json:"content"`
    Embedding  pgvector.Vector     `db:"embedding" json:"-"`
    Metadata   ChunkMetadata       `db:"metadata" json:"metadata"`
    CreatedAt  time.Time           `db:"created_at" json:"createdAt"`
}

type SimilarChunk struct {
    DocumentChunk
    Similarity float64 `db:"similarity" json:"similarity"`
}
```

### 2. Repository Interface

**internal/domain/repository/chunk_repository.go**
```go
package repository

import (
    "context"
    "github.com/yourusername/be-rag-go/internal/domain/entity"
    "github.com/pgvector/pgvector-go"
)

type ChunkRepository interface {
    Create(ctx context.Context, chunk *entity.DocumentChunk) error
    SearchSimilar(ctx context.Context, params SimilaritySearchParams) ([]entity.SimilarChunk, error)
    DeleteByDocumentID(ctx context.Context, documentID string) error
}

type SimilaritySearchParams struct {
    UserID             string
    UserRole           entity.UserRole
    UserMajor          string
    QueryEmbedding     pgvector.Vector
    TopK               int
    SimilarityThreshold float64
}
```

### 3. Repository Implementation dengan pgvector

**internal/adapter/repository/postgres/chunk_repository.go**
```go
package postgres

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"

    "github.com/jmoiron/sqlx"
    "github.com/pgvector/pgvector-go"
    "github.com/yourusername/be-rag-go/internal/domain/entity"
    "github.com/yourusername/be-rag-go/internal/domain/repository"
)

type chunkRepository struct {
    db *sqlx.DB
}

func NewChunkRepository(db *sqlx.DB) repository.ChunkRepository {
    return &chunkRepository{db: db}
}

func (r *chunkRepository) Create(ctx context.Context, chunk *entity.DocumentChunk) error {
    query := `
        INSERT INTO document_chunks (
            id, document_id, chunk_index, content, embedding, metadata, created_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7)
    `

    metadataJSON, err := json.Marshal(chunk.Metadata)
    if err != nil {
        return err
    }

    _, err = r.db.ExecContext(ctx, query,
        chunk.ID,
        chunk.DocumentID,
        chunk.ChunkIndex,
        chunk.Content,
        chunk.Embedding,
        metadataJSON,
        chunk.CreatedAt,
    )

    return err
}

func (r *chunkRepository) SearchSimilar(
    ctx context.Context,
    params repository.SimilaritySearchParams,
) ([]entity.SimilarChunk, error) {
    var visibilityFilter string
    var args []interface{}
    argIdx := 1

    // Base query
    query := `
        SELECT
            dc.id,
            dc.document_id,
            dc.chunk_index,
            dc.content,
            dc.metadata,
            dc.created_at,
            1 - (dc.embedding <=> $%d) as similarity
        FROM document_chunks dc
        INNER JOIN documents d ON dc.document_id = d.id
        INNER JOIN users u ON d.user_id = u.id
        WHERE d.status = 'COMPLETED'
            AND (1 - (dc.embedding <=> $%d)) >= $%d
    `

    args = append(args, params.QueryEmbedding, params.SimilarityThreshold)
    argIdx = 3

    // Add visibility filter based on role
    if params.UserRole != entity.RoleAdmin {
        query += fmt.Sprintf(`
            AND (
                d.user_id = $%d
                OR (d.visibility = 'PUBLIC' AND u.major = $%d)
            )
        `, argIdx, argIdx+1)
        args = append(args, params.UserID, params.UserMajor)
        argIdx += 2
    }

    query += fmt.Sprintf(`
        ORDER BY dc.embedding <=> $1
        LIMIT $%d
    `, argIdx)
    args = append(args, params.TopK)

    // Format query with arg placeholders
    query = fmt.Sprintf(query, 1, 1, 2)

    rows, err := r.db.QueryContext(ctx, query, args...)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var chunks []entity.SimilarChunk
    for rows.Next() {
        var chunk entity.SimilarChunk
        var metadataJSON []byte

        err := rows.Scan(
            &chunk.ID,
            &chunk.DocumentID,
            &chunk.ChunkIndex,
            &chunk.Content,
            &metadataJSON,
            &chunk.CreatedAt,
            &chunk.Similarity,
        )
        if err != nil {
            return nil, err
        }

        if len(metadataJSON) > 0 {
            json.Unmarshal(metadataJSON, &chunk.Metadata)
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

### 4. Document Usecase (RAG Pipeline)

**internal/usecase/document/document_usecase.go**
```go
package document

import (
    "context"
    "fmt"
    "strings"
    "time"

    "github.com/google/uuid"
    "github.com/pgvector/pgvector-go"
    "github.com/yourusername/be-rag-go/internal/domain/entity"
    "github.com/yourusername/be-rag-go/internal/domain/repository"
)

type DocumentUsecase struct {
    docRepo      repository.DocumentRepository
    chunkRepo    repository.ChunkRepository
    embedder     EmbeddingService
    textExtractor TextExtractorService
    chunker      ChunkerService
    chunkSize    int
    chunkOverlap int
    topK         int
    simThreshold float64
}

func NewDocumentUsecase(
    docRepo repository.DocumentRepository,
    chunkRepo repository.ChunkRepository,
    embedder EmbeddingService,
    textExtractor TextExtractorService,
    chunker ChunkerService,
    cfg Config,
) *DocumentUsecase {
    return &DocumentUsecase{
        docRepo:       docRepo,
        chunkRepo:     chunkRepo,
        embedder:      embedder,
        textExtractor: textExtractor,
        chunker:       chunker,
        chunkSize:     cfg.ChunkSize,
        chunkOverlap:  cfg.ChunkOverlap,
        topK:          cfg.TopK,
        simThreshold:  cfg.SimilarityThreshold,
    }
}

type Config struct {
    ChunkSize           int
    ChunkOverlap        int
    TopK                int
    SimilarityThreshold float64
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
        ID:           uuid.New().String(),
        UserID:       userID,
        Filename:     fmt.Sprintf("%d-%s", time.Now().Unix(), filename),
        OriginalName: filename,
        FileSize:     int64(len(fileData)),
        MimeType:     mimeType,
        Status:       entity.StatusProcessing,
        Visibility:   visibility,
        TotalChunks:  0,
        CreatedAt:    time.Now(),
        UpdatedAt:    time.Now(),
    }

    if err := uc.docRepo.Create(ctx, doc); err != nil {
        return nil, err
    }

    // Process document asynchronously (enqueue to worker)
    // Worker will call ProcessDocument method

    return doc, nil
}

func (uc *DocumentUsecase) ProcessDocument(
    ctx context.Context,
    documentID string,
    fileData []byte,
    mimeType string,
) error {
    // Extract text from document
    extractedContents, err := uc.textExtractor.Extract(ctx, fileData, mimeType)
    if err != nil {
        uc.docRepo.UpdateStatus(ctx, documentID, entity.StatusFailed)
        return err
    }

    chunkIndex := 0

    // Process each extracted content
    for _, content := range extractedContents {
        // Split into chunks
        chunks := uc.chunker.Split(content.Text, uc.chunkSize, uc.chunkOverlap)

        // Generate embeddings and store
        for _, chunkText := range chunks {
            embedding, err := uc.embedder.GenerateEmbedding(ctx, chunkText)
            if err != nil {
                continue // Skip failed embeddings
            }

            chunk := &entity.DocumentChunk{
                ID:         uuid.New().String(),
                DocumentID: documentID,
                ChunkIndex: chunkIndex,
                Content:    chunkText,
                Embedding:  pgvector.NewVector(embedding),
                Metadata: entity.ChunkMetadata{
                    Source:     content.Source,
                    PageNumber: content.PageNumber,
                    Confidence: content.Confidence,
                },
                CreatedAt: time.Now(),
            }

            if err := uc.chunkRepo.Create(ctx, chunk); err != nil {
                return err
            }

            chunkIndex++
        }
    }

    // Update document status
    if err := uc.docRepo.UpdateStatus(ctx, documentID, entity.StatusCompleted); err != nil {
        return err
    }

    if err := uc.docRepo.UpdateTotalChunks(ctx, documentID, chunkIndex); err != nil {
        return err
    }

    return nil
}

func (uc *DocumentUsecase) QueryDocuments(
    ctx context.Context,
    userID string,
    userRole entity.UserRole,
    userMajor string,
    query string,
) (*QueryResult, error) {
    // Generate query embedding
    embedding, err := uc.embedder.GenerateEmbedding(ctx, query)
    if err != nil {
        return nil, err
    }

    // Search similar chunks
    chunks, err := uc.chunkRepo.SearchSimilar(ctx, repository.SimilaritySearchParams{
        UserID:              userID,
        UserRole:            userRole,
        UserMajor:           userMajor,
        QueryEmbedding:      pgvector.NewVector(embedding),
        TopK:                uc.topK,
        SimilarityThreshold: uc.simThreshold,
    })
    if err != nil {
        return nil, err
    }

    if len(chunks) == 0 {
        return &QueryResult{
            Query:   query,
            Answer:  "Mohon maaf, saya tidak menemukan informasi yang relevan.",
            Sources: []Source{},
        }, nil
    }

    // Build context
    var contextBuilder strings.Builder
    for i, chunk := range chunks {
        contextBuilder.WriteString(fmt.Sprintf("[%d] %s\n\n", i+1, chunk.Content))
    }

    return &QueryResult{
        Query:   query,
        Context: contextBuilder.String(),
        Sources: chunksToSources(chunks),
    }, nil
}

type QueryResult struct {
    Query   string   `json:"query"`
    Answer  string   `json:"answer"`
    Context string   `json:"-"`
    Sources []Source `json:"sources"`
}

type Source struct {
    DocumentID   string  `json:"documentId"`
    DocumentName string  `json:"documentName"`
    ChunkIndex   int     `json:"chunkIndex"`
    Similarity   float64 `json:"similarity"`
    Content      string  `json:"content"`
}

func chunksToSources(chunks []entity.SimilarChunk) []Source {
    sources := make([]Source, len(chunks))
    for i, chunk := range chunks {
        content := chunk.Content
        if len(content) > 200 {
            content = content[:200] + "..."
        }
        sources[i] = Source{
            DocumentID: chunk.DocumentID,
            ChunkIndex: chunk.ChunkIndex,
            Similarity: chunk.Similarity,
            Content:    content,
        }
    }
    return sources
}
```

### 5. OpenAI Adapter

**internal/adapter/openai/embedding.go**
```go
package openai

import (
    "context"

    "github.com/sashabaranov/go-openai"
)

type EmbeddingClient struct {
    client *openai.Client
    model  string
}

func NewEmbeddingClient(apiKey string, model string) *EmbeddingClient {
    return &EmbeddingClient{
        client: openai.NewClient(apiKey),
        model:  model,
    }
}

func (c *EmbeddingClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
    resp, err := c.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
        Input: []string{text},
        Model: openai.EmbeddingModel(c.model),
    })
    if err != nil {
        return nil, err
    }

    return resp.Data[0].Embedding, nil
}
```

**internal/adapter/openai/chat.go**
```go
package openai

import (
    "context"
    "fmt"

    "github.com/sashabaranov/go-openai"
)

type ChatClient struct {
    client *openai.Client
    model  string
}

func NewChatClient(apiKey string, model string) *ChatClient {
    return &ChatClient{
        client: openai.NewClient(apiKey),
        model:  model,
    }
}

func (c *ChatClient) GenerateAnswer(
    ctx context.Context,
    query string,
    context string,
    history []openai.ChatCompletionMessage,
) (string, error) {
    systemPrompt := `Anda adalah asisten AI yang membantu menjawab pertanyaan berdasarkan dokumen yang diberikan.

INSTRUKSI:
- Jawab pertanyaan berdasarkan HANYA context yang diberikan
- Jika context tidak cukup untuk menjawab, katakan "Saya tidak menemukan informasi yang cukup untuk menjawab pertanyaan ini"
- Berikan jawaban yang concise dan to-the-point
- Gunakan Bahasa Indonesia yang baik dan benar`

    messages := []openai.ChatCompletionMessage{
        {
            Role:    openai.ChatMessageRoleSystem,
            Content: systemPrompt,
        },
    }

    // Add history
    messages = append(messages, history...)

    // Add current query with context
    userContent := fmt.Sprintf("CONTEXT:\n%s\n\nPERTANYAAN:\n%s", context, query)
    messages = append(messages, openai.ChatCompletionMessage{
        Role:    openai.ChatMessageRoleUser,
        Content: userContent,
    })

    resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model:       c.model,
        Messages:    messages,
        Temperature: 0.3,
        MaxTokens:   500,
    })
    if err != nil {
        return "", err
    }

    return resp.Choices[0].Message.Content, nil
}
```

### 6. HTTP Handler

**internal/delivery/http/handler/document_handler.go**
```go
package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/yourusername/be-rag-go/internal/domain/entity"
    "github.com/yourusername/be-rag-go/internal/usecase/document"
)

type DocumentHandler struct {
    docUsecase *document.DocumentUsecase
}

func NewDocumentHandler(docUsecase *document.DocumentUsecase) *DocumentHandler {
    return &DocumentHandler{docUsecase: docUsecase}
}

func (h *DocumentHandler) Upload(c *gin.Context) {
    userID := c.GetString("userID") // From JWT middleware

    file, err := c.FormFile("file")
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
        return
    }

    visibility := entity.VisibilityPrivate
    if c.PostForm("visibility") == "PUBLIC" {
        visibility = entity.VisibilityPublic
    }

    // Read file
    fileData, err := file.Open()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
        return
    }
    defer fileData.Close()

    buf := make([]byte, file.Size)
    fileData.Read(buf)

    // Upload document
    doc, err := h.docUsecase.UploadDocument(
        c.Request.Context(),
        userID,
        file.Filename,
        buf,
        file.Header.Get("Content-Type"),
        visibility,
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusCreated, gin.H{
        "id":      doc.ID,
        "status":  doc.Status,
        "message": "Document uploaded successfully. Processing in background.",
    })
}

func (h *DocumentHandler) Query(c *gin.Context) {
    userID := c.GetString("userID")
    userRole := entity.UserRole(c.GetString("userRole"))
    userMajor := c.GetString("userMajor")

    var req struct {
        Query string `json:"query" binding:"required"`
    }

    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    result, err := h.docUsecase.QueryDocuments(
        c.Request.Context(),
        userID,
        userRole,
        userMajor,
        req.Query,
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, result)
}
```

### 7. Database Migration dengan pgvector

**internal/adapter/repository/migrations/001_init.sql**
```sql
-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- User roles enum
CREATE TYPE user_role AS ENUM ('STUDENT', 'TEACHER', 'ADMIN');
CREATE TYPE document_status AS ENUM ('PROCESSING', 'COMPLETED', 'FAILED');
CREATE TYPE document_visibility AS ENUM ('PUBLIC', 'PRIVATE');
CREATE TYPE message_role AS ENUM ('USER', 'ASSISTANT');

-- Users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    major VARCHAR(255) NOT NULL,
    role user_role DEFAULT 'STUDENT',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Documents table
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    status document_status DEFAULT 'PROCESSING',
    total_chunks INT DEFAULT 0,
    visibility document_visibility DEFAULT 'PRIVATE',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_documents_user_id ON documents(user_id);
CREATE INDEX idx_documents_status ON documents(status);

-- Document chunks with vector embeddings
CREATE TABLE document_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    chunk_index INT NOT NULL,
    content TEXT NOT NULL,
    embedding vector(1536), -- OpenAI text-embedding-3-small dimension
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(document_id, chunk_index)
);

CREATE INDEX idx_chunks_document_id ON document_chunks(document_id);

-- Create HNSW index for fast vector similarity search
CREATE INDEX ON document_chunks USING hnsw (embedding vector_cosine_ops);

-- Conversations table
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_conversations_user_id ON conversations(user_id);
CREATE INDEX idx_conversations_created_at ON conversations(created_at);

-- Messages table
CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role message_role NOT NULL,
    content TEXT NOT NULL,
    sources JSONB,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX idx_messages_created_at ON messages(created_at);
```

### 8. Main Entry Point

**cmd/api/main.go**
```go
package main

import (
    "log"

    "github.com/gin-gonic/gin"
    "github.com/jmoiron/sqlx"
    _ "github.com/lib/pq"

    "github.com/yourusername/be-rag-go/internal/adapter/openai"
    "github.com/yourusername/be-rag-go/internal/adapter/repository/postgres"
    "github.com/yourusername/be-rag-go/internal/delivery/http/handler"
    "github.com/yourusername/be-rag-go/internal/delivery/http/middleware"
    "github.com/yourusername/be-rag-go/internal/usecase/document"
    "github.com/yourusername/be-rag-go/pkg/config"
)

func main() {
    // Load config
    cfg := config.Load()

    // Connect to database
    db, err := sqlx.Connect("postgres", cfg.DatabaseURL)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()

    // Initialize repositories
    docRepo := postgres.NewDocumentRepository(db)
    chunkRepo := postgres.NewChunkRepository(db)

    // Initialize external services
    embedder := openai.NewEmbeddingClient(cfg.OpenAIKey, "text-embedding-3-small")
    chatClient := openai.NewChatClient(cfg.OpenAIKey, "gpt-4o-mini")

    // Initialize usecases
    docUsecase := document.NewDocumentUsecase(
        docRepo,
        chunkRepo,
        embedder,
        nil, // text extractor
        nil, // chunker
        document.Config{
            ChunkSize:           cfg.ChunkSize,
            ChunkOverlap:        cfg.ChunkOverlap,
            TopK:                cfg.TopK,
            SimilarityThreshold: cfg.SimilarityThreshold,
        },
    )

    // Initialize handlers
    docHandler := handler.NewDocumentHandler(docUsecase)

    // Setup router
    r := gin.Default()
    r.Use(middleware.CORS())

    // Public routes
    api := r.Group("/api")
    {
        api.POST("/auth/login", nil) // TODO: implement
        api.POST("/users", nil)       // TODO: implement
    }

    // Protected routes
    protected := api.Group("")
    protected.Use(middleware.JWTAuth(cfg.JWTSecret))
    {
        // Documents
        protected.POST("/documents/upload", docHandler.Upload)
        protected.POST("/documents/query", docHandler.Query)
        protected.GET("/documents", nil)
        protected.GET("/documents/:id", nil)
        protected.DELETE("/documents/:id", nil)

        // Chat
        protected.POST("/chat/conversations", nil)
        protected.GET("/chat/conversations", nil)
        // ... more routes
    }

    log.Printf("Server starting on port %s", cfg.Port)
    r.Run(":" + cfg.Port)
}
```

### 9. Configuration

**pkg/config/config.go**
```go
package config

import (
    "os"
    "strconv"
)

type Config struct {
    DatabaseURL         string
    JWTSecret           string
    OpenAIKey           string
    Port                string
    ChunkSize           int
    ChunkOverlap        int
    TopK                int
    SimilarityThreshold float64
}

func Load() *Config {
    return &Config{
        DatabaseURL:         getEnv("DATABASE_URL", ""),
        JWTSecret:           getEnv("JWT_SECRET", "secret"),
        OpenAIKey:           getEnv("OPENAI_API_KEY", ""),
        Port:                getEnv("PORT", "8080"),
        ChunkSize:           getEnvInt("CHUNK_SIZE", 1000),
        ChunkOverlap:        getEnvInt("CHUNK_OVERLAP", 200),
        TopK:                getEnvInt("TOP_K_RESULTS", 6),
        SimilarityThreshold: getEnvFloat("SIMILARITY_THRESHOLD", 0.5),
    }
}

func getEnv(key, defaultVal string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
    if val := os.Getenv(key); val != "" {
        if i, err := strconv.Atoi(val); err == nil {
            return i
        }
    }
    return defaultVal
}

func getEnvFloat(key string, defaultVal float64) float64 {
    if val := os.Getenv(key); val != "" {
        if f, err := strconv.ParseFloat(val, 64); err == nil {
            return f
        }
    }
    return defaultVal
}
```

### 10. Environment Variables

**.env.example**
```bash
# Database
DATABASE_URL=postgres://user:password@localhost:5432/rag_db?sslmode=disable

# JWT
JWT_SECRET=your-secret-key-change-in-production

# OpenAI
OPENAI_API_KEY=sk-your-openai-api-key
OPENAI_EMBEDDING_MODEL=text-embedding-3-small
OPENAI_CHAT_MODEL=gpt-4o-mini

# Server
PORT=8080

# RAG Configuration
CHUNK_SIZE=1000
CHUNK_OVERLAP=200
TOP_K_RESULTS=6
SIMILARITY_THRESHOLD=0.5
MAX_CONTEXT_MESSAGES=10

# Redis (for Asynq job queue)
REDIS_URL=redis://localhost:6379
```

## Implementasi Fitur Kompleks

### Text Chunker

**internal/usecase/document/chunker.go**
```go
package document

import "strings"

type ChunkerService interface {
    Split(text string, chunkSize, overlap int) []string
}

type chunker struct{}

func NewChunker() ChunkerService {
    return &chunker{}
}

func (c *chunker) Split(text string, chunkSize, overlap int) []string {
    words := strings.Fields(text)
    var chunks []string

    for i := 0; i < len(words); i += chunkSize - overlap {
        end := i + chunkSize
        if end > len(words) {
            end = len(words)
        }

        chunk := strings.Join(words[i:end], " ")
        if len(strings.TrimSpace(chunk)) > 0 {
            chunks = append(chunks, chunk)
        }

        if end >= len(words) {
            break
        }
    }

    return chunks
}
```

### Background Job Processing dengan Asynq

**internal/delivery/worker/document_processor.go**
```go
package worker

import (
    "context"
    "encoding/json"

    "github.com/hibiken/asynq"
    "github.com/yourusername/be-rag-go/internal/usecase/document"
)

const TypeDocumentProcess = "document:process"

type DocumentProcessPayload struct {
    DocumentID string `json:"documentId"`
    FileData   []byte `json:"fileData"`
    MimeType   string `json:"mimeType"`
}

type DocumentProcessor struct {
    docUsecase *document.DocumentUsecase
}

func NewDocumentProcessor(docUsecase *document.DocumentUsecase) *DocumentProcessor {
    return &DocumentProcessor{docUsecase: docUsecase}
}

func (p *DocumentProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
    var payload DocumentProcessPayload
    if err := json.Unmarshal(t.Payload(), &payload); err != nil {
        return err
    }

    return p.docUsecase.ProcessDocument(
        ctx,
        payload.DocumentID,
        payload.FileData,
        payload.MimeType,
    )
}

func NewDocumentProcessTask(documentID string, fileData []byte, mimeType string) (*asynq.Task, error) {
    payload, err := json.Marshal(DocumentProcessPayload{
        DocumentID: documentID,
        FileData:   fileData,
        MimeType:   mimeType,
    })
    if err != nil {
        return nil, err
    }
    return asynq.NewTask(TypeDocumentProcess, payload), nil
}
```

**cmd/worker/main.go**
```go
package main

import (
    "log"

    "github.com/hibiken/asynq"
    "github.com/yourusername/be-rag-go/internal/delivery/worker"
    "github.com/yourusername/be-rag-go/pkg/config"
)

func main() {
    cfg := config.Load()

    // Initialize dependencies (repos, usecases, etc.)
    // ...

    srv := asynq.NewServer(
        asynq.RedisClientOpt{Addr: cfg.RedisURL},
        asynq.Config{
            Concurrency: 10,
        },
    )

    mux := asynq.NewServeMux()
    processor := worker.NewDocumentProcessor(nil) // pass docUsecase
    mux.HandleFunc(worker.TypeDocumentProcess, processor.ProcessTask)

    log.Println("Worker starting...")
    if err := srv.Run(mux); err != nil {
        log.Fatalf("could not run server: %v", err)
    }
}
```

## Docker Compose

**docker-compose.yml**
```yaml
version: '3.8'

services:
  postgres:
    image: ankane/pgvector:latest
    environment:
      POSTGRES_USER: raguser
      POSTGRES_PASSWORD: ragpass
      POSTGRES_DB: rag_db
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  api:
    build: .
    command: /app/api
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgres://raguser:ragpass@postgres:5432/rag_db?sslmode=disable
      REDIS_URL: redis://redis:6379
    env_file:
      - .env
    depends_on:
      - postgres
      - redis

  worker:
    build: .
    command: /app/worker
    environment:
      DATABASE_URL: postgres://raguser:ragpass@postgres:5432/rag_db?sslmode=disable
      REDIS_URL: redis://redis:6379
    env_file:
      - .env
    depends_on:
      - postgres
      - redis

volumes:
  postgres_data:
```

**Dockerfile**
```dockerfile
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git tesseract-ocr tesseract-ocr-data-ind

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o worker ./cmd/worker

FROM alpine:latest

RUN apk --no-cache add ca-certificates tesseract-ocr tesseract-ocr-data-ind

WORKDIR /root/

COPY --from=builder /app/api .
COPY --from=builder /app/worker .

CMD ["./api"]
```

## Verification & Testing

### Unit Test Example

**internal/usecase/document/chunker_test.go**
```go
package document

import (
    "testing"
)

func TestChunker_Split(t *testing.T) {
    chunker := NewChunker()

    text := "word1 word2 word3 word4 word5 word6 word7 word8"
    chunks := chunker.Split(text, 3, 1)

    expected := []string{
        "word1 word2 word3",
        "word3 word4 word5",
        "word5 word6 word7",
        "word7 word8",
    }

    if len(chunks) != len(expected) {
        t.Errorf("Expected %d chunks, got %d", len(expected), len(chunks))
    }
}
```

### Integration Test

**test/integration/document_test.go**
```go
package integration

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/yourusername/be-rag-go/internal/domain/entity"
)

func TestDocumentUploadAndQuery(t *testing.T) {
    // Setup test database
    // ...

    // Upload document
    doc, err := docUsecase.UploadDocument(
        context.Background(),
        "user-123",
        "test.pdf",
        []byte("test content"),
        "application/pdf",
        entity.VisibilityPrivate,
    )

    assert.NoError(t, err)
    assert.NotEmpty(t, doc.ID)

    // Query document
    result, err := docUsecase.QueryDocuments(
        context.Background(),
        "user-123",
        entity.RoleStudent,
        "Computer Science",
        "test query",
    )

    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

## Langkah-langkah Implementasi

1. **Setup Project** (Week 1)
   - Initialize Go module
   - Setup folder structure
   - Configure Docker Compose
   - Setup PostgreSQL + pgvector

2. **Core Domain & Database** (Week 2)
   - Implement domain entities
   - Create repository interfaces
   - Implement PostgreSQL repositories
   - Run migrations

3. **Basic API** (Week 3)
   - Implement auth usecase & handler
   - Implement user management
   - Setup JWT middleware
   - Test authentication flow

4. **Document Processing** (Week 4-5)
   - Implement text extractor (PDF, images)
   - Implement OCR service
   - Implement chunker
   - Setup Asynq worker
   - Test document upload & processing

5. **RAG Implementation** (Week 6)
   - Implement OpenAI embedding client
   - Implement vector search
   - Test similarity search queries

6. **Chat Feature** (Week 7)
   - Implement chat usecase
   - Implement conversation management
   - Integrate RAG with chat
   - Test end-to-end chat flow

7. **Testing & Polish** (Week 8)
   - Write unit tests
   - Write integration tests
   - Performance optimization
   - Documentation

## Critical Files Summary

File yang **HARUS** dibuat:
1. All entity files (5 files)
2. All repository interfaces (5 files)
3. All repository implementations (5 files)
4. Migration SQL (1 file)
5. Document & Chat usecases (2 files)
6. OpenAI adapters (2 files)
7. HTTP handlers (4 files)
8. Middleware (3 files)
9. Main entry points (2 files)
10. Config & utilities (4 files)

**Total: ~35-40 file utama + tests**
