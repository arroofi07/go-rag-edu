# 📚 RAG System Implementation Guide - Complete

> **Panduan Lengkap** implementasi sistem RAG (Retrieval-Augmented Generation) dari nol dengan Golang

---

## 📖 Daftar Guide

Ikuti guide ini secara berurutan. Setiap step harus selesai dan ditest sebelum lanjut ke step berikutnya.

### **STEP 1 & 2: Setup Project & Authentication**
📄 File: [`STEP_1_2_SETUP_AND_AUTH.md`](./STEP_1_2_SETUP_AND_AUTH.md)

**Isi:**
- Setup project & dependencies
- Database migration
- Core utilities (config, database, JWT, password)
- User entity & repository
- Auth usecase (register & login)
- Auth handler & middleware
- Testing auth endpoints

**Output:**
- ✅ Server bisa running
- ✅ Register user berhasil
- ✅ Login berhasil dapat JWT token
- ✅ Protected routes berfungsi

---

### **STEP 3: Document Upload**
📄 File: [`STEP_3_DOCUMENT_UPLOAD.md`](./STEP_3_DOCUMENT_UPLOAD.md)

**Isi:**
- Document entity & repository
- Document usecase (upload, list, get, delete)
- Document handler & DTO
- File upload handling

**Output:**
- ✅ Upload dokumen berhasil
- ✅ List dokumen berhasil
- ✅ Get & delete dokumen berhasil

---

### **STEP 4: Document Processing**
📄 File: [`STEP_4_DOCUMENT_PROCESSING.md`](./STEP_4_DOCUMENT_PROCESSING.md)

**Isi:**
- Document chunk entity & repository
- OpenAI embedding client
- Text extractor service (PDF)
- Chunker service
- Background processing logic
- Batch embedding generation

**Output:**
- ✅ PDF text extraction berhasil
- ✅ Text chunking berhasil
- ✅ Embeddings ter-generate
- ✅ Chunks tersimpan di database
- ✅ Document status update ke COMPLETED

---

### **STEP 5: RAG Query**
📄 File: [`STEP_5_RAG_QUERY.md`](./STEP_5_RAG_QUERY.md)

**Isi:**
- OpenAI chat client
- Similarity search implementation
- Query usecase dengan RAG
- Query handler & DTO
- Context building dari chunks

**Output:**
- ✅ Query dokumen berhasil
- ✅ Similarity search berfungsi
- ✅ AI answer generation berhasil
- ✅ Sources ditampilkan dengan similarity score

---

### **STEP 6: Chat Conversation**
📄 File: [`STEP_6_CHAT_CONVERSATION.md`](./STEP_6_CHAT_CONVERSATION.md)

**Isi:**
- Conversation & message entities
- Conversation & message repositories
- Chat usecase dengan history support
- Chat handler & DTO
- Greeting detection
- Conversational RAG

**Output:**
- ✅ Create conversation berhasil
- ✅ Send message dengan RAG berhasil
- ✅ Conversation history tersimpan
- ✅ List & get conversations berhasil
- ✅ Delete conversation berhasil

---

## 🎯 Cara Menggunakan Guide Ini

### 1. **Mulai dari STEP 1 & 2**
```bash
# Baca file
cat STEP_1_2_SETUP_AND_AUTH.md

# Ikuti semua instruksi
# Test setiap endpoint
# Pastikan semua checklist ✅
```

### 2. **Lanjut ke STEP 3**
```bash
cat STEP_3_DOCUMENT_UPLOAD.md
# Implementasi
# Test
```

### 3. **Lanjut ke STEP 4, 5, 6**
Ulangi proses yang sama untuk setiap step.

---

## 📁 Struktur Project Akhir

