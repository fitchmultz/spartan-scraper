package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	piai "github.com/fitchmultz/spartan-scraper/internal/ai"
	"github.com/fitchmultz/spartan-scraper/internal/aiauthoring"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
)

type fakeAuthoringProvider struct {
	extractResult   extract.AIExtractResult
	templateResults []extract.AITemplateGenerateResult
	templateCalls   int
	lastTemplateReq extract.AITemplateGenerateRequest
}

type fakeAutomationClient struct {
	renderProfileResult piai.GenerateRenderProfileResult
	pipelineJSResult    piai.GeneratePipelineJSResult
	renderProfileReq    piai.GenerateRenderProfileRequest
	pipelineJSReq       piai.GeneratePipelineJSRequest
	renderProfileCalls  int
	pipelineJSCalls     int
}

func (f *fakeAuthoringProvider) Extract(ctx context.Context, req extract.AIExtractRequest) (extract.AIExtractResult, error) {
	return f.extractResult, nil
}

func (f *fakeAuthoringProvider) GenerateTemplate(ctx context.Context, req extract.AITemplateGenerateRequest) (extract.AITemplateGenerateResult, error) {
	f.templateCalls++
	f.lastTemplateReq = req
	if len(f.templateResults) == 0 {
		return extract.AITemplateGenerateResult{}, nil
	}
	idx := f.templateCalls - 1
	if idx >= len(f.templateResults) {
		idx = len(f.templateResults) - 1
	}
	return f.templateResults[idx], nil
}

func (f *fakeAuthoringProvider) HealthCheck(ctx context.Context) error {
	return nil
}

func (f *fakeAuthoringProvider) RouteFingerprint(capability string) string {
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

func TestAIAuthoringToolsList(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	tools := srv.toolsList()
	toolNames := make(map[string]bool, len(tools))
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}
	for _, name := range []string{"ai_extract_preview", "ai_template_generate", "ai_template_debug", "ai_render_profile_generate", "ai_render_profile_debug", "ai_pipeline_js_generate", "ai_pipeline_js_debug"} {
		if !toolNames[name] {
			t.Fatalf("expected tool %s in list", name)
		}
	}
}

