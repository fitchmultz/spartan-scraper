// Package api provides HTTP handler tests for template preview endpoints.
//
// Purpose:
// - Verify template preview endpoints normalize request inputs consistently.
//
// Responsibilities:
// - Confirm preview and selector-test requests accept URLs with surrounding whitespace.
// - Confirm selector-test requests trim selector input before querying the DOM.
// - Confirm hostless URLs fail as validation before any fetch attempt.
//
// Scope:
// - `/v1/template-preview` and `/v1/template-preview/test-selector` only.
//
// Usage:
// - Run with `go test ./internal/api`.
//
// Invariants/Assumptions:
// - URL validation should match the trimmed operator input, not the raw transport whitespace.
// - Selector testing should reject whitespace-only selectors but accept trimmed selector text.
// - Hostless `http`/`https` inputs are malformed for preview and selector-test flows.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	urlpkg "net/url"
	"testing"
)

func TestTemplatePreviewHandlersTrimWhitespace(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>Example</title></head><body><main>Hello</main></body></html>`))
	}))
	defer target.Close()

	previewReq := httptest.NewRequest(
		http.MethodGet,
		fmt.Sprintf("/v1/template-preview?url=%s", urlpkg.QueryEscape("  "+target.URL+"  ")),
		nil,
	)
	previewRR := httptest.NewRecorder()
	srv.Routes().ServeHTTP(previewRR, previewReq)
	if previewRR.Code != http.StatusOK {
		t.Fatalf("expected preview 200, got %d: %s", previewRR.Code, previewRR.Body.String())
	}

	var previewResp TemplatePreviewResponse
	if err := json.Unmarshal(previewRR.Body.Bytes(), &previewResp); err != nil {
		t.Fatalf("failed to decode preview response: %v", err)
	}
	if previewResp.URL != target.URL {
		t.Fatalf("expected trimmed preview URL %q, got %q", target.URL, previewResp.URL)
	}

	selectorBody, _ := json.Marshal(TestSelectorRequest{
		URL:      "  " + target.URL + "  ",
		Selector: "  main  ",
	})
	selectorReq := httptest.NewRequest(
		http.MethodPost,
		"/v1/template-preview/test-selector",
		bytes.NewReader(selectorBody),
	)
	selectorReq.Header.Set("Content-Type", "application/json")
	selectorRR := httptest.NewRecorder()
	srv.Routes().ServeHTTP(selectorRR, selectorReq)
	if selectorRR.Code != http.StatusOK {
		t.Fatalf("expected selector test 200, got %d: %s", selectorRR.Code, selectorRR.Body.String())
	}

	var selectorResp TestSelectorResponse
	if err := json.Unmarshal(selectorRR.Body.Bytes(), &selectorResp); err != nil {
		t.Fatalf("failed to decode selector response: %v", err)
	}
	if selectorResp.Selector != "main" {
		t.Fatalf("expected trimmed selector %q, got %q", "main", selectorResp.Selector)
	}
	if selectorResp.Matches != 1 {
		t.Fatalf("expected 1 match, got %d", selectorResp.Matches)
	}
}

func TestTemplatePreviewHandlersRejectHostlessURLs(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	previewReq := httptest.NewRequest(http.MethodGet, "/v1/template-preview?url=https:", nil)
	previewRR := httptest.NewRecorder()
	srv.Routes().ServeHTTP(previewRR, previewReq)
	if previewRR.Code != http.StatusBadRequest {
		t.Fatalf("expected preview 400, got %d: %s", previewRR.Code, previewRR.Body.String())
	}

	selectorBody, _ := json.Marshal(TestSelectorRequest{URL: "https:", Selector: "main"})
	selectorReq := httptest.NewRequest(
		http.MethodPost,
		"/v1/template-preview/test-selector",
		bytes.NewReader(selectorBody),
	)
	selectorReq.Header.Set("Content-Type", "application/json")
	selectorRR := httptest.NewRecorder()
	srv.Routes().ServeHTTP(selectorRR, selectorReq)
	if selectorRR.Code != http.StatusBadRequest {
		t.Fatalf("expected selector test 400, got %d: %s", selectorRR.Code, selectorRR.Body.String())
	}
}