```
be-go/
├── cmd/
│   └── api/
│       └── main.go                          # ✅ Entry point
├── internal/
│   ├── domain/
│   │   ├── entity/
│   │   │   ├── user.go                      # ✅ STEP 2
│   │   │   ├── document.go                  # ✅ STEP 3
│   │   │   ├── document_chunk.go            # ✅ STEP 4
│   │   │   ├── conversation.go              # ✅ STEP 6
│   │   │   └── message.go                   # ✅ STEP 6
│   │   └── repository/
│   │       ├── user_repository.go           # ✅ STEP 2
│   │       ├── document_repository.go       # ✅ STEP 3
│   │       ├── chunk_repository.go          # ✅ STEP 4
│   │       ├── conversation_repository.go   # ✅ STEP 6
│   │       └── message_repository.go        # ✅ STEP 6
│   ├── adapter/
│   │   ├── repository/postgres/
│   │   │   ├── user_repository.go           # ✅ STEP 2
│   │   │   ├── document_repository.go       # ✅ STEP 3
│   │   │   ├── chunk_repository.go          # ✅ STEP 4
│   │   │   ├── conversation_repository.go   # ✅ STEP 6
│   │   │   └── message_repository.go        # ✅ STEP 6
│   │   └── openai/
│   │       ├── embedding.go                 # ✅ STEP 4
│   │       └── chat.go                      # ✅ STEP 5
│   ├── usecase/
│   │   ├── auth/
│   │   │   └── auth_usecase.go              # ✅ STEP 2
│   │   ├── document/
│   │   │   ├── document_usecase.go          # ✅ STEP 3, 4, 5
│   │   │   ├── text_extractor.go            # ✅ STEP 4
│   │   │   └── chunker.go                   # ✅ STEP 4
│   │   └── chat/
│   │       └── chat_usecase.go              # ✅ STEP 6
│   └── delivery/http/
│       ├── dto/
│       │   ├── auth_dto.go                  # ✅ STEP 2
│       │   ├── document_dto.go              # ✅ STEP 3, 5
│       │   └── chat_dto.go                  # ✅ STEP 6
│       ├── handler/
│       │   ├── auth_handler.go              # ✅ STEP 2
│       │   ├── document_handler.go          # ✅ STEP 3, 5
│       │   └── chat_handler.go              # ✅ STEP 6
│       └── middleware/
│           └── auth.go                      # ✅ STEP 2
├── pkg/
│   ├── config/
│   │   └── config.go                        # ✅ STEP 2, updated STEP 3
│   ├── database/
│   │   └── postgres.go                      # ✅ STEP 2
│   ├── jwt/
│   │   └── jwt.go                           # ✅ STEP 2
│   └── password/
│       └── bcrypt.go                        # ✅ STEP 2
├── migrations/
│   └── 001_init.sql                         # ✅ STEP 1
├── .env                                      # ✅ STEP 1
├── go.mod                                    # ✅ STEP 1
└── go.sum                                    # ✅ Auto-generated
```

---

## 🚀 API Endpoints (Setelah Semua Step Selesai)

### **Authentication**
- `POST /api/auth/register` - Register user baru
- `POST /api/auth/login` - Login dan dapatkan JWT token
- `GET /api/auth/me` - Get user info (protected)

### **Documents**
- `POST /api/documents/upload` - Upload dokumen (PDF)
- `GET /api/documents` - List semua dokumen user
- `GET /api/documents/:id` - Get detail dokumen
- `DELETE /api/documents/:id` - Delete dokumen
- `POST /api/documents/query` - Query dokumen dengan RAG

### **Chat**
- `POST /api/chat/conversations` - Create conversation baru
- `POST /api/chat/conversations/:id/messages` - Send message
- `GET /api/chat/conversations` - List semua conversations
- `GET /api/chat/conversations/:id` - Get conversation detail
- `DELETE /api/chat/conversations/:id` - Delete conversation

---

## 🧪 Testing Flow

### 1. **Test Auth (STEP 2)**
```bash
# Register
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"password123","name":"Test","major":"CS","role":"STUDENT"}'

# Login
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@test.com","password":"password123"}'
```

### 2. **Test Document Upload (STEP 3)**
```bash
TOKEN="your-jwt-token"

curl -X POST http://localhost:8080/api/documents/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@test.pdf" \
  -F "visibility=PRIVATE"
```

### 3. **Test RAG Query (STEP 5)**
```bash
curl -X POST http://localhost:8080/api/documents/query \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query":"Apa itu machine learning?"}'
```

### 4. **Test Chat (STEP 6)**
```bash
curl -X POST http://localhost:8080/api/chat/conversations \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message":"Halo!"}'
```

---

## 📝 Environment Variables

```env
# Database
DATABASE_URL=postgresql://user:pass@host:port/db

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRATION=168h

# OpenAI
OPENAI_API_KEY=sk-...
OPENAI_EMBEDDING_MODEL=text-embedding-3-small
OPENAI_CHAT_MODEL=gpt-4o-mini

# Server
PORT=8080

# RAG Config
CHUNK_SIZE=1000
CHUNK_OVERLAP=200
TOP_K_RESULTS=6
SIMILARITY_THRESHOLD=0.5
```

---

## 🎓 Tips Implementasi

1. **Ikuti urutan step** - Jangan skip step
2. **Test setiap step** - Pastikan berfungsi sebelum lanjut
3. **Baca error message** - Go error messages sangat jelas
4. **Check database** - Verifikasi data tersimpan dengan benar
5. **Use Postman/Insomnia** - Lebih mudah untuk testing API

---

## 🐛 Common Issues

### Issue 1: Database connection failed
```bash
# Check connection string di .env
# Pastikan PostgreSQL running
# Test connection: psql $DATABASE_URL
```

### Issue 2: pgvector not found
```sql
-- Run di PostgreSQL
CREATE EXTENSION IF NOT EXISTS vector;
```

### Issue 3: OpenAI API error
```bash
# Check API key di .env
# Pastikan ada credit di OpenAI account
```

---

## 🎉 Setelah Selesai

Setelah semua step selesai, Anda akan punya:
- ✅ Complete RAG system
- ✅ JWT authentication
- ✅ Document processing dengan embeddings
- ✅ Similarity search dengan pgvector
- ✅ AI-powered chat dengan conversation history
- ✅ Clean Architecture implementation

**Next Steps:**
- Add OCR support untuk images
- Add file storage (S3/local)
- Add rate limiting
- Add logging & monitoring
- Add unit tests
- Dockerize application

---

**Selamat coding! 🚀**
