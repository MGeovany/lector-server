package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteError(t *testing.T) {
	rr := httptest.NewRecorder()
	writeError(rr, http.StatusTeapot, "nope")

	if rr.Code != http.StatusTeapot {
		t.Fatalf("expected status %d, got %d", http.StatusTeapot, rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected content type application/json, got %s", ct)
	}
	if strings.TrimSpace(rr.Body.String()) != `{"error":"nope"}` {
		t.Fatalf("unexpected response body: %s", rr.Body.String())
	}
}
