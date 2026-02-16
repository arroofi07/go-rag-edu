package document

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"rag-api/internal/domain/entity"
	"rag-api/internal/domain/repository"

	"github.com/pgvector/pgvector-go"
)

type EmbeddingService interface {
	GenerateBatchEmbeddings(ctx context.Context, texts []string) ([]pgvector.Vector, error)
}

type DocumentUsecase struct {
	docRepo   repository.DocumentRepository
	chunkRepo repository.ChunkRepository
	embedder  EmbeddingService
	extractor *TextExtractor
	chunker   *Chunker
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

// upload document
func (uc *DocumentUsecase) UploadDocument(
	ctx context.Context,
	userID string,
	filename string,
	fileData []byte,
	mimeType string,
	visibility entity.DocumentVisibility,
) (*entity.Document, error) {

	// create document record
	doc := &entity.Document{
		UserID:       userID,
		Filename:     fmt.Sprintf("%d_%d_%s", userID, time.Now().Unix(), filename),
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

	// process document in background
	go func() {
		// recovery for panic in background process
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in document processing for doc %s: %v", doc.ID, r)
				uc.docRepo.UpdateStatus(context.Background(), doc.ID, entity.StatusFailed)
			}
		}()

		if err := uc.ProcessDocument(context.Background(), doc.ID, fileData, mimeType); err != nil {
			log.Printf("Error processing document %s: %v", doc.ID, err)
			uc.docRepo.UpdateStatus(context.Background(), doc.ID, entity.StatusFailed)
		}
	}()

	return doc, nil

}

// process document
func (uc DocumentUsecase) ProcessDocument(
	ctx context.Context,
	documentID string,
	fileData []byte,
	mimeType string,
) error {
	log.Printf("Starting processing for document %s", documentID)

	// 1 extract text
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
	log.Printf("Extracted %d characters from document %s", len(text), documentID)

	// 2 chunk text
	textChunks := uc.chunker.ChunkText(text)
	if len(textChunks) == 0 {
		return fmt.Errorf("no chunks generated")
	}
	log.Printf("Generated %d chunks from document %s", len(textChunks), documentID)

	// 3 generate embeddings
	embeddings, err := uc.embedder.GenerateBatchEmbeddings(ctx, textChunks)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}
	log.Printf("Generated %d embeddings from document %s", len(embeddings), documentID)

	// 4 create chunks with embeddings
	var chunks []entity.DocumentChunk
	for i, content := range textChunks {
		metadata, _ := json.Marshal(entity.ChunkMetadata{
			Source:     "text",
			PageNumber: i/10 + 1,
		})
		chunks = append(chunks, entity.DocumentChunk{
			DocumentID: documentID,
			ChunkIndex: i,
			Content:    content,
			Embedding:  embeddings[i],
			Metadata:   metadata,
		})
	}

	// 5 save chunks
	if err := uc.chunkRepo.CreateBatch(ctx, chunks); err != nil {
		return fmt.Errorf("failed to save chunks: %w", err)
	}
	log.Printf("Saved %d chunks to database for document %s", len(chunks), documentID)

	// 6 update document status
	if err := uc.docRepo.UpdateTotalChunks(ctx, documentID, len(chunks)); err != nil {
		return err
	}

	if err := uc.docRepo.UpdateStatus(ctx, documentID, entity.StatusCompleted); err != nil {
		return err
	}

	log.Printf("Document %s processed successfully with %d chunks", documentID, len(chunks))
	return nil

}

// list document
func (uc *DocumentUsecase) ListDocuments(
	ctx context.Context,
	userID string,
	page, limit int,
) ([]entity.Document, int, error) {
	return uc.docRepo.List(ctx, userID, page, limit)
}

// get document by id
func (uc *DocumentUsecase) GetDocumentByID(
	ctx context.Context,
	documentID string,
	userID string,
) (*entity.Document, error) {
	doc, err := uc.docRepo.FindByIDAndUserID(ctx, documentID, userID)
	if err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, nil
	}

	return doc, nil

}

// delete
func (uc *DocumentUsecase) DeleteDocument(
	ctx context.Context,
	documentID string,
	userID string,
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
