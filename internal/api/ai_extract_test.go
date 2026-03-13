// Package api provides HTTP handlers for AI-powered extraction endpoints.
//
// This test file validates body size limits for AI extract endpoints to prevent
// oversized payload attacks.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
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

type fakeAIProvider struct {
	extractResult     extract.AIExtractResult
	templateResponses []extract.AITemplateGenerateResult
	templateCalls     int
	lastTemplateReq   extract.AITemplateGenerateRequest
}

func (f *fakeAIProvider) Extract(ctx context.Context, req extract.AIExtractRequest) (extract.AIExtractResult, error) {
	return f.extractResult, nil
}

func (f *fakeAIProvider) GenerateTemplate(ctx context.Context, req extract.AITemplateGenerateRequest) (extract.AITemplateGenerateResult, error) {
	f.templateCalls++
	f.lastTemplateReq = req
	if len(f.templateResponses) == 0 {
		return extract.AITemplateGenerateResult{}, nil
	}
	index := f.templateCalls - 1
	if index >= len(f.templateResponses) {
		index = len(f.templateResponses) - 1
	}
	return f.templateResponses[index], nil
}

func (f *fakeAIProvider) HealthCheck(ctx context.Context) error {
	return nil
}

func (f *fakeAIProvider) RouteFingerprint(capability string) string {
	return "test-route"
}

func TestAIExtractPreviewIncludesProviderMetadata(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	provider := &fakeAIProvider{
		extractResult: extract.AIExtractResult{
			Fields: map[string]extract.FieldValue{
				"title": {
					Values: []string{"Example"},
					Source: extract.FieldSourceLLM,
				},
			},
			Confidence: 0.9,
			RouteID:    "openai/gpt-5.4",
			Provider:   "openai",
			Model:      "gpt-5.4",
		},
	}
	srv.aiExtractor = extract.NewAIExtractorWithProvider(
		config.AIConfig{Enabled: true, Routing: config.DefaultAIRoutingConfig()},
		srv.cfg.DataDir,
		provider,
	)
	srv.cfg.AI = config.AIConfig{
		Enabled:            true,
		RequestTimeoutSecs: 30,
		Routing:            config.DefaultAIRoutingConfig(),
	}

	body := `{"url":"https://example.com","html":"<html><h1>Example</h1></html>","mode":"natural_language","fields":["title"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/extract/ai-preview", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp AIExtractPreviewResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.RouteID != "openai/gpt-5.4" || resp.Provider != "openai" || resp.Model != "gpt-5.4" {
		t.Fatalf("expected route/provider/model metadata, got %q %q/%q", resp.RouteID, resp.Provider, resp.Model)
	}
	if got := rr.Header().Get("X-Spartan-AI-Route"); got != "openai/gpt-5.4" {
		t.Fatalf("expected X-Spartan-AI-Route header, got %q", got)
	}
}

func TestAIExtractPreviewFetchesHTMLWhenNotProvided(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><h1>Fetched Example</h1></body></html>`))
	}))
	defer source.Close()

	provider := &fakeAIProvider{
		extractResult: extract.AIExtractResult{
			Fields: map[string]extract.FieldValue{
				"title": {
					Values: []string{"Fetched Example"},
					Source: extract.FieldSourceLLM,
				},
			},
			Confidence: 0.91,
			RouteID:    "openai/gpt-5.4",
			Provider:   "openai",
			Model:      "gpt-5.4",
		},
	}
	srv.aiExtractor = extract.NewAIExtractorWithProvider(
		config.AIConfig{Enabled: true, Routing: config.DefaultAIRoutingConfig()},
		srv.cfg.DataDir,
		provider,
	)
	srv.cfg.AI = config.AIConfig{
		Enabled:            true,
		RequestTimeoutSecs: 30,
		Routing:            config.DefaultAIRoutingConfig(),
	}

	body := `{"url":"` + source.URL + `","mode":"natural_language","fields":["title"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/extract/ai-preview", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAITemplateGenerateFetchesHTMLAndRetriesValidation(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><h1 class="product-title">Widget</h1><span class="price">$10</span></body></html>`))
	}))
	defer source.Close()

	provider := &fakeAIProvider{
		templateResponses: []extract.AITemplateGenerateResult{
			{
				Template: extract.Template{
					Name: "broken-template",
					Selectors: []extract.SelectorRule{
						{Name: "title", Selector: ".missing", Attr: "text", Trim: true},
					},
				},
			},
			{
				Template: extract.Template{
					Name: "product-template",
					Selectors: []extract.SelectorRule{
						{Name: "title", Selector: ".product-title", Attr: "text", Trim: true},
						{Name: "price", Selector: ".price", Attr: "text", Trim: true},
					},
				},
				Explanation: "Selectors updated after validation feedback.",
				RouteID:     "openai/gpt-5.4",
				Provider:    "openai",
				Model:       "gpt-5.4",
			},
		},
	}
	srv.aiExtractor = extract.NewAIExtractorWithProvider(
		config.AIConfig{Enabled: true, Routing: config.DefaultAIRoutingConfig()},
		srv.cfg.DataDir,
		provider,
	)
	srv.cfg.AI = config.AIConfig{
		Enabled:            true,
		RequestTimeoutSecs: 30,
		Routing:            config.DefaultAIRoutingConfig(),
	}

	body := `{"url":"` + source.URL + `","description":"Extract the product title and price"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/extract/ai-template-generate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if provider.templateCalls != 2 {
		t.Fatalf("expected template generation retry, got %d calls", provider.templateCalls)
	}
	if provider.lastTemplateReq.Feedback == "" {
		t.Fatal("expected validation feedback on retry request")
	}

	var resp AIExtractTemplateGenerateResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Template.Selectors) != 2 {
		t.Fatalf("expected validated selectors in response, got %#v", resp.Template.Selectors)
	}
	if resp.RouteID != "openai/gpt-5.4" || resp.Provider != "openai" || resp.Model != "gpt-5.4" {
		t.Fatalf("expected route/provider/model metadata, got %q %q/%q", resp.RouteID, resp.Provider, resp.Model)
	}
	if got := rr.Header().Get("X-Spartan-AI-Model"); got != "gpt-5.4" {
		t.Fatalf("expected X-Spartan-AI-Model header, got %q", got)
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

func TestFetchHTMLForAIRejectsInternalTargetsWhenAPIServerIsRemote(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	srv.cfg.BindAddr = "0.0.0.0"

	_, err := srv.fetchHTMLForAI(context.Background(), "http://127.0.0.1:12345", false, false)
	if !webhook.IsSSRFError(err) {
		t.Fatalf("expected SSRF validation error, got %v", err)
	}
}
