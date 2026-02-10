# 🔍 STEP 5: Document Query (RAG - Retrieval Augmented Generation)

> Fitur untuk query dokumen dengan RAG: similarity search + AI answer generation

## 5.1 Create OpenAI Chat Client

**File**: `internal/adapter/openai/chat.go`

```go
package openai

import (
	"context"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

type ChatClient struct {
	client *openai.Client
	model  string
}

func NewChatClient(apiKey, model string) *ChatClient {
	return &ChatClient{
		client: openai.NewClient(apiKey),
		model:  model,
	}
}

func (c *ChatClient) GenerateAnswer(
	ctx context.Context,
	query string,
	context string,
) (string, error) {
	systemPrompt := `Anda adalah asisten AI yang membantu menjawab pertanyaan berdasarkan dokumen yang diberikan.

Instruksi:
1. Jawab pertanyaan HANYA berdasarkan konteks yang diberikan
2. Jika informasi tidak ada dalam konteks, katakan "Maaf, saya tidak menemukan informasi tersebut dalam dokumen"
3. Berikan jawaban yang jelas, ringkas, dan terstruktur
4. Gunakan bahasa Indonesia yang baik dan benar`

	userPrompt := fmt.Sprintf(`Konteks dari dokumen:
%s

Pertanyaan: %s

Jawaban:`, context, query)

	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: systemPrompt,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: userPrompt,
			},
		},
		Temperature: 0.7,
		MaxTokens:   500,
	})

	if err != nil {
		return "", fmt.Errorf("failed to generate answer: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return resp.Choices[0].Message.Content, nil
}
```

## 5.2 Add Query Method to Document Usecase

**Update File**: `internal/usecase/document/document_usecase.go`

Tambahkan interface dan method baru:

```go
type ChatService interface {
	GenerateAnswer(ctx context.Context, query, context string) (string, error)
}

// Update struct
type DocumentUsecase struct {
	docRepo      repository.DocumentRepository
	chunkRepo    repository.ChunkRepository
	embedder     EmbeddingService
	chatService  ChatService  // TAMBAHKAN
	extractor    *TextExtractor
	chunker      *Chunker
	topK         int
	threshold    float64
}

// Update constructor
func NewDocumentUsecase(
	docRepo repository.DocumentRepository,
	chunkRepo repository.ChunkRepository,
	embedder EmbeddingService,
	chatService ChatService,  // TAMBAHKAN
	chunkSize, chunkOverlap int,
	topK int,
	threshold float64,
) *DocumentUsecase {
	return &DocumentUsecase{
		docRepo:     docRepo,
		chunkRepo:   chunkRepo,
		embedder:    embedder,
		chatService: chatService,  // TAMBAHKAN
		extractor:   NewTextExtractor(),
		chunker:     NewChunker(chunkSize, chunkOverlap),
		topK:        topK,
		threshold:   threshold,
	}
}

// TAMBAHKAN method baru
func (uc *DocumentUsecase) QueryDocuments(
	ctx context.Context,
	query string,
) (string, []entity.SimilarChunk, error) {
	// 1. Generate embedding untuk query
	queryEmbedding, err := uc.embedder.GenerateEmbedding(ctx, query)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// 2. Search similar chunks
	chunks, err := uc.chunkRepo.SearchSimilar(ctx, queryEmbedding, uc.topK, uc.threshold)
	if err != nil {
		return "", nil, fmt.Errorf("failed to search similar chunks: %w", err)
	}

	if len(chunks) == 0 {
		return "Maaf, saya tidak menemukan informasi yang relevan dalam dokumen.", nil, nil
	}

	// 3. Build context from chunks
	var contextBuilder strings.Builder
	for i, chunk := range chunks {
		contextBuilder.WriteString(fmt.Sprintf("[Dokumen %d - Similarity: %.2f]\n%s\n\n",
			i+1, chunk.Similarity, chunk.Content))
	}

	// 4. Generate answer using LLM
	answer, err := uc.chatService.GenerateAnswer(ctx, query, contextBuilder.String())
	if err != nil {
		return "", chunks, fmt.Errorf("failed to generate answer: %w", err)
	}

	return answer, chunks, nil
}
```

## 5.3 Update Document DTO

**Update File**: `internal/delivery/http/dto/document_dto.go`

Tambahkan DTO untuk query:

```go
type QueryDocumentRequest struct {
	Query string `json:"query" binding:"required"`
}

type QueryDocumentResponse struct {
	Query   string        `json:"query"`
	Answer  string        `json:"answer"`
	Sources []ChunkSource `json:"sources"`
}

type ChunkSource struct {
	DocumentID string  `json:"documentId"`
	Content    string  `json:"content"`
	Similarity float64 `json:"similarity"`
	ChunkIndex int     `json:"chunkIndex"`
}
```

## 5.4 Update Document Handler

**Update File**: `internal/delivery/http/handler/document_handler.go`

Tambahkan method Query:

