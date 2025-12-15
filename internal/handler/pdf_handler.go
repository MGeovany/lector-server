package handler

import (
	"net/http"

	"pdf-text-reader/internal/domain"
)

// PDFHandler handles HTTP requests for PDF operations
type PDFHandler struct {
	pdfService domain.PDFProcessor
	logger     domain.Logger
}

// NewPDFHandler creates a new PDF handler instance
func NewPDFHandler(pdfService domain.PDFProcessor, logger domain.Logger) *PDFHandler {
	return &PDFHandler{
		pdfService: pdfService,
		logger:     logger,
	}
}

// UploadPDF handles PDF file uploads
func (h *PDFHandler) UploadPDF(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement PDF upload handling in subsequent tasks
	h.logger.Debug("UploadPDF handler called")
	w.WriteHeader(http.StatusNotImplemented)
}

// GetDocument handles document retrieval requests
func (h *PDFHandler) GetDocument(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement document retrieval in subsequent tasks
	h.logger.Debug("GetDocument handler called")
	w.WriteHeader(http.StatusNotImplemented)
}

// GetDocumentMetadata handles document metadata requests
func (h *PDFHandler) GetDocumentMetadata(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement metadata retrieval in subsequent tasks
	h.logger.Debug("GetDocumentMetadata handler called")
	w.WriteHeader(http.StatusNotImplemented)
}
