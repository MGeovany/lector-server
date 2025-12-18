package service

import (
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"pdf-text-reader/internal/domain"

	"github.com/gen2brain/go-fitz"
)

// PDFProcessor handles PDF text extraction
type PDFProcessor struct {
	logger domain.Logger
}

// NewPDFProcessor creates a new PDF processor
func NewPDFProcessor(logger domain.Logger) *PDFProcessor {
	return &PDFProcessor{
		logger: logger,
	}
}

// TextBlock represents a block of text from a PDF
type TextBlock struct {
	Type     string `json:"type"`     // "paragraph" or "heading"
	Content  string `json:"content"`  // The text content
	Level    int    `json:"level"`    // Heading level (0 for paragraphs)
	PageNum  int    `json:"page_num"` // Page number (1-indexed)
	Position int    `json:"position"` // Position within the page
}

// PDFMetadata contains extracted PDF metadata
type PDFMetadata struct {
	Author      string `json:"author"`
	PageCount   int    `json:"page_count"`
	HasPassword bool   `json:"has_password"`
	Title       string `json:"title"`
}

// ProcessPDF extracts text and metadata from a PDF file
func (p *PDFProcessor) ProcessPDF(pdfBytes []byte) ([]TextBlock, PDFMetadata, error) {
	// Open PDF document from bytes
	doc, err := fitz.NewFromMemory(pdfBytes)
	if err != nil {
		return nil, PDFMetadata{}, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer doc.Close()

	// Get metadata
	docMetadata := doc.Metadata()
	metadata := PDFMetadata{
		PageCount:   doc.NumPage(),
		HasPassword: false, // go-fitz doesn't expose this directly
	}

	// Extract title and author from metadata
	if title, ok := docMetadata["title"]; ok && title != "" {
		metadata.Title = title
	}
	if author, ok := docMetadata["author"]; ok && author != "" {
		metadata.Author = author
	}

	var blocks []TextBlock
	positionCounter := 0

	// Process each page
	for pageNum := 0; pageNum < doc.NumPage(); pageNum++ {
		// Extract text from page
		text, err := doc.Text(pageNum)
		if err != nil {
			p.logger.Warn("Failed to extract text from page", "page_num", pageNum, "error", err)
			continue
		}

		// Split text into paragraphs and process
		paragraphs := p.splitIntoParagraphs(text)

		for _, para := range paragraphs {
			para = strings.TrimSpace(para)
			if para == "" {
				continue
			}

			// Determine if it's a heading (short text, all caps, or starts with number)
			blockType := "paragraph"
			level := 0
			if p.isHeading(para) {
				blockType = "heading"
				level = 1 // Default heading level
			}

			// Sanitize content to remove problematic Unicode sequences
			// Replace control characters and normalize Unicode
			sanitizedContent := p.sanitizeText(para)

			blocks = append(blocks, TextBlock{
				Type:     blockType,
				Content:  sanitizedContent,
				Level:    level,
				PageNum:  pageNum + 1, // 1-indexed for frontend
				Position: positionCounter,
			})
			positionCounter++
		}
	}

	return blocks, metadata, nil
}

// splitIntoParagraphs splits text into paragraphs based on double newlines
func (p *PDFProcessor) splitIntoParagraphs(text string) []string {
	// Normalize line breaks
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Split by double newlines (paragraph breaks)
	paragraphs := strings.Split(text, "\n\n")

	var result []string
	for _, para := range paragraphs {
		// Clean up single newlines within paragraphs (replace with space)
		para = strings.ReplaceAll(para, "\n", " ")
		para = strings.TrimSpace(para)
		if para != "" {
			result = append(result, para)
		}
	}

	return result
}

// isHeading determines if a text block is likely a heading
func (p *PDFProcessor) isHeading(text string) bool {
	// Heuristics for detecting headings:
	// 1. Very short text (less than 100 chars)
	// 2. All uppercase
	// 3. Ends with no punctuation
	// 4. Single line

	text = strings.TrimSpace(text)
	if len(text) == 0 {
		return false
	}

	// Single line check
	if strings.Contains(text, "\n") {
		return false
	}

	// Short text is more likely to be a heading
	if len(text) < 100 {
		// Check if all uppercase (common for headings)
		if text == strings.ToUpper(text) && len(text) > 3 {
			return true
		}
		// Very short text is likely a heading
		if len(text) < 50 {
			return true
		}
	}

	return false
}

// ProcessPDFFromReader processes a PDF from an io.Reader
func (p *PDFProcessor) ProcessPDFFromReader(reader io.Reader) ([]TextBlock, PDFMetadata, error) {
	pdfBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, PDFMetadata{}, fmt.Errorf("failed to read PDF: %w", err)
	}
	return p.ProcessPDF(pdfBytes)
}

