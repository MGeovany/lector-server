package domain

import (
	"encoding/json"
	"testing"
	"time"
)

// TestDocument_Validate tests that the Document.Validate() method works correctly.
// It tests:
// - Valid documents with all required fields
// - Required field validation (ID, UserID, Title)
// - Nested metadata validation
func TestDocument_Validate(t *testing.T) {
	tests := []struct {
		name    string
		doc     Document
		wantErr bool
		errMsg  string
	}{
		{
			// Tests that a document with all required fields and valid metadata passes validation
			name: "Valid document",
			doc: Document{
				ID:      "test-id",
				UserID:  "user-id",
				Title:   "Test Document",
				Content: json.RawMessage(`[{"text": "test"}]`),
				Metadata: DocumentMetadata{
					PageCount: 10,
					FileSize:  1024,
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			// Tests that validation fails when document ID is missing
			name: "Missing ID",
			doc: Document{
				UserID:    "user-id",
				Title:     "Test Document",
				Content:   json.RawMessage(`[{"text": "test"}]`),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "id: document ID is required",
		},
		{
			// Tests that validation fails when UserID is missing
			name: "Missing UserID",
			doc: Document{
				ID:        "test-id",
				Title:     "Test Document",
				Content:   json.RawMessage(`[{"text": "test"}]`),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "user_id: user ID is required",
		},
		{
			// Tests that validation fails when title is empty
			name: "Empty title",
			doc: Document{
				ID:        "test-id",
				UserID:    "user-id",
				Title:     "",
				Content:   json.RawMessage(`[{"text": "test"}]`),
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "title: title is required",
		},
		{
			// Tests that validation fails when metadata has invalid values (negative file size)
			name: "Invalid metadata",
			doc: Document{
				ID:      "test-id",
				UserID:  "user-id",
				Title:   "Test Document",
				Content: json.RawMessage(`[{"text": "test"}]`),
				Metadata: DocumentMetadata{
					FileSize: -1, // Invalid: file size cannot be negative
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			wantErr: true,
			errMsg:  "file_size: file size cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.doc.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Document.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && err.Error() != tt.errMsg {
				t.Errorf("Document.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

// TestReadingPosition_Validate tests that the ReadingPosition.Validate() method works correctly.
// It tests:
// - Valid reading positions
// - Required field validation (UserID, DocumentID)
// - Range validation: Progress must be between 0 and 1
// - PageNumber validation: cannot be negative
// - Boundary cases for Progress values (0.0 and 1.0)
func TestReadingPosition_Validate(t *testing.T) {
	tests := []struct {
		name    string
		pos     ReadingPosition
		wantErr bool
		errMsg  string
	}{
		{
			// Tests that a reading position with all valid fields passes validation
			name: "Valid reading position",
			pos: ReadingPosition{
				UserID:     "user-id",
				DocumentID: "doc-id",
				Progress:   0.5,
				PageNumber: 5,
				UpdatedAt:  time.Now(),
			},
			wantErr: false,
		},
		{
			// Tests that validation fails when UserID is missing
			name: "Missing UserID",
			pos: ReadingPosition{
				DocumentID: "doc-id",
				Progress:   0.5,
				PageNumber: 5,
				UpdatedAt:  time.Now(),
			},
			wantErr: true,
			errMsg:  "user_id: user ID is required",
		},
		{
			// Tests that validation fails when DocumentID is missing
			name: "Missing DocumentID",
			pos: ReadingPosition{
				UserID:     "user-id",
				Progress:   0.5,
				PageNumber: 5,
				UpdatedAt:  time.Now(),
			},
			wantErr: true,
			errMsg:  "document_id: document ID is required",
		},
		{
			// Tests that validation fails when Progress exceeds the maximum value (1.0)
			name: "Progress too high",
			pos: ReadingPosition{
				UserID:     "user-id",
				DocumentID: "doc-id",
				Progress:   1.5, // Invalid: > 1.0
				PageNumber: 5,
				UpdatedAt:  time.Now(),
			},
			wantErr: true,
			errMsg:  "progress: progress must be between 0 and 1",
		},
		{
			// Tests that validation fails when Progress is negative
			name: "Progress negative",
			pos: ReadingPosition{
				UserID:     "user-id",
				DocumentID: "doc-id",
				Progress:   -0.1, // Invalid: < 0
				PageNumber: 5,
				UpdatedAt:  time.Now(),
			},
			wantErr: true,
			errMsg:  "progress: progress must be between 0 and 1",
		},
		{
			// Tests that Progress at the lower boundary (0.0) is valid
			name: "Progress at boundaries",
			pos: ReadingPosition{
				UserID:     "user-id",
				DocumentID: "doc-id",
				Progress:   0.0, // Valid boundary: minimum value
				PageNumber: 0,
				UpdatedAt:  time.Now(),
			},
			wantErr: false,
		},
		{
			// Tests that Progress at the upper boundary (1.0) is valid
			name: "Progress at upper boundary",
			pos: ReadingPosition{
				UserID:     "user-id",
				DocumentID: "doc-id",
				Progress:   1.0, // Valid boundary: maximum value
				PageNumber: 10,
				UpdatedAt:  time.Now(),
			},
			wantErr: false,
		},
		{
			// Tests that validation fails when PageNumber is negative
			name: "Negative page number",
			pos: ReadingPosition{
				UserID:     "user-id",
				DocumentID: "doc-id",
				Progress:   0.5,
				PageNumber: -1, // Invalid: page numbers cannot be negative
				UpdatedAt:  time.Now(),
			},
			wantErr: true,
			errMsg:  "page_number: page number cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pos.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadingPosition.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && err.Error() != tt.errMsg {
				t.Errorf("ReadingPosition.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

// TestDocumentMetadata_Validate tests that the DocumentMetadata.Validate() method works correctly.
// It tests:
// - Valid metadata with all fields
// - Valid metadata with minimal required fields
// - Zero values are valid (0 is acceptable for counts and sizes)
// - Negative values are rejected (FileSize, PageCount, WordCount)
// - Multiple validation errors (returns first error encountered)
func TestDocumentMetadata_Validate(t *testing.T) {
	tests := []struct {
		name    string
		meta    DocumentMetadata
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid metadata",
			meta: DocumentMetadata{
				OriginalTitle:  "Test PDF",
				OriginalAuthor: "Test Author",
				Language:       "en",
				PageCount:      10,
				WordCount:      1000,
				FileSize:       1024,
				Format:         "pdf",
				Source:         "upload",
				HasPassword:    false,
			},
			wantErr: false,
		},
		{
			name: "Valid metadata with minimal fields",
			meta: DocumentMetadata{
				FileSize: 1024,
				Format:   "pdf",
			},
			wantErr: false,
		},
		{
			name: "Valid metadata with zero values",
			meta: DocumentMetadata{
				FileSize:  0,
				PageCount: 0,
				WordCount: 0,
			},
			wantErr: false,
		},
		{
			name: "Negative file size",
			meta: DocumentMetadata{
				FileSize: -1,
			},
			wantErr: true,
			errMsg:  "file_size: file size cannot be negative",
		},
		{
			name: "Negative page count",
			meta: DocumentMetadata{
				PageCount: -1,
				FileSize:  1024,
			},
			wantErr: true,
			errMsg:  "page_count: page count cannot be negative",
		},
		{
			name: "Negative word count",
			meta: DocumentMetadata{
				WordCount: -1,
				FileSize:  1024,
			},
			wantErr: true,
			errMsg:  "word_count: word count cannot be negative",
		},
		{
			name: "Multiple negative values",
			meta: DocumentMetadata{
				FileSize:  -1,
				PageCount: -1,
				WordCount: -1,
			},
			wantErr: true,
			// Should return first error encountered
			errMsg: "file_size: file size cannot be negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.meta.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DocumentMetadata.Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && err.Error() != tt.errMsg {
				t.Errorf("DocumentMetadata.Validate() error = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

// TestDocument_JSONSerialization tests that Document can be correctly serialized to JSON
// and deserialized back without losing data. This ensures the JSON tags are correct
// and the structure can be used in API responses.
func TestDocument_JSONSerialization(t *testing.T) {
	doc := Document{
		ID:      "test-id",
		UserID:  "user-id",
		Title:   "Test Document",
		Content: json.RawMessage(`[{"type":"paragraph","content":"test","page_number":1}]`),
		Metadata: DocumentMetadata{
			PageCount: 10,
			FileSize:  1024,
			Format:    "pdf",
		},
		CreatedAt: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 1, 2, 12, 0, 0, 0, time.UTC),
	}

	// Test marshaling
	jsonData, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("Failed to marshal document: %v", err)
	}

	// Test unmarshaling
	var unmarshaled Document
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal document: %v", err)
	}

	// Verify fields
	if unmarshaled.ID != doc.ID {
		t.Errorf("ID mismatch: got %v, want %v", unmarshaled.ID, doc.ID)
	}
	if unmarshaled.UserID != doc.UserID {
		t.Errorf("UserID mismatch: got %v, want %v", unmarshaled.UserID, doc.UserID)
	}
	if unmarshaled.Title != doc.Title {
		t.Errorf("Title mismatch: got %v, want %v", unmarshaled.Title, doc.Title)
	}
	if unmarshaled.Metadata.FileSize != doc.Metadata.FileSize {
		t.Errorf("FileSize mismatch: got %v, want %v", unmarshaled.Metadata.FileSize, doc.Metadata.FileSize)
	}
}

// TestDocumentWithPosition_JSONSerialization tests that DocumentWithPosition can be
// correctly serialized to JSON and deserialized back. This structure combines a document
// with its reading position, which is commonly used in API responses.
func TestDocumentWithPosition_JSONSerialization(t *testing.T) {
	doc := &Document{
		ID:        "doc-id",
		UserID:    "user-id",
		Title:     "Test Document",
		Content:   json.RawMessage(`[]`),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	pos := &ReadingPosition{
		UserID:     "user-id",
		DocumentID: "doc-id",
		Progress:   0.75,
		PageNumber: 5,
		UpdatedAt:  time.Now(),
	}

	docWithPos := DocumentWithPosition{
		DocumentData:    doc,
		ReadingPosition: pos,
	}

	// Test marshaling
	jsonData, err := json.Marshal(docWithPos)
	if err != nil {
		t.Fatalf("Failed to marshal DocumentWithPosition: %v", err)
	}

	// Test unmarshaling
	var unmarshaled DocumentWithPosition
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal DocumentWithPosition: %v", err)
	}

	// Verify document data
	if unmarshaled.DocumentData == nil {
		t.Fatal("DocumentData is nil after unmarshaling")
	}
	if unmarshaled.DocumentData.ID != doc.ID {
		t.Errorf("Document ID mismatch: got %v, want %v", unmarshaled.DocumentData.ID, doc.ID)
	}

	// Verify reading position
	if unmarshaled.ReadingPosition == nil {
		t.Fatal("ReadingPosition is nil after unmarshaling")
	}
	if unmarshaled.ReadingPosition.Progress != pos.Progress {
		t.Errorf("Progress mismatch: got %v, want %v", unmarshaled.ReadingPosition.Progress, pos.Progress)
	}
	if unmarshaled.ReadingPosition.PageNumber != pos.PageNumber {
		t.Errorf("PageNumber mismatch: got %v, want %v", unmarshaled.ReadingPosition.PageNumber, pos.PageNumber)
	}
}

// TestLibraryResponse_JSONSerialization tests that LibraryResponse can be correctly
// serialized to JSON and deserialized back. This structure represents a collection of
// documents with their reading positions, which is the main response from the library endpoint.
// It tests handling of multiple documents and optional reading positions (nil values).
func TestLibraryResponse_JSONSerialization(t *testing.T) {
	doc1 := &Document{
		ID:        "doc-1",
		UserID:    "user-1",
		Title:     "Document 1",
		Content:   json.RawMessage(`[]`),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	doc2 := &Document{
		ID:        "doc-2",
		UserID:    "user-1",
		Title:     "Document 2",
		Content:   json.RawMessage(`[]`),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	response := LibraryResponse{
		Documents: []DocumentWithPosition{
			{
				DocumentData:    doc1,
				ReadingPosition: &ReadingPosition{UserID: "user-1", DocumentID: "doc-1", Progress: 0.5, PageNumber: 3, UpdatedAt: time.Now()},
			},
			{
				DocumentData:    doc2,
				ReadingPosition: nil, // Some documents may not have reading position
			},
		},
	}

	// Test marshaling
	jsonData, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal LibraryResponse: %v", err)
	}

	// Test unmarshaling
	var unmarshaled LibraryResponse
	if err := json.Unmarshal(jsonData, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal LibraryResponse: %v", err)
	}

	// Verify structure
	if len(unmarshaled.Documents) != 2 {
		t.Errorf("Expected 2 documents, got %d", len(unmarshaled.Documents))
	}
	if unmarshaled.Documents[0].DocumentData.ID != "doc-1" {
		t.Errorf("First document ID mismatch: got %v, want doc-1", unmarshaled.Documents[0].DocumentData.ID)
	}
	if unmarshaled.Documents[1].DocumentData.ID != "doc-2" {
		t.Errorf("Second document ID mismatch: got %v, want doc-2", unmarshaled.Documents[1].DocumentData.ID)
	}
	// First document should have reading position
	if unmarshaled.Documents[0].ReadingPosition == nil {
		t.Error("First document should have reading position")
	}
	// Second document may not have reading position (nil is valid)
}

// TestValidationError_Error tests that ValidationError correctly formats error messages.
// It tests different scenarios:
// - Error with both field and message (formatted as "field: message")
// - Error with only message (returns message as-is)
// - Error with empty field name (returns message only)
func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name    string
		err     *ValidationError
		wantMsg string
	}{
		{
			name:    "Error with field and message",
			err:     &ValidationError{Field: "title", Message: "title is required"},
			wantMsg: "title: title is required",
		},
		{
			name:    "Error with only message",
			err:     &ValidationError{Message: "validation failed"},
			wantMsg: "validation failed",
		},
		{
			name:    "Error with empty field",
			err:     &ValidationError{Field: "", Message: "something went wrong"},
			wantMsg: "something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("ValidationError.Error() = %v, want %v", got, tt.wantMsg)
			}
		})
	}
}

// TestDocument_ContentJSONValidity tests that Document.Content can contain valid JSON.
// This test verifies that the Content field (json.RawMessage) can hold various JSON structures.
// Note: Document validation does not check JSON validity - that's handled separately.
// This test ensures the Content field can store valid JSON arrays and objects.
func TestDocument_ContentJSONValidity(t *testing.T) {
	tests := []struct {
		name    string
		content json.RawMessage
		wantErr bool
	}{
		{
			name:    "Valid JSON array",
			content: json.RawMessage(`[{"type":"paragraph","content":"test","page_number":1}]`),
			wantErr: false,
		},
		{
			name:    "Valid empty array",
			content: json.RawMessage(`[]`),
			wantErr: false,
		},
		{
			name:    "Invalid JSON - missing bracket",
			content: json.RawMessage(`[{"type":"paragraph"`),
			wantErr: true,
		},
		{
			name:    "Valid complex JSON",
			content: json.RawMessage(`[{"type":"heading","content":"Chapter 1","level":1,"page_number":1},{"type":"paragraph","content":"Content here","page_number":1}]`),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc := Document{
				ID:        "test-id",
				UserID:    "user-id",
				Title:     "Test",
				Content:   tt.content,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}

			// Try to unmarshal the content to verify it's valid JSON
			var data interface{}
			err := json.Unmarshal(tt.content, &data)

			if (err != nil) != tt.wantErr {
				t.Errorf("Content JSON validity check: error = %v, wantErr %v", err, tt.wantErr)
			}

			// Document should still validate even if content JSON is invalid
			// (validation doesn't check JSON validity, that's a separate concern)
			if err := doc.Validate(); err != nil {
				t.Errorf("Document.Validate() should not fail due to content JSON: %v", err)
			}
		})
	}
}
