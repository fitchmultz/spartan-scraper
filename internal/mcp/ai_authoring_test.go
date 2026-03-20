package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	piai "github.com/fitchmultz/spartan-scraper/internal/ai"
	"github.com/fitchmultz/spartan-scraper/internal/aiauthoring"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

type fakeAuthoringProvider struct {
	extractResult   extract.AIExtractResult
	templateResults []extract.AITemplateGenerateResult
	templateCalls   int
	extractCalls    int
	lastExtractReq  extract.AIExtractRequest
	lastTemplateReq extract.AITemplateGenerateRequest
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

func (f *fakeAuthoringProvider) Extract(ctx context.Context, req extract.AIExtractRequest) (extract.AIExtractResult, error) {
	f.extractCalls++
	f.lastExtractReq = req
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

func (f *fakeAuthoringProvider) HealthStatus(ctx context.Context) (extract.AIHealthSnapshot, error) {
	return extract.BuildConfiguredAIHealth(config.AIConfig{
		Enabled: true,
		Mode:    "sdk",
		Routing: config.DefaultAIRoutingConfig(),
	}), nil
}

func (f *fakeAuthoringProvider) HealthCheck(ctx context.Context) error {
	_, err := f.HealthStatus(ctx)
	return err
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

func TestAIAuthoringToolsList(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	tools := srv.toolsList()
	toolNames := make(map[string]bool, len(tools))
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}
	for _, name := range []string{"ai_extract_preview", "ai_template_generate", "ai_template_debug", "ai_render_profile_generate", "ai_render_profile_debug", "ai_pipeline_js_generate", "ai_pipeline_js_debug", "ai_research_refine", "ai_export_shape", "ai_transform_generate", "proxy_pool_status"} {
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
		researchRefineResult: piai.ResearchRefineResult{
			Refined: piai.ResearchRefinedContent{
				Summary:        "Enterprise pricing appears to be sales-led, with support commitments documented across the supplied evidence.",
				ConciseSummary: "Sales-led pricing with documented support commitments.",
				KeyFindings:    []string{"Pricing is handled through direct sales rather than self-serve checkout."},
				EvidenceHighlights: []piai.ResearchEvidenceHighlight{{
					URL:         "https://example.com/pricing",
					Title:       "Pricing",
					Finding:     "The pricing page routes buyers to contact sales.",
					CitationURL: "https://example.com/pricing",
				}},
				Confidence: 0.81,
			},
			Explanation: "Condensed the supplied research result into an operator brief.",
			RouteID:     "openai/gpt-5.4",
			Provider:    "openai",
			Model:       "gpt-5.4",
		},
		exportShapeResult: piai.ExportShapeResult{
			Shape: piai.BridgeExportShapeConfig{
				TopLevelFields:   []string{"url", "title", "status"},
				NormalizedFields: []string{"field.price"},
				SummaryFields:    []string{"title", "field.price"},
				FieldLabels:      map[string]string{"field.price": "Price"},
			},
			Explanation: "Selected export-ready fields for the representative sample.",
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
				"images": []map[string]interface{}{{
					"data":      "ZmFrZQ==",
					"mime_type": "image/png",
				}},
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
	if provider.extractCalls != 1 {
		t.Fatalf("expected single extract call, got %d", provider.extractCalls)
	}
	if len(provider.lastExtractReq.Images) != 1 || provider.lastExtractReq.Images[0].MimeType != "image/png" {
		t.Fatalf("expected preview image to reach provider, got %#v", provider.lastExtractReq.Images)
	}

	templateBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "ai_template_generate",
			"arguments": map[string]interface{}{
				"html":        "<html><body><h1>Example</h1></body></html>",
				"description": "Extract the title",
				"images": []map[string]interface{}{{
					"data":      "ZmFrZQ==",
					"mime_type": "image/png",
				}},
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
	if len(provider.lastTemplateReq.Images) != 1 || provider.lastTemplateReq.Images[0].MimeType != "image/png" {
		t.Fatalf("expected template image to reach provider, got %#v", provider.lastTemplateReq.Images)
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
				"images": []map[string]interface{}{{
					"data":      "ZmFrZQ==",
					"mime_type": "image/png",
				}},
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
	if len(provider.lastTemplateReq.Images) != 1 || provider.lastTemplateReq.Images[0].MimeType != "image/png" {
		t.Fatalf("expected template debug image to reach provider, got %#v", provider.lastTemplateReq.Images)
	}

	renderProfileBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "ai_render_profile_generate",
			"arguments": map[string]interface{}{
				"url":          source.URL,
				"name":         "example-app",
				"hostPatterns": []string{"example.com", "*.example.com"},
				"instructions": "Wait for the main shell and prefer headless mode",
				"images": []map[string]interface{}{{
					"data":      "ZmFrZQ==",
					"mime_type": "image/png",
				}},
				"visual": true,
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
	directImageFound := false
	for _, image := range automationClient.renderProfileReq.Images {
		if image.MimeType == "image/png" && image.Data == "ZmFrZQ==" {
			directImageFound = true
			break
		}
	}
	if !directImageFound {
		t.Fatalf("expected render profile direct image to reach automation client, got %#v", automationClient.renderProfileReq.Images)
	}

	renderProfileDebugBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "ai_render_profile_debug",
			"arguments": map[string]interface{}{
				"url": source.URL,
				"images": []map[string]interface{}{{
					"data":      "ZmFrZQ==",
					"mime_type": "image/png",
				}},
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
	if len(automationClient.renderProfileReq.Images) != 1 || automationClient.renderProfileReq.Images[0].MimeType != "image/png" {
		t.Fatalf("expected render profile debug image to reach automation client, got %#v", automationClient.renderProfileReq.Images)
	}

	pipelineBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "ai_pipeline_js_generate",
			"arguments": map[string]interface{}{
				"url":          source.URL,
				"instructions": "Wait for the main shell and reset scroll position",
				"images": []map[string]interface{}{{
					"data":      "ZmFrZQ==",
					"mime_type": "image/png",
				}},
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
	if len(automationClient.pipelineJSReq.Images) != 1 || automationClient.pipelineJSReq.Images[0].MimeType != "image/png" {
		t.Fatalf("expected pipeline JS image to reach automation client, got %#v", automationClient.pipelineJSReq.Images)
	}

	pipelineDebugBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "ai_pipeline_js_debug",
			"arguments": map[string]interface{}{
				"url": source.URL,
				"images": []map[string]interface{}{{
					"data":      "ZmFrZQ==",
					"mime_type": "image/png",
				}},
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
	if len(automationClient.pipelineJSReq.Images) != 1 || automationClient.pipelineJSReq.Images[0].MimeType != "image/png" {
		t.Fatalf("expected pipeline JS debug image to reach automation client, got %#v", automationClient.pipelineJSReq.Images)
	}

	researchRefineBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "ai_research_refine",
			"arguments": map[string]interface{}{
				"result": map[string]interface{}{
					"query":   "pricing and support commitments",
					"summary": "Original summary",
					"evidence": []map[string]interface{}{{
						"url":         "https://example.com/pricing",
						"title":       "Pricing",
						"snippet":     "Contact sales for enterprise pricing.",
						"citationUrl": "https://example.com/pricing",
					}},
					"citations": []map[string]interface{}{{
						"canonical": "https://example.com/pricing",
						"url":       "https://example.com/pricing",
					}},
				},
				"instructions": "Condense this into a concise operator brief",
			},
		}),
	}
	researchRefineResult, err := srv.handleToolCall(context.Background(), researchRefineBase)
	if err != nil {
		t.Fatalf("ai_research_refine failed: %v", err)
	}
	researchRefineResp, ok := researchRefineResult.(aiauthoring.ResearchRefineResult)
	if !ok {
		t.Fatalf("expected research refine result type, got %#v", researchRefineResult)
	}
	if researchRefineResp.Refined.ConciseSummary != "Sales-led pricing with documented support commitments." {
		t.Fatalf("unexpected concise summary: %q", researchRefineResp.Refined.ConciseSummary)
	}
	if automationClient.researchRefineCalls != 1 {
		t.Fatalf("expected single research refine call, got %d", automationClient.researchRefineCalls)
	}
	if automationClient.researchRefineReq.Instructions != "Condense this into a concise operator brief" {
		t.Fatalf("unexpected research refine instructions: %q", automationClient.researchRefineReq.Instructions)
	}

	resultDir := filepath.Join(tmpDir, "jobs", "job-export-shape")
	if err := os.MkdirAll(resultDir, 0o755); err != nil {
		t.Fatalf("mkdir result dir: %v", err)
	}
	resultPath := filepath.Join(resultDir, "result.json")
	if err := os.WriteFile(resultPath, []byte(`{"url":"https://example.com","status":200,"title":"Example","text":"Body","normalized":{"fields":{"price":{"values":["$10"]}}}}`), 0o644); err != nil {
		t.Fatalf("write result file: %v", err)
	}
	now := time.Now().UTC()
	if err := srv.store.Create(context.Background(), model.Job{
		ID:          "job-export-shape",
		Kind:        model.KindScrape,
		Status:      model.StatusSucceeded,
		CreatedAt:   now,
		UpdatedAt:   now,
		SpecVersion: 1,
		ResultPath:  resultPath,
	}); err != nil {
		t.Fatalf("create job: %v", err)
	}

	exportShapeBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "ai_export_shape",
			"arguments": map[string]interface{}{
				"jobId":        "job-export-shape",
				"format":       "md",
				"instructions": "Focus on pricing fields",
			},
		}),
	}
	exportShapeResult, err := srv.handleToolCall(context.Background(), exportShapeBase)
	if err != nil {
		t.Fatalf("ai_export_shape failed: %v", err)
	}
	exportShapeResp, ok := exportShapeResult.(aiauthoring.ExportShapeResult)
	if !ok {
		t.Fatalf("expected export shape result type, got %#v", exportShapeResult)
	}
	if len(exportShapeResp.Shape.NormalizedFields) != 1 || exportShapeResp.Shape.NormalizedFields[0] != "field.price" {
		t.Fatalf("unexpected export shape: %#v", exportShapeResp.Shape)
	}
	if automationClient.exportShapeCalls != 1 {
		t.Fatalf("expected single export shape call, got %d", automationClient.exportShapeCalls)
	}
	if automationClient.exportShapeReq.JobKind != string(model.KindScrape) {
		t.Fatalf("unexpected export shape job kind: %q", automationClient.exportShapeReq.JobKind)
	}

	automationClient.transformResult = piai.GenerateTransformResult{
		Transform: piai.BridgeTransformConfig{
			Expression: "{title: title, url: url}",
			Language:   "jmespath",
		},
		Explanation: "Projected the URL and title fields.",
		RouteID:     "openai/gpt-5.4",
		Provider:    "openai",
		Model:       "gpt-5.4",
	}
	transformBase := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "ai_transform_generate",
			"arguments": map[string]interface{}{
				"jobId":             "job-export-shape",
				"preferredLanguage": "jmespath",
				"currentTransform": map[string]interface{}{
					"expression": "[",
					"language":   "jmespath",
				},
				"instructions": "Project the URL and title for export",
			},
		}),
	}
	transformResult, err := srv.handleToolCall(context.Background(), transformBase)
	if err != nil {
		t.Fatalf("ai_transform_generate failed: %v", err)
	}
	transformResp, ok := transformResult.(aiauthoring.TransformResult)
	if !ok {
		t.Fatalf("expected transform result type, got %#v", transformResult)
	}
	if transformResp.Transform.Expression != "{title: title, url: url}" || transformResp.Transform.Language != "jmespath" {
		t.Fatalf("unexpected transform response: %#v", transformResp)
	}
	if automationClient.transformCalls != 1 {
		t.Fatalf("expected single transform call, got %d", automationClient.transformCalls)
	}
	if automationClient.transformReq.JobKind != string(model.KindScrape) {
		t.Fatalf("unexpected transform job kind: %q", automationClient.transformReq.JobKind)
	}
	if automationClient.transformReq.PreferredLanguage != "jmespath" {
		t.Fatalf("unexpected preferred language: %q", automationClient.transformReq.PreferredLanguage)
	}
}