// sanitizeText removes problematic Unicode characters and control sequences
// Specifically removes \u0000 (NULL) and other characters that cause PostgreSQL 22P05 errors
// This function ensures that the text can be safely JSON-encoded and stored in PostgreSQL JSONB
func (p *PDFProcessor) sanitizeText(text string) string {
	// First pass: remove NULL characters and other problematic control characters
	var result strings.Builder
	result.Grow(len(text))

	for _, r := range text {
		// Remove NULL character (0x00) - PostgreSQL JSONB cannot handle it (causes 22P05 error)
		if r == 0x00 {
			continue
		}

		// Remove other problematic control characters except whitespace
		// Keep: tab (0x09), newline (0x0A), carriage return (0x0D)
		if r == 0x09 || r == 0x0A || r == 0x0D {
			result.WriteRune(r)
		} else if r >= 0x20 && r < 0x7F {
			// Printable ASCII (0x20-0x7E)
			result.WriteRune(r)
		} else if r >= 0x7F && r <= 0x10FFFF {
			// Valid Unicode above ASCII
			// Exclude surrogates (0xD800-0xDFFF) which are invalid in JSON
			if r < 0xD800 || r > 0xDFFF {
				result.WriteRune(r)
			}
		}
		// All other characters (control chars, surrogates, etc.) are skipped
	}

	sanitized := result.String()

	// Second pass: ensure the string can be safely JSON-encoded without Unicode escape issues
	// Marshal to JSON to check for problematic escape sequences
	testJSON, err := json.Marshal(sanitized)
	if err != nil {
		// If marshaling fails, return empty string
		return ""
	}

	// Check if the JSON contains problematic Unicode escape sequences
	jsonStr := string(testJSON)
	
	// Remove all control character Unicode escapes (0000-001F) that PostgreSQL rejects
	// Use regex to match \u followed by 0000-001F
	re := regexp.MustCompile(`\\u00[0-1][0-9a-fA-F]`)
	jsonStr = re.ReplaceAllString(jsonStr, "")
	
	// Remove surrogate pairs (D800-DFFF) which are invalid in JSON
	reSurrogates := regexp.MustCompile(`\\u[dD][89aAbBcCdDeEfF][0-9a-fA-F]{2}`)
	jsonStr = reSurrogates.ReplaceAllString(jsonStr, "")
	
	// Remove any literal NULL bytes
	jsonStr = strings.ReplaceAll(jsonStr, "\x00", "")

	// Unmarshal to verify it's valid and get the cleaned string
	var testStr string
	if err := json.Unmarshal([]byte(jsonStr), &testStr); err != nil {
		// If unmarshaling fails, try to clean more aggressively
		// Remove any remaining problematic sequences
		cleaned := strings.ReplaceAll(sanitized, "\x00", "")
		cleaned = strings.ReplaceAll(cleaned, "\u0000", "")
		// Try marshaling again
		if testJSON2, err := json.Marshal(cleaned); err == nil {
			testStr2 := string(testJSON2)
			testStr2 = re.ReplaceAllString(testStr2, "")
			testStr2 = reSurrogates.ReplaceAllString(testStr2, "")
			if err := json.Unmarshal([]byte(testStr2), &testStr); err != nil {
				return cleaned // Return the cleaned string even if JSON unmarshal fails
			}
		} else {
			return cleaned
		}
	}

	return testStr
}

// ConvertToJSON converts TextBlocks to JSON format expected by frontend
// Ensures the JSON is safe for PostgreSQL JSONB (no \u0000 or problematic escape sequences)
func (p *PDFProcessor) ConvertToJSON(blocks []TextBlock) (json.RawMessage, error) {
	// Marshal to JSON
	jsonBytes, err := json.Marshal(blocks)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal blocks: %w", err)
	}

	// Remove problematic Unicode escape sequences that PostgreSQL rejects
	jsonStr := string(jsonBytes)
	
	// Remove all control character Unicode escapes (0000-001F)
	reControlChars := regexp.MustCompile(`\\u00[0-1][0-9a-fA-F]`)
	jsonStr = reControlChars.ReplaceAllString(jsonStr, "")
	
	// Remove surrogate pairs (D800-DFFF) which are invalid in JSON
	reSurrogates := regexp.MustCompile(`\\u[dD][89aAbBcCdDeEfF][0-9a-fA-F]{2}`)
	jsonStr = reSurrogates.ReplaceAllString(jsonStr, "")
	
	// Remove any literal NULL bytes
	jsonStr = strings.ReplaceAll(jsonStr, "\x00", "")
	jsonStr = strings.ReplaceAll(jsonStr, "\\u0000", "")
	jsonStr = strings.ReplaceAll(jsonStr, "\\u000", "")

	// Verify the cleaned JSON is valid by unmarshaling and re-marshaling
	var verify []TextBlock
	if err := json.Unmarshal([]byte(jsonStr), &verify); err != nil {
		// If verification fails, try to recover by cleaning more aggressively
		// Remove ALL Unicode escapes and re-encode
		reAllUnicode := regexp.MustCompile(`\\u[0-9a-fA-F]{4}`)
		jsonStr = reAllUnicode.ReplaceAllStringFunc(jsonStr, func(match string) string {
			// Check if it's a control character or surrogate
			hexStr := match[2:]
			if len(hexStr) == 4 {
				if (hexStr[0] == '0' && hexStr[1] == '0' && hexStr[2] <= '1') ||
					((hexStr[0] == 'd' || hexStr[0] == 'D') && (hexStr[1] >= '8' && hexStr[1] <= 'f' || hexStr[1] >= '8' && hexStr[1] <= 'F')) {
					return "" // Remove problematic sequences
				}
			}
			return match // Keep other Unicode escapes
		})
		
		// Try unmarshaling again
		if err := json.Unmarshal([]byte(jsonStr), &verify); err != nil {
			p.logger.Warn("Failed to verify cleaned JSON after aggressive cleaning", "error", err)
			// Return empty array as fallback
			return json.RawMessage("[]"), nil
		}
	}

	// Re-marshal to ensure clean JSON
	cleanedJSON, err := json.Marshal(verify)
	if err != nil {
		// If re-marshaling fails, return the cleaned string version
		return json.RawMessage(jsonStr), nil
	}

	// Final pass: clean the re-marshaled JSON one more time
	finalJSONStr := string(cleanedJSON)
	finalJSONStr = reControlChars.ReplaceAllString(finalJSONStr, "")
	finalJSONStr = reSurrogates.ReplaceAllString(finalJSONStr, "")
	finalJSONStr = strings.ReplaceAll(finalJSONStr, "\x00", "")

	return json.RawMessage(finalJSONStr), nil
}
