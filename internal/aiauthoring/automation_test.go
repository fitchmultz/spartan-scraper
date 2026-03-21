// Package aiauthoring verifies resolved-goal behavior for automation authoring flows.
//
// Purpose:
// - Ensure automation generation and debugging results expose the exact goal Spartan sent to the AI model.
//
// Responsibilities:
// - Cover explicit versus derived goal sources, retry stability, and debug scaffold reporting.
//
// Scope:
// - Render-profile and pipeline-JS resolved-goal tests only.
//
// Usage:
// - Run with `go test ./internal/aiauthoring`.
//
// Invariants/Assumptions:
// - `resolved_goal.text` must match the original model instruction string.
// - `resolved_goal.source` must be `explicit` when operators supplied guidance and `derived` otherwise.
package aiauthoring

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	piai "github.com/fitchmultz/spartan-scraper/internal/ai"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

type fakeAutomationResolvedGoalClient struct {
	renderProfileResults []piai.GenerateRenderProfileResult
	pipelineJSResults    []piai.GeneratePipelineJSResult
	renderProfileReq     piai.GenerateRenderProfileRequest
	pipelineJSReq        piai.GeneratePipelineJSRequest
	renderProfileCalls   int
	pipelineJSCalls      int
}

func (f *fakeAutomationResolvedGoalClient) GenerateRenderProfile(_ context.Context, req piai.GenerateRenderProfileRequest) (piai.GenerateRenderProfileResult, error) {
	f.renderProfileCalls++
	f.renderProfileReq = req
	idx := f.renderProfileCalls - 1
	if idx >= len(f.renderProfileResults) {
		idx = len(f.renderProfileResults) - 1
	}
	return f.renderProfileResults[idx], nil
}

func (f *fakeAutomationResolvedGoalClient) GeneratePipelineJS(_ context.Context, req piai.GeneratePipelineJSRequest) (piai.GeneratePipelineJSResult, error) {
	f.pipelineJSCalls++
	f.pipelineJSReq = req
	idx := f.pipelineJSCalls - 1
	if idx >= len(f.pipelineJSResults) {
		idx = len(f.pipelineJSResults) - 1
	}
	return f.pipelineJSResults[idx], nil
}

func (f *fakeAutomationResolvedGoalClient) GenerateResearchRefinement(context.Context, piai.ResearchRefineRequest) (piai.ResearchRefineResult, error) {
	return piai.ResearchRefineResult{}, nil
}

func (f *fakeAutomationResolvedGoalClient) GenerateExportShape(context.Context, piai.ExportShapeRequest) (piai.ExportShapeResult, error) {
	return piai.ExportShapeResult{}, nil
}

func (f *fakeAutomationResolvedGoalClient) GenerateTransform(context.Context, piai.GenerateTransformRequest) (piai.GenerateTransformResult, error) {
	return piai.GenerateTransformResult{}, nil
}

func newAutomationResolvedGoalService(t *testing.T, client AutomationClient) *Service {
	t.Helper()
	return NewServiceWithAutomationClient(
		config.Config{
			DataDir:            t.TempDir(),
			UserAgent:          "test-agent",
			RequestTimeoutSecs: 30,
			AI:                 config.AIConfig{Enabled: true, RequestTimeoutSecs: 30},
		},
		nil,
		client,
		true,
	)
}

