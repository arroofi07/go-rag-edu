# 🚀 Guide Implementasi RAG System - Step by Step

> **Panduan Lengkap** untuk implementasi sistem RAG dari nol dengan pendekatan **feature-by-feature**. Setiap fitur akan diimplementasi, ditest, dan diverifikasi sebelum lanjut ke fitur berikutnya.
>
> **Tech Stack**: Go Fiber v2 + Swagger (swaggo) + PostgreSQL (pgx) + JWT

---

## 📋 Daftar Fitur yang Akan Dibangun

1. ✅ **Setup Project & Database** - Foundation
2. ✅ **Authentication (Register & Login)** - Fitur pertama yang akan kita bangun
3. ⏳ **Document Upload** - Upload dan simpan metadata
4. ⏳ **Document Processing** - Extract text, chunking, embedding
5. ⏳ **Document Query (RAG)** - Similarity search + AI answer
6. ⏳ **Chat Conversation** - Conversational RAG dengan history

---

## 🎯 STEP 1: Setup Project & Database

### 1.1 Initialize Project

```bash
cd r:/RAG/be-go

# Initialize Go module
go mod init rag-api

# Install dependencies dasar
go get github.com/gofiber/fiber/v2@latest
go get github.com/jmoiron/sqlx@v1.3.5
go get github.com/jackc/pgx/v5@latest
go get github.com/pgvector/pgvector-go@v0.2.1
go get github.com/golang-jwt/jwt/v5@v5.2.0
go get golang.org/x/crypto@latest
go get github.com/joho/godotenv@v1.5.1
go get github.com/google/uuid@v1.6.0

# Swagger
go get github.com/swaggo/swag/cmd/swag@latest
go get github.com/swaggo/fiber-swagger@latest
```

### 1.2 Create .env File

**File**: `.env`

```env
# Database
DATABASE_URL=postgresql://postgres.jzgmhmmzryrsdpsmmqar:Dynamite07@aws-0-ap-southeast-1.pooler.supabase.com:6543/postgres?pgbouncer=true

# JWT
JWT_SECRET=Dynamite07
JWT_EXPIRATION=168h

# Server
PORT=8080
```

### 1.3 Create Makefile

**File**: `makefile`

```makefile
.PHONY: dev run build swagger swagger-run clean tidy test

# Hot reload development (auto-restart on file changes)
dev:
	go run github.com/air-verse/air@latest

# Run without hot reload
run:
	go run cmd/api/main.go

# Build binary
build:
	go build -o tmp/main.exe cmd/api/main.go

# Generate swagger docs
swagger:
	go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/api/main.go -o docs

# Generate swagger docs then run with hot reload
swagger-run: swagger dev

# Tidy dependencies
tidy:
	go mod tidy

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -rf tmp/
```

### 1.4 Create Database Migration

**File**: `migrations/001_init.sql`

```sql
-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Create ENUMs
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

CREATE INDEX idx_users_email ON users(email);

-- Documents table (untuk fitur selanjutnya)
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

-- Document chunks (untuk fitur selanjutnya)
CREATE TABLE document_chunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    chunk_index INT NOT NULL,
    content TEXT NOT NULL,
    embedding vector(1536),
    metadata JSONB,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(document_id, chunk_index)
);

CREATE INDEX idx_chunks_document_id ON document_chunks(document_id);
CREATE INDEX idx_chunks_embedding ON document_chunks USING hnsw (embedding vector_cosine_ops);

-- Conversations (untuk fitur chat)
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_conversations_user_id ON conversations(user_id);

-- Messages (untuk fitur chat)
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
```

**Run Migration**:
```bash
# Gunakan DIRECT_URL dari NestJS project untuk migration
psql "postgresql://postgres.jzgmhmmzryrsdpsmmqar:Dynamite07@db.jzgmhmmzryrsdpsmmqar.supabase.co:5432/postgres" -f migrations/001_init.sql
```

---

## 🔐 STEP 2: Fitur Authentication (Register & Login)

Kita akan implementasi fitur auth lengkap dari layer paling bawah (database) sampai HTTP handler, lalu test.

### 2.1 Create Core Utilities

#### **File**: `pkg/config/config.go`

```go
package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL   string
	JWTSecret     string
	JWTExpiration time.Duration
	Port          int
}

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
	}
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
```

#### **File**: `pkg/database/postgres.go`

```go
package database

import (
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
)

func Connect(databaseUrl string) (*sqlx.DB, error) {
	db, err := sqlx.Open("pgx", databaseUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
```

#### **File**: `pkg/jwt/jwt.go`

```go
package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string `json:"sub"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Major  string `json:"major"`
	jwt.RegisteredClaims
}

