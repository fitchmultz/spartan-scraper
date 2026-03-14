package ai

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/aiauthoring"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

type fakeAuthoringRunner struct {
	previewReq          aiauthoring.PreviewRequest
	templateReq         aiauthoring.TemplateRequest
	debugReq            aiauthoring.TemplateDebugRequest
	renderProfileReq    aiauthoring.RenderProfileRequest
	pipelineJSReq       aiauthoring.PipelineJSRequest
	previewResult       aiauthoring.PreviewResult
	templateResult      aiauthoring.TemplateResult
	debugResult         aiauthoring.TemplateDebugResult
	renderProfileResult aiauthoring.RenderProfileResult
	pipelineJSResult    aiauthoring.PipelineJSResult
	previewErr          error
	templateErr         error
	debugErr            error
	renderProfileErr    error
	pipelineJSErr       error
	previewCalled       bool
	templateCalled      bool
	debugCalled         bool
	renderProfileCalled bool
	pipelineJSCalled    bool
}

func (f *fakeAuthoringRunner) Preview(ctx context.Context, req aiauthoring.PreviewRequest) (aiauthoring.PreviewResult, error) {
	f.previewCalled = true
	f.previewReq = req
	return f.previewResult, f.previewErr
}

func (f *fakeAuthoringRunner) GenerateTemplate(ctx context.Context, req aiauthoring.TemplateRequest) (aiauthoring.TemplateResult, error) {
	f.templateCalled = true
	f.templateReq = req
	return f.templateResult, f.templateErr
}

func (f *fakeAuthoringRunner) DebugTemplate(ctx context.Context, req aiauthoring.TemplateDebugRequest) (aiauthoring.TemplateDebugResult, error) {
	f.debugCalled = true
	f.debugReq = req
	return f.debugResult, f.debugErr
}

func (f *fakeAuthoringRunner) GenerateRenderProfile(ctx context.Context, req aiauthoring.RenderProfileRequest) (aiauthoring.RenderProfileResult, error) {
	f.renderProfileCalled = true
	f.renderProfileReq = req
	return f.renderProfileResult, f.renderProfileErr
}

func (f *fakeAuthoringRunner) GeneratePipelineJS(ctx context.Context, req aiauthoring.PipelineJSRequest) (aiauthoring.PipelineJSResult, error) {
	f.pipelineJSCalled = true
	f.pipelineJSReq = req
	return f.pipelineJSResult, f.pipelineJSErr
}

func withFakeRunner(t *testing.T, runner *fakeAuthoringRunner) {
	t.Helper()
	original := newAuthoringRunner
	newAuthoringRunner = func(cfg config.Config) (authoringRunner, error) {
		return runner, nil
	}
	t.Cleanup(func() {
		newAuthoringRunner = original
	})
}

func captureOutput(t *testing.T, target **os.File, fn func() int) (int, string) {
	t.Helper()
	original := *target
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	*target = w
	code := fn()
	_ = w.Close()
	*target = original
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return code, buf.String()
}

func TestRunPreviewRequiresURLOrHTML(t *testing.T) {
	runner := &fakeAuthoringRunner{previewErr: errors.New("url or html is required")}
	withFakeRunner(t, runner)
	code, stderr := captureOutput(t, &os.Stderr, func() int {
		return RunAI(context.Background(), config.Config{}, []string{"preview", "--prompt", "Extract title"})
	})
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr, "url or html is required") {
		t.Fatalf("expected missing input error, got %q", stderr)
	}
}