func TestGenerateRenderProfileResolvedGoalUsesExplicitInstructions(t *testing.T) {
	client := &fakeAutomationResolvedGoalClient{renderProfileResults: []piai.GenerateRenderProfileResult{{
		Profile: piai.BridgeRenderProfile{
			PreferHeadless: true,
			Wait:           piai.BridgeRenderWaitPolicy{Mode: "selector", Selector: "main"},
		},
	}}}
	service := newAutomationResolvedGoalService(t, client)
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><main>Dashboard</main></body></html>`))
	}))
	defer source.Close()

	result, err := service.GenerateRenderProfile(context.Background(), RenderProfileRequest{
		URL:          source.URL,
		Name:         "example-app",
		HostPatterns: []string{"example.com"},
		Instructions: "Wait for the dashboard shell and prefer headless mode",
	})
	if err != nil {
		t.Fatalf("GenerateRenderProfile() error = %v", err)
	}
	if result.ResolvedGoal == nil {
		t.Fatal("expected resolved goal")
	}
	if result.ResolvedGoal.Source != resolvedGoalSourceExplicit {
		t.Fatalf("expected explicit goal source, got %#v", result.ResolvedGoal)
	}
	if result.ResolvedGoal.Text != "Wait for the dashboard shell and prefer headless mode" {
		t.Fatalf("unexpected resolved goal text: %q", result.ResolvedGoal.Text)
	}
	if client.renderProfileReq.Instructions != result.ResolvedGoal.Text {
		t.Fatalf("expected model request instructions to match resolved goal, got %q", client.renderProfileReq.Instructions)
	}
}

func TestGenerateRenderProfileResolvedGoalStaysStableAcrossRetry(t *testing.T) {
	client := &fakeAutomationResolvedGoalClient{renderProfileResults: []piai.GenerateRenderProfileResult{
		{
			Profile: piai.BridgeRenderProfile{Wait: piai.BridgeRenderWaitPolicy{Mode: "selector", Selector: ".missing"}},
		},
		{
			Profile: piai.BridgeRenderProfile{PreferHeadless: true, Wait: piai.BridgeRenderWaitPolicy{Mode: "selector", Selector: "main"}},
		},
	}}
	service := newAutomationResolvedGoalService(t, client)
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><main>Dashboard</main><script src="/app.js"></script></body></html>`))
	}))
	defer source.Close()

	result, err := service.GenerateRenderProfile(context.Background(), RenderProfileRequest{URL: source.URL})
	if err != nil {
		t.Fatalf("GenerateRenderProfile() error = %v", err)
	}
	if client.renderProfileCalls != 2 {
		t.Fatalf("expected retry after invalid candidate, got %d calls", client.renderProfileCalls)
	}
	if strings.TrimSpace(client.renderProfileReq.Feedback) == "" {
		t.Fatal("expected retry feedback on final request")
	}
	if result.ResolvedGoal == nil {
		t.Fatal("expected resolved goal")
	}
	if result.ResolvedGoal.Source != resolvedGoalSourceDerived {
		t.Fatalf("expected derived goal source, got %#v", result.ResolvedGoal)
	}
	if !strings.Contains(result.ResolvedGoal.Text, "Generate a render profile") {
		t.Fatalf("unexpected derived goal text: %q", result.ResolvedGoal.Text)
	}
	if client.renderProfileReq.Instructions != result.ResolvedGoal.Text {
		t.Fatalf("expected retry to preserve resolved goal text, got %q", client.renderProfileReq.Instructions)
	}
}

