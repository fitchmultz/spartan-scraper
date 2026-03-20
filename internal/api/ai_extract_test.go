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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	piai "github.com/fitchmultz/spartan-scraper/internal/ai"
	"github.com/fitchmultz/spartan-scraper/internal/aiauthoring"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

func TestAIExtractPreviewBodySize(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a very large request body that exceeds maxAIAuthoringRequestBodySize
	largeBody := make([]byte, maxAIAuthoringRequestBodySize+1000)
	for i := range largeBody {
		largeBody[i] = 'a'
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/ai/extract-preview", bytes.NewReader(largeBody))
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
	healthSnapshot    extract.AIHealthSnapshot
	templateCalls     int
	extractCalls      int
	lastExtractReq    extract.AIExtractRequest
	lastTemplateReq   extract.AITemplateGenerateRequest
}

type fakeAutomationClient struct {
	renderProfileResult  piai.GenerateRenderProfileResult
	pipelineJSResult     piai.GeneratePipelineJSResult
	researchRefineResult piai.ResearchRefineResult
	exportShapeResult    piai.ExportShapeResult
	transformResult      piai.GenerateTransformResult
	renderProfileReq     piai.GenerateRenderProfileRequest
	pipelineJSReq        piai.GeneratePipelineJSRequest
	researchRefineReq    piai.ResearchRefineRequest
	exportShapeReq       piai.ExportShapeRequest
	transformReq         piai.GenerateTransformRequest
	renderProfileCalls   int
	pipelineJSCalls      int
	researchRefineCalls  int
	exportShapeCalls     int
	transformCalls       int
}

func (f *fakeAIProvider) Extract(ctx context.Context, req extract.AIExtractRequest) (extract.AIExtractResult, error) {
	f.extractCalls++
	f.lastExtractReq = req
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

func (f *fakeAIProvider) HealthStatus(ctx context.Context) (extract.AIHealthSnapshot, error) {
	if len(f.healthSnapshot.Capabilities) == 0 {
		routing := config.DefaultAIRoutingConfig()
		resolved := make(map[string][]string, len(routing.Routes))
		available := make(map[string][]string, len(routing.Routes))
		for capability, routes := range routing.Routes {
			resolved[capability] = append([]string(nil), routes...)
			available[capability] = append([]string(nil), routes...)
		}
		return extract.BuildAIHealthSnapshot(config.AIConfig{
			Enabled: true,
			Mode:    "sdk",
			Routing: routing,
		}, piai.HealthResponse{
			Mode:      "sdk",
			Resolved:  resolved,
			Available: available,
		}), nil
	}
	return f.healthSnapshot, nil
}

func (f *fakeAIProvider) HealthCheck(ctx context.Context) error {
	_, err := f.HealthStatus(ctx)
	return err
}

func (f *fakeAIProvider) RouteFingerprint(capability string) string {
	return "test-route"
}

func (f *fakeAutomationClient) GenerateRenderProfile(ctx context.Context, req piai.GenerateRenderProfileRequest) (piai.GenerateRenderProfileResult, error) {
	f.renderProfileCalls++
	f.renderProfileReq = req
	return f.renderProfileResult, nil
}

func (f *fakeAutomationClient) GeneratePipelineJS(ctx context.Context, req piai.GeneratePipelineJSRequest) (piai.GeneratePipelineJSResult, error) {
	f.pipelineJSCalls++
	f.pipelineJSReq = req
	return f.pipelineJSResult, nil
}

func (f *fakeAutomationClient) GenerateResearchRefinement(ctx context.Context, req piai.ResearchRefineRequest) (piai.ResearchRefineResult, error) {
	f.researchRefineCalls++
	f.researchRefineReq = req
	return f.researchRefineResult, nil
}

func (f *fakeAutomationClient) GenerateExportShape(ctx context.Context, req piai.ExportShapeRequest) (piai.ExportShapeResult, error) {
	f.exportShapeCalls++
	f.exportShapeReq = req
	return f.exportShapeResult, nil
}

func (f *fakeAutomationClient) GenerateTransform(ctx context.Context, req piai.GenerateTransformRequest) (piai.GenerateTransformResult, error) {
	f.transformCalls++
	f.transformReq = req
	return f.transformResult, nil
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
	req := httptest.NewRequest(http.MethodPost, "/v1/ai/extract-preview", bytes.NewBufferString(body))
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

func TestAIExtractPreviewIncludesDirectImages(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	provider := &fakeAIProvider{
		extractResult: extract.AIExtractResult{
			Fields: map[string]extract.FieldValue{
				"title": {Values: []string{"Example"}, Source: extract.FieldSourceLLM},
			},
			Confidence: 0.9,
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

	body := `{"html":"<html><h1>Example</h1></html>","mode":"natural_language","prompt":"Extract the title","images":[{"data":"ZmFrZQ==","mime_type":"image/png"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/ai/extract-preview", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if provider.extractCalls != 1 {
		t.Fatalf("expected single extract call, got %d", provider.extractCalls)
	}
	if len(provider.lastExtractReq.Images) != 1 || provider.lastExtractReq.Images[0].MimeType != "image/png" {
		t.Fatalf("expected direct image to reach provider, got %#v", provider.lastExtractReq.Images)
	}

	var resp AIExtractPreviewResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.VisualContextUsed {
		t.Fatal("expected visual context to be reported when direct images are attached")
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
	req := httptest.NewRequest(http.MethodPost, "/v1/ai/extract-preview", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestAITemplateGenerateSupportsDirectHTML(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	provider := &fakeAIProvider{
		templateResponses: []extract.AITemplateGenerateResult{{
			Template: extract.Template{
				Name: "product-template",
				Selectors: []extract.SelectorRule{
					{Name: "title", Selector: "h1", Attr: "text", Trim: true},
				},
			},
			RouteID:  "openai/gpt-5.4",
			Provider: "openai",
			Model:    "gpt-5.4",
		}},
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

	body := `{"html":"<html><body><h1>Widget</h1></body></html>","description":"Extract the product title","images":[{"data":"ZmFrZQ==","mime_type":"image/png"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/ai/template-generate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if provider.templateCalls != 1 {
		t.Fatalf("expected single template generation call, got %d", provider.templateCalls)
	}
	if provider.lastTemplateReq.URL != "" {
		t.Fatalf("expected empty URL for direct HTML mode, got %q", provider.lastTemplateReq.URL)
	}
	if len(provider.lastTemplateReq.Images) != 1 || provider.lastTemplateReq.Images[0].MimeType != "image/png" {
		t.Fatalf("expected direct image to reach template provider, got %#v", provider.lastTemplateReq.Images)
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
	req := httptest.NewRequest(http.MethodPost, "/v1/ai/template-generate", bytes.NewBufferString(body))
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

func TestAITemplateDebugSuggestsRepairsForBrokenTemplate(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	provider := &fakeAIProvider{
		templateResponses: []extract.AITemplateGenerateResult{{
			Template: extract.Template{
				Name: "product-template",
				Selectors: []extract.SelectorRule{
					{Name: "title", Selector: "h1", Attr: "text", Trim: true},
				},
			},
			Explanation: "Updated the selector to use the visible heading.",
			RouteID:     "openai/gpt-5.4",
			Provider:    "openai",
			Model:       "gpt-5.4",
		}},
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

	body := `{"html":"<html><body><h1>Widget</h1></body></html>","template":{"name":"product-template","selectors":[{"name":"title","selector":".missing","attr":"text"}]},"instructions":"Prefer the visible heading","images":[{"data":"ZmFrZQ==","mime_type":"image/png"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/ai/template-debug", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if provider.templateCalls != 1 {
		t.Fatalf("expected single repair call, got %d", provider.templateCalls)
	}
	if provider.lastTemplateReq.Feedback == "" {
		t.Fatal("expected debug feedback on repair request")
	}
	if len(provider.lastTemplateReq.Images) != 1 || provider.lastTemplateReq.Images[0].MimeType != "image/png" {
		t.Fatalf("expected direct image to reach template debug provider, got %#v", provider.lastTemplateReq.Images)
	}

	var resp AIExtractTemplateDebugResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Issues) == 0 {
		t.Fatal("expected local issues in debug response")
	}
	if resp.SuggestedTemplate == nil || resp.SuggestedTemplate.Name != "product-template" {
		t.Fatalf("unexpected suggested template: %#v", resp.SuggestedTemplate)
	}
}

func TestAIRenderProfileGenerateReturnsValidatedProfile(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	automationClient := &fakeAutomationClient{
		renderProfileResult: piai.GenerateRenderProfileResult{
			Profile:     piai.BridgeRenderProfile{PreferHeadless: true, Wait: piai.BridgeRenderWaitPolicy{Mode: "selector", Selector: "main"}},
			Explanation: "Use headless mode and wait for the main content.",
			RouteID:     "openai/gpt-5.4",
			Provider:    "openai",
			Model:       "gpt-5.4",
		},
	}
	srv.cfg.AI = config.AIConfig{Enabled: true, Routing: config.DefaultAIRoutingConfig(), RequestTimeoutSecs: 30}
	srv.aiAuthoring = aiauthoring.NewServiceWithAutomationClient(srv.cfg, nil, automationClient, true)

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><main>Dashboard</main></body></html>`))
	}))
	defer source.Close()

	body := `{"url":"` + source.URL + `","name":"dashboard","host_patterns":["example.com"],"instructions":"Wait for the dashboard shell and prefer headless mode","images":[{"data":"ZmFrZQ==","mime_type":"image/png"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/ai/render-profile-generate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if automationClient.renderProfileCalls != 1 {
		t.Fatalf("expected single render profile generation call, got %d", automationClient.renderProfileCalls)
	}
	if len(automationClient.renderProfileReq.Images) != 1 || automationClient.renderProfileReq.Images[0].MimeType != "image/png" {
		t.Fatalf("expected direct image to reach automation client, got %#v", automationClient.renderProfileReq.Images)
	}

	var resp AIRenderProfileGenerateResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Profile.Name != "dashboard" {
		t.Fatalf("unexpected profile name: %q", resp.Profile.Name)
	}
	if !resp.Profile.PreferHeadless {
		t.Fatal("expected preferHeadless to be preserved")
	}
	if resp.RouteID != "openai/gpt-5.4" || resp.Provider != "openai" || resp.Model != "gpt-5.4" {
		t.Fatalf("expected route/provider/model metadata, got %q %q/%q", resp.RouteID, resp.Provider, resp.Model)
	}
}

func TestAIPipelineJSGenerateReturnsValidatedScript(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	automationClient := &fakeAutomationClient{
		pipelineJSResult: piai.GeneratePipelineJSResult{
			Script:      piai.BridgePipelineJSScript{Selectors: []string{"main"}, PostNav: "window.scrollTo(0, 0);"},
			Explanation: "Wait for the main content and reset scroll position.",
			RouteID:     "openai/gpt-5.4",
			Provider:    "openai",
			Model:       "gpt-5.4",
		},
	}
	srv.cfg.AI = config.AIConfig{Enabled: true, Routing: config.DefaultAIRoutingConfig(), RequestTimeoutSecs: 30}
	srv.aiAuthoring = aiauthoring.NewServiceWithAutomationClient(srv.cfg, nil, automationClient, true)

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><main>Dashboard</main></body></html>`))
	}))
	defer source.Close()

	body := `{"url":"` + source.URL + `","instructions":"Wait for the dashboard shell and reset scroll position","images":[{"data":"ZmFrZQ==","mime_type":"image/png"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/ai/pipeline-js-generate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if automationClient.pipelineJSCalls != 1 {
		t.Fatalf("expected single pipeline JS generation call, got %d", automationClient.pipelineJSCalls)
	}
	if len(automationClient.pipelineJSReq.Images) != 1 || automationClient.pipelineJSReq.Images[0].MimeType != "image/png" {
		t.Fatalf("expected direct image to reach pipeline JS automation client, got %#v", automationClient.pipelineJSReq.Images)
	}

	var resp AIPipelineJSGenerateResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Script.Name == "" {
		t.Fatalf("expected generated script name, got %q", resp.Script.Name)
	}
	if len(resp.Script.Selectors) != 1 || resp.Script.Selectors[0] != "main" {
		t.Fatalf("unexpected selectors: %#v", resp.Script.Selectors)
	}
}

func TestAIRenderProfileDebugReturnsIssuesAndSuggestion(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	automationClient := &fakeAutomationClient{
		renderProfileResult: piai.GenerateRenderProfileResult{
			Profile:     piai.BridgeRenderProfile{PreferHeadless: true, Wait: piai.BridgeRenderWaitPolicy{Mode: "selector", Selector: "main"}},
			Explanation: "Prefer headless mode and wait for the main shell.",
			RouteID:     "openai/gpt-5.4",
			Provider:    "openai",
			Model:       "gpt-5.4",
		},
	}
	srv.cfg.AI = config.AIConfig{Enabled: true, Routing: config.DefaultAIRoutingConfig(), RequestTimeoutSecs: 30}
	srv.aiAuthoring = aiauthoring.NewServiceWithAutomationClient(srv.cfg, nil, automationClient, true)

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><main>Dashboard</main></body></html>`))
	}))
	defer source.Close()

	body := `{"url":"` + source.URL + `","profile":{"name":"example-app","hostPatterns":["127.0.0.1"],"wait":{"mode":"selector","selector":".missing"}},"instructions":"Prefer the visible main shell","images":[{"data":"ZmFrZQ==","mime_type":"image/png"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/ai/render-profile-debug", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if automationClient.renderProfileCalls != 1 {
		t.Fatalf("expected single render profile debug call, got %d", automationClient.renderProfileCalls)
	}
	if automationClient.renderProfileReq.Feedback == "" {
		t.Fatal("expected render profile debug feedback")
	}
	if len(automationClient.renderProfileReq.Images) != 1 || automationClient.renderProfileReq.Images[0].MimeType != "image/png" {
		t.Fatalf("expected direct image to reach render profile debug automation client, got %#v", automationClient.renderProfileReq.Images)
	}

	var resp AIRenderProfileDebugResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Issues) == 0 {
		t.Fatal("expected local render profile issues")
	}
	if resp.SuggestedProfile == nil || resp.SuggestedProfile.Wait.Selector != "main" {
		t.Fatalf("unexpected suggested profile: %#v", resp.SuggestedProfile)
	}
	if resp.RecheckStatus != http.StatusOK {
		t.Fatalf("expected recheck status 200, got %d", resp.RecheckStatus)
	}
}

func TestAIPipelineJSDebugReturnsIssuesAndSuggestion(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	automationClient := &fakeAutomationClient{
		pipelineJSResult: piai.GeneratePipelineJSResult{
			Script:      piai.BridgePipelineJSScript{Selectors: []string{"main"}, PostNav: "window.scrollTo(0, 0);"},
			Explanation: "Wait for the main shell and normalize scroll.",
			RouteID:     "openai/gpt-5.4",
			Provider:    "openai",
			Model:       "gpt-5.4",
		},
	}
	srv.cfg.AI = config.AIConfig{Enabled: true, Routing: config.DefaultAIRoutingConfig(), RequestTimeoutSecs: 30}
	srv.aiAuthoring = aiauthoring.NewServiceWithAutomationClient(srv.cfg, nil, automationClient, true)

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><main>Dashboard</main></body></html>`))
	}))
	defer source.Close()

	body := `{"url":"` + source.URL + `","script":{"name":"example-app","hostPatterns":["127.0.0.1"],"selectors":[".missing"]},"images":[{"data":"ZmFrZQ==","mime_type":"image/png"}]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/ai/pipeline-js-debug", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if automationClient.pipelineJSCalls != 1 {
		t.Fatalf("expected single pipeline JS debug call, got %d", automationClient.pipelineJSCalls)
	}
	if automationClient.pipelineJSReq.Feedback == "" {
		t.Fatal("expected pipeline JS debug feedback")
	}
	if len(automationClient.pipelineJSReq.Images) != 1 || automationClient.pipelineJSReq.Images[0].MimeType != "image/png" {
		t.Fatalf("expected direct image to reach pipeline JS debug automation client, got %#v", automationClient.pipelineJSReq.Images)
	}

	var resp AIPipelineJSDebugResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Issues) == 0 {
		t.Fatal("expected local pipeline JS issues")
	}
	if resp.SuggestedScript == nil || len(resp.SuggestedScript.Selectors) != 1 || resp.SuggestedScript.Selectors[0] != "main" {
		t.Fatalf("unexpected suggested script: %#v", resp.SuggestedScript)
	}
}

func TestAIResearchRefineReturnsStructuredRefinement(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	automationClient := &fakeAutomationClient{
		researchRefineResult: piai.ResearchRefineResult{
			Refined: piai.ResearchRefinedContent{
				Summary:        "Enterprise pricing appears to be sales-led, with support commitments documented across the supplied evidence.",
				ConciseSummary: "Sales-led pricing with documented support commitments.",
				KeyFindings: []string{
					"Pricing is handled through direct sales rather than self-serve checkout.",
				},
				OpenQuestions: []string{"Are SLA terms publicly documented?"},
				RecommendedNextSteps: []string{
					"Confirm final SLA language with the vendor sales team.",
				},
				EvidenceHighlights: []piai.ResearchEvidenceHighlight{{
					URL:         "https://example.com/pricing",
					Title:       "Pricing",
					Finding:     "The pricing page routes buyers to contact sales.",
					CitationURL: "https://example.com/pricing",
				}},
				Confidence: 0.81,
			},
			Explanation: "Condensed the existing research result into an operator brief.",
			RouteID:     "openai/gpt-5.4",
			Provider:    "openai",
			Model:       "gpt-5.4",
		},
	}
	srv.cfg.AI = config.AIConfig{Enabled: true, Routing: config.DefaultAIRoutingConfig(), RequestTimeoutSecs: 30}
	srv.aiAuthoring = aiauthoring.NewServiceWithAutomationClient(srv.cfg, nil, automationClient, true)

	body := `{"result":{"query":"pricing and support commitments","summary":"Original summary","confidence":0.78,"evidence":[{"url":"https://example.com/pricing","title":"Pricing","snippet":"Contact sales for enterprise pricing.","citationUrl":"https://example.com/pricing"}],"citations":[{"canonical":"https://example.com/pricing","url":"https://example.com/pricing"}]},"instructions":"Condense this into a concise operator brief"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/ai/research-refine", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if automationClient.researchRefineCalls != 1 {
		t.Fatalf("expected single research refine call, got %d", automationClient.researchRefineCalls)
	}
	if automationClient.researchRefineReq.Instructions != "Condense this into a concise operator brief" {
		t.Fatalf("unexpected instructions: %q", automationClient.researchRefineReq.Instructions)
	}

	var resp AIResearchRefineResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Refined.ConciseSummary != "Sales-led pricing with documented support commitments." {
		t.Fatalf("unexpected concise summary: %q", resp.Refined.ConciseSummary)
	}
	if resp.InputStats.EvidenceCount != 1 || resp.InputStats.EvidenceUsedCount != 1 {
		t.Fatalf("unexpected input stats: %#v", resp.InputStats)
	}
	if !strings.Contains(resp.Markdown, "Refined Research Brief") {
		t.Fatalf("expected markdown output, got %q", resp.Markdown)
	}
	if got := rr.Header().Get("X-Spartan-AI-Route"); got != "openai/gpt-5.4" {
		t.Fatalf("expected X-Spartan-AI-Route header, got %q", got)
	}
}