func GenerateToken(userID, email, role, major, secret string, expiry time.Duration) (string, error) {
	claims := &Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		Major:  major,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ValidateToken(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
```

#### **File**: `pkg/password/bcrypt.go`

```go
package password

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func ComparePassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
```

### 2.2 Create Domain Layer (Entity & Repository Interface)

#### **File**: `internal/domain/entity/user.go`

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

#### **File**: `internal/domain/repository/user_repository.go`

```go
package repository

import (
	"context"
	"rag-api/internal/domain/entity"
)

type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	FindByEmail(ctx context.Context, email string) (*entity.User, error)
	FindById(ctx context.Context, id string) (*entity.User, error)
}
```

### 2.3 Implement Repository (PostgreSQL)

#### **File**: `internal/adapter/repository/postgres/user_repository.go`

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

type userRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) repository.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *entity.User) error {
	user.ID = uuid.New().String()
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	query := `
		INSERT INTO users (id, email, password, name, major, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := r.db.ExecContext(ctx, query,
		user.ID, user.Email, user.Password, user.Name,
		user.Major, user.Role, user.CreatedAt, user.UpdatedAt,
	)

	return err
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*entity.User, error) {
	var user entity.User
	query := `SELECT * FROM users WHERE email = $1`

	err := r.db.GetContext(ctx, &user, query, email)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *userRepository) FindById(ctx context.Context, id string) (*entity.User, error) {
	var user entity.User
	query := `SELECT * FROM users WHERE id = $1`

	err := r.db.GetContext(ctx, &user, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}
```

### 2.4 Create Auth Usecase

#### **File**: `internal/usecase/auth/auth_usecase.go`

```go
package auth

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"rag-api/internal/domain/entity"
	"rag-api/internal/domain/repository"
	"rag-api/pkg/jwt"
	"rag-api/pkg/password"
)

type AuthUsecase struct {
	userRepo  repository.UserRepository
	jwtSecret string
	jwtExpiry time.Duration
}

func NewAuthUsecase(
	userRepo repository.UserRepository,
	jwtSecret string,
	jwtExpiry time.Duration,
) *AuthUsecase {
	return &AuthUsecase{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
		jwtExpiry: jwtExpiry,
	}
}

// register user
func (uc *AuthUsecase) Register(
	ctx context.Context,
	email, pass, name, major string,
	role entity.UserRole,
) (*entity.User, error) {
	// Validate input
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || pass == "" || name == "" || major == "" {
		return nil, errors.New("all fields are required")
	}

	// Check if email already exists
	existing, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if existing != nil && err == nil {
		return nil, errors.New("email already registered")
	}

	// Hash password
	hashedPassword, err := password.HashPassword(pass)
	if err != nil {
		return nil, err
	}

	// Create user
	user := &entity.User{
		Email:    email,
		Password: hashedPassword,
		Name:     name,
		Major:    major,
		Role:     role,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

// login user
func (uc *AuthUsecase) Login(
	ctx context.Context,
	email, pass string,
) (string, *entity.User, error) {
	// Validate input
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || pass == "" {
		return "", nil, errors.New("email and password are required")
	}

	// Find user
	user, err := uc.userRepo.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil, errors.New("invalid credentials")
		}
		return "", nil, err
	}
	if user == nil {
		return "", nil, errors.New("invalid credentials")
	}

	// Verify password
	if err := password.ComparePassword(user.Password, pass); err != nil {
		return "", nil, errors.New("invalid credentials")
	}

	// Generate JWT token
	token, err := jwt.GenerateToken(
		user.ID,
		user.Email,
		string(user.Role),
		user.Major,
		uc.jwtSecret,
		uc.jwtExpiry,
	)
	if err != nil {
		return "", nil, err
	}

	return token, user, nil
}

// get user info
func (uc *AuthUsecase) GetUserInfo(
	ctx context.Context,
	userID string,
) (*entity.User, error) {
	return uc.userRepo.FindById(ctx, userID)
}
```

### 2.5 Create HTTP Layer (DTO, Handler, Middleware)

#### **File**: `internal/delivery/http/dto/auth_dto.go`

```go
package dto

// tipe data untuk request register
type RegisterRequest struct {
	Email    string `json:"email" binding:"required" example:"user@example.com"`
	Password string `json:"password" binding:"required" example:"password123"`
	Name     string `json:"name" binding:"required" example:"John Doe"`
	Major    string `json:"major"  example:"Computer Science"`
	Role     string `json:"role" example:"STUDENT" enums:"STUDENT,TEACHER,ADMIN"`
}

// tipe data untuk request login
type LoginRequest struct {
	Email    string `json:"email" binding:"required" example:"user@example.com"`
	Password string `json:"password" binding:"required" example:"password123"`
}

// tipe data untuk response login
type AuthResponse struct {
	AccessToken string   `json:"access_token" example:"eyJhbGciOiJIUzI1NiIs..."`
	User        UserInfo `json:"user"`
}

// tipe data untuk response user info
type UserInfo struct {
	ID    string `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Email string `json:"email" example:"user@example.com"`
	Name  string `json:"name" example:"John Doe"`
	Major string `json:"major" example:"Computer Science"`
	Role  string `json:"role" example:"STUDENT"`
}

// generic response
type MessageResponse struct {
	Message string `json:"message" example:"Operation successful"`
}

// error response
type ErrorResponse struct {
	Error string `json:"error" example:"Something went wrong"`
}

// register success response
type RegisterSuccessResponse struct {
	Message string   `json:"message" example:"User registered successfully"`
	User    UserInfo `json:"user"`
}

// login success response
type LoginSuccessResponse struct {
	Message string   `json:"message" example:"User logged in successfully"`
	Token   string   `json:"token" example:"eyJhbGciOiJIUzI1NiIs..."`
	User    UserInfo `json:"user"`
}

// me response
type MeResponse struct {
	User UserInfo `json:"user"`
}
```

#### **File**: `internal/delivery/http/handler/auth_handler.go`

```go
package handler

import (
	"rag-api/internal/delivery/http/dto"
	"rag-api/internal/domain/entity"
	"rag-api/internal/usecase/auth"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	authUsecase *auth.AuthUsecase
}

func NewAuthHandler(authUsecase *auth.AuthUsecase) *AuthHandler {
	return &AuthHandler{authUsecase: authUsecase}
}

// Register godoc
// @Summary      Register a new user
// @Description  Create a new user account with email, password, name, major, and role
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.RegisterRequest        true  "Register Request"
// @Success      200      {object}  dto.RegisterSuccessResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /api/auth/register [post]
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	user, err := h.authUsecase.Register(
		c.Context(),
		req.Email,
		req.Password,
		req.Name,
		req.Major,
		entity.UserRole(req.Role),
	)

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "User registered successfully", "user": dto.UserInfo{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
		Major: user.Major,
		Role:  string(user.Role),
	}})
}

// Login godoc
// @Summary      Login user
// @Description  Authenticate a user with email and password, returns a JWT token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        request  body      dto.LoginRequest         true  "Login Request"
// @Success      200      {object}  dto.LoginSuccessResponse
// @Failure      400      {object}  dto.ErrorResponse
// @Failure      401      {object}  dto.ErrorResponse
// @Router       /api/auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	token, user, err := h.authUsecase.Login(
		c.Context(),
		req.Email,
		req.Password,
	)

	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"message": "User logged in successfully", "token": token, "user": dto.UserInfo{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
		Major: user.Major,
		Role:  string(user.Role),
	}})
}

// Me godoc
// @Summary      Get user info
// @Description  Get current authenticated user info
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Success      200      {object}  dto.MeResponse
// @Failure      401      {object}  dto.ErrorResponse
// @Failure      500      {object}  dto.ErrorResponse
// @Router       /api/auth/me [get]
func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID, ok := c.Locals("userID").(string)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid user session"})
	}

	user, err := h.authUsecase.GetUserInfo(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{"user": dto.UserInfo{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
		Major: user.Major,
		Role:  string(user.Role),
	}})
}
```

#### **File**: `internal/delivery/http/middleware/auth.go`

```go
package middleware

import (
	"strings"

	"rag-api/pkg/jwt"

	"github.com/gofiber/fiber/v2"
)

func JWTAuth(secret string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Authorization header required"})
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid authorization header format"})
		}

		tokenString := parts[1]

		// Validate token
		claims, err := jwt.ValidateToken(tokenString, secret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
		}

		// Set user info to context via Locals
		c.Locals("userID", claims.UserID)
		c.Locals("email", claims.Email)
		c.Locals("role", claims.Role)
		c.Locals("major", claims.Major)

		return c.Next()
	}
}
```

### 2.6 Create Main Application

#### **File**: `cmd/api/main.go`

```go
package main

import (
	"fmt"
	"log"

	"rag-api/internal/adapter/repository/postgres"
	"rag-api/internal/delivery/http/handler"
	"rag-api/internal/delivery/http/middleware"
	"rag-api/internal/usecase/auth"
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

	// connect to database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("connected to database")

	// initialize repository
	userRepo := postgres.NewUserRepository(db)

	// initialize usecase
	authUsecase := auth.NewAuthUsecase(userRepo, cfg.JWTSecret, cfg.JWTExpiration)

	// initialize handler
	authHandler := handler.NewAuthHandler(authUsecase)

	// initialize fiber app
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

	// Start server
	log.Printf("🚀 Server starting on port %d", cfg.Port)
	log.Printf("📚 Swagger UI: http://localhost:%d/swagger/index.html", cfg.Port)
	if err := app.Listen(fmt.Sprintf(":%d", cfg.Port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

### 2.7 Generate Swagger & Test

**Generate swagger docs**:
```bash
make swagger
```

**Run server**:
```bash
make swagger-run
```

**Swagger UI**: Buka `http://localhost:8080/swagger/index.html` di browser.

**Test 1: Register User**
```bash
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "student@test.com",
    "password": "password123",
    "name": "Test Student",
    "major": "Computer Science",
    "role": "STUDENT"
  }'
```

**Expected Response**:
```json
{
  "message": "User registered successfully",
  "user": {
    "id": "uuid-here",
    "email": "student@test.com",
    "name": "Test Student",
    "major": "Computer Science",
    "role": "STUDENT"
  }
}
```

**Test 2: Login**
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "student@test.com",
    "password": "password123"
  }'
```

**Expected Response**:
```json
{
  "message": "User logged in successfully",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": "uuid-here",
    "email": "student@test.com",
    "name": "Test Student",
    "major": "Computer Science",
    "role": "STUDENT"
  }
}
```

**Test 3: Access Protected Route**
```bash
# Copy token dari response login
curl -X GET http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

