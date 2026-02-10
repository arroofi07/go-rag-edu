# 💬 STEP 6: Chat Conversation (Conversational RAG)

> Fitur untuk chat dengan AI menggunakan RAG dan conversation history

## 6.1 Create Conversation & Message Repositories

**File**: `internal/domain/repository/conversation_repository.go`

```go
package repository

import (
	"context"
	"rag-api/internal/domain/entity"
)

type ConversationRepository interface {
	Create(ctx context.Context, conv *entity.Conversation) error
	FindByID(ctx context.Context, id string) (*entity.Conversation, error)
	FindByIDAndUserID(ctx context.Context, id, userID string) (*entity.Conversation, error)
	List(ctx context.Context, userID string, page, limit int) ([]entity.Conversation, int, error)
	Update(ctx context.Context, conv *entity.Conversation) error
	Delete(ctx context.Context, id string) error
}
```

**File**: `internal/domain/repository/message_repository.go`

```go
package repository

import (
	"context"
	"rag-api/internal/domain/entity"
)

type MessageRepository interface {
	Create(ctx context.Context, msg *entity.Message) error
	ListByConversation(ctx context.Context, conversationID string, limit int) ([]entity.Message, error)
}
```

**File**: `internal/adapter/repository/postgres/conversation_repository.go`

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

type conversationRepository struct {
	db *sqlx.DB
}

func NewConversationRepository(db *sqlx.DB) repository.ConversationRepository {
	return &conversationRepository{db: db}
}

func (r *conversationRepository) Create(ctx context.Context, conv *entity.Conversation) error {
	conv.ID = uuid.New().String()
	conv.CreatedAt = time.Now()
	conv.UpdatedAt = time.Now()

	query := `
		INSERT INTO conversations (id, user_id, title, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(ctx, query,
		conv.ID, conv.UserID, conv.Title, conv.CreatedAt, conv.UpdatedAt,
	)

	return err
}

func (r *conversationRepository) FindByID(ctx context.Context, id string) (*entity.Conversation, error) {
	var conv entity.Conversation
	query := `SELECT * FROM conversations WHERE id = $1`

	err := r.db.GetContext(ctx, &conv, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &conv, nil
}

func (r *conversationRepository) FindByIDAndUserID(ctx context.Context, id, userID string) (*entity.Conversation, error) {
	var conv entity.Conversation
	query := `SELECT * FROM conversations WHERE id = $1 AND user_id = $2`

	err := r.db.GetContext(ctx, &conv, query, id, userID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &conv, nil
}

func (r *conversationRepository) List(ctx context.Context, userID string, page, limit int) ([]entity.Conversation, int, error) {
	offset := (page - 1) * limit

	var convs []entity.Conversation
	query := `
		SELECT * FROM conversations 
		WHERE user_id = $1 
		ORDER BY updated_at DESC 
		LIMIT $2 OFFSET $3
	`
	err := r.db.SelectContext(ctx, &convs, query, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM conversations WHERE user_id = $1`
	err = r.db.GetContext(ctx, &total, countQuery, userID)
	if err != nil {
		return nil, 0, err
	}

	return convs, total, nil
}

func (r *conversationRepository) Update(ctx context.Context, conv *entity.Conversation) error {
	conv.UpdatedAt = time.Now()

	query := `UPDATE conversations SET title = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, conv.Title, conv.UpdatedAt, conv.ID)
	return err
}

func (r *conversationRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM conversations WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
```

**File**: `internal/adapter/repository/postgres/message_repository.go`

```go
package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"rag-api/internal/domain/entity"
	"rag-api/internal/domain/repository"
)

type messageRepository struct {
	db *sqlx.DB
}

func NewMessageRepository(db *sqlx.DB) repository.MessageRepository {
	return &messageRepository{db: db}
}

func (r *messageRepository) Create(ctx context.Context, msg *entity.Message) error {
	msg.ID = uuid.New().String()
	msg.CreatedAt = time.Now()

	query := `
		INSERT INTO messages (id, conversation_id, role, content, sources, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query,
		msg.ID, msg.ConversationID, msg.Role, msg.Content,
		msg.Sources, msg.Metadata, msg.CreatedAt,
	)

	return err
}

