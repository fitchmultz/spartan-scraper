// Package api provides HTTP handlers for AI-powered extraction endpoints.
//
// This test file validates body size limits for AI extract endpoints to prevent
// oversized payload attacks.
package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAIExtractPreviewBodySize(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a very large request body that exceeds maxRequestBodySize
	largeBody := make([]byte, maxRequestBodySize+1000)
	for i := range largeBody {
		largeBody[i] = 'a'
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/extract/ai-preview", bytes.NewReader(largeBody))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len(largeBody)) // Explicitly set Content-Length
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	// Should fail due to size limit (returns 413 for request entity too large)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status 413 for oversized request, got %d", rr.Code)
	}
}

func TestAITemplateGenerateBodySize(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a very large request body that exceeds maxRequestBodySize
	largeBody := make([]byte, maxRequestBodySize+1000)
	for i := range largeBody {
		largeBody[i] = 'a'
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/extract/ai-template-generate", bytes.NewReader(largeBody))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len(largeBody)) // Explicitly set Content-Length
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	// Should fail due to size limit (returns 413 for request entity too large)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status 413 for oversized request, got %d", rr.Code)
	}
}
