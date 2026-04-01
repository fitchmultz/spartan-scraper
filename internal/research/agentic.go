// Package research provides research functionality for Spartan Scraper.
//
// Purpose:
// - Gather research evidence documents and build the deterministic baseline result used by agentic follow-up rounds.
//
// Responsibilities:
// - Define agentic research document types, collect scrape/crawl evidence, and build the baseline clustered result.
//
// Scope:
// - Evidence gathering and baseline result construction only; agentic planning, synthesis, and prompt rendering live in adjacent files.
//
// Usage:
// - Used internally by research execution when agentic refinement is enabled.
//
// Invariants/Assumptions:
// - Evidence gathering must preserve request fetch settings.
// - The deterministic baseline result remains the source of truth for later agentic rounds.
package research

import (
	"context"
	"log/slog"
	"sort"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/crawl"
	"github.com/fitchmultz/spartan-scraper/internal/scrape"
)

const (
	agenticStatusCompleted = "completed"
	agenticStatusFailed    = "failed"
	agenticStatusSkipped   = "skipped"
)

type researchDocument struct {
	Evidence Evidence
	Links    []string
}

type agenticPlan struct {
	Objective    string
	FocusAreas   []string
	FollowUpURLs []string
	Reasoning    string
	RouteID      string
	Provider     string
	Model        string
	Cached       bool
	Confidence   float64
}

type agenticSynthesis struct {
	Summary              string
	Objective            string
	FocusAreas           []string
	KeyFindings          []string
	OpenQuestions        []string
	RecommendedNextSteps []string
	RouteID              string
	Provider             string
	Model                string
	Cached               bool
	Confidence           float64
}

func gatherResearchDocuments(ctx context.Context, req Request, targets []string, maxDepth int, maxPages int) ([]researchDocument, int, int, error) {
	items := make([]researchDocument, 0)
	queryTokens := tokenize(req.Query)
	var successCount, failCount int

	for _, target := range targets {
		if ctx.Err() != nil {
			return nil, successCount, failCount, apperrors.Wrap(apperrors.KindInternal, "research cancelled", ctx.Err())
		}
		if strings.TrimSpace(target) == "" {
			continue
		}

		if maxDepth > 0 {
			slog.Debug("research crawling target", "url", apperrors.SanitizeURL(target), "maxDepth", maxDepth)
			pages, err := crawl.Run(ctx, crawl.Request{
				URL:              target,
				RequestID:        req.RequestID,
				MaxDepth:         maxDepth,
				MaxPages:         maxPages,
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
				Screenshot:       req.Screenshot,
				Device:           req.Device,
				NetworkIntercept: req.NetworkIntercept,
				ProxyPool:        req.ProxyPool,
				AIExtractor:      req.AIExtractor,
			})
			if err != nil {
				if ctx.Err() != nil {
					return nil, successCount, failCount, apperrors.Wrap(apperrors.KindInternal, "research cancelled", ctx.Err())
				}
				slog.Error("research crawl failed", "url", apperrors.SanitizeURL(target), "error", err)
				failCount++
				continue
			}
			successCount++
			for _, page := range pages {
				if page.Status == 304 {
					continue
				}
				fields := cloneEvidenceFields(page.Normalized.Fields)
				searchText := evidenceSearchText(page.Normalized.Title, page.Normalized.Text, fields)
				items = append(items, researchDocument{
					Evidence: Evidence{
						URL:     page.URL,
						Title:   page.Normalized.Title,
						Snippet: makeEvidenceSnippet(page.Normalized.Text, fields),
						Score:   scoreText(queryTokens, searchText) + researchSourcePriorityBoost(target, page.URL),
						Fields:  fields,
					},
					Links: normalizeDocumentLinks(page.URL, page.Normalized.Links),
				})
			}
			continue
		}

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
			Screenshot:       req.Screenshot,
			Device:           req.Device,
			NetworkIntercept: req.NetworkIntercept,
			ProxyPool:        req.ProxyPool,
			AIExtractor:      req.AIExtractor,
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil, successCount, failCount, apperrors.Wrap(apperrors.KindInternal, "research cancelled", ctx.Err())
			}
			slog.Error("research scrape failed", "url", apperrors.SanitizeURL(target), "error", err)
			failCount++
			continue
		}
		successCount++
		if res.Status == 304 {
			continue
		}
		fields := cloneEvidenceFields(res.Normalized.Fields)
		searchText := evidenceSearchText(res.Normalized.Title, res.Normalized.Text, fields)
		items = append(items, researchDocument{
			Evidence: Evidence{
				URL:     res.URL,
				Title:   res.Normalized.Title,
				Snippet: makeEvidenceSnippet(res.Normalized.Text, fields),
				Score:   scoreText(queryTokens, searchText) + researchSourcePriorityBoost(target, res.URL),
				Fields:  fields,
			},
			Links: normalizeDocumentLinks(res.URL, res.Normalized.Links),
		})
	}

	return items, successCount, failCount, nil
}

func buildResearchResult(query string, docs []researchDocument) Result {
	items := make([]Evidence, 0, len(docs))
	for _, doc := range docs {
		items = append(items, doc.Evidence)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Score > items[j].Score
	})

	items = enrichEvidence(items)
	items = dedupEvidence(items, 3)
	clusters, items := clusterEvidence(items, 8, 1)
	citations := buildCitations(items)
	confidence := overallConfidence(items, clusters)
	summary := summarize(tokenize(query), items)

	return Result{
		Query:      query,
		Summary:    summary,
		Evidence:   items,
		Clusters:   clusters,
		Citations:  citations,
		Confidence: confidence,
	}
}
