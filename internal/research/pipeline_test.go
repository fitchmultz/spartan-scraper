// Package research provides unit tests for research pipeline output processing.
// Tests cover pipeline plugin integration, type mismatch handling, and output transformations.
// Does NOT test the research crawler or evidence gathering (research_test.go covers that).
package research

import (
	"context"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

type testMismatchPlugin struct{}

func (p testMismatchPlugin) Name() string {
	return "test_mismatch"
}

func (p testMismatchPlugin) Stages() []pipeline.Stage {
	return []pipeline.Stage{pipeline.StagePreOutput}
}

func (p testMismatchPlugin) Priority() int {
	return 0
}

func (p testMismatchPlugin) Enabled(target pipeline.Target, opts pipeline.Options) bool {
	return true
}

func (p testMismatchPlugin) PreFetch(ctx pipeline.HookContext, in pipeline.FetchInput) (pipeline.FetchInput, error) {
	return in, nil
}

func (p testMismatchPlugin) PostFetch(ctx pipeline.HookContext, in pipeline.FetchInput, out pipeline.FetchOutput) (pipeline.FetchOutput, error) {
	return out, nil
}

func (p testMismatchPlugin) PreExtract(ctx pipeline.HookContext, in pipeline.ExtractInput) (pipeline.ExtractInput, error) {
	return in, nil
}

func (p testMismatchPlugin) PostExtract(ctx pipeline.HookContext, in pipeline.ExtractInput, out pipeline.ExtractOutput) (pipeline.ExtractOutput, error) {
	return out, nil
}

func (p testMismatchPlugin) PreOutput(ctx pipeline.HookContext, in pipeline.OutputInput) (pipeline.OutputInput, error) {
	return pipeline.OutputInput{Structured: "string not Result"}, nil
}

func (p testMismatchPlugin) PostOutput(ctx pipeline.HookContext, in pipeline.OutputInput, out pipeline.OutputOutput) (pipeline.OutputOutput, error) {
	return out, nil
}

func TestApplyResearchOutputPipeline_Success(t *testing.T) {
	ctx := context.Background()
	registry := pipeline.NewRegistry()

	baseCtx := pipeline.HookContext{
		Target: pipeline.Target{URL: "http://example.com"},
	}

	result := Result{
		Query:   "test query",
		Summary: "test summary",
	}

	got, err := applyResearchOutputPipeline(ctx, registry, baseCtx, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Query != result.Query {
		t.Errorf("expected query %q, got %q", result.Query, got.Query)
	}

	if got.Summary != result.Summary {
		t.Errorf("expected summary %q, got %q", result.Summary, got.Summary)
	}
}

func TestApplyResearchOutputPipeline_TypeMismatch(t *testing.T) {
	ctx := context.Background()
	registry := pipeline.NewRegistry()
	registry.Register(testMismatchPlugin{})

	baseCtx := pipeline.HookContext{
		Target: pipeline.Target{URL: "http://example.com"},
	}

	result := Result{Query: "test"}
	_, err := applyResearchOutputPipeline(ctx, registry, baseCtx, result)

	if err == nil {
		t.Fatal("expected error for type mismatch, got nil")
	}

	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("expected KindInternal, got %v", apperrors.KindOf(err))
	}

	if err.Error() != "pipeline output type mismatch for research" {
		t.Errorf("unexpected error message: %v", err.Error())
	}
}

func TestApplyResearchOutputPipeline_NilStructured(t *testing.T) {
	ctx := context.Background()
	registry := pipeline.NewRegistry()

	baseCtx := pipeline.HookContext{
		Target: pipeline.Target{URL: "http://example.com"},
	}

	result := Result{
		Query:   "test query",
		Summary: "test summary",
	}

	got, err := applyResearchOutputPipeline(ctx, registry, baseCtx, result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got.Query != result.Query {
		t.Errorf("expected query %q, got %q", result.Query, got.Query)
	}
}