**Expected Response**:
```json
{
  "user": {
    "id": "uuid-here",
    "email": "student@test.com",
    "name": "Test Student",
    "major": "Computer Science",
    "role": "STUDENT"
  }
}
```

### ✅ Checklist Fitur Authentication

- [ ] Database migration berhasil
- [ ] Swagger docs ter-generate (`make swagger`)
- [ ] Server bisa running tanpa error
- [ ] Swagger UI bisa diakses di `/swagger/index.html`
- [ ] Register user berhasil (status 200)
- [ ] Login berhasil dan dapat JWT token (status 200)
- [ ] Access protected route `/api/auth/me` dengan token berhasil (status 200)
- [ ] Access protected route tanpa token ditolak (status 401)
- [ ] Login dengan password salah ditolak (status 401)
- [ ] Register dengan email yang sudah ada ditolak

**Jika semua test di atas berhasil, fitur Authentication sudah selesai! ✅**

---

## 📄 STEP 3: Fitur Document Upload (Coming Next)

Setelah fitur authentication selesai dan berhasil ditest, kita akan lanjut ke fitur document upload. Fitur ini akan mencakup:

1. Upload file (PDF/image)
2. Simpan metadata ke database
3. Return document info

**File yang akan dibuat**:
- `internal/domain/entity/document.go`
- `internal/domain/repository/document_repository.go`
- `internal/adapter/repository/postgres/document_repository.go`
- `internal/usecase/document/document_usecase.go`
- `internal/delivery/http/dto/document_dto.go`
- `internal/delivery/http/handler/document_handler.go`