func (r *messageRepository) ListByConversation(ctx context.Context, conversationID string, limit int) ([]entity.Message, error) {
	var messages []entity.Message
	query := `
		SELECT * FROM messages 
		WHERE conversation_id = $1 
		ORDER BY created_at ASC
		LIMIT $2
	`

	err := r.db.SelectContext(ctx, &messages, query, conversationID, limit)
	if err != nil {
		return nil, err
	}

	return messages, nil
}
```

## 6.2 Update OpenAI Chat Client

**Update File**: `internal/adapter/openai/chat.go`

Tambahkan method untuk chat dengan history:

```go
func (c *ChatClient) GenerateAnswerWithHistory(
	ctx context.Context,
	query string,
	context string,
	history []openai.ChatCompletionMessage,
) (string, error) {
	systemPrompt := `Anda adalah asisten AI yang membantu menjawab pertanyaan berdasarkan dokumen yang diberikan.

Instruksi:
1. Jawab pertanyaan HANYA berdasarkan konteks yang diberikan
2. Gunakan riwayat percakapan untuk memberikan jawaban yang lebih kontekstual
3. Jika informasi tidak ada dalam konteks, katakan "Maaf, saya tidak menemukan informasi tersebut dalam dokumen"
4. Berikan jawaban yang jelas, ringkas, dan terstruktur
5. Gunakan bahasa Indonesia yang baik dan benar`

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
	}

	// Add conversation history
	messages = append(messages, history...)

	// Add current query with context
	userPrompt := fmt.Sprintf(`Konteks dari dokumen:
%s

Pertanyaan: %s`, context, query)

	messages = append(messages, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: userPrompt,
	})

	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:       c.model,
		Messages:    messages,
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

## 6.3 Create Chat Usecase

**File**: `internal/usecase/chat/chat_usecase.go`