```go
func (h *DocumentHandler) Query(c *gin.Context) {
	var req dto.QueryDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	answer, chunks, err := h.docUsecase.QueryDocuments(c.Request.Context(), req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Convert chunks to sources
	var sources []dto.ChunkSource
	for _, chunk := range chunks {
		sources = append(sources, dto.ChunkSource{
			DocumentID: chunk.DocumentID,
			Content:    chunk.Content,
			Similarity: chunk.Similarity,
			ChunkIndex: chunk.ChunkIndex,
		})
	}

	c.JSON(http.StatusOK, dto.QueryDocumentResponse{
		Query:   req.Query,
		Answer:  answer,
		Sources: sources,
	})
}
```

## 5.5 Update Main Application

**Update File**: `cmd/api/main.go`

```go
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
	chatClient := openai.NewChatClient(cfg.OpenAIKey, cfg.OpenAIChatModel)  // TAMBAHKAN

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
		chatClient,  // TAMBAHKAN
		cfg.ChunkSize,
		cfg.ChunkOverlap,
		cfg.TopKResults,
		cfg.SimilarityThreshold,
	)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authUsecase)
	docHandler := handler.NewDocumentHandler(docUsecase)

	// Setup router
	r := gin.Default()

	api := r.Group("/api")
	{
		api.POST("/auth/register", authHandler.Register)
		api.POST("/auth/login", authHandler.Login)
	}

	protected := api.Group("")
	protected.Use(middleware.JWTAuth(cfg.JWTSecret))
	{
		protected.GET("/auth/me", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"userID": c.GetString("userID"),
				"email":  c.GetString("email"),
				"role":   c.GetString("role"),
				"major":  c.GetString("major"),
			})
		})

		protected.POST("/documents/upload", docHandler.Upload)
		protected.GET("/documents", docHandler.List)
		protected.GET("/documents/:id", docHandler.GetByID)
		protected.DELETE("/documents/:id", docHandler.Delete)
		protected.POST("/documents/query", docHandler.Query)  // TAMBAHKAN
	}

	log.Printf("🚀 Server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

## 5.6 Fix Embedding Interface

**Update File**: `internal/adapter/openai/embedding.go`

Tambahkan method GenerateEmbedding (singular):

```go
func (c *EmbeddingClient) GenerateEmbedding(ctx context.Context, text string) (pgvector.Vector, error) {
	resp, err := c.client.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.EmbeddingModel(c.model),
	})
	if err != nil {
		return pgvector.Vector{}, err
	}

	if len(resp.Data) == 0 {
		return pgvector.Vector{}, fmt.Errorf("no embedding returned")
	}

	embedding := make([]float32, len(resp.Data[0].Embedding))
	for i, v := range resp.Data[0].Embedding {
		embedding[i] = v
	}

	return pgvector.NewVector(embedding), nil
}
```

## 5.7 Test RAG Query

**Test 1: Query Documents**

```bash
# Login dan upload document dulu (jika belum)
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"student@test.com","password":"password123"}' \
  | jq -r '.access_token')

# Query documents
curl -X POST http://localhost:8080/api/documents/query \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Apa itu machine learning?"
  }'
```

**Expected Response**:
```json
{
  "query": "Apa itu machine learning?",
  "answer": "Machine learning adalah cabang dari artificial intelligence yang memungkinkan komputer untuk belajar dari data tanpa diprogram secara eksplisit. Sistem machine learning menggunakan algoritma untuk mengidentifikasi pola dalam data dan membuat prediksi atau keputusan berdasarkan pola tersebut.",
  "sources": [
    {
      "documentId": "abc-123",
      "content": "Machine learning merupakan subset dari AI yang fokus pada pengembangan sistem yang dapat belajar dan meningkatkan performa mereka dari pengalaman...",
      "similarity": 0.89,
      "chunkIndex": 5
    },
    {
      "documentId": "abc-123",
      "content": "Dalam machine learning, model dilatih menggunakan data historis untuk membuat prediksi pada data baru...",
      "similarity": 0.85,
      "chunkIndex": 6
    }
  ]
}
```

**Test 2: Query dengan Pertanyaan yang Tidak Ada di Dokumen**

```bash
curl -X POST http://localhost:8080/api/documents/query \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "query": "Siapa presiden Indonesia?"
  }'
```

**Expected Response**:
```json
{
  "query": "Siapa presiden Indonesia?",
  "answer": "Maaf, saya tidak menemukan informasi tersebut dalam dokumen.",
  "sources": []
}
```

### ✅ Checklist STEP 5

- [ ] OpenAI chat client created
- [ ] QueryDocuments method implemented
- [ ] Query DTO created
- [ ] Query handler implemented
- [ ] Route added to main.go
- [ ] Query berhasil return answer (status 200)
- [ ] Sources/chunks ditampilkan dengan similarity score
- [ ] Answer relevan dengan query
- [ ] Handling untuk query yang tidak ada di dokumen

---

**STEP 5 selesai! Lanjut ke STEP 6 untuk Chat Conversation.**
