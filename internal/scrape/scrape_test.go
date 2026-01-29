package scrape

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func TestRun(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<html><head><title>Test Title</title></head><body><h1>Hello</h1></body></html>`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	req := Request{
		URL:       srv.URL,
		Timeout:   5 * time.Second,
		UserAgent: "SpartanTest/1.0",
		DataDir:   t.TempDir(),
	}

	result, err := Run(context.Background(), req)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.Status != http.StatusOK {
		t.Errorf("expected status 200, got %d", result.Status)
	}

	if result.Title != "Test Title" {
		t.Errorf("expected title 'Test Title', got %q", result.Title)
	}

	if result.Normalized.Title != "Test Title" {
		t.Errorf("expected normalized title 'Test Title', got %q", result.Normalized.Title)
	}
}

type wrongTypeTransformer struct {
	pipeline.BaseTransformer
}

func (w *wrongTypeTransformer) Name() string {
	return "wrong_type_transformer"
}

func (w *wrongTypeTransformer) Transform(ctx pipeline.HookContext, in pipeline.OutputInput) (pipeline.OutputOutput, error) {
	return pipeline.OutputOutput{
		Raw:        in.Raw,
		Structured: "not a Result",
	}, nil
}

func TestApplyScrapeOutputPipeline_TypeMismatch(t *testing.T) {
	registry := pipeline.NewRegistry()
	registry.RegisterTransformer(&wrongTypeTransformer{})

	result := Result{
		URL:    "http://example.com",
		Status: 200,
		Title:  "Test",
		Text:   "Test text",
	}

	ctx := pipeline.HookContext{
		Context:   context.Background(),
		RequestID: "test-123",
		Stage:     pipeline.StagePreOutput,
		Target:    pipeline.NewTarget("http://example.com", "scrape"),
		Now:       time.Now(),
		DataDir:   t.TempDir(),
		Options:   pipeline.Options{},
	}

	_, err := applyScrapeOutputPipeline(context.Background(), registry, ctx, result)
	if err == nil {
		t.Fatal("expected error for type mismatch, got nil")
	}

	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("expected KindInternal error, got %v", apperrors.KindOf(err))
	}
}