```go
package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	"rag-api/internal/domain/entity"
	"rag-api/internal/domain/repository"
)

type EmbeddingService interface {
	GenerateEmbedding(ctx context.Context, text string) (interface{}, error)
}

type ChatService interface {
	GenerateAnswerWithHistory(ctx context.Context, query, context string, history []openai.ChatCompletionMessage) (string, error)
}

type ChatUsecase struct {
	convRepo  repository.ConversationRepository
	msgRepo   repository.MessageRepository
	chunkRepo repository.ChunkRepository
	embedder  EmbeddingService
	chatSvc   ChatService
	topK      int
	threshold float64
}

func NewChatUsecase(
	convRepo repository.ConversationRepository,
	msgRepo repository.MessageRepository,
	chunkRepo repository.ChunkRepository,
	embedder EmbeddingService,
	chatSvc ChatService,
	topK int,
	threshold float64,
) *ChatUsecase {
	return &ChatUsecase{
		convRepo:  convRepo,
		msgRepo:   msgRepo,
		chunkRepo: chunkRepo,
		embedder:  embedder,
		chatSvc:   chatSvc,
		topK:      topK,
		threshold: threshold,
	}
}

func (uc *ChatUsecase) CreateConversation(
	ctx context.Context,
	userID, message string,
) (*entity.Conversation, *entity.Message, *entity.Message, error) {
	// Create conversation with auto-generated title
	title := generateTitle(message)
	conv := &entity.Conversation{
		UserID: userID,
		Title:  title,
	}

	if err := uc.convRepo.Create(ctx, conv); err != nil {
		return nil, nil, nil, err
	}

	// Process first message
	userMsg, assistantMsg, err := uc.processMessage(ctx, conv.ID, message, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	return conv, userMsg, assistantMsg, nil
}

func (uc *ChatUsecase) SendMessage(
	ctx context.Context,
	conversationID, userID, message string,
) (*entity.Message, *entity.Message, error) {
	// Verify conversation ownership
	conv, err := uc.convRepo.FindByIDAndUserID(ctx, conversationID, userID)
	if err != nil {
		return nil, nil, err
	}
	if conv == nil {
		return nil, nil, fmt.Errorf("conversation not found")
	}

	// Get conversation history
	history, err := uc.msgRepo.ListByConversation(ctx, conversationID, 10)
	if err != nil {
		return nil, nil, err
	}

	// Process message
	userMsg, assistantMsg, err := uc.processMessage(ctx, conversationID, message, history)
	if err != nil {
		return nil, nil, err
	}

	return userMsg, assistantMsg, nil
}

func (uc *ChatUsecase) processMessage(
	ctx context.Context,
	conversationID, message string,
	history []entity.Message,
) (*entity.Message, *entity.Message, error) {
	// 1. Save user message
	userMsg := &entity.Message{
		ConversationID: conversationID,
		Role:           entity.MessageRoleUser,
		Content:        message,
	}
	if err := uc.msgRepo.Create(ctx, userMsg); err != nil {
		return nil, nil, err
	}

	// 2. Check if greeting
	if isGreeting(message) {
		assistantMsg := &entity.Message{
			ConversationID: conversationID,
			Role:           entity.MessageRoleAssistant,
			Content:        "Halo! Saya siap membantu Anda. Silakan tanyakan apa saja tentang dokumen yang telah Anda upload.",
		}
		if err := uc.msgRepo.Create(ctx, assistantMsg); err != nil {
			return nil, nil, err
		}
		return userMsg, assistantMsg, nil
	}

	// 3. Generate embedding for query
	queryEmbedding, err := uc.embedder.GenerateEmbedding(ctx, message)
	if err != nil {
		return nil, nil, err
	}

	// 4. Search similar chunks
	chunks, err := uc.chunkRepo.SearchSimilar(ctx, queryEmbedding, uc.topK, uc.threshold)
	if err != nil {
		return nil, nil, err
	}

	// 5. Build context
	var contextBuilder strings.Builder
	var sources []map[string]interface{}

	for i, chunk := range chunks {
		contextBuilder.WriteString(fmt.Sprintf("[Dokumen %d]\n%s\n\n", i+1, chunk.Content))
		sources = append(sources, map[string]interface{}{
			"documentId": chunk.DocumentID,
			"chunkIndex": chunk.ChunkIndex,
			"similarity": chunk.Similarity,
		})
	}

	// 6. Convert history to OpenAI format
	var chatHistory []openai.ChatCompletionMessage
	for _, msg := range history {
		role := openai.ChatMessageRoleUser
		if msg.Role == entity.MessageRoleAssistant {
			role = openai.ChatMessageRoleAssistant
		}
		chatHistory = append(chatHistory, openai.ChatCompletionMessage{
			Role:    role,
			Content: msg.Content,
		})
	}

	// 7. Generate answer
	var answer string
	if len(chunks) == 0 {
		answer = "Maaf, saya tidak menemukan informasi yang relevan dalam dokumen untuk menjawab pertanyaan Anda."
	} else {
		answer, err = uc.chatSvc.GenerateAnswerWithHistory(ctx, message, contextBuilder.String(), chatHistory)
		if err != nil {
			return nil, nil, err
		}
	}

	// 8. Save assistant message
	sourcesJSON, _ := json.Marshal(sources)
	assistantMsg := &entity.Message{
		ConversationID: conversationID,
		Role:           entity.MessageRoleAssistant,
		Content:        answer,
		Sources:        sourcesJSON,
	}
	if err := uc.msgRepo.Create(ctx, assistantMsg); err != nil {
		return nil, nil, err
	}

	return userMsg, assistantMsg, nil
}

func (uc *ChatUsecase) ListConversations(
	ctx context.Context,
	userID string,
	page, limit int,
) ([]entity.Conversation, int, error) {
	return uc.convRepo.List(ctx, userID, page, limit)
}

func (uc *ChatUsecase) GetConversation(
	ctx context.Context,
	conversationID, userID string,
) (*entity.Conversation, []entity.Message, error) {
	conv, err := uc.convRepo.FindByIDAndUserID(ctx, conversationID, userID)
	if err != nil {
		return nil, nil, err
	}
	if conv == nil {
		return nil, nil, fmt.Errorf("conversation not found")
	}

	messages, err := uc.msgRepo.ListByConversation(ctx, conversationID, 100)
	if err != nil {
		return nil, nil, err
	}

	return conv, messages, nil
}

func (uc *ChatUsecase) DeleteConversation(
	ctx context.Context,
	conversationID, userID string,
) error {
	conv, err := uc.convRepo.FindByIDAndUserID(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if conv == nil {
		return fmt.Errorf("conversation not found")
	}

	return uc.convRepo.Delete(ctx, conversationID)
}

// Helper functions
func generateTitle(message string) string {
	if len(message) > 50 {
		return message[:50] + "..."
	}
	return message
}

func isGreeting(message string) bool {
	greetings := []string{"halo", "hai", "hello", "hi", "selamat"}
	lower := strings.ToLower(message)
	for _, greeting := range greetings {
		if strings.Contains(lower, greeting) && len(message) < 20 {
			return true
		}
	}
	return false
}
```

