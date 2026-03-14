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

	piai "github.com/fitchmultz/spartan-scraper/internal/ai"
	"github.com/fitchmultz/spartan-scraper/internal/aiauthoring"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
)

type fakeAuthoringRunner struct {
	previewReq               aiauthoring.PreviewRequest
	templateReq              aiauthoring.TemplateRequest
	debugReq                 aiauthoring.TemplateDebugRequest
	renderProfileReq         aiauthoring.RenderProfileRequest
	renderProfileDebugReq    aiauthoring.RenderProfileDebugRequest
	pipelineJSReq            aiauthoring.PipelineJSRequest
	pipelineJSDebugReq       aiauthoring.PipelineJSDebugRequest
	researchRefineReq        aiauthoring.ResearchRefineRequest
	exportShapeReq           aiauthoring.ExportShapeRequest
	transformReq             aiauthoring.TransformRequest
	previewResult            aiauthoring.PreviewResult
	templateResult           aiauthoring.TemplateResult
	debugResult              aiauthoring.TemplateDebugResult
	renderProfileResult      aiauthoring.RenderProfileResult
	renderProfileDebugResult aiauthoring.RenderProfileDebugResult
	pipelineJSResult         aiauthoring.PipelineJSResult
	pipelineJSDebugResult    aiauthoring.PipelineJSDebugResult
	researchRefineResult     aiauthoring.ResearchRefineResult
	exportShapeResult        aiauthoring.ExportShapeResult
	transformResult          aiauthoring.TransformResult
	previewErr               error
	templateErr              error
	debugErr                 error
	renderProfileErr         error
	renderProfileDebugErr    error
	pipelineJSErr            error
	pipelineJSDebugErr       error
	researchRefineErr        error
	exportShapeErr           error
	transformErr             error
	previewCalled            bool
	templateCalled           bool
	debugCalled              bool
	renderProfileCalled      bool
	renderProfileDebugCalled bool
	pipelineJSCalled         bool
	pipelineJSDebugCalled    bool
	researchRefineCalled     bool
	exportShapeCalled        bool
	transformCalled          bool
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

func (f *fakeAuthoringRunner) DebugRenderProfile(ctx context.Context, req aiauthoring.RenderProfileDebugRequest) (aiauthoring.RenderProfileDebugResult, error) {
	f.renderProfileDebugCalled = true
	f.renderProfileDebugReq = req
	return f.renderProfileDebugResult, f.renderProfileDebugErr
}

func (f *fakeAuthoringRunner) GeneratePipelineJS(ctx context.Context, req aiauthoring.PipelineJSRequest) (aiauthoring.PipelineJSResult, error) {
	f.pipelineJSCalled = true
	f.pipelineJSReq = req
	return f.pipelineJSResult, f.pipelineJSErr
}

func (f *fakeAuthoringRunner) DebugPipelineJS(ctx context.Context, req aiauthoring.PipelineJSDebugRequest) (aiauthoring.PipelineJSDebugResult, error) {
	f.pipelineJSDebugCalled = true
	f.pipelineJSDebugReq = req
	return f.pipelineJSDebugResult, f.pipelineJSDebugErr
}

func (f *fakeAuthoringRunner) RefineResearch(ctx context.Context, req aiauthoring.ResearchRefineRequest) (aiauthoring.ResearchRefineResult, error) {
	f.researchRefineCalled = true
	f.researchRefineReq = req
	return f.researchRefineResult, f.researchRefineErr
}

func (f *fakeAuthoringRunner) GenerateExportShape(ctx context.Context, req aiauthoring.ExportShapeRequest) (aiauthoring.ExportShapeResult, error) {
	f.exportShapeCalled = true
	f.exportShapeReq = req
	return f.exportShapeResult, f.exportShapeErr
}

func (f *fakeAuthoringRunner) GenerateTransform(ctx context.Context, req aiauthoring.TransformRequest) (aiauthoring.TransformResult, error) {
	f.transformCalled = true
	f.transformReq = req
	return f.transformResult, f.transformErr
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

func TestRunPreviewLoadsImageFiles(t *testing.T) {
	runner := &fakeAuthoringRunner{previewResult: aiauthoring.PreviewResult{Confidence: 0.9}}
	withFakeRunner(t, runner)

	tmpDir := t.TempDir()
	imagePath := filepath.Join(tmpDir, "ref.png")
	if err := os.WriteFile(imagePath, []byte("fake"), 0o644); err != nil {
		t.Fatalf("write image file: %v", err)
	}

	code, stderr := captureOutput(t, &os.Stderr, func() int {
		return RunAI(context.Background(), config.Config{}, []string{"preview", "--html", "<html><h1>Example</h1></html>", "--prompt", "Extract title", "--image-file", imagePath})
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr)
	}
	if len(runner.previewReq.Images) != 1 {
		t.Fatalf("expected single image, got %#v", runner.previewReq.Images)
	}
	if runner.previewReq.Images[0].MimeType != "image/png" {
		t.Fatalf("unexpected image mime type: %q", runner.previewReq.Images[0].MimeType)
	}
	if runner.previewReq.Images[0].Data != "ZmFrZQ==" {
		t.Fatalf("unexpected image data: %q", runner.previewReq.Images[0].Data)
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

func TestRunRenderProfileDebugLoadsSavedProfile(t *testing.T) {
	runner := &fakeAuthoringRunner{
		renderProfileDebugResult: aiauthoring.RenderProfileDebugResult{
			Issues: []string{"wait.selector matched no elements"},
			SuggestedProfile: &fetch.RenderProfile{
				Name:           "example-app",
				HostPatterns:   []string{"example.com"},
				PreferHeadless: true,
			},
		},
	}
	withFakeRunner(t, runner)

	tmpDir := t.TempDir()
	if err := fetch.SaveRenderProfilesFile(tmpDir, fetch.RenderProfilesFile{Profiles: []fetch.RenderProfile{{
		Name:         "example-app",
		HostPatterns: []string{"example.com"},
		Wait:         fetch.RenderWaitPolicy{Mode: fetch.RenderWaitModeSelector, Selector: ".missing"},
	}}}); err != nil {
		t.Fatalf("save render profile: %v", err)
	}

	code, stdout := captureOutput(t, &os.Stdout, func() int {
		return RunAI(context.Background(), config.Config{DataDir: tmpDir}, []string{"render-profile-debug", "--url", "https://example.com/app", "--profile-name", "example-app", "--instructions", "Prefer a stable wait selector", "--visual"})
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !runner.renderProfileDebugCalled {
		t.Fatal("expected render profile debug runner to be called")
	}
	if runner.renderProfileDebugReq.Profile.Name != "example-app" {
		t.Fatalf("unexpected profile name: %q", runner.renderProfileDebugReq.Profile.Name)
	}
	if !runner.renderProfileDebugReq.Visual {
		t.Fatal("expected visual flag to be forwarded")
	}
	if !strings.Contains(stdout, `"suggested_profile":`) {
		t.Fatalf("expected debug JSON output, got %s", stdout)
	}
}

func TestRunPipelineJSDebugLoadsScriptFile(t *testing.T) {
	runner := &fakeAuthoringRunner{
		pipelineJSDebugResult: aiauthoring.PipelineJSDebugResult{
			Issues: []string{"selectors[0] matched no elements"},
			SuggestedScript: &pipeline.JSTargetScript{
				Name:         "example-app",
				HostPatterns: []string{"example.com"},
				Selectors:    []string{"main"},
			},
		},
	}
	withFakeRunner(t, runner)

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "pipeline-js.json")
	if err := os.WriteFile(scriptPath, []byte(`{"name":"example-app","hostPatterns":["example.com"],"selectors":[".missing"]}`), 0o644); err != nil {
		t.Fatalf("write pipeline JS file: %v", err)
	}

	code, stdout := captureOutput(t, &os.Stdout, func() int {
		return RunAI(context.Background(), config.Config{DataDir: tmpDir}, []string{"pipeline-js-debug", "--url", "https://example.com/app", "--script-file", scriptPath, "--headless"})
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !runner.pipelineJSDebugCalled {
		t.Fatal("expected pipeline JS debug runner to be called")
	}
	if runner.pipelineJSDebugReq.Script.Name != "example-app" {
		t.Fatalf("unexpected script name: %q", runner.pipelineJSDebugReq.Script.Name)
	}
	if !runner.pipelineJSDebugReq.Headless {
		t.Fatal("expected headless flag to be forwarded")
	}
	if !strings.Contains(stdout, `"suggested_script":`) {
		t.Fatalf("expected debug JSON output, got %s", stdout)
	}
}

func TestRunResearchRefineLoadsResultFile(t *testing.T) {
	runner := &fakeAuthoringRunner{
		researchRefineResult: aiauthoring.ResearchRefineResult{
			Refined: piai.ResearchRefinedContent{
				Summary:        "Refined summary",
				ConciseSummary: "Concise summary",
				KeyFindings:    []string{"Enterprise pricing is sales-led."},
			},
			Markdown: "# Refined Research Brief\n",
		},
	}
	withFakeRunner(t, runner)

	tmpDir := t.TempDir()
	resultPath := filepath.Join(tmpDir, "research-result.json")
	if err := os.WriteFile(resultPath, []byte(`{"query":"pricing model","summary":"Original summary","confidence":0.8,"evidence":[{"url":"https://example.com/pricing","title":"Pricing","snippet":"Contact sales"}],"citations":[{"canonical":"https://example.com/pricing","url":"https://example.com/pricing"}]}`), 0o644); err != nil {
		t.Fatalf("write research result file: %v", err)
	}

	code, stdout := captureOutput(t, &os.Stdout, func() int {
		return RunAI(context.Background(), config.Config{DataDir: tmpDir}, []string{"research-refine", "--result-file", resultPath, "--instructions", "Condense this into an operator brief"})
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !runner.researchRefineCalled {
		t.Fatal("expected research refine runner to be called")
	}
	if runner.researchRefineReq.Result.Query != "pricing model" {
		t.Fatalf("unexpected research query: %q", runner.researchRefineReq.Result.Query)
	}
	if runner.researchRefineReq.Instructions != "Condense this into an operator brief" {
		t.Fatalf("unexpected research instructions: %q", runner.researchRefineReq.Instructions)
	}
	if !strings.Contains(stdout, `"markdown": "# Refined Research Brief`) {
		t.Fatalf("expected research refine JSON output, got %s", stdout)
	}
}

func TestRunExportShapeLoadsShapeAndResultFile(t *testing.T) {
	runner := &fakeAuthoringRunner{
		exportShapeResult: aiauthoring.ExportShapeResult{
			Shape: exporter.ShapeConfig{
				TopLevelFields:   []string{"url", "title"},
				NormalizedFields: []string{"field.price"},
				FieldLabels:      map[string]string{"field.price": "Price"},
			},
		},
	}
	withFakeRunner(t, runner)

	tmpDir := t.TempDir()
	resultPath := filepath.Join(tmpDir, "crawl.jsonl")
	shapePath := filepath.Join(tmpDir, "shape.json")
	if err := os.WriteFile(resultPath, []byte("{\"url\":\"https://example.com\",\"status\":200,\"title\":\"Example\",\"text\":\"Body\",\"normalized\":{\"fields\":{\"price\":{\"values\":[\"$10\"]}}}}\n{\"url\":\"https://example.com/2\",\"status\":200,\"title\":\"Example 2\",\"text\":\"Body\",\"normalized\":{\"fields\":{\"price\":{\"values\":[\"$12\"]}}}}\n"), 0o644); err != nil {
		t.Fatalf("write result file: %v", err)
	}
	if err := os.WriteFile(shapePath, []byte(`{"topLevelFields":["url"],"normalizedFields":["field.price"]}`), 0o644); err != nil {
		t.Fatalf("write shape file: %v", err)
	}

	code, stdout := captureOutput(t, &os.Stdout, func() int {
		return RunAI(context.Background(), config.Config{DataDir: tmpDir}, []string{"export-shape", "--result-file", resultPath, "--kind", string(model.KindCrawl), "--format", "csv", "--shape-file", shapePath, "--instructions", "Focus on pricing"})
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !runner.exportShapeCalled {
		t.Fatal("expected export shape runner to be called")
	}
	if runner.exportShapeReq.JobKind != model.KindCrawl {
		t.Fatalf("unexpected job kind: %q", runner.exportShapeReq.JobKind)
	}
	if runner.exportShapeReq.Format != "csv" {
		t.Fatalf("unexpected format: %q", runner.exportShapeReq.Format)
	}
	if runner.exportShapeReq.CurrentShape.TopLevelFields[0] != "url" {
		t.Fatalf("unexpected current shape: %#v", runner.exportShapeReq.CurrentShape)
	}
	if runner.exportShapeReq.Instructions != "Focus on pricing" {
		t.Fatalf("unexpected instructions: %q", runner.exportShapeReq.Instructions)
	}
	if !strings.Contains(stdout, `"shape":`) {
		t.Fatalf("expected export shape JSON output, got %s", stdout)
	}
}

func TestRunTransformLoadsResultFileAndForwardsCurrentTransform(t *testing.T) {
	runner := &fakeAuthoringRunner{
		transformResult: aiauthoring.TransformResult{
			Transform: exporter.TransformConfig{
				Expression: "{title: title, url: url}",
				Language:   "jmespath",
			},
			Preview: []any{map[string]any{"title": "Example", "url": "https://example.com"}},
		},
	}
	withFakeRunner(t, runner)

	tmpDir := t.TempDir()
	resultPath := filepath.Join(tmpDir, "crawl.jsonl")
	if err := os.WriteFile(resultPath, []byte("{\"url\":\"https://example.com\",\"title\":\"Example\",\"status\":200}\n"), 0o644); err != nil {
		t.Fatalf("write result file: %v", err)
	}

	code, stdout := captureOutput(t, &os.Stdout, func() int {
		return RunAI(context.Background(), config.Config{DataDir: tmpDir}, []string{"transform", "--result-file", resultPath, "--language", "jmespath", "--expression", "[", "--instructions", "Project the URL and title"})
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !runner.transformCalled {
		t.Fatal("expected transform runner to be called")
	}
	if runner.transformReq.PreferredLanguage != "jmespath" {
		t.Fatalf("unexpected preferred language: %q", runner.transformReq.PreferredLanguage)
	}
	if runner.transformReq.CurrentTransform.Expression != "[" || runner.transformReq.CurrentTransform.Language != "jmespath" {
		t.Fatalf("unexpected current transform: %#v", runner.transformReq.CurrentTransform)
	}
	if runner.transformReq.Instructions != "Project the URL and title" {
		t.Fatalf("unexpected instructions: %q", runner.transformReq.Instructions)
	}
	if !strings.Contains(stdout, `"transform":`) {
		t.Fatalf("expected transform JSON output, got %s", stdout)
	}
}

func TestRunTransformLoadsCurrentTransformFromSchedule(t *testing.T) {
	runner := &fakeAuthoringRunner{
		transformResult: aiauthoring.TransformResult{
			Transform: exporter.TransformConfig{
				Expression: "{title: title}",
				Language:   "jmespath",
			},
		},
	}
	withFakeRunner(t, runner)

	tmpDir := t.TempDir()
	resultPath := filepath.Join(tmpDir, "crawl.jsonl")
	if err := os.WriteFile(resultPath, []byte("{\"url\":\"https://example.com\",\"title\":\"Example\",\"status\":200}\n"), 0o644); err != nil {
		t.Fatalf("write result file: %v", err)
	}
	storage := scheduler.NewExportStorage(tmpDir)
	created, err := storage.Add(scheduler.ExportSchedule{
		Name:    "Projected Export",
		Enabled: true,
		Filters: scheduler.ExportFilters{JobKinds: []string{"scrape"}},
		Export: scheduler.ExportConfig{
			Format:          "csv",
			DestinationType: "local",
			Transform: exporter.TransformConfig{
				Expression: "{title: title, url: url}",
				Language:   "jsonata",
			},
		},
	})
	if err != nil {
		t.Fatalf("storage.Add(): %v", err)
	}

	code, stderr := captureOutput(t, &os.Stderr, func() int {
		return RunAI(context.Background(), config.Config{DataDir: tmpDir}, []string{"transform", "--result-file", resultPath, "--schedule-id", created.ID})
	})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr)
	}
	if runner.transformReq.CurrentTransform.Expression != "{title: title, url: url}" {
		t.Fatalf("unexpected current transform: %#v", runner.transformReq.CurrentTransform)
	}
	if runner.transformReq.PreferredLanguage != "jsonata" {
		t.Fatalf("unexpected preferred language: %q", runner.transformReq.PreferredLanguage)
	}
}
