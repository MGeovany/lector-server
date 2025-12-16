package domain

import (
	"context"
	"io"
)

type StorageService interface {
	Upload(ctx context.Context, path string, file io.Reader) error
}

type SupabaseStorage struct {
	baseURL string
	apiKey  string
}
