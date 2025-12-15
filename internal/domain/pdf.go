package domain

import (
	"io"
	"time"
)

// PDFFormat represents different PDF format types
type PDFFormat string

const (
	PDFFormatStandard PDFFormat = "standard"
	PDFFormatScanned  PDFFormat = "scanned"
	PDFFormatEncrypted PDFFormat = "encrypted"
)

// BlockType represents different types of text blocks
type BlockType string

const (
	BlockTypeParagraph BlockType = "paragraph"
	BlockTypeHeading   BlockType = "heading"
	BlockTypeList      BlockType = "list"
	BlockTypeTable     BlockType = "table"
)

// TextBlock represents a structured piece of extracted text
type TextBlock struct {
	Content  string    `json:"content"`
	Type     BlockType `json:"type"`
	Level    int       `json:"level"`    // for headings
	PageNum  int       `json:"page_num"`
	Position int       `json:"position"`
}

// DocumentMetadata contains information about the PDF document
type DocumentMetadata struct {
	Title       string `json:"title"`
	Author      string `json:"author"`
	PageCount   int    `json:"page_count"`
	FileSize    int64  `json:"file_size"`
	Format      string `json:"format"`
	HasPassword bool   `json:"has_password"`
}

// ExtractedDocument represents the complete extracted document
type ExtractedDocument struct {
	ID           string           `json:"id"`
	OriginalName string           `json:"original_name"`
	Content      []TextBlock      `json:"content"`
	Metadata     DocumentMetadata `json:"metadata"`
	CreatedAt    time.Time        `json:"created_at"`
}

// ExtractedText represents the raw extracted text with metadata
type ExtractedText struct {
	Content   string           `json:"content"`
	Blocks    []TextBlock      `json:"blocks"`
	Metadata  DocumentMetadata `json:"metadata"`
	PageCount int              `json:"page_count"`
}

// PDFDocument represents a PDF document being processed
type PDFDocument struct {
	Reader   io.Reader
	Filename string
	Size     int64
	Format   PDFFormat
}

// FileInfo represents information about an uploaded file
type FileInfo struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	Path     string `json:"path"`
}