func TestAIExportShapeLoadsJobResultAndReturnsShape(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	resultDir := filepath.Join(srv.cfg.DataDir, "jobs", "job-export-shape")
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		t.Fatalf("mkdir result dir: %v", err)
	}
	resultPath := filepath.Join(resultDir, "result.jsonl")
	resultBody := `{"url":"https://example.com","status":200,"title":"Example","text":"Body","normalized":{"title":"Example","fields":{"price":{"values":["$10"]},"plan":{"values":["Pro"]}}}}`
	if err := os.WriteFile(resultPath, []byte(resultBody), 0o644); err != nil {
		t.Fatalf("write result file: %v", err)
	}
	now := time.Now().UTC()
	job := model.Job{
		ID:          "job-export-shape",
		Kind:        model.KindScrape,
		Status:      model.StatusSucceeded,
		CreatedAt:   now,
		UpdatedAt:   now,
		SpecVersion: 1,
		ResultPath:  resultPath,
	}
	if err := srv.store.Create(context.Background(), job); err != nil {
		t.Fatalf("create job: %v", err)
	}

	automationClient := &fakeAutomationClient{
		exportShapeResult: piai.ExportShapeResult{
			Shape: piai.BridgeExportShapeConfig{
				TopLevelFields:   []string{"url", "title", "status"},
				NormalizedFields: []string{"field.price", "field.plan"},
				SummaryFields:    []string{"title", "field.price"},
				FieldLabels:      map[string]string{"field.price": "Price"},
				Formatting:       piai.ExportFormattingHints{MarkdownTitle: "Pricing Export"},
			},
			Explanation: "Selected high-signal export fields for the representative scrape result.",
			RouteID:     "openai/gpt-5.4",
			Provider:    "openai",
			Model:       "gpt-5.4",
		},
	}
	srv.cfg.AI = config.AIConfig{Enabled: true, Routing: config.DefaultAIRoutingConfig(), RequestTimeoutSecs: 30}
	srv.aiAuthoring = aiauthoring.NewServiceWithAutomationClient(srv.cfg, nil, automationClient, true)

	body := `{"job_id":"job-export-shape","format":"md","instructions":"Focus on pricing-related export fields"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/ai/export-shape", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if automationClient.exportShapeCalls != 1 {
		t.Fatalf("expected single export shape call, got %d", automationClient.exportShapeCalls)
	}
	if automationClient.exportShapeReq.JobKind != string(model.KindScrape) {
		t.Fatalf("unexpected job kind: %q", automationClient.exportShapeReq.JobKind)
	}
	if automationClient.exportShapeReq.Format != "md" {
		t.Fatalf("unexpected format: %q", automationClient.exportShapeReq.Format)
	}
	if automationClient.exportShapeReq.Instructions != "Focus on pricing-related export fields" {
		t.Fatalf("unexpected instructions: %q", automationClient.exportShapeReq.Instructions)
	}
	var resp AIExportShapeResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.Shape.NormalizedFields) != 2 || resp.Shape.NormalizedFields[0] != "field.price" {
		t.Fatalf("unexpected shape: %#v", resp.Shape)
	}
	if got := rr.Header().Get("X-Spartan-AI-Route"); got != "openai/gpt-5.4" {
		t.Fatalf("expected X-Spartan-AI-Route header, got %q", got)
	}
}

func TestAITransformGenerateLoadsJobResultAndReturnsTransform(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	resultDir := filepath.Join(srv.cfg.DataDir, "jobs", "job-transform")
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		t.Fatalf("mkdir result dir: %v", err)
	}
	resultPath := filepath.Join(resultDir, "result.jsonl")
	resultBody := "{\"url\":\"https://example.com\",\"title\":\"Example\",\"status\":200}\n{\"url\":\"https://example.com/2\",\"title\":\"Example 2\",\"status\":200}\n"
	if err := os.WriteFile(resultPath, []byte(resultBody), 0o644); err != nil {
		t.Fatalf("write result file: %v", err)
	}
	now := time.Now().UTC()
	if err := srv.store.Create(context.Background(), model.Job{
		ID:          "job-transform",
		Kind:        model.KindCrawl,
		Status:      model.StatusSucceeded,
		CreatedAt:   now,
		UpdatedAt:   now,
		SpecVersion: 1,
		ResultPath:  resultPath,
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	automationClient := &fakeAutomationClient{
		transformResult: piai.GenerateTransformResult{
			Transform: piai.BridgeTransformConfig{
				Expression: "{title: title, url: url}",
				Language:   "jmespath",
			},
			Explanation: "Projected the title and URL for export.",
			RouteID:     "openai/gpt-5.4",
			Provider:    "openai",
			Model:       "gpt-5.4",
		},
	}
	srv.cfg.AI = config.AIConfig{Enabled: true, Routing: config.DefaultAIRoutingConfig(), RequestTimeoutSecs: 30}
	srv.aiAuthoring = aiauthoring.NewServiceWithAutomationClient(srv.cfg, nil, automationClient, true)

	body := `{"job_id":"job-transform","preferredLanguage":"jmespath","currentTransform":{"expression":"[","language":"jmespath"},"instructions":"Project the URL and title"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/ai/transform-generate", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if automationClient.transformCalls != 1 {
		t.Fatalf("expected single transform call, got %d", automationClient.transformCalls)
	}
	if automationClient.transformReq.JobKind != string(model.KindCrawl) {
		t.Fatalf("unexpected job kind: %q", automationClient.transformReq.JobKind)
	}
	if automationClient.transformReq.PreferredLanguage != "jmespath" {
		t.Fatalf("unexpected preferred language: %q", automationClient.transformReq.PreferredLanguage)
	}
	if automationClient.transformReq.CurrentTransform.Expression != "[" {
		t.Fatalf("unexpected current transform: %#v", automationClient.transformReq.CurrentTransform)
	}
	var resp AITransformGenerateResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Transform.Language != "jmespath" || resp.Transform.Expression != "{title: title, url: url}" {
		t.Fatalf("unexpected transform: %#v", resp.Transform)
	}
	if len(resp.Preview) != 2 {
		t.Fatalf("expected preview output, got %#v", resp.Preview)
	}
	if !resp.InputStats.CurrentTransformProvided {
		t.Fatalf("expected current transform indicator, got %#v", resp.InputStats)
	}
	if got := rr.Header().Get("X-Spartan-AI-Route"); got != "openai/gpt-5.4" {
		t.Fatalf("expected X-Spartan-AI-Route header, got %q", got)
	}
}

func TestAITemplateGenerateBodySize(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a very large request body that exceeds maxAIAuthoringRequestBodySize
	largeBody := make([]byte, maxAIAuthoringRequestBodySize+1000)
	for i := range largeBody {
		largeBody[i] = 'a'
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/ai/template-generate", bytes.NewReader(largeBody))
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
