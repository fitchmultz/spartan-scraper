// Package research provides main orchestration for research workflows.
package research

import (
	"context"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/crawl"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/scrape"
)

// Run executes a research operation by crawling/scraping targets and aggregating results.
func Run(ctx context.Context, req Request) (Result, error) {
	slog.Info("research.Run start", "query", req.Query, "urls", req.URLs)
	items := make([]Evidence, 0)
	queryTokens := tokenize(req.Query)

	for _, target := range req.URLs {
		if strings.TrimSpace(target) == "" {
			continue
		}

		if req.MaxDepth > 0 {
			slog.Debug("research crawling target", "url", apperrors.SanitizeURL(target), "maxDepth", req.MaxDepth)
			pages, err := crawl.Run(ctx, crawl.Request{
				URL:              target,
				RequestID:        req.RequestID,
				MaxDepth:         req.MaxDepth,
				MaxPages:         req.MaxPages,
				Concurrency:      req.Concurrency,
				Headless:         req.Headless,
				UsePlaywright:    req.UsePlaywright,
				Auth:             req.Auth,
				Extract:          req.Extract,
				Pipeline:         req.Pipeline,
				Timeout:          req.Timeout,
				UserAgent:        req.UserAgent,
				Limiter:          req.Limiter,
				MaxRetries:       req.MaxRetries,
				RetryBase:        req.RetryBase,
				MaxResponseBytes: req.MaxResponseBytes,
				DataDir:          req.DataDir,
				Store:            req.Store,
				Registry:         req.Registry,
				JSRegistry:       req.JSRegistry,
				TemplateRegistry: req.TemplateRegistry,
			})
			if err != nil {
				slog.Error("research crawl failed", "url", apperrors.SanitizeURL(target), "error", err)
				continue
			}
			for _, page := range pages {
				if page.Status == 304 {
					continue
				}
				items = append(items, Evidence{
					URL:     page.URL,
					Title:   page.Title,
					Snippet: makeSnippet(page.Text),
					Score:   scoreText(queryTokens, page.Text),
				})
			}
		} else {
			slog.Debug("research scraping target", "url", apperrors.SanitizeURL(target))
			res, err := scrape.Run(ctx, scrape.Request{
				URL:              target,
				RequestID:        req.RequestID,
				Headless:         req.Headless,
				UsePlaywright:    req.UsePlaywright,
				Auth:             req.Auth,
				Extract:          req.Extract,
				Pipeline:         req.Pipeline,
				Timeout:          req.Timeout,
				UserAgent:        req.UserAgent,
				Limiter:          req.Limiter,
				MaxRetries:       req.MaxRetries,
				RetryBase:        req.RetryBase,
				MaxResponseBytes: req.MaxResponseBytes,
				DataDir:          req.DataDir,
				Store:            req.Store,
				Registry:         req.Registry,
				JSRegistry:       req.JSRegistry,
				TemplateRegistry: req.TemplateRegistry,
			})
			if err != nil {
				slog.Error("research scrape failed", "url", apperrors.SanitizeURL(target), "error", err)
				continue
			}
			if res.Status != 304 {
				items = append(items, Evidence{
					URL:     res.URL,
					Title:   res.Title,
					Snippet: makeSnippet(res.Text),
					Score:   scoreText(queryTokens, res.Text),
				})
			}
		}
	}

	slog.Info("research gathering complete", "evidenceCount", len(items))
	sort.Slice(items, func(i, j int) bool {
		return items[i].Score > items[j].Score
	})

	items = enrichEvidence(items)
	items = dedupEvidence(items, 3)
	clusters, items := clusterEvidence(items, 8, 1)
	citations := buildCitations(items)
	confidence := overallConfidence(items, clusters)

	summary := summarize(queryTokens, items)
	result := Result{
		Query:      req.Query,
		Summary:    summary,
		Evidence:   items,
		Clusters:   clusters,
		Citations:  citations,
		Confidence: confidence,
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