**Apakah Anda ingin saya lanjutkan dengan STEP 3 sekarang, atau Anda ingin test STEP 2 (Authentication) dulu?**

---

## 📚 Reference

### Struktur Project
```
be-go/
├── cmd/api/main.go                          # ✅ Sudah dibuat
├── docs/                                    # ✅ Swagger docs (auto-generated)
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── pkg/
│   ├── config/config.go                     # ✅ Sudah dibuat
│   ├── database/postgres.go                 # ✅ Sudah dibuat
│   ├── jwt/jwt.go                          # ✅ Sudah dibuat
│   └── password/bcrypt.go                  # ✅ Sudah dibuat
├── internal/
│   ├── domain/
│   │   ├── entity/user.go                  # ✅ Sudah dibuat
│   │   └── repository/user_repository.go   # ✅ Sudah dibuat
│   ├── adapter/repository/postgres/
│   │   └── user_repository.go              # ✅ Sudah dibuat
│   ├── usecase/auth/auth_usecase.go        # ✅ Sudah dibuat
│   └── delivery/http/
│       ├── dto/auth_dto.go                 # ✅ Sudah dibuat
│       ├── handler/auth_handler.go         # ✅ Sudah dibuat
│       └── middleware/auth.go              # ✅ Sudah dibuat
├── migrations/001_init.sql                  # ✅ Sudah dibuat
├── makefile                                 # ✅ Sudah dibuat
├── .env                                     # ✅ Sudah dibuat
└── go.mod                                   # ✅ Sudah dibuat
```

### Environment Variables
```env
DATABASE_URL=postgresql://...
JWT_SECRET=your-secret
JWT_EXPIRATION=168h
PORT=8080
```

### API Endpoints (Sudah Tersedia)
- `POST /api/auth/register` - Register user baru
- `POST /api/auth/login` - Login dan dapatkan JWT token
- `GET /api/auth/me` - Get user info (protected)
- `GET /swagger/*` - Swagger UI documentation
