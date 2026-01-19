// Package crawl provides functionality for crawling multiple pages of a website.
// It implements a concurrent crawler that respects depth and page limits,
// avoids cycles by tracking visited URLs, and supports incremental crawling
// using ETags and content hashes.
package crawl

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/fetch"
	"spartan-scraper/internal/model"
	"spartan-scraper/internal/pipeline"
)

// Request represents a website crawl request.
type Request struct {
	URL           string
	RequestID     string
	MaxDepth      int
	MaxPages      int
	Concurrency   int
	Headless      bool
	UsePlaywright bool
	Auth          fetch.AuthOptions
	Extract       extract.ExtractOptions
	Pipeline      pipeline.Options
	Timeout       time.Duration
	UserAgent     string
	Limiter       *fetch.HostLimiter
	MaxRetries    int
	RetryBase     time.Duration
	DataDir       string
	Incremental   bool
	Store         CrawlStateStore
	Registry      *pipeline.Registry
	JSRegistry    *pipeline.JSRegistry
}

// CrawlStateStore defines the interface for persisting and retrieving crawl states.
type CrawlStateStore interface {
	GetCrawlState(ctx context.Context, url string) (model.CrawlState, error)
	UpsertCrawlState(ctx context.Context, state model.CrawlState) error
}

// PageResult represents the scraping result for a single page during a crawl.
type PageResult struct {
	URL        string                     `json:"url"`
	Status     int                        `json:"status"`
	Title      string                     `json:"title"`
	Text       string                     `json:"text"`
	Links      []string                   `json:"links"`
	Metadata   extract.Result             `json:"metadata"` // Legacy
	Extracted  extract.Extracted          `json:"extracted"`
	Normalized extract.NormalizedDocument `json:"normalized"`
}