## 6.4 Create Chat DTO & Handler

**File**: `internal/delivery/http/dto/chat_dto.go`

```go
package dto

type CreateConversationRequest struct {
	Message string `json:"message" binding:"required"`
}

type SendMessageRequest struct {
	Message string `json:"message" binding:"required"`
}

type ChatResponse struct {
	ConversationID   string          `json:"conversationId"`
	UserMessage      MessageResponse `json:"userMessage"`
	AssistantMessage MessageResponse `json:"assistantMessage"`
}

type MessageResponse struct {
	ID      string                   `json:"id"`
	Role    string                   `json:"role"`
	Content string                   `json:"content"`
	Sources []map[string]interface{} `json:"sources,omitempty"`
}

type ConversationInfo struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type ConversationDetail struct {
	Conversation ConversationInfo  `json:"conversation"`
	Messages     []MessageResponse `json:"messages"`
}
```

**File**: `internal/delivery/http/handler/chat_handler.go`

```go
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"rag-api/internal/delivery/http/dto"
	"rag-api/internal/usecase/chat"
)

type ChatHandler struct {
	chatUsecase *chat.ChatUsecase
}

func NewChatHandler(chatUsecase *chat.ChatUsecase) *ChatHandler {
	return &ChatHandler{chatUsecase: chatUsecase}
}

func (h *ChatHandler) CreateConversation(c *gin.Context) {
	userID := c.GetString("userID")

	var req dto.CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	conv, userMsg, assistantMsg, err := h.chatUsecase.CreateConversation(c.Request.Context(), userID, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var sources []map[string]interface{}
	if len(assistantMsg.Sources) > 0 {
		json.Unmarshal(assistantMsg.Sources, &sources)
	}

	c.JSON(http.StatusCreated, dto.ChatResponse{
		ConversationID: conv.ID,
		UserMessage: dto.MessageResponse{
			ID:      userMsg.ID,
			Role:    string(userMsg.Role),
			Content: userMsg.Content,
		},
		AssistantMessage: dto.MessageResponse{
			ID:      assistantMsg.ID,
			Role:    string(assistantMsg.Role),
			Content: assistantMsg.Content,
			Sources: sources,
		},
	})
}

func (h *ChatHandler) SendMessage(c *gin.Context) {
	userID := c.GetString("userID")
	conversationID := c.Param("id")

	var req dto.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userMsg, assistantMsg, err := h.chatUsecase.SendMessage(c.Request.Context(), conversationID, userID, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var sources []map[string]interface{}
	if len(assistantMsg.Sources) > 0 {
		json.Unmarshal(assistantMsg.Sources, &sources)
	}

	c.JSON(http.StatusOK, dto.ChatResponse{
		ConversationID: conversationID,
		UserMessage: dto.MessageResponse{
			ID:      userMsg.ID,
			Role:    string(userMsg.Role),
			Content: userMsg.Content,
		},
		AssistantMessage: dto.MessageResponse{
			ID:      assistantMsg.ID,
			Role:    string(assistantMsg.Role),
			Content: assistantMsg.Content,
			Sources: sources,
		},
	})
}

func (h *ChatHandler) ListConversations(c *gin.Context) {
	userID := c.GetString("userID")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	convs, total, err := h.chatUsecase.ListConversations(c.Request.Context(), userID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var convInfos []dto.ConversationInfo
	for _, conv := range convs {
		convInfos = append(convInfos, dto.ConversationInfo{
			ID:        conv.ID,
			Title:     conv.Title,
			CreatedAt: conv.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: conv.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": convInfos,
		"meta": gin.H{
			"total": total,
			"page":  page,
			"limit": limit,
		},
	})
}

func (h *ChatHandler) GetConversation(c *gin.Context) {
	userID := c.GetString("userID")
	conversationID := c.Param("id")

	conv, messages, err := h.chatUsecase.GetConversation(c.Request.Context(), conversationID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var msgResponses []dto.MessageResponse
	for _, msg := range messages {
		var sources []map[string]interface{}
		if len(msg.Sources) > 0 {
			json.Unmarshal(msg.Sources, &sources)
		}

		msgResponses = append(msgResponses, dto.MessageResponse{
			ID:      msg.ID,
			Role:    string(msg.Role),
			Content: msg.Content,
			Sources: sources,
		})
	}

	c.JSON(http.StatusOK, dto.ConversationDetail{
		Conversation: dto.ConversationInfo{
			ID:        conv.ID,
			Title:     conv.Title,
			CreatedAt: conv.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt: conv.UpdatedAt.Format("2006-01-02 15:04:05"),
		},
		Messages: msgResponses,
	})
}

func (h *ChatHandler) DeleteConversation(c *gin.Context) {
	userID := c.GetString("userID")
	conversationID := c.Param("id")

	if err := h.chatUsecase.DeleteConversation(c.Request.Context(), conversationID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Conversation deleted successfully"})
}
```

