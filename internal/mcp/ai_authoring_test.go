package mcp

import (
	"context"
	"encoding/json"
	"os"
	"testing"

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

func TestAIAuthoringToolsList(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	tools := srv.toolsList()
	toolNames := make(map[string]bool, len(tools))
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}
	for _, name := range []string{"ai_extract_preview", "ai_template_generate"} {
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
	aiExtractor := extract.NewAIExtractorWithProvider(
		config.AIConfig{Enabled: true, Routing: config.DefaultAIRoutingConfig(), RequestTimeoutSecs: 30},
		tmpDir,
		provider,
	)
	srv.aiAuthoring = aiauthoring.NewService(config.Config{DataDir: tmpDir, UserAgent: "test-agent", RequestTimeoutSecs: 30, AI: config.AIConfig{RequestTimeoutSecs: 30}}, aiExtractor, true)

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
}
