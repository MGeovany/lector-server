package service

import "testing"

func TestNewStorageService(t *testing.T) {
	svc := NewStorageService("http://localhost:54321", "test-key")
	if svc.baseURL != "http://localhost:54321" {
		t.Fatalf("expected base url to be set, got %s", svc.baseURL)
	}
	if svc.apiKey != "test-key" {
		t.Fatalf("expected api key to be set, got %s", svc.apiKey)
	}
	if svc.storageClient == nil {
		t.Fatalf("expected storage client to be initialized")
	}
}
