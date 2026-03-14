package aiauthoring

import (
	"context"
	"testing"

	piai "github.com/fitchmultz/spartan-scraper/internal/ai"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
)

type fakeTransformAutomationClient struct {
	results []piai.GenerateTransformResult
	calls   int
	lastReq piai.GenerateTransformRequest
}

func (f *fakeTransformAutomationClient) GenerateRenderProfile(context.Context, piai.GenerateRenderProfileRequest) (piai.GenerateRenderProfileResult, error) {
	return piai.GenerateRenderProfileResult{}, nil
}

func (f *fakeTransformAutomationClient) GeneratePipelineJS(context.Context, piai.GeneratePipelineJSRequest) (piai.GeneratePipelineJSResult, error) {
	return piai.GeneratePipelineJSResult{}, nil
}

func (f *fakeTransformAutomationClient) GenerateResearchRefinement(context.Context, piai.ResearchRefineRequest) (piai.ResearchRefineResult, error) {
	return piai.ResearchRefineResult{}, nil
}

func (f *fakeTransformAutomationClient) GenerateExportShape(context.Context, piai.ExportShapeRequest) (piai.ExportShapeResult, error) {
	return piai.ExportShapeResult{}, nil
}

func (f *fakeTransformAutomationClient) GenerateTransform(_ context.Context, req piai.GenerateTransformRequest) (piai.GenerateTransformResult, error) {
	f.calls++
	f.lastReq = req
	idx := f.calls - 1
	if idx >= len(f.results) {
		idx = len(f.results) - 1
	}
	return f.results[idx], nil
}

func TestGenerateTransformBuildsValidatedPreview(t *testing.T) {
	client := &fakeTransformAutomationClient{results: []piai.GenerateTransformResult{{
		Transform: piai.BridgeTransformConfig{
			Expression: "{title: title, url: url}",
			Language:   "jmespath",
		},
		Explanation: "Projected the export-friendly fields.",
		RouteID:     "openai/gpt-5.4",
		Provider:    "openai",
		Model:       "gpt-5.4",
	}}}
	service := NewServiceWithAutomationClient(
		config.Config{AI: config.AIConfig{Enabled: true, RequestTimeoutSecs: 30}},
		nil,
		client,
		true,
	)

	result, err := service.GenerateTransform(context.Background(), TransformRequest{
		RawResult:         []byte("{\"url\":\"https://example.com\",\"title\":\"Example\",\"status\":200}\n{\"url\":\"https://example.com/2\",\"title\":\"Example 2\",\"status\":200}\n"),
		PreferredLanguage: "jmespath",
		Instructions:      "Project the URL and title for export.",
	})
	if err != nil {
		t.Fatalf("GenerateTransform() error = %v", err)
	}
	if client.calls != 1 {
		t.Fatalf("expected single automation call, got %d", client.calls)
	}
	if got := len(client.lastReq.SampleRecords); got != 2 {
		t.Fatalf("expected 2 sample records, got %d", got)
	}
	if client.lastReq.PreferredLanguage != "jmespath" {
		t.Fatalf("unexpected preferred language: %q", client.lastReq.PreferredLanguage)
	}
	if result.Transform.Expression != "{title: title, url: url}" || result.Transform.Language != "jmespath" {
		t.Fatalf("unexpected transform: %#v", result.Transform)
	}
	if len(result.Preview) != 2 {
		t.Fatalf("expected preview for both sample records, got %#v", result.Preview)
	}
	first, ok := result.Preview[0].(map[string]any)
	if !ok {
		t.Fatalf("expected preview map, got %#v", result.Preview[0])
	}
	if first["title"] != "Example" || first["url"] != "https://example.com" {
		t.Fatalf("unexpected preview item: %#v", first)
	}
	if result.InputStats.SampleRecordCount != 2 {
		t.Fatalf("unexpected sample record count: %#v", result.InputStats)
	}
	if result.InputStats.FieldPathCount == 0 {
		t.Fatalf("expected non-zero field path count: %#v", result.InputStats)
	}
}

func TestGenerateTransformRetriesAfterInvalidCandidate(t *testing.T) {
	client := &fakeTransformAutomationClient{results: []piai.GenerateTransformResult{
		{
			Transform: piai.BridgeTransformConfig{
				Expression: "{title: title}",
				Language:   "jsonata",
			},
		},
		{
			Transform: piai.BridgeTransformConfig{
				Expression: "{title: title}",
				Language:   "jmespath",
			},
		},
	}}
	service := NewServiceWithAutomationClient(
		config.Config{AI: config.AIConfig{Enabled: true, RequestTimeoutSecs: 30}},
		nil,
		client,
		true,
	)

	result, err := service.GenerateTransform(context.Background(), TransformRequest{
		RawResult:         []byte("{\"title\":\"Example\"}\n"),
		PreferredLanguage: "jmespath",
		CurrentTransform: exporter.TransformConfig{
			Expression: "[",
			Language:   "jmespath",
		},
	})
	if err != nil {
		t.Fatalf("GenerateTransform() error = %v", err)
	}
	if client.calls != 2 {
		t.Fatalf("expected retry after invalid candidate, got %d calls", client.calls)
	}
	if client.lastReq.Feedback == "" {
		t.Fatal("expected retry feedback to be sent to automation client")
	}
	if len(result.Issues) == 0 {
		t.Fatal("expected diagnostic issues from invalid current transform")
	}
	if result.Transform.Language != "jmespath" {
		t.Fatalf("expected retried language to match preference, got %#v", result.Transform)
	}
}
