-- Drop existing tables if they exist (be careful with this in production!)
DROP TABLE IF EXISTS "document_chunks" CASCADE;
DROP TABLE IF EXISTS "documents" CASCADE;
DROP TABLE IF EXISTS "users" CASCADE;

-- Drop existing types if they exist
DROP TYPE IF EXISTS "UserRole" CASCADE;
DROP TYPE IF EXISTS "DocumentStatus" CASCADE;

-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- Create UserRole enum
CREATE TYPE "UserRole" AS ENUM ('STUDENT', 'TEACHER', 'ADMIN');

-- Create DocumentStatus enum
CREATE TYPE "DocumentStatus" AS ENUM ('PROCESSING', 'COMPLETED', 'FAILED');

-- Create DocumentVsibility enum
CREATE TYPE "DocumentVisibility" AS ENUM ('PUBLIC', 'PRIVATE');

-- Create users table
CREATE TABLE "users" (
    "id" TEXT NOT NULL,
    "email" TEXT NOT NULL,
    "password" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "role" "UserRole" NOT NULL DEFAULT 'STUDENT',
    "major" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "users_pkey" PRIMARY KEY ("id")
);

-- Create documents table
CREATE TABLE "documents" (
    "id" TEXT NOT NULL,
    "userId" TEXT NOT NULL,
    "filename" TEXT NOT NULL,
    "originalName" TEXT NOT NULL,
    "fileSize" INTEGER NOT NULL,
    "mimeType" TEXT NOT NULL,
    "status" "DocumentStatus" NOT NULL DEFAULT 'PROCESSING',
    "totalChunks" INTEGER NOT NULL DEFAULT 0,
    "visibility" "DocumentVisibility"  NOT NULL DEFAULT 'PRIVATE',
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "documents_pkey" PRIMARY KEY ("id")
);

-- Create document_chunks table
CREATE TABLE "document_chunks" (
    "id" TEXT NOT NULL,
    "documentId" TEXT NOT NULL,
    "chunkIndex" INTEGER NOT NULL,
    "content" TEXT NOT NULL,
    "embedding" vector(1536),
    "metadata" JSONB,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "document_chunks_pkey" PRIMARY KEY ("id")
);

-- Create unique indexes
CREATE UNIQUE INDEX "users_email_key" ON "users"("email");
CREATE UNIQUE INDEX "document_chunks_documentId_chunkIndex_key" ON "document_chunks"("documentId", "chunkIndex");

-- Create indexes
CREATE INDEX "documents_userId_idx" ON "documents"("userId");
CREATE INDEX "documents_status_idx" ON "documents"("status");
CREATE INDEX "document_chunks_documentId_idx" ON "document_chunks"("documentId");

-- Add foreign key constraints
ALTER TABLE "documents" ADD CONSTRAINT "documents_userId_fkey" FOREIGN KEY ("userId") REFERENCES "users"("id") ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "document_chunks" ADD CONSTRAINT "document_chunks_documentId_fkey" FOREIGN KEY ("documentId") REFERENCES "documents"("id") ON DELETE CASCADE ON UPDATE CASCADE;