## 6.5 Update Main Application (Final)

**Update File**: `cmd/api/main.go`

```go
package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"rag-api/internal/adapter/openai"
	"rag-api/internal/adapter/repository/postgres"
	"rag-api/internal/delivery/http/handler"
	"rag-api/internal/delivery/http/middleware"
	"rag-api/internal/usecase/auth"
	"rag-api/internal/usecase/chat"
	"rag-api/internal/usecase/document"
	"rag-api/pkg/config"
	"rag-api/pkg/database"
)

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
	chatClient := openai.NewChatClient(cfg.OpenAIKey, cfg.OpenAIChatModel)

	// Initialize repositories
	userRepo := postgres.NewUserRepository(db)
	docRepo := postgres.NewDocumentRepository(db)
	chunkRepo := postgres.NewChunkRepository(db)
	convRepo := postgres.NewConversationRepository(db)
	msgRepo := postgres.NewMessageRepository(db)

	// Initialize usecases
	authUsecase := auth.NewAuthUsecase(userRepo, cfg.JWTSecret, cfg.JWTExpiration)
	docUsecase := document.NewDocumentUsecase(
		docRepo,
		chunkRepo,
		embeddingClient,
		chatClient,
		cfg.ChunkSize,
		cfg.ChunkOverlap,
		cfg.TopKResults,
		cfg.SimilarityThreshold,
	)
	chatUsecase := chat.NewChatUsecase(
		convRepo,
		msgRepo,
		chunkRepo,
		embeddingClient,
		chatClient,
		cfg.TopKResults,
		cfg.SimilarityThreshold,
	)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authUsecase)
	docHandler := handler.NewDocumentHandler(docUsecase)
	chatHandler := handler.NewChatHandler(chatUsecase)

	// Setup router
	r := gin.Default()

	api := r.Group("/api")
	{
		// Public routes
		api.POST("/auth/register", authHandler.Register)
		api.POST("/auth/login", authHandler.Login)
	}

	// Protected routes
	protected := api.Group("")
	protected.Use(middleware.JWTAuth(cfg.JWTSecret))
	{
		// Auth
		protected.GET("/auth/me", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"userID": c.GetString("userID"),
				"email":  c.GetString("email"),
				"role":   c.GetString("role"),
				"major":  c.GetString("major"),
			})
		})

		// Documents
		protected.POST("/documents/upload", docHandler.Upload)
		protected.GET("/documents", docHandler.List)
		protected.GET("/documents/:id", docHandler.GetByID)
		protected.DELETE("/documents/:id", docHandler.Delete)
		protected.POST("/documents/query", docHandler.Query)

		// Chat
		protected.POST("/chat/conversations", chatHandler.CreateConversation)
		protected.POST("/chat/conversations/:id/messages", chatHandler.SendMessage)
		protected.GET("/chat/conversations", chatHandler.ListConversations)
		protected.GET("/chat/conversations/:id", chatHandler.GetConversation)
		protected.DELETE("/chat/conversations/:id", chatHandler.DeleteConversation)
	}

	log.Printf("🚀 Server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

## 6.6 Test Chat Conversation

**Test 1: Create Conversation**

```bash
curl -X POST http://localhost:8080/api/chat/conversations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Halo!"
  }'
