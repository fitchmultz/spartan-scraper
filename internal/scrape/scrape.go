// Package scrape provides functionality for scraping a single web page.
// It handles fetching the page content (optionally using a headless browser),
// running it through a pipeline of pre-processors, extracting metadata and content
// based on templates, and running post-processors and transformers on the output.
package scrape

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

// Request represents a single page scrape request.
type Request struct {
	URL              string
	Method           string // HTTP method (GET, POST, PUT, DELETE, PATCH, etc.)
	Body             []byte // Request body for POST/PUT/PATCH
	ContentType      string // Content-Type header for request body
	RequestID        string
	Headless         bool
	UsePlaywright    bool
	Auth             fetch.AuthOptions
	Extract          extract.ExtractOptions
	Pipeline         pipeline.Options
	Timeout          time.Duration
	UserAgent        string
	Limiter          *fetch.HostLimiter
	MaxRetries       int
	RetryBase        time.Duration
	MaxResponseBytes int64
	DataDir          string
	Incremental      bool
	Store            CrawlStateStore
	Registry         *pipeline.Registry
	JSRegistry       *pipeline.JSRegistry
	TemplateRegistry *extract.TemplateRegistry
	// MetricsCallback is called for each fetch operation to record metrics.
	MetricsCallback fetch.MetricsCallback
	// Screenshot config for headless fetchers (chromedp, playwright).
	Screenshot *fetch.ScreenshotConfig
}

// Result contains the outcome of a scrape operation.
type Result struct {
	URL        string                     `json:"url"`
	Status     int                        `json:"status"`
	Title      string                     `json:"title"`
	Text       string                     `json:"text"`
	Links      []string                   `json:"links"`
	Metadata   extract.Result             `json:"metadata"` // Legacy
	Extracted  extract.Extracted          `json:"extracted"`
	Normalized extract.NormalizedDocument `json:"normalized"`
}

// CrawlStateStore defines the interface for persisting and retrieving crawl states.
type CrawlStateStore interface {
	GetCrawlState(ctx context.Context, url string) (model.CrawlState, error)
	UpsertCrawlState(ctx context.Context, state model.CrawlState) error
}

