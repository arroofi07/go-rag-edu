# 📄 STEP 3: Document Upload & Management

> Fitur untuk upload dokumen (PDF/image), simpan metadata, dan manage documents

## 3.1 Install Dependencies Tambahan

```bash
# Untuk document processing nanti
go get github.com/sashabaranov/go-openai@v1.20.0
```

## 3.2 Update Config untuk OpenAI

**Update File**: `pkg/config/config.go`

Tambahkan field OpenAI di struct Config:

```go
type Config struct {
	DatabaseURL   string
	JWTSecret     string
	JWTExpiration time.Duration
	Port          int

	// OpenAI - TAMBAHKAN INI
	OpenAIKey            string
	OpenAIEmbeddingModel string
	OpenAIChatModel      string

	// RAG Config - TAMBAHKAN INI
	ChunkSize           int
	ChunkOverlap        int
	TopKResults         int
	SimilarityThreshold float64
}
```

Update fungsi Load():

```go
func Load() *Config {
	godotenv.Load()
	jwtExp, _ := time.ParseDuration(getEnv("JWT_EXPIRATION", "168h"))

	port, err := strconv.Atoi(getEnv("PORT", "8080"))
	if err != nil {
		port = 8080
	}

	return &Config{
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		JWTSecret:     getEnv("JWT_SECRET", ""),
		JWTExpiration: jwtExp,
		Port:          port,

		// OpenAI
		OpenAIKey:            getEnv("OPENAI_API_KEY", ""),
		OpenAIEmbeddingModel: getEnv("OPENAI_EMBEDDING_MODEL", "text-embedding-3-small"),
		OpenAIChatModel:      getEnv("OPENAI_CHAT_MODEL", "gpt-4o-mini"),

		// RAG Config
		ChunkSize:           getEnvInt("CHUNK_SIZE", 1000),
		ChunkOverlap:        getEnvInt("CHUNK_OVERLAP", 200),
		TopKResults:         getEnvInt("TOP_K_RESULTS", 6),
		SimilarityThreshold: getEnvFloat("SIMILARITY_THRESHOLD", 0.5),
	}
}

// Tambahkan helper function
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

**Update .env**:
```env
# Tambahkan ini
OPENAI_API_KEY=sk-proj-your-openai-api-key-here
OPENAI_EMBEDDING_MODEL=text-embedding-3-small
OPENAI_CHAT_MODEL=gpt-4o-mini

CHUNK_SIZE=1000
CHUNK_OVERLAP=200
TOP_K_RESULTS=6
SIMILARITY_THRESHOLD=0.5
```

## 3.3 Create Document Entity & Repository

**File**: `internal/domain/entity/document.go`

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

**File**: `internal/domain/repository/document_repository.go`

```go
package repository

import (
	"context"
	"rag-api/internal/domain/entity"
)

type DocumentRepository interface {
	Create(ctx context.Context, doc *entity.Document) error
	FindByID(ctx context.Context, id string) (*entity.Document, error)
	FindByIDAndUserID(ctx context.Context, id, userID string) (*entity.Document, error)
	List(ctx context.Context, userID string, page, limit int) ([]entity.Document, int, error)
	UpdateStatus(ctx context.Context, id string, status entity.DocumentStatus) error
	UpdateTotalChunks(ctx context.Context, id string, totalChunks int) error
	Delete(ctx context.Context, id string) error
}
```

**File**: `internal/adapter/repository/postgres/document_repository.go`

```go
package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"rag-api/internal/domain/entity"
	"rag-api/internal/domain/repository"
)

type documentRepository struct {
	db *sqlx.DB
}

func NewDocumentRepository(db *sqlx.DB) repository.DocumentRepository {
	return &documentRepository{db: db}
}

func (r *documentRepository) Create(ctx context.Context, doc *entity.Document) error {
	doc.ID = uuid.New().String()
	doc.CreatedAt = time.Now()
	doc.UpdatedAt = time.Now()

	query := `
		INSERT INTO documents (
			id, user_id, filename, original_name, file_size, mime_type,
			status, total_chunks, visibility, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.ExecContext(ctx, query,
		doc.ID, doc.UserID, doc.Filename, doc.OriginalName, doc.FileSize,
		doc.MimeType, doc.Status, doc.TotalChunks, doc.Visibility,
		doc.CreatedAt, doc.UpdatedAt,
	)

	return err
}

func (r *documentRepository) FindByID(ctx context.Context, id string) (*entity.Document, error) {
	var doc entity.Document
	query := `SELECT * FROM documents WHERE id = $1`

	err := r.db.GetContext(ctx, &doc, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &doc, nil
}

func (r *documentRepository) FindByIDAndUserID(ctx context.Context, id, userID string) (*entity.Document, error) {
	var doc entity.Document
	query := `SELECT * FROM documents WHERE id = $1 AND user_id = $2`

	err := r.db.GetContext(ctx, &doc, query, id, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &doc, nil
}

func (r *documentRepository) List(ctx context.Context, userID string, page, limit int) ([]entity.Document, int, error) {
	offset := (page - 1) * limit

	// Get documents
	var docs []entity.Document
	query := `
		SELECT * FROM documents
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	err := r.db.SelectContext(ctx, &docs, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	// Get total count
	var total int
	countQuery := `SELECT COUNT(*) FROM documents WHERE user_id = $1`
	err = r.db.GetContext(ctx, &total, countQuery, userID)
	if err != nil {
		return nil, 0, err
	}

	return docs, total, nil
}

func (r *documentRepository) UpdateStatus(ctx context.Context, id string, status entity.DocumentStatus) error {
	query := `UPDATE documents SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, status, time.Now(), id)
	return err
}

func (r *documentRepository) UpdateTotalChunks(ctx context.Context, id string, totalChunks int) error {
	query := `UPDATE documents SET total_chunks = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, totalChunks, time.Now(), id)
	return err
}

