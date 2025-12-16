package service

import (
	"context"
	"fmt"
	"io"

	storage_go "github.com/supabase-community/storage-go"
)

type StorageService interface {
	Upload(ctx context.Context, path string, file io.Reader, token string) error
}

type SupabaseStorage struct {
	baseURL       string
	apiKey        string
	storageClient *storage_go.Client
}

func NewStorageService(
	baseURL string,
	apiKey string,
) *SupabaseStorage {
	storageURL := baseURL + "/storage/v1"
	storageClient := storage_go.NewClient(storageURL, apiKey, nil)

	return &SupabaseStorage{
		baseURL:       baseURL,
		apiKey:        apiKey,
		storageClient: storageClient,
	}
}

func (s *SupabaseStorage) Upload(
	ctx context.Context,
	path string,
	file io.Reader,
	token string,
) error {
	bucketName := "documents"

	// Create a client with the user's access token for RLS policies
	// Use anon key (not service role) when using user token
	storageURL := s.baseURL + "/storage/v1"
	headers := map[string]string{
		"Authorization": "Bearer " + token,
	}
	storageClient := storage_go.NewClient(storageURL, s.apiKey, headers)

	_, err := storageClient.UploadFile(bucketName, path, file)
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	return nil
}
