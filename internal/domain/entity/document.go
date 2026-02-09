package entity

import "time"

type DocumentStatus string
type DocumentVisibility string

const (
	StatusProcessing DocumentStatus = "processing"
	StatusCompleted  DocumentStatus = "completed"
	StatusFailed     DocumentStatus = "failed"

	VisibilityPublic  DocumentVisibility = "public"
	VisibilityPrivate DocumentVisibility = "private"
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
