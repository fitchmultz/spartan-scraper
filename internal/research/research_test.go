// Package research provides integration tests for multi-source research workflows.
// Tests cover evidence gathering, deduplication, summarization, context cancellation,
// and partial failure handling across multiple target URLs.
// Does NOT test the pipeline output processing (pipeline_test.go covers that).
package research

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/simhash"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"Hello World!", []string{"hello", "world"}},
		{"Multiple tokens, with symbols.", []string{"multiple", "tokens", "with", "symbols"}},
		{"Duplicate duplicate", []string{"duplicate"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := tokenize(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("tokenize(%q) = %v; want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSimHash(t *testing.T) {
	h1 := simhash.Compute("this is a test")
	h2 := simhash.Compute("this is another test")
	h3 := simhash.Compute("completely different text")

	d12 := simhash.HammingDistance(h1, h2)
	d13 := simhash.HammingDistance(h1, h3)

	if d12 >= d13 {
		t.Errorf("expected similar text to have smaller hamming distance: d12=%d, d13=%d", d12, d13)
	}
}

func TestDedupEvidence(t *testing.T) {
	items := []Evidence{
		{URL: "u1", SimHash: simhash.Compute("duplicate text")},
		{URL: "u2", SimHash: simhash.Compute("duplicate text")},
		{URL: "u3", SimHash: simhash.Compute("unique text")},
	}

	deduped := dedupEvidence(items, 3)
	if len(deduped) != 2 {
		t.Errorf("expected 2 items after dedup, got %d", len(deduped))
	}
}

func TestRun(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Results for test</h1><p>This is a test page with evidence.</p></body></html>`)
	}))
	defer srv.Close()

	req := Request{
		Query:    "test evidence",
		URLs:     []string{srv.URL},
		MaxDepth: 0, // scrape only
		Timeout:  5 * time.Second,
		DataDir:  t.TempDir(),
	}

	result, err := Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.Query != "test evidence" {
		t.Errorf("expected query 'test evidence', got %q", result.Query)
	}

	if len(result.Evidence) == 0 {
		t.Errorf("expected evidence, got 0 items")
	}

	if result.Summary == "" {
		t.Errorf("expected summary, got empty")
	}
}

func TestRunContextCancellation(t *testing.T) {
	// Create a server that delays responses
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		fmt.Fprint(w, `<html><body><h1>Results for test</h1><p>This is a test page with evidence.</p></body></html>`)
	}))
	defer srv.Close()

	// Create a cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Start the research in a goroutine and cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	req := Request{
		Query:    "test evidence",
		URLs:     []string{srv.URL, srv.URL, srv.URL},
		MaxDepth: 0,
		Timeout:  5 * time.Second,
		DataDir:  t.TempDir(),
	}

	_, err := Run(ctx, req)
	if err == nil {
		t.Fatal("expected error due to context cancellation, got nil")
	}

	// Verify it's a context cancellation error
	if ctx.Err() == nil {
		t.Error("expected context to be cancelled")
	}
}

func TestRunAllTargetsFail(t *testing.T) {
	// Use invalid URLs that will cause connection errors
	req := Request{
		Query:    "test evidence",
		URLs:     []string{"http://localhost:1", "http://localhost:2"},
		MaxDepth: 0,
		Timeout:  1 * time.Second,
		DataDir:  t.TempDir(),
	}

	_, err := Run(context.Background(), req)
	if err == nil {
		t.Fatal("expected error when all targets fail, got nil")
	}

	expectedMsg := "all research targets failed"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

type fakeResearchAIProvider struct {
	extractResult extract.AIExtractResult
}

func (f *fakeResearchAIProvider) Extract(ctx context.Context, req extract.AIExtractRequest) (extract.AIExtractResult, error) {
	return f.extractResult, nil
}

func (f *fakeResearchAIProvider) GenerateTemplate(ctx context.Context, req extract.AITemplateGenerateRequest) (extract.AITemplateGenerateResult, error) {
	return extract.AITemplateGenerateResult{}, nil
}

func (f *fakeResearchAIProvider) HealthCheck(ctx context.Context) error {
	return nil
}

func (f *fakeResearchAIProvider) RouteFingerprint(capability string) string {
	return "test-route"
}

func TestRunIncludesAIExtractedFieldsInEvidence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Enterprise Pricing</h1><p>Talk to sales for pricing details.</p></body></html>`)
	}))
	defer srv.Close()

	provider := &fakeResearchAIProvider{
		extractResult: extract.AIExtractResult{
			Fields: map[string]extract.FieldValue{
				"pricing_model": {
					Values: []string{"Usage-based enterprise contract"},
					Source: extract.FieldSourceLLM,
				},
				"support_terms": {
					Values: []string{"Dedicated support with SLA"},
					Source: extract.FieldSourceLLM,
				},
			},
			Confidence: 0.91,
		},
	}
	aiExtractor := extract.NewAIExtractorWithProvider(
		config.AIConfig{Enabled: true, Routing: config.DefaultAIRoutingConfig()},
		t.TempDir(),
		provider,
	)

	result, err := Run(context.Background(), Request{
		Query:       "pricing model",
		URLs:        []string{srv.URL},
		MaxDepth:    0,
		Timeout:     5 * time.Second,
		DataDir:     t.TempDir(),
		AIExtractor: aiExtractor,
		Extract: extract.ExtractOptions{
			AI: &extract.AIExtractOptions{
				Enabled: true,
				Mode:    extract.AIModeNaturalLanguage,
				Prompt:  "Extract the pricing model and support terms",
			},
		},
	})
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if len(result.Evidence) != 1 {
		t.Fatalf("expected 1 evidence item, got %d", len(result.Evidence))
	}
	fields := result.Evidence[0].Fields
	if len(fields) < 2 {
		t.Fatalf("expected AI fields to be merged into evidence, got %d total fields", len(fields))
	}
	if values := fields["pricing_model"].Values; len(values) != 1 || values[0] != "Usage-based enterprise contract" {
		t.Fatalf("unexpected pricing_model values: %#v", values)
	}
	if values := fields["support_terms"].Values; len(values) != 1 || values[0] != "Dedicated support with SLA" {
		t.Fatalf("unexpected support_terms values: %#v", values)
	}
	if !strings.Contains(result.Summary, "pricing model") {
		t.Fatalf("expected summary to reference AI field summary, got %q", result.Summary)
	}
}

func TestRunPartialFailure(t *testing.T) {
	// Create two servers: one that succeeds, one that fails
	successSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><body><h1>Success</h1><p>This is successful evidence.</p></body></html>`)
	}))
	defer successSrv.Close()

	failSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer failSrv.Close()

	req := Request{
		Query:    "test evidence",
		URLs:     []string{failSrv.URL, successSrv.URL, failSrv.URL},
		MaxDepth: 0,
		Timeout:  5 * time.Second,
		DataDir:  t.TempDir(),
	}

	result, err := Run(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error with partial failure, got %v", err)
	}

	if len(result.Evidence) == 0 {
		t.Error("expected evidence from successful target, got none")
	}
}