// Run executes a scrape request. It fetches the page, runs it through the pipeline,
// extracts data, and returns the result.
func Run(ctx context.Context, req Request) (Result, error) {
	slog.Debug("scrape.Run start", "url", req.URL)
	registry := req.Registry
	if registry == nil {
		registry = pipeline.NewRegistry()
	}
	jsRegistry := req.JSRegistry
	if jsRegistry == nil {
		loaded, err := pipeline.LoadJSRegistry(req.DataDir)
		if err != nil {
			slog.Error("failed to load JS registry", "error", err)
			return Result{}, err
		}
		jsRegistry = loaded
	}

	var state model.CrawlState
	var ifNoneMatch, ifModifiedSince string

	if req.Incremental && req.Store != nil {
		existingState, err := req.Store.GetCrawlState(ctx, req.URL)
		if err == nil {
			state = existingState
			ifNoneMatch = state.ETag
			ifModifiedSince = state.LastModified
			slog.Debug("incremental scrape", "url", req.URL, "etag", ifNoneMatch, "lastModified", ifModifiedSince)
		}
	}

	var fetcher fetch.Fetcher
	if req.MetricsCallback != nil {
		fetcher = fetch.NewFetcherWithMetrics(req.DataDir, req.MetricsCallback)
	} else {
		fetcher = fetch.NewFetcher(req.DataDir)
	}

	fetchReq := fetch.Request{
		URL:              req.URL,
		Method:           req.Method,
		Body:             req.Body,
		ContentType:      req.ContentType,
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
	}

	target := pipeline.NewTarget(fetchReq.URL, string(model.KindScrape))
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
		slog.Error("pre-fetch pipeline failed", "url", req.URL, "error", err)
		return Result{}, err
	}
	fetchReq = fetchInput.Request
	target = pipeline.NewTarget(fetchReq.URL, string(model.KindScrape))
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

	slog.Debug("fetching", "url", fetchReq.URL)
	res, err := fetcher.Fetch(ctx, fetchReq)
	if err != nil {
		slog.Error("fetch failed", "url", req.URL, "error", err)
		return Result{}, err
	}
	slog.Debug("fetch complete", "url", res.URL, "status", res.Status)

	postFetchCtx := baseCtx
	postFetchCtx.Stage = pipeline.StagePostFetch
	fetchOut, err := registry.RunPostFetch(postFetchCtx, fetchInput, pipeline.FetchOutput{Result: res})
	if err != nil {
		slog.Error("post-fetch pipeline failed", "url", req.URL, "error", err)
		return Result{}, err
	}
	res = fetchOut.Result

	if res.Status == 304 {
		slog.Info("content not modified (304)", "url", res.URL)
		if req.Incremental && req.Store != nil {
			state.LastScraped = time.Now()
			if err := req.Store.UpsertCrawlState(ctx, state); err != nil {
				slog.Error("failed to update crawl state", "url", req.URL, "error", err)
			}
		}
		return applyScrapeOutputPipeline(ctx, registry, baseCtx, Result{
			URL:    res.URL,
			Status: 304,
		})
	}

	currentHash := sha256.Sum256([]byte(res.HTML))
	currentHashStr := hex.EncodeToString(currentHash[:])

	if req.Incremental && state.ContentHash == currentHashStr && state.ContentHash != "" {
		slog.Info("content not modified (hash match)", "url", res.URL)
		if req.Incremental && req.Store != nil {
			state.LastScraped = time.Now()
			if err := req.Store.UpsertCrawlState(ctx, state); err != nil {
				slog.Error("failed to update crawl state", "url", req.URL, "error", err)
			}
		}
		return applyScrapeOutputPipeline(ctx, registry, baseCtx, Result{
			URL:    res.URL,
			Status: 200,
		})
	}

	slog.Debug("extracting", "url", res.URL)
	preExtractCtx := baseCtx
	preExtractCtx.Stage = pipeline.StagePreExtract
	extractInput, err := registry.RunPreExtract(preExtractCtx, pipeline.ExtractInput{
		Target:  target,
		HTML:    res.HTML,
		Options: req.Extract,
		DataDir: req.DataDir,
	})
	if err != nil {
		slog.Error("pre-extract pipeline failed", "url", req.URL, "error", err)
		return Result{}, err
	}

	output, err := extract.Execute(extract.ExecuteInput{
		URL:      res.URL,
		HTML:     extractInput.HTML,
		Options:  extractInput.Options,
		DataDir:  extractInput.DataDir,
		Registry: req.TemplateRegistry,
	})
	if err != nil {
		slog.Error("extraction failed", "url", req.URL, "error", err)
		return Result{}, err
	}

	postExtractCtx := baseCtx
	postExtractCtx.Stage = pipeline.StagePostExtract
	extractOut, err := registry.RunPostExtract(postExtractCtx, extractInput, pipeline.ExtractOutput{
		Extracted:  output.Extracted,
		Normalized: output.Normalized,
	})
	if err != nil {
		slog.Error("post-extract pipeline failed", "url", req.URL, "error", err)
		return Result{}, err
	}
	output.Extracted = extractOut.Extracted
	output.Normalized = extractOut.Normalized

	if req.Incremental && req.Store != nil {
		newState := model.CrawlState{
			URL:          req.URL,
			ETag:         res.ETag,
			LastModified: res.LastModified,
			ContentHash:  currentHashStr,
			LastScraped:  time.Now(),
		}
		if err := req.Store.UpsertCrawlState(ctx, newState); err != nil {
			slog.Error("failed to update crawl state", "url", req.URL, "error", err)
		}
	}

	result := Result{
		URL:    res.URL,
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
	}
	slog.Info("scrape complete", "url", res.URL, "status", res.Status, "title", result.Title)
	return applyScrapeOutputPipeline(ctx, registry, baseCtx, result)
}

func applyScrapeOutputPipeline(ctx context.Context, registry *pipeline.Registry, baseCtx pipeline.HookContext, result Result) (Result, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return Result{}, apperrors.Wrap(apperrors.KindInternal, "failed to marshal scrape result", err)
	}
	input := pipeline.OutputInput{
		Target:     baseCtx.Target,
		Kind:       string(model.KindScrape),
		Raw:        raw,
		Structured: result,
	}

	preCtx := baseCtx
	preCtx.Stage = pipeline.StagePreOutput
	outInput, err := registry.RunPreOutput(preCtx, input)
	if err != nil {
		return Result{}, err
	}
	if typed, ok := outInput.Structured.(Result); ok {
		result = typed
		outInput.Structured = result
	}

	transformCtx := baseCtx
	transformCtx.Stage = pipeline.StagePreOutput
	out, err := registry.RunTransformers(transformCtx, outInput)
	if err != nil {
		return Result{}, err
	}

	postCtx := baseCtx
	postCtx.Stage = pipeline.StagePostOutput
	out, err = registry.RunPostOutput(postCtx, outInput, out)
	if err != nil {
		return Result{}, err
	}

	if out.Structured == nil {
		return result, nil
	}
	typed, ok := out.Structured.(Result)
	if !ok {
		return Result{}, apperrors.Internal("pipeline output type mismatch for scrape")
	}
	return typed, nil
}