func TestGeneratePipelineJSResolvedGoalUsesDerivedInstructions(t *testing.T) {
	client := &fakeAutomationResolvedGoalClient{pipelineJSResults: []piai.GeneratePipelineJSResult{{
		Script: piai.BridgePipelineJSScript{Selectors: []string{"main"}},
	}}}
	service := newAutomationResolvedGoalService(t, client)
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><main>Dashboard</main><script src="/app.js"></script></body></html>`))
	}))
	defer source.Close()

	result, err := service.GeneratePipelineJS(context.Background(), PipelineJSRequest{URL: source.URL})
	if err != nil {
		t.Fatalf("GeneratePipelineJS() error = %v", err)
	}
	if result.ResolvedGoal == nil {
		t.Fatal("expected resolved goal")
	}
	if result.ResolvedGoal.Source != resolvedGoalSourceDerived {
		t.Fatalf("expected derived goal source, got %#v", result.ResolvedGoal)
	}
	if !strings.Contains(result.ResolvedGoal.Text, "pipeline JS") {
		t.Fatalf("unexpected resolved goal text: %q", result.ResolvedGoal.Text)
	}
	if client.pipelineJSReq.Instructions != result.ResolvedGoal.Text {
		t.Fatalf("expected model request instructions to match resolved goal, got %q", client.pipelineJSReq.Instructions)
	}
}

func TestDebugRenderProfileResolvedGoalUsesDebugScaffoldWhenGuidanceOmitted(t *testing.T) {
	client := &fakeAutomationResolvedGoalClient{renderProfileResults: []piai.GenerateRenderProfileResult{{
		Profile: piai.BridgeRenderProfile{
			PreferHeadless: true,
			Wait:           piai.BridgeRenderWaitPolicy{Mode: "selector", Selector: "main"},
		},
	}}}
	service := newAutomationResolvedGoalService(t, client)
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><main>Dashboard</main></body></html>`))
	}))
	defer source.Close()

	result, err := service.DebugRenderProfile(context.Background(), RenderProfileDebugRequest{
		URL: source.URL,
		Profile: fetch.RenderProfile{
			Name:         "example-app",
			HostPatterns: []string{"127.0.0.1"},
			Wait:         fetch.RenderWaitPolicy{Mode: fetch.RenderWaitModeSelector, Selector: ".missing"},
		},
	})
	if err != nil {
		t.Fatalf("DebugRenderProfile() error = %v", err)
	}
	if result.ResolvedGoal == nil {
		t.Fatal("expected resolved goal")
	}
	if result.ResolvedGoal.Source != resolvedGoalSourceDerived {
		t.Fatalf("expected derived goal source, got %#v", result.ResolvedGoal)
	}
	if !strings.Contains(result.ResolvedGoal.Text, `Tune the render profile named "example-app"`) {
		t.Fatalf("unexpected resolved goal text: %q", result.ResolvedGoal.Text)
	}
	if client.renderProfileReq.Instructions != result.ResolvedGoal.Text {
		t.Fatalf("expected debug request instructions to match resolved goal, got %q", client.renderProfileReq.Instructions)
	}
}

func TestDebugPipelineJSResolvedGoalIncludesExplicitGuidance(t *testing.T) {
	client := &fakeAutomationResolvedGoalClient{pipelineJSResults: []piai.GeneratePipelineJSResult{{
		Script: piai.BridgePipelineJSScript{Selectors: []string{"main"}, PostNav: "window.scrollTo(0, 0);"},
	}}}
	service := newAutomationResolvedGoalService(t, client)
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body><main>Dashboard</main></body></html>`))
	}))
	defer source.Close()

	result, err := service.DebugPipelineJS(context.Background(), PipelineJSDebugRequest{
		URL:          source.URL,
		Instructions: "Prefer selector waits over custom JavaScript",
		Script: pipeline.JSTargetScript{
			Name:         "example-app",
			HostPatterns: []string{"127.0.0.1"},
			Selectors:    []string{".missing"},
		},
	})
	if err != nil {
		t.Fatalf("DebugPipelineJS() error = %v", err)
	}
	if result.ResolvedGoal == nil {
		t.Fatal("expected resolved goal")
	}
	if result.ResolvedGoal.Source != resolvedGoalSourceExplicit {
		t.Fatalf("expected explicit goal source, got %#v", result.ResolvedGoal)
	}
	if !strings.Contains(result.ResolvedGoal.Text, "Operator guidance: Prefer selector waits over custom JavaScript") {
		t.Fatalf("unexpected resolved goal text: %q", result.ResolvedGoal.Text)
	}
	if client.pipelineJSReq.Instructions != result.ResolvedGoal.Text {
		t.Fatalf("expected debug request instructions to match resolved goal, got %q", client.pipelineJSReq.Instructions)
	}
}