```

**Expected Response**:
```json
{
  "conversationId": "conv-123",
  "userMessage": {
    "id": "msg-1",
    "role": "USER",
    "content": "Halo!"
  },
  "assistantMessage": {
    "id": "msg-2",
    "role": "ASSISTANT",
    "content": "Halo! Saya siap membantu Anda. Silakan tanyakan apa saja tentang dokumen yang telah Anda upload."
  }
}
```

**Test 2: Send Message (RAG Query)**

```bash
curl -X POST http://localhost:8080/api/chat/conversations/conv-123/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Apa itu machine learning?"
  }'
```

**Test 3: List Conversations**

```bash
curl -X GET "http://localhost:8080/api/chat/conversations?page=1&limit=10" \
  -H "Authorization: Bearer $TOKEN"
```

**Test 4: Get Conversation Detail**

```bash
curl -X GET http://localhost:8080/api/chat/conversations/conv-123 \
  -H "Authorization: Bearer $TOKEN"
```

**Test 5: Delete Conversation**

```bash
curl -X DELETE http://localhost:8080/api/chat/conversations/conv-123 \
  -H "Authorization: Bearer $TOKEN"
```

### ✅ Checklist STEP 6

- [ ] Conversation & message repositories created
- [ ] Chat usecase implemented dengan history support
- [ ] Chat DTO & handler created
- [ ] Routes added to main.go
- [ ] Create conversation berhasil (status 201)
- [ ] Send message berhasil (status 200)
- [ ] List conversations berhasil (status 200)
- [ ] Get conversation detail berhasil (status 200)
- [ ] Delete conversation berhasil (status 200)
- [ ] Greeting detection works
- [ ] RAG integration works dengan history
- [ ] Sources ditampilkan di response

---

## 🎉 SEMUA STEP SELESAI!

Anda sekarang memiliki sistem RAG lengkap dengan:
- ✅ Authentication (JWT)
- ✅ Document Upload
- ✅ Document Processing (text extraction, chunking, embedding)
- ✅ RAG Query (similarity search + AI answer)
- ✅ Chat Conversation (conversational RAG dengan history)

### API Endpoints Lengkap

**Auth**
- `POST /api/auth/register`
- `POST /api/auth/login`
- `GET /api/auth/me`

**Documents**
- `POST /api/documents/upload`
- `GET /api/documents`
- `GET /api/documents/:id`
- `DELETE /api/documents/:id`
- `POST /api/documents/query`

**Chat**
- `POST /api/chat/conversations`
- `POST /api/chat/conversations/:id/messages`
- `GET /api/chat/conversations`
- `GET /api/chat/conversations/:id`
- `DELETE /api/chat/conversations/:id`

### Next Steps (Optional Improvements)

1. **Add OCR Support** - Process images dengan Tesseract
2. **Add File Storage** - Simpan file ke S3/local storage
3. **Add Rate Limiting** - Protect API dari abuse
4. **Add Logging** - Structured logging dengan zerolog
5. **Add Tests** - Unit tests dan integration tests
6. **Add Docker** - Containerize application
7. **Add CI/CD** - Automated deployment