func TestRunPreviewCallsRunnerAndPrintsJSON(t *testing.T) {
	runner := &fakeAuthoringRunner{
		previewResult: aiauthoring.PreviewResult{
			Fields:     map[string]extract.FieldValue{"title": {Values: []string{"Example"}, Source: extract.FieldSourceLLM}},
			Confidence: 0.9,
			RouteID:    "openai/gpt-5.4",
			Provider:   "openai",
			Model:      "gpt-5.4",
		},
	}
	withFakeRunner(t, runner)
	code, stdout := captureOutput(t, &os.Stdout, func() int {
		return RunAI(context.Background(), config.Config{}, []string{"preview", "--url", "https://example.com", "--prompt", "Extract title", "--fields", "title,price"})
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !runner.previewCalled {
		t.Fatal("expected preview runner to be called")
	}
	if runner.previewReq.URL != "https://example.com" {
		t.Fatalf("unexpected preview URL: %q", runner.previewReq.URL)
	}
	if runner.previewReq.Prompt != "Extract title" {
		t.Fatalf("unexpected prompt: %q", runner.previewReq.Prompt)
	}
	if len(runner.previewReq.Fields) != 2 || runner.previewReq.Fields[0] != "title" || runner.previewReq.Fields[1] != "price" {
		t.Fatalf("unexpected fields: %#v", runner.previewReq.Fields)
	}
	if !strings.Contains(stdout, `"route_id": "openai/gpt-5.4"`) {
		t.Fatalf("expected JSON output, got %s", stdout)
	}
}

func TestRunTemplateReadsHTMLFileAndWritesOutputFile(t *testing.T) {
	runner := &fakeAuthoringRunner{
		templateResult: aiauthoring.TemplateResult{
			Template: extract.Template{Name: "product-template", Selectors: []extract.SelectorRule{{Name: "title", Selector: "h1", Attr: "text"}}},
			RouteID:  "openai/gpt-5.4",
		},
	}
	withFakeRunner(t, runner)
	tmpDir := t.TempDir()
	htmlPath := filepath.Join(tmpDir, "page.html")
	if err := os.WriteFile(htmlPath, []byte("<html><h1>Example</h1></html>"), 0o644); err != nil {
		t.Fatalf("write html file: %v", err)
	}
	outPath := filepath.Join(tmpDir, "template.json")

	code, stderr := captureOutput(t, &os.Stderr, func() int {
		return RunAI(context.Background(), config.Config{}, []string{"template", "--html-file", htmlPath, "--description", "Extract title", "--sample-fields", "title,price", "--out", outPath})
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr)
	}
	if !runner.templateCalled {
		t.Fatal("expected template runner to be called")
	}
	if !strings.Contains(runner.templateReq.HTML, "<h1>Example</h1>") {
		t.Fatalf("expected HTML from file, got %q", runner.templateReq.HTML)
	}
	if runner.templateReq.Description != "Extract title" {
		t.Fatalf("unexpected description: %q", runner.templateReq.Description)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !strings.Contains(string(data), `"name": "product-template"`) {
		t.Fatalf("unexpected output file contents: %s", string(data))
	}
}

func TestRunTemplateDebugLoadsTemplateByName(t *testing.T) {
	runner := &fakeAuthoringRunner{
		debugResult: aiauthoring.TemplateDebugResult{
			Issues: []string{"selector title matched no elements"},
			SuggestedTemplate: &extract.Template{
				Name:      "product",
				Selectors: []extract.SelectorRule{{Name: "title", Selector: "h1", Attr: "text"}},
			},
		},
	}
	withFakeRunner(t, runner)

	tmpDir := t.TempDir()
	registryPath := filepath.Join(tmpDir, "extract_templates.json")
	if err := os.WriteFile(registryPath, []byte(`{"templates":[{"name":"product","selectors":[{"name":"title","selector":".missing","attr":"text"}]}]}`), 0o644); err != nil {
		t.Fatalf("write template registry: %v", err)
	}

	code, stdout := captureOutput(t, &os.Stdout, func() int {
		return RunAI(context.Background(), config.Config{DataDir: tmpDir}, []string{"template-debug", "--html", "<html><h1>Widget</h1></html>", "--template-name", "product", "--instructions", "Prefer the visible h1", "--visual"})
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !runner.debugCalled {
		t.Fatal("expected debug runner to be called")
	}
	if runner.debugReq.Template.Name != "product" {
		t.Fatalf("unexpected template name: %q", runner.debugReq.Template.Name)
	}
	if !runner.debugReq.Visual {
		t.Fatal("expected visual mode to be forwarded")
	}
	if !strings.Contains(stdout, `"issues": [`) {
		t.Fatalf("expected debug JSON output, got %s", stdout)
	}
}

func TestRunRenderProfileForwardsOptions(t *testing.T) {
	runner := &fakeAuthoringRunner{
		renderProfileResult: aiauthoring.RenderProfileResult{
			Profile: fetch.RenderProfile{Name: "example.com", HostPatterns: []string{"example.com"}, PreferHeadless: true},
		},
	}
	withFakeRunner(t, runner)

	code, stdout := captureOutput(t, &os.Stdout, func() int {
		return RunAI(context.Background(), config.Config{}, []string{"render-profile", "--url", "https://example.com/app", "--name", "example-app", "--host-patterns", "example.com,*.example.com", "--instructions", "Wait for the dashboard shell", "--headless", "--playwright", "--visual"})
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !runner.renderProfileCalled {
		t.Fatal("expected render profile runner to be called")
	}
	if runner.renderProfileReq.Name != "example-app" {
		t.Fatalf("unexpected profile name: %q", runner.renderProfileReq.Name)
	}
	if len(runner.renderProfileReq.HostPatterns) != 2 {
		t.Fatalf("unexpected host patterns: %#v", runner.renderProfileReq.HostPatterns)
	}
	if !runner.renderProfileReq.Visual || !runner.renderProfileReq.Headless || !runner.renderProfileReq.UsePlaywright {
		t.Fatalf("expected browser flags to be forwarded: %#v", runner.renderProfileReq)
	}
	if !strings.Contains(stdout, `"profile":`) {
		t.Fatalf("expected profile JSON output, got %s", stdout)
	}
}

func TestRunPipelineJSForwardsOptions(t *testing.T) {
	runner := &fakeAuthoringRunner{
		pipelineJSResult: aiauthoring.PipelineJSResult{
			Script: pipeline.JSTargetScript{Name: "example.com", HostPatterns: []string{"example.com"}, Selectors: []string{"main"}},
		},
	}
	withFakeRunner(t, runner)

	code, stdout := captureOutput(t, &os.Stdout, func() int {
		return RunAI(context.Background(), config.Config{}, []string{"pipeline-js", "--url", "https://example.com/app", "--instructions", "Wait for the main dashboard", "--visual"})
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !runner.pipelineJSCalled {
		t.Fatal("expected pipeline JS runner to be called")
	}
	if runner.pipelineJSReq.URL != "https://example.com/app" {
		t.Fatalf("unexpected URL: %q", runner.pipelineJSReq.URL)
	}
	if !runner.pipelineJSReq.Visual {
		t.Fatal("expected visual flag to be forwarded")
	}
	if !strings.Contains(stdout, `"script":`) {
		t.Fatalf("expected script JSON output, got %s", stdout)
	}
}
