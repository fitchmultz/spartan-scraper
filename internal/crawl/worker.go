// Package crawl provides crawl functionality for Spartan Scraper.
//
// Purpose:
// - Implement worker support for package crawl.
//
// Responsibilities:
// - Define the file-local types, functions, and helpers that belong to this package concern.
//
// Scope:
// - Package-internal behavior owned by this file; broader orchestration stays in adjacent package files.
//
// Usage:
// - Used by other files in package `crawl` and any exported callers that depend on this package.
//
// Invariants/Assumptions:
// - This file should preserve the package contract and rely on surrounding package configuration as the source of truth.

package crawl

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/simhash"
)

// processPage processes a single crawl task.
// Returns the page result, a boolean indicating whether to enqueue children,
// and a boolean indicating if the page was skipped (e.g., 304 Not Modified).
func processPage(
	ctx context.Context,
	item task,
	req Request,
	fetcher fetch.Fetcher,
	registry *pipeline.Registry,
	jsRegistry *pipeline.JSRegistry,
	seenSimHashes map[uint64]string,
	simhashMu *sync.Mutex,
) (PageResult, bool, bool) {
	slog.Debug("processing crawl page", "url", apperrors.SanitizeURL(item.URL), "depth", item.Depth)
	// Check state if incremental
	var state model.CrawlState
	var ifNoneMatch, ifModifiedSince string

	if req.Incremental && req.Store != nil {
		existingState, err := req.Store.GetCrawlState(ctx, item.URL)
		if err == nil {
			state = existingState
			ifNoneMatch = state.ETag
			ifModifiedSince = state.LastModified
			slog.Debug("incremental crawl", "url", apperrors.SanitizeURL(item.URL), "etag", ifNoneMatch, "lastModified", ifModifiedSince)
		}
	}

	fetchReq := fetch.Request{
		URL:              item.URL,
		Timeout:          req.Timeout,
		UserAgent:        req.UserAgent,
		Headless:         req.Headless,
		UsePlaywright:    req.UsePlaywright,
		Auth:             req.Auth,
		Limiter:          req.Limiter,
		MaxRetries:       req.MaxRetries,
		RetryBaseDelay:   req.RetryBase,
		MaxResponseBytes: req.MaxResponseBytes,
		DataDir:          req.DataDir,
		IfNoneMatch:      ifNoneMatch,
		IfModifiedSince:  ifModifiedSince,
		Screenshot:       req.Screenshot,
		Device:           req.Device,
		NetworkIntercept: req.NetworkIntercept,
	}

	target := pipeline.NewTarget(fetchReq.URL, string(model.KindCrawl))
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

	preFetchCtx := baseCtx
	preFetchCtx.Stage = pipeline.StagePreFetch
	fetchInput, err := registry.RunPreFetch(preFetchCtx, pipeline.FetchInput{
		Target:     target,
		Request:    fetchReq,
		Auth:       req.Auth,
		Timeout:    req.Timeout,
		UserAgent:  req.UserAgent,
		Headless:   req.Headless,
		Playwright: req.UsePlaywright,
		DataDir:    req.DataDir,
	})
	if err != nil {
		slog.Error("pre-fetch pipeline failed", "url", apperrors.SanitizeURL(item.URL), "error", err)
		return PageResult{}, false, false
	}
	fetchReq = fetchInput.Request
	target = pipeline.NewTarget(fetchReq.URL, string(model.KindCrawl))
	baseCtx.Target = target

	if fetchReq.Headless && jsRegistry != nil {
		engine := pipeline.EngineChromedp
		if fetchReq.UsePlaywright {
			engine = pipeline.EnginePlaywright
		}
		preScripts, postScripts, selectors := pipeline.SelectScripts(jsRegistry.Match(fetchReq.URL), engine)
		if len(preScripts) > 0 {
			fetchReq.PreNavJS = preScripts
		}
		if len(postScripts) > 0 {
			fetchReq.PostNavJS = postScripts
		}
		if len(selectors) > 0 {
			fetchReq.WaitSelectors = selectors
		}
	}

	slog.Debug("fetching crawl page", "url", apperrors.SanitizeURL(fetchReq.URL))
	res, err := fetcher.Fetch(ctx, fetchReq)
	if err != nil {
		slog.Error("fetch failed", "url", apperrors.SanitizeURL(item.URL), "error", err)
		return PageResult{}, false, false // Don't enqueue children if fetch failed
	}
	if res.Status >= 400 {
		slog.Warn("fetch returned error status", "url", apperrors.SanitizeURL(item.URL), "status", res.Status)
		return PageResult{}, false, false
	}
	slog.Debug("fetch complete", "url", apperrors.SanitizeURL(res.URL), "status", res.Status)

	postFetchCtx := baseCtx
	postFetchCtx.Stage = pipeline.StagePostFetch
	fetchOut, err := registry.RunPostFetch(postFetchCtx, fetchInput, pipeline.FetchOutput{Result: res})
	if err != nil {
		slog.Error("post-fetch pipeline failed", "url", apperrors.SanitizeURL(item.URL), "error", err)
		return PageResult{}, false, false
	}
	res = fetchOut.Result

	// Check 304 or Hash Match
	isUnchanged := res.Status == 304
	var currentHash string
	if !isUnchanged {
		sum := sha256.Sum256([]byte(res.HTML))
		currentHash = hex.EncodeToString(sum[:])
		if req.Incremental && state.ContentHash == currentHash && state.ContentHash != "" {
			isUnchanged = true
		}
	}

	if isUnchanged {
		slog.Info("crawl page unchanged", "url", apperrors.SanitizeURL(item.URL))
		// Update LastScraped timestamp
		if req.Incremental && req.Store != nil {
			state.LastScraped = time.Now()
			state.Depth = item.Depth
			state.JobID = req.RequestID
			if err := req.Store.UpsertCrawlState(ctx, state); err != nil {
				slog.Error("failed to update crawl state", "url", apperrors.SanitizeURL(item.URL), "error", err)
			}
		}
		// Return a page result indicating it was skipped
		return PageResult{
			URL:    item.URL,
			Status: 304,
		}, false, true // skip processing/extracting
	}

	slog.Debug("extracting crawl page", "url", apperrors.SanitizeURL(res.URL))
	preExtractCtx := baseCtx
	preExtractCtx.Stage = pipeline.StagePreExtract
	extractInput, err := registry.RunPreExtract(preExtractCtx, pipeline.ExtractInput{
		Target:  target,
		HTML:    res.HTML,
		Options: req.Extract,
		DataDir: req.DataDir,
	})
	if err != nil {
		slog.Error("pre-extract pipeline failed", "url", apperrors.SanitizeURL(item.URL), "error", err)
		return PageResult{}, false, false
	}

	// If changed (or first run), extract and save state
	output, extractErr := extract.Execute(extract.ExecuteInput{
		URL:         item.URL,
		HTML:        extractInput.HTML,
		Options:     extractInput.Options,
		DataDir:     extractInput.DataDir,
		Registry:    req.TemplateRegistry,
		AIExtractor: req.AIExtractor,
		Context:     ctx,
	})
	if extractErr != nil {
		slog.Error("extraction failed", "url", apperrors.SanitizeURL(item.URL), "error", extractErr)
		return PageResult{}, false, false
	}

	postExtractCtx := baseCtx
	postExtractCtx.Stage = pipeline.StagePostExtract
	extractOut, err := registry.RunPostExtract(postExtractCtx, extractInput, pipeline.ExtractOutput{
		Extracted:  output.Extracted,
		Normalized: output.Normalized,
	})
	if err != nil {
		slog.Error("post-extract pipeline failed", "url", apperrors.SanitizeURL(item.URL), "error", err)
		return PageResult{}, false, false
	}
	output.Extracted = extractOut.Extracted
	output.Normalized = extractOut.Normalized

	// Update state
	if req.Incremental && req.Store != nil {
		newState := model.CrawlState{
			URL:          item.URL,
			ETag:         res.ETag,
			LastModified: res.LastModified,
			ContentHash:  currentHash,
			LastScraped:  time.Now(),
			Depth:        item.Depth,
			JobID:        req.RequestID,
		}
		if err := req.Store.UpsertCrawlState(ctx, newState); err != nil {
			slog.Error("failed to update crawl state", "url", apperrors.SanitizeURL(item.URL), "error", err)
		}
	}

	// Compute simhash for content deduplication
	contentText := strings.TrimSpace(output.Normalized.Title + " " + output.Normalized.Text)
	pageSimHash := simhash.Compute(contentText)

	result := PageResult{
		URL:    item.URL,
		Status: res.Status,
		Title:  output.Normalized.Title,
		Text:   output.Normalized.Text,
		Links:  output.Normalized.Links,
		Metadata: extract.Result{
			Title:       output.Normalized.Title,
			Description: output.Normalized.Description,
			Text:        output.Normalized.Text,
			Links:       output.Normalized.Links,
		},
		Extracted:  output.Extracted,
		Normalized: output.Normalized,
		SimHash:    pageSimHash,
	}

	// Check for near-duplicate content if enabled (within same crawl)
	if req.SkipDuplicates && pageSimHash != 0 {
		simhashMu.Lock()
		isDuplicate := false
		var originalURL string

		// Check against all seen simhashes
		for seenHash, seenURL := range seenSimHashes {
			if simhash.HammingDistance(pageSimHash, seenHash) <= req.SimHashThreshold {
				isDuplicate = true
				originalURL = seenURL
				break
			}
		}

		if isDuplicate {
			result.DuplicateOf = originalURL
			slog.Info("found duplicate content", "url", apperrors.SanitizeURL(item.URL), "duplicateOf", apperrors.SanitizeURL(originalURL), "simhash", pageSimHash)
		} else {
			// Add to seen index
			seenSimHashes[pageSimHash] = item.URL
		}
		simhashMu.Unlock()
	}

	finalResult, err := applyCrawlOutputPipeline(ctx, registry, baseCtx, result)
	if err != nil {
		slog.Error("crawl output pipeline failed", "url", apperrors.SanitizeURL(item.URL), "error", err)
		return PageResult{}, false, false
	}

	slog.Info("crawl page complete", "url", apperrors.SanitizeURL(item.URL), "status", res.Status, "title", result.Title)
	return finalResult, true, false
}