func (r *documentRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM documents WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
```

## 3.4 Create Document Usecase

**File**: `internal/usecase/document/document_usecase.go`

```go
package document

import (
	"context"
	"fmt"
	"time"

	"rag-api/internal/domain/entity"
	"rag-api/internal/domain/repository"
)

type DocumentUsecase struct {
	docRepo repository.DocumentRepository
}

func NewDocumentUsecase(docRepo repository.DocumentRepository) *DocumentUsecase {
	return &DocumentUsecase{
		docRepo: docRepo,
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

	// TODO: Process document in background (STEP 4)
	// For now, just return the document

	return doc, nil
}

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
	// Verify ownership
	doc, err := uc.docRepo.FindByIDAndUserID(ctx, documentID, userID)
	if err != nil {
		return err
	}
	if doc == nil {
		return fmt.Errorf("document not found")
	}

	return uc.docRepo.Delete(ctx, documentID)
}
```

## 3.5 Create HTTP Layer

**File**: `internal/delivery/http/dto/document_dto.go`

```go
package dto

type UploadDocumentResponse struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Status   string `json:"status"`
	Message  string `json:"message"`
}

type DocumentInfo struct {
	ID           string `json:"id"`
	Filename     string `json:"filename"`
	OriginalName string `json:"originalName"`
	FileSize     int64  `json:"fileSize"`
	MimeType     string `json:"mimeType"`
	Status       string `json:"status"`
	TotalChunks  int    `json:"totalChunks"`
	Visibility   string `json:"visibility"`
	CreatedAt    string `json:"createdAt"`
}

type ListDocumentsResponse struct {
	Data []DocumentInfo `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

type PaginationMeta struct {
	Total      int `json:"total"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	TotalPages int `json:"totalPages"`
}
```

**File**: `internal/delivery/http/handler/document_handler.go`

```go
package handler

import (
	"io"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"rag-api/internal/delivery/http/dto"
	"rag-api/internal/domain/entity"
	"rag-api/internal/usecase/document"
)

type DocumentHandler struct {
	docUsecase *document.DocumentUsecase
}

func NewDocumentHandler(docUsecase *document.DocumentUsecase) *DocumentHandler {
	return &DocumentHandler{docUsecase: docUsecase}
}

// Upload godoc
// @Summary      Upload a document
// @Description  Upload a PDF or image file for processing
// @Tags         Documents
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        file        formData  file    true  "File to upload"
// @Param        visibility  formData  string  false "Visibility (PUBLIC or PRIVATE)" default(PRIVATE)
// @Success      201  {object}  dto.UploadDocumentResponse
// @Failure      400  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/documents/upload [post]
func (h *DocumentHandler) Upload(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)

	// Get file from form
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "File is required"})
	}

	// Get visibility (default: PRIVATE)
	visibility := entity.VisibilityPrivate
	if c.FormValue("visibility") == "PUBLIC" {
		visibility = entity.VisibilityPublic
	}

	// Read file data
	fileData, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read file"})
	}
	defer fileData.Close()

	buf, err := io.ReadAll(fileData)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to read file"})
	}

	// Upload document
	doc, err := h.docUsecase.UploadDocument(
		c.Context(),
		userID,
		file.Filename,
		buf,
		file.Header.Get("Content-Type"),
		visibility,
	)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(dto.UploadDocumentResponse{
		ID:       doc.ID,
		Filename: doc.Filename,
		Status:   string(doc.Status),
		Message:  "Document uploaded successfully. Processing in background.",
	})
}

