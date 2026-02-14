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
	return &DocumentUsecase{docRepo: docRepo}
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

	return doc, nil

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

	return uc.docRepo.Delete(ctx, documentID)

}