// Run executes a crawl request. It concurrently fetches and processes pages
// starting from a root URL, following links up to a maximum depth and page count.
func Run(ctx context.Context, req Request) ([]PageResult, error) {
	slog.Info("crawl.Run start", "url", req.URL, "maxDepth", req.MaxDepth, "maxPages", req.MaxPages)
	if req.MaxDepth <= 0 {
		req.MaxDepth = 1
	}
	if req.MaxPages <= 0 {
		req.MaxPages = 100
	}
	if req.Concurrency <= 0 {
		req.Concurrency = 4
	}

	registry := req.Registry
	if registry == nil {
		registry = pipeline.NewRegistry()
	}
	jsRegistry := req.JSRegistry
	if jsRegistry == nil {
		loaded, err := pipeline.LoadJSRegistry(req.DataDir)
		if err != nil {
			slog.Error("failed to load JS registry", "error", err)
			return nil, err
		}
		jsRegistry = loaded
	}

	startURL, err := url.Parse(req.URL)
	if err != nil {
		slog.Error("failed to parse start URL", "url", req.URL, "error", err)
		return nil, err
	}

	fetcher := fetch.NewFetcher()

	type task struct {
		URL   string
		Depth int
	}

	processPage := func(item task) (PageResult, bool, bool) {
		slog.Debug("processing crawl page", "url", item.URL, "depth", item.Depth)
		// Check state if incremental
		var state model.CrawlState
		var ifNoneMatch, ifModifiedSince string

		if req.Incremental && req.Store != nil {
			existingState, err := req.Store.GetCrawlState(ctx, item.URL)
			if err == nil {
				state = existingState
				ifNoneMatch = state.ETag
				ifModifiedSince = state.LastModified
				slog.Debug("incremental crawl", "url", item.URL, "etag", ifNoneMatch, "lastModified", ifModifiedSince)
			}
		}

		fetchReq := fetch.Request{
			URL:             item.URL,
			Timeout:         req.Timeout,
			UserAgent:       req.UserAgent,
			Headless:        req.Headless,
			UsePlaywright:   req.UsePlaywright,
			Auth:            req.Auth,
			Limiter:         req.Limiter,
			MaxRetries:      req.MaxRetries,
			RetryBaseDelay:  req.RetryBase,
			DataDir:         req.DataDir,
			IfNoneMatch:     ifNoneMatch,
			IfModifiedSince: ifModifiedSince,
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
			slog.Error("pre-fetch pipeline failed", "url", item.URL, "error", err)
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

		slog.Debug("fetching crawl page", "url", fetchReq.URL)
		res, err := fetcher.Fetch(ctx, fetchReq)
		if err != nil {
			slog.Error("fetch failed", "url", item.URL, "error", err)
			return PageResult{}, false, false // Don't enqueue children if fetch failed
		}
		if res.Status >= 400 {
			slog.Warn("fetch returned error status", "url", item.URL, "status", res.Status)
			return PageResult{}, false, false
		}
		slog.Debug("fetch complete", "url", res.URL, "status", res.Status)

		postFetchCtx := baseCtx
		postFetchCtx.Stage = pipeline.StagePostFetch
		fetchOut, err := registry.RunPostFetch(postFetchCtx, fetchInput, pipeline.FetchOutput{Result: res})
		if err != nil {
			slog.Error("post-fetch pipeline failed", "url", item.URL, "error", err)
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
			slog.Info("crawl page unchanged", "url", item.URL)
			// Update LastScraped timestamp
			if req.Incremental && req.Store != nil {
				state.LastScraped = time.Now()
				if err := req.Store.UpsertCrawlState(ctx, state); err != nil {
					slog.Error("failed to update crawl state", "url", item.URL, "error", err)
				}
			}
			// Return a page result indicating it was skipped
			return PageResult{
				URL:    item.URL,
				Status: 304,
			}, false, true // skip processing/extracting
		}

		slog.Debug("extracting crawl page", "url", res.URL)
		preExtractCtx := baseCtx
		preExtractCtx.Stage = pipeline.StagePreExtract
		extractInput, err := registry.RunPreExtract(preExtractCtx, pipeline.ExtractInput{
			Target:  target,
			HTML:    res.HTML,
			Options: req.Extract,
			DataDir: req.DataDir,
		})
		if err != nil {
			slog.Error("pre-extract pipeline failed", "url", item.URL, "error", err)
			return PageResult{}, false, false
		}

		// If changed (or first run), extract and save state
		output, extractErr := extract.Execute(extract.ExecuteInput{
			URL:     item.URL,
			HTML:    extractInput.HTML,
			Options: extractInput.Options,
			DataDir: extractInput.DataDir,
		})
		if extractErr != nil {
			slog.Error("extraction failed", "url", item.URL, "error", extractErr)
			return PageResult{}, false, false
		}

		postExtractCtx := baseCtx
		postExtractCtx.Stage = pipeline.StagePostExtract
		extractOut, err := registry.RunPostExtract(postExtractCtx, extractInput, pipeline.ExtractOutput{
			Extracted:  output.Extracted,
			Normalized: output.Normalized,
		})
		if err != nil {
			slog.Error("post-extract pipeline failed", "url", item.URL, "error", err)
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
			}
			if err := req.Store.UpsertCrawlState(ctx, newState); err != nil {
				slog.Error("failed to update crawl state", "url", item.URL, "error", err)
			}
		}

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
		}

		finalResult, err := applyCrawlOutputPipeline(ctx, registry, baseCtx, result)
		if err != nil {
			slog.Error("crawl output pipeline failed", "url", item.URL, "error", err)
			return PageResult{}, false, false
		}

		slog.Info("crawl page complete", "url", item.URL, "status", res.Status, "title", result.Title)
		return finalResult, true, false
	}

	tasks := make(chan task, req.MaxPages)
	results := make(chan PageResult, req.MaxPages)
	var wg sync.WaitGroup
	var visitedMu sync.Mutex
	visited := map[string]bool{}
	var processed int32

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	enqueue := func(url string, depth int) {
		if atomic.LoadInt32(&processed) >= int32(req.MaxPages) {
			return
		}
		norm := normalizeURL(url)
		visitedMu.Lock()
		if visited[norm] {
			visitedMu.Unlock()
			return
		}
		visited[norm] = true
		visitedMu.Unlock()

		slog.Debug("enqueuing crawl task", "url", url, "depth", depth)
		wg.Add(1)
		select {
		case tasks <- task{URL: url, Depth: depth}:
		default:
			slog.Warn("crawl task channel full", "url", url)
			wg.Done()
		case <-ctx.Done():
			wg.Done()
		}
	}

	enqueue(req.URL, 0)

	for i := 0; i < req.Concurrency; i++ {
		go func(workerID int) {
			slog.Debug("starting crawl worker", "workerID", workerID)
			for {
				select {
				case <-ctx.Done():
					return
				case item, ok := <-tasks:
					if !ok {
						return
					}
					if atomic.LoadInt32(&processed) >= int32(req.MaxPages) {
						wg.Done()
						continue
					}

					res, enqueueChildren, skipped := processPage(item)
					if skipped || res.URL == "" {
						wg.Done()
						continue
					}

					if atomic.AddInt32(&processed, 1) <= int32(req.MaxPages) {
						results <- res
					} else {
						wg.Done()
						continue
					}

					if enqueueChildren && item.Depth < req.MaxDepth {
						for _, href := range res.Links {
							resolved := resolveURL(startURL, href)
							if resolved == "" {
								continue
							}
							if !sameHost(startURL, resolved) {
								continue
							}
							enqueue(resolved, item.Depth+1)
						}
					}

					wg.Done()
				}
			}
		}(i)
	}

	go func() {
		wg.Wait()
		slog.Info("crawl completed", "totalProcessed", atomic.LoadInt32(&processed))
		close(tasks)
		close(results)
		cancel()
	}()

	collected := make([]PageResult, 0)
	for item := range results {
		if len(collected) >= req.MaxPages {
			break
		}
		collected = append(collected, item)
	}

	return collected, nil
}

func normalizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	return u.String()
}

func resolveURL(base *url.URL, href string) string {
	u, err := url.Parse(strings.TrimSpace(href))
	if err != nil {
		return ""
	}
	return base.ResolveReference(u).String()
}

func sameHost(base *url.URL, raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	return u.Host == base.Host
}

func applyCrawlOutputPipeline(ctx context.Context, registry *pipeline.Registry, baseCtx pipeline.HookContext, result PageResult) (PageResult, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return PageResult{}, fmt.Errorf("failed to marshal result: %w", err)
	}
	input := pipeline.OutputInput{
		Target:     baseCtx.Target,
		Kind:       string(model.KindCrawl),
		Raw:        raw,
		Structured: result,
	}

	preCtx := baseCtx
	preCtx.Stage = pipeline.StagePreOutput
	outInput, err := registry.RunPreOutput(preCtx, input)
	if err != nil {
		return PageResult{}, err
	}
	if typed, ok := outInput.Structured.(PageResult); ok {
		result = typed
		outInput.Structured = result
	}

	transformCtx := baseCtx
	transformCtx.Stage = pipeline.StagePreOutput
	out, err := registry.RunTransformers(transformCtx, outInput)
	if err != nil {
		return PageResult{}, err
	}

	postCtx := baseCtx
	postCtx.Stage = pipeline.StagePostOutput
	out, err = registry.RunPostOutput(postCtx, outInput, out)
	if err != nil {
		return PageResult{}, err
	}

	if out.Structured == nil {
		return result, nil
	}
	typed, ok := out.Structured.(PageResult)
	if !ok {
		return PageResult{}, fmt.Errorf("pipeline output type mismatch for crawl")
	}
	return typed, nil
}