// List godoc
// @Summary      List documents
// @Description  Get a list of documents for the authenticated user
// @Tags         Documents
// @Produce      json
// @Security     BearerAuth
// @Param        page   query  int  false  "Page number" default(1)
// @Param        limit  query  int  false  "Items per page" default(10)
// @Success      200  {object}  dto.ListDocumentsResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/documents [get]
func (h *DocumentHandler) List(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))

	docs, total, err := h.docUsecase.ListDocuments(c.Context(), userID, page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	// Convert to DTO
	var docInfos []dto.DocumentInfo
	for _, doc := range docs {
		docInfos = append(docInfos, dto.DocumentInfo{
			ID:           doc.ID,
			Filename:     doc.Filename,
			OriginalName: doc.OriginalName,
			FileSize:     doc.FileSize,
			MimeType:     doc.MimeType,
			Status:       string(doc.Status),
			TotalChunks:  doc.TotalChunks,
			Visibility:   string(doc.Visibility),
			CreatedAt:    doc.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	totalPages := (total + limit - 1) / limit

	return c.Status(fiber.StatusOK).JSON(dto.ListDocumentsResponse{
		Data: docInfos,
		Meta: dto.PaginationMeta{
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
		},
	})
}

// GetByID godoc
// @Summary      Get document by ID
// @Description  Get a single document's details
// @Tags         Documents
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Document ID"
// @Success      200  {object}  dto.DocumentInfo
// @Failure      404  {object}  dto.ErrorResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/documents/{id} [get]
func (h *DocumentHandler) GetByID(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	documentID := c.Params("id")

	doc, err := h.docUsecase.GetDocument(c.Context(), documentID, userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	if doc == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Document not found"})
	}

	return c.Status(fiber.StatusOK).JSON(dto.DocumentInfo{
		ID:           doc.ID,
		Filename:     doc.Filename,
		OriginalName: doc.OriginalName,
		FileSize:     doc.FileSize,
		MimeType:     doc.MimeType,
		Status:       string(doc.Status),
		TotalChunks:  doc.TotalChunks,
		Visibility:   string(doc.Visibility),
		CreatedAt:    doc.CreatedAt.Format("2006-01-02 15:04:05"),
	})
}

// Delete godoc
// @Summary      Delete a document
// @Description  Delete a document by ID
// @Tags         Documents
// @Produce      json
// @Security     BearerAuth
// @Param        id  path  string  true  "Document ID"
// @Success      200  {object}  dto.MessageResponse
// @Failure      500  {object}  dto.ErrorResponse
// @Router       /api/documents/{id} [delete]
func (h *DocumentHandler) Delete(c *fiber.Ctx) error {
	userID, _ := c.Locals("userID").(string)
	documentID := c.Params("id")

	if err := h.docUsecase.DeleteDocument(c.Context(), documentID, userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "Document deleted successfully"})
}
```

## 3.6 Update Main Application

**Update File**: `cmd/api/main.go`

```go
package main

import (
	"fmt"
	"log"

	"rag-api/internal/adapter/repository/postgres"
	"rag-api/internal/delivery/http/handler"
	"rag-api/internal/delivery/http/middleware"
	"rag-api/internal/usecase/auth"
	"rag-api/internal/usecase/document"  // TAMBAHKAN
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

	// Initialize repositories
	userRepo := postgres.NewUserRepository(db)
	docRepo := postgres.NewDocumentRepository(db)  // TAMBAHKAN

	// Initialize usecases
	authUsecase := auth.NewAuthUsecase(userRepo, cfg.JWTSecret, cfg.JWTExpiration)
	docUsecase := document.NewDocumentUsecase(docRepo)  // TAMBAHKAN

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authUsecase)
	docHandler := handler.NewDocumentHandler(docUsecase)  // TAMBAHKAN

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

	// Document routes - TAMBAHKAN
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

## 3.7 Test Document Upload

**Test 1: Upload Document**
```bash
# Login dulu untuk dapat token
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"student@test.com","password":"password123"}' \
  | jq -r '.token')

# Upload file
curl -X POST http://localhost:8080/api/documents/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@test.pdf" \
  -F "visibility=PRIVATE"
```

**Expected Response**:
```json
{
  "id": "uuid-here",
  "filename": "1234567890-test.pdf",
  "status": "PROCESSING",
  "message": "Document uploaded successfully. Processing in background."
}
```

**Test 2: List Documents**
```bash
curl -X GET "http://localhost:8080/api/documents?page=1&limit=10" \
  -H "Authorization: Bearer $TOKEN"
```

**Test 3: Get Document by ID**
```bash
curl -X GET http://localhost:8080/api/documents/{DOCUMENT_ID} \
  -H "Authorization: Bearer $TOKEN"
```

**Test 4: Delete Document**
```bash
curl -X DELETE http://localhost:8080/api/documents/{DOCUMENT_ID} \
  -H "Authorization: Bearer $TOKEN"
```

### ✅ Checklist STEP 3

- [ ] Config updated dengan OpenAI settings
- [ ] Document entity & repository created
- [ ] Document usecase implemented
- [ ] Document handler created (Fiber + Swagger annotations)
- [ ] Routes added to main.go (Fiber)
- [ ] Upload document berhasil (status 201)
- [ ] List documents berhasil (status 200)
- [ ] Get document by ID berhasil (status 200)
- [ ] Delete document berhasil (status 200)

---

**File ini berisi STEP 3. Saya akan lanjutkan dengan membuat file terpisah untuk STEP 4, 5, dan 6 agar lebih mudah dibaca.**