func TestHandleToolCallAIPreviewAndTemplateGeneration(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	provider := &fakeAuthoringProvider{
		extractResult: extract.AIExtractResult{
			Fields: map[string]extract.FieldValue{
				"title": {Values: []string{"Example"}, Source: extract.FieldSourceLLM},
			},
			Confidence: 0.93,
			RouteID:    "openai/gpt-5.4",
			Provider:   "openai",
			Model:      "gpt-5.4",
		},
		templateResults: []extract.AITemplateGenerateResult{
			{
				Template: extract.Template{
					Name:      "broken-template",
					Selectors: []extract.SelectorRule{{Name: "title", Selector: ".missing", Attr: "text"}},
				},
			},
			{
				Template: extract.Template{
					Name:      "product-template",
					Selectors: []extract.SelectorRule{{Name: "title", Selector: "h1", Attr: "text"}},
				},
				Explanation: "Updated selectors after validation feedback",
				RouteID:     "openai/gpt-5.4",
				Provider:    "openai",
				Model:       "gpt-5.4",
			},
		},
	}
	automationClient := &fakeAutomationClient{
		renderProfileResult: piai.GenerateRenderProfileResult{
			Profile:     piai.BridgeRenderProfile{PreferHeadless: true, Wait: piai.BridgeRenderWaitPolicy{Mode: "selector", Selector: "main"}},
			Explanation: "Prefer headless mode and wait for the main element.",
			RouteID:     "openai/gpt-5.4",
			Provider:    "openai",
			Model:       "gpt-5.4",
		},
		pipelineJSResult: piai.GeneratePipelineJSResult{
			Script:      piai.BridgePipelineJSScript{Selectors: []string{"main"}, PostNav: "window.scrollTo(0, 0);"},
			Explanation: "Wait for the main element and normalize scroll position.",
			RouteID:     "openai/gpt-5.4",
			Provider:    "openai",
			Model:       "gpt-5.4",
		},
	}
	aiExtractor := extract.NewAIExtractorWithProvider(
		config.AIConfig{Enabled: true, Routing: config.DefaultAIRoutingConfig(), RequestTimeoutSecs: 30},
		tmpDir,
		provider,
	)
	srv.aiAuthoring = aiauthoring.NewServiceWithAutomationClient(config.Config{DataDir: tmpDir, UserAgent: "test-agent", RequestTimeoutSecs: 30, AI: config.AIConfig{Enabled: true, Routing: config.DefaultAIRoutingConfig(), RequestTimeoutSecs: 30}}, aiExtractor, automationClient, true)

	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><main>Example</main><h1>Example</h1></body></html>`))
	}))
	defer source.Close()

	previewBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "ai_extract_preview",
			"arguments": map[string]interface{}{
				"html":   "<html><body><h1>Example</h1></body></html>",
				"mode":   "natural_language",
				"prompt": "Extract the title",
				"fields": []string{"title"},
			},
		}),
	}
	previewResult, err := srv.handleToolCall(context.Background(), previewBase)
	if err != nil {
		t.Fatalf("ai_extract_preview failed: %v", err)
	}
	previewMap, ok := previewResult.(aiauthoring.PreviewResult)
	if !ok {
		t.Fatalf("expected preview result type, got %#v", previewResult)
	}
	if previewMap.RouteID != "openai/gpt-5.4" {
		t.Fatalf("unexpected route id: %q", previewMap.RouteID)
	}

	templateBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "ai_template_generate",
			"arguments": map[string]interface{}{
				"html":        "<html><body><h1>Example</h1></body></html>",
				"description": "Extract the title",
			},
		}),
	}
	templateResult, err := srv.handleToolCall(context.Background(), templateBase)
	if err != nil {
		t.Fatalf("ai_template_generate failed: %v", err)
	}
	templateResp, ok := templateResult.(aiauthoring.TemplateResult)
	if !ok {
		t.Fatalf("expected template result type, got %#v", templateResult)
	}
	if templateResp.Template.Name != "product-template" {
		t.Fatalf("unexpected template name: %q", templateResp.Template.Name)
	}
	if provider.templateCalls != 2 {
		t.Fatalf("expected validation retry, got %d calls", provider.templateCalls)
	}
	if provider.lastTemplateReq.Feedback == "" {
		t.Fatal("expected validation feedback on retry")
	}

	provider.templateResults = []extract.AITemplateGenerateResult{{
		Template: extract.Template{
			Name:      "product-template",
			Selectors: []extract.SelectorRule{{Name: "title", Selector: "h1", Attr: "text"}},
		},
		Explanation: "Updated selector to target the visible heading.",
		RouteID:     "openai/gpt-5.4",
		Provider:    "openai",
		Model:       "gpt-5.4",
	}}
	provider.templateCalls = 0
	provider.lastTemplateReq = extract.AITemplateGenerateRequest{}

	debugBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "ai_template_debug",
			"arguments": map[string]interface{}{
				"html": "<html><body><h1>Example</h1></body></html>",
				"template": map[string]interface{}{
					"name": "product-template",
					"selectors": []map[string]interface{}{{
						"name":     "title",
						"selector": ".missing",
						"attr":     "text",
					}},
				},
				"instructions": "Prefer the visible heading",
			},
		}),
	}
	debugResult, err := srv.handleToolCall(context.Background(), debugBase)
	if err != nil {
		t.Fatalf("ai_template_debug failed: %v", err)
	}
	debugResp, ok := debugResult.(aiauthoring.TemplateDebugResult)
	if !ok {
		t.Fatalf("expected debug result type, got %#v", debugResult)
	}
	if len(debugResp.Issues) == 0 {
		t.Fatal("expected local template issues in debug response")
	}
	if debugResp.SuggestedTemplate == nil || debugResp.SuggestedTemplate.Name != "product-template" {
		t.Fatalf("unexpected suggested template: %#v", debugResp.SuggestedTemplate)
	}
	if provider.templateCalls != 1 {
		t.Fatalf("expected single debug generation call, got %d", provider.templateCalls)
	}
	if provider.lastTemplateReq.Feedback == "" {
		t.Fatal("expected template debug feedback to reach the provider")
	}

	renderProfileBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "ai_render_profile_generate",
			"arguments": map[string]interface{}{
				"url":          source.URL,
				"name":         "example-app",
				"hostPatterns": []string{"example.com", "*.example.com"},
				"instructions": "Wait for the main shell and prefer headless mode",
				"visual":       true,
			},
		}),
	}
	renderProfileResult, err := srv.handleToolCall(context.Background(), renderProfileBase)
	if err != nil {
		t.Fatalf("ai_render_profile_generate failed: %v", err)
	}
	renderProfileResp, ok := renderProfileResult.(aiauthoring.RenderProfileResult)
	if !ok {
		t.Fatalf("expected render profile result type, got %#v", renderProfileResult)
	}
	if renderProfileResp.Profile.Name != "example-app" {
		t.Fatalf("unexpected render profile name: %q", renderProfileResp.Profile.Name)
	}
	if renderProfileResp.Profile.Wait.Selector != "main" {
		t.Fatalf("unexpected render profile wait selector: %#v", renderProfileResp.Profile.Wait)
	}
	if automationClient.renderProfileCalls != 1 {
		t.Fatalf("expected single render profile generation call, got %d", automationClient.renderProfileCalls)
	}
	if automationClient.renderProfileReq.Instructions == "" {
		t.Fatal("expected render profile instructions to reach the automation client")
	}

	renderProfileDebugBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "ai_render_profile_debug",
			"arguments": map[string]interface{}{
				"url": source.URL,
				"profile": map[string]interface{}{
					"name":         "example-app",
					"hostPatterns": []string{"127.0.0.1"},
					"wait": map[string]interface{}{
						"mode":     "selector",
						"selector": ".missing",
					},
				},
				"instructions": "Prefer the visible main shell",
			},
		}),
	}
	renderProfileDebugResult, err := srv.handleToolCall(context.Background(), renderProfileDebugBase)
	if err != nil {
		t.Fatalf("ai_render_profile_debug failed: %v", err)
	}
	renderProfileDebugResp, ok := renderProfileDebugResult.(aiauthoring.RenderProfileDebugResult)
	if !ok {
		t.Fatalf("expected render profile debug result type, got %#v", renderProfileDebugResult)
	}
	if len(renderProfileDebugResp.Issues) == 0 {
		t.Fatal("expected local render profile issues")
	}
	if renderProfileDebugResp.SuggestedProfile == nil || renderProfileDebugResp.SuggestedProfile.Wait.Selector != "main" {
		t.Fatalf("unexpected render profile suggestion: %#v", renderProfileDebugResp.SuggestedProfile)
	}
	if automationClient.renderProfileCalls != 2 {
		t.Fatalf("expected debug call to reuse automation client, got %d total calls", automationClient.renderProfileCalls)
	}
	if automationClient.renderProfileReq.Feedback == "" {
		t.Fatal("expected render profile debug feedback to reach the automation client")
	}

	pipelineBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "ai_pipeline_js_generate",
			"arguments": map[string]interface{}{
				"url":          source.URL,
				"instructions": "Wait for the main shell and reset scroll position",
			},
		}),
	}
	pipelineResult, err := srv.handleToolCall(context.Background(), pipelineBase)
	if err != nil {
		t.Fatalf("ai_pipeline_js_generate failed: %v", err)
	}
	pipelineResp, ok := pipelineResult.(aiauthoring.PipelineJSResult)
	if !ok {
		t.Fatalf("expected pipeline JS result type, got %#v", pipelineResult)
	}
	if len(pipelineResp.Script.Selectors) != 1 || pipelineResp.Script.Selectors[0] != "main" {
		t.Fatalf("unexpected pipeline JS selectors: %#v", pipelineResp.Script.Selectors)
	}
	if automationClient.pipelineJSCalls != 1 {
		t.Fatalf("expected single pipeline JS generation call, got %d", automationClient.pipelineJSCalls)
	}

	pipelineDebugBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "ai_pipeline_js_debug",
			"arguments": map[string]interface{}{
				"url": source.URL,
				"script": map[string]interface{}{
					"name":         "example-app",
					"hostPatterns": []string{"127.0.0.1"},
					"selectors":    []string{".missing"},
				},
			},
		}),
	}
	pipelineDebugResult, err := srv.handleToolCall(context.Background(), pipelineDebugBase)
	if err != nil {
		t.Fatalf("ai_pipeline_js_debug failed: %v", err)
	}
	pipelineDebugResp, ok := pipelineDebugResult.(aiauthoring.PipelineJSDebugResult)
	if !ok {
		t.Fatalf("expected pipeline JS debug result type, got %#v", pipelineDebugResult)
	}
	if len(pipelineDebugResp.Issues) == 0 {
		t.Fatal("expected local pipeline JS issues")
	}
	if pipelineDebugResp.SuggestedScript == nil || len(pipelineDebugResp.SuggestedScript.Selectors) != 1 || pipelineDebugResp.SuggestedScript.Selectors[0] != "main" {
		t.Fatalf("unexpected pipeline JS suggestion: %#v", pipelineDebugResp.SuggestedScript)
	}
	if automationClient.pipelineJSCalls != 2 {
		t.Fatalf("expected debug call to reuse automation client, got %d total calls", automationClient.pipelineJSCalls)
	}
	if automationClient.pipelineJSReq.Feedback == "" {
		t.Fatal("expected pipeline JS debug feedback to reach the automation client")
	}
}
