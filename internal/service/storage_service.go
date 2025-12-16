package service

import (
	"context"
	"errors"
	"io"
	"net/http"
)

type StorageService interface {
	Upload(ctx context.Context, path string, file io.Reader) error
}

type SupabaseStorage struct {
	baseURL string
	apiKey  string
}

func NewStorageService(
	baseURL string,
	apiKey string,
) *SupabaseStorage {
	return &SupabaseStorage{
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

func (s *SupabaseStorage) Upload(
	ctx context.Context,
	path string,
	file io.Reader,
) error {

	req, _ := http.NewRequest(
		"POST",
		s.baseURL+"/storage/v1/object/"+path,
		file,
	)

	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/pdf")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return errors.New("storage upload failed")
	}

	return nil
}
