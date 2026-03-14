// Package research provides main orchestration for research workflows.
package research

import (
	"context"
	"log/slog"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

// Run executes a research operation by crawling/scraping targets and aggregating results.
func Run(ctx context.Context, req Request) (Result, error) {
	slog.Info("research.Run start", "query", req.Query, "urls", req.URLs)

	docs, successCount, failCount, err := gatherResearchDocuments(ctx, req, req.URLs, req.MaxDepth, req.MaxPages)
	if err != nil {
		return Result{}, err
	}
	if successCount == 0 && failCount > 0 {
		return Result{}, apperrors.Internal("all research targets failed")
	}

	slog.Info("research gathering complete", "evidenceCount", len(docs), "successCount", successCount, "failCount", failCount)
	result := buildResearchResult(req.Query, docs)
	if req.Agentic != nil && req.Agentic.Enabled {
		agentic, enrichedDocs := runAgenticResearch(ctx, req, docs, result)
		result = buildResearchResult(req.Query, enrichedDocs)
		result.Agentic = agentic
	}

	registry := req.Registry
	if registry == nil {
		registry = pipeline.NewRegistry()
	}
	target := pipeline.NewTarget("", string(model.KindResearch))
	baseCtx := pipeline.HookContext{
		Context:     ctx,
		RequestID:   req.RequestID,
		Target:      target,
		Now:         time.Now(),
		DataDir:     req.DataDir,
		Options:     req.Pipeline,
		Attributes:  map[string]string{},
		Diagnostics: map[string]any{},
	}
	slog.Info("research complete", "confidence", result.Confidence)
	return applyResearchOutputPipeline(ctx, registry, baseCtx, result)
}
