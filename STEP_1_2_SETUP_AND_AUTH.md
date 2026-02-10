# 🚀 Guide Implementasi RAG System - Step by Step

> **Panduan Lengkap** untuk implementasi sistem RAG dari nol dengan pendekatan **feature-by-feature**. Setiap fitur akan diimplementasi, ditest, dan diverifikasi sebelum lanjut ke fitur berikutnya.

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
go get github.com/gin-gonic/gin@v1.10.0
go get github.com/jmoiron/sqlx@v1.3.5
go get github.com/lib/pq@v1.10.9
go get github.com/pgvector/pgvector-go@v0.3.0
go get github.com/golang-jwt/jwt/v5@v5.2.0
go get golang.org/x/crypto@v0.21.0
go get github.com/joho/godotenv@v1.5.1
go get github.com/go-playground/validator/v10@v10.19.0
go get github.com/google/uuid@v1.6.0
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

### 1.3 Create Database Migration

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
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL   string
	JWTSecret     string
	JWTExpiration time.Duration
	Port          string
}

func Load() *Config {
	godotenv.Load()

	jwtExp, _ := time.ParseDuration(getEnv("JWT_EXPIRATION", "168h"))

	return &Config{
		DatabaseURL:   getEnv("DATABASE_URL", ""),
		JWTSecret:     getEnv("JWT_SECRET", "secret"),
		JWTExpiration: jwtExp,
		Port:          getEnv("PORT", "8080"),
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

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func Connect(databaseURL string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
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
	claims := Claims{
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
	FindByID(ctx context.Context, id string) (*entity.User, error)
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

func (r *userRepository) FindByID(ctx context.Context, id string) (*entity.User, error) {
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
	if err != nil {
		return nil, err
	}
	if existing != nil {
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
```

### 2.5 Create HTTP Layer (DTO, Handler, Middleware)

#### **File**: `internal/delivery/http/dto/auth_dto.go`

```go
package dto

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name" binding:"required"`
	Major    string `json:"major" binding:"required"`
	Role     string `json:"role" binding:"required,oneof=STUDENT TEACHER ADMIN"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	AccessToken string   `json:"access_token"`
	User        UserInfo `json:"user"`
}

type UserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Major string `json:"major"`
	Role  string `json:"role"`
}
```

#### **File**: `internal/delivery/http/handler/auth_handler.go`

```go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"rag-api/internal/delivery/http/dto"
	"rag-api/internal/domain/entity"
	"rag-api/internal/usecase/auth"
)

type AuthHandler struct {
	authUsecase *auth.AuthUsecase
}

func NewAuthHandler(authUsecase *auth.AuthUsecase) *AuthHandler {
	return &AuthHandler{authUsecase: authUsecase}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.authUsecase.Register(
		c.Request.Context(),
		req.Email,
		req.Password,
		req.Name,
		req.Major,
		entity.UserRole(req.Role),
	)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user": dto.UserInfo{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
			Major: user.Major,
			Role:  string(user.Role),
		},
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	token, user, err := h.authUsecase.Login(
		c.Request.Context(),
		req.Email,
		req.Password,
	)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, dto.AuthResponse{
		AccessToken: token,
		User: dto.UserInfo{
			ID:    user.ID,
			Email: user.Email,
			Name:  user.Name,
			Major: user.Major,
			Role:  string(user.Role),
		},
	})
}
```

#### **File**: `internal/delivery/http/middleware/auth.go`

```go
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"rag-api/pkg/jwt"
)

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := jwt.ValidateToken(tokenString, secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Set user info to context
		c.Set("userID", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Set("major", claims.Major)

		c.Next()
	}
}
```

### 2.6 Create Main Application

#### **File**: `cmd/api/main.go`

```go
package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"rag-api/internal/adapter/repository/postgres"
	"rag-api/internal/delivery/http/handler"
	"rag-api/internal/delivery/http/middleware"
	"rag-api/internal/usecase/auth"
	"rag-api/pkg/config"
	"rag-api/pkg/database"
)

func main() {
	// Load config
	cfg := config.Load()

	// Connect to database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("✅ Connected to database")

	// Initialize repositories
	userRepo := postgres.NewUserRepository(db)

	// Initialize usecases
	authUsecase := auth.NewAuthUsecase(userRepo, cfg.JWTSecret, cfg.JWTExpiration)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authUsecase)

	// Setup router
	r := gin.Default()

	// Public routes
	api := r.Group("/api")
	{
		api.POST("/auth/register", authHandler.Register)
		api.POST("/auth/login", authHandler.Login)
	}

	// Protected routes (untuk testing JWT middleware)
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
	}

	// Start server
	log.Printf("🚀 Server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

### 2.7 Test Fitur Authentication

**Run server**:
```bash
go run cmd/api/main.go
```

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
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
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
  "userID": "uuid-here",
  "email": "student@test.com",
  "role": "STUDENT",
  "major": "Computer Science"
}
```

### ✅ Checklist Fitur Authentication

- [ ] Database migration berhasil
- [ ] Server bisa running tanpa error
- [ ] Register user berhasil (status 201)
- [ ] Login berhasil dan dapat JWT token (status 200)
- [ ] Access protected route dengan token berhasil (status 200)
- [ ] Access protected route tanpa token ditolak (status 401)
- [ ] Login dengan password salah ditolak (status 401)
- [ ] Register dengan email yang sudah ada ditolak (status 400)

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
