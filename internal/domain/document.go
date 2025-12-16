package domain

import (
	"context"
	"io"
	"time"
)

type DocumentData struct {
	ID        string
	UserID    string
	Title     string
	FilePath  string
	FileSize  int64
	Metadata  DocumentMetadata
	CreatedAt time.Time
	UpdatedAt time.Time
}

type DocumentMetadata struct {
	Author      string
	PageCount   int
	HasPassword bool
}

type DocumentService interface {
	Upload(ctx context.Context, userID string, file io.Reader, token string, originalName string) (*Document, error)
}
