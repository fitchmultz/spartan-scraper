// Package crawl provides functionality for crawling multiple pages of a website.
// It implements a concurrent crawler that respects depth and page limits,
// avoids cycles by tracking visited URLs, supports incremental crawling
// using ETags and content hashes, and detects near-duplicate content using simhash.
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

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/dedup"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/simhash"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

// Request represents a website crawl request.
type Request struct {
	URL              string
	RequestID        string
	MaxDepth         int
	MaxPages         int
	Concurrency      int
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
	// SitemapURL is an optional URL to a sitemap.xml file.
	// If provided, URLs from the sitemap will be used as crawl seeds.
	SitemapURL string
	// SitemapOnly indicates whether to only crawl URLs from the sitemap.
	// If false (default), the root URL is also crawled plus sitemap URLs.
	SitemapOnly bool
	// IncludePatterns are glob patterns for URL paths to include.
	// If specified, only URLs matching at least one pattern are crawled.
	// Supports * (matches any chars except /) and ** (matches any chars including /).
	IncludePatterns []string
	// ExcludePatterns are glob patterns for URL paths to exclude.
	// Excluded URLs take precedence over included ones.
	// Supports * (matches any chars except /) and ** (matches any chars including /).
	ExcludePatterns []string
	// Screenshot config for headless fetchers (chromedp, playwright).
	Screenshot *fetch.ScreenshotConfig
	// RobotsCache is an optional cache for robots.txt compliance checking.
	// If nil, robots.txt is not checked.
	RobotsCache *Cache
	// SkipDuplicates enables near-duplicate content detection during crawling.
	// When enabled, pages with content similar to already-crawled pages are marked as duplicates.
	SkipDuplicates bool
	// SimHashThreshold is the maximum Hamming distance for content to be considered a duplicate.
	// Default is 3. Lower values require more similarity (0 = exact match).
	SimHashThreshold int
	// CrossJobDedup enables cross-job duplicate detection using ContentIndex.
	// When enabled, the crawler queries the ContentIndex before fetching to detect
	// if similar content has been indexed by previous jobs.
	CrossJobDedup bool
	// CrossJobDedupThreshold is the Hamming distance threshold for cross-job duplicate detection.
	// Default is 3 (near-duplicates).
	CrossJobDedupThreshold int
	// ProxyPool for proxy rotation. If nil, no proxy pool is used.
	ProxyPool *fetch.ProxyPool
	// WebhookDispatcher is an optional dispatcher for page crawled events.
	// If nil, no webhook events are dispatched.
	WebhookDispatcher interface {
		Dispatch(ctx context.Context, url string, payload webhook.Payload, secret string)
	}
	// WebhookConfig holds webhook configuration for the crawl.
	WebhookConfig *model.WebhookConfig
	// AIExtractor for AI-powered extraction. If nil, AI extraction is disabled.
	AIExtractor *extract.AIExtractor
	// ContentIndex for cross-job deduplication. If nil, cross-job dedup is disabled.
	ContentIndex ContentIndex
}

// ContentIndex provides cross-job content deduplication capabilities.
// This is a subset of the dedup.ContentIndex interface for crawl's needs.
type ContentIndex interface {
	// Index stores a content fingerprint for a URL.
	Index(ctx context.Context, jobID, url string, simhash uint64) error
	// FindDuplicates returns URLs with similar content across all jobs.
	FindDuplicates(ctx context.Context, simhash uint64, threshold int) ([]dedup.DuplicateMatch, error)
}

// CrawlStateStore defines the interface for persisting and retrieving crawl states.
type CrawlStateStore interface {
	GetCrawlState(ctx context.Context, url string) (model.CrawlState, error)
	UpsertCrawlState(ctx context.Context, state model.CrawlState) error
}

// PageResult represents the scraping result for a single page during a crawl.
type PageResult struct {
	URL                string                     `json:"url"`
	Status             int                        `json:"status"`
	Title              string                     `json:"title"`
	Text               string                     `json:"text"`
	Links              []string                   `json:"links"`
	Metadata           extract.Result             `json:"metadata"` // Legacy
	Extracted          extract.Extracted          `json:"extracted"`
	Normalized         extract.NormalizedDocument `json:"normalized"`
	SimHash            uint64                     `json:"simhash"`                      // Content fingerprint for duplicate detection
	DuplicateOf        string                     `json:"duplicateOf,omitempty"`        // URL of original page if this is a duplicate (same crawl)
	CrossJobDuplicates []CrossJobDuplicate        `json:"crossJobDuplicates,omitempty"` // Duplicates found in other jobs
}

// CrossJobDuplicate represents a duplicate match from a different job.
type CrossJobDuplicate struct {
	JobID     string `json:"jobId"`
	URL       string `json:"url"`
	Distance  int    `json:"distance"`
	IndexedAt string `json:"indexedAt"`
}

// Run executes a crawl request. It concurrently fetches and processes pages
// starting from a root URL, following links up to a maximum depth and page count.
func Run(ctx context.Context, req Request) ([]PageResult, error) {
	slog.Info("crawl.Run start", "url", apperrors.SanitizeURL(req.URL), "maxDepth", req.MaxDepth, "maxPages", req.MaxPages)
	if req.MaxDepth <= 0 {
		req.MaxDepth = 1
	}
	if req.MaxPages <= 0 {
		req.MaxPages = 100
	}
	if req.Concurrency <= 0 {
		req.Concurrency = 4
	}
	if req.SimHashThreshold < 0 {
		req.SimHashThreshold = 3 // default threshold
	}
	if req.CrossJobDedupThreshold <= 0 {
		req.CrossJobDedupThreshold = 3 // default threshold
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
		slog.Error("failed to parse start URL", "url", apperrors.SanitizeURL(req.URL), "error", err)
		return nil, err
	}

	// Compile pattern matcher for URL filtering
	patternMatcher, err := NewPatternMatcher(req.IncludePatterns, req.ExcludePatterns)
	if err != nil {
		slog.Error("failed to compile URL patterns", "error", err)
		return nil, apperrors.Wrap(apperrors.KindValidation, "invalid URL pattern", err)
	}

	var fetcher fetch.Fetcher
	if req.MetricsCallback != nil {
		if req.ProxyPool != nil {
			fetcher = fetch.NewFetcherWithMetricsAndProxyPool(req.DataDir, req.MetricsCallback, req.ProxyPool)
		} else {
			fetcher = fetch.NewFetcherWithMetrics(req.DataDir, req.MetricsCallback)
		}
	} else {
		if req.ProxyPool != nil {
			fetcher = fetch.NewFetcherWithProxyPool(req.DataDir, req.ProxyPool)
		} else {
			fetcher = fetch.NewFetcher(req.DataDir)
		}
	}

	type task struct {
		URL   string
		Depth int
	}

	// Simhash index for duplicate detection
	var simhashMu sync.Mutex
	seenSimHashes := make(map[uint64]string) // simhash -> URL mapping

	processPage := func(item task) (PageResult, bool, bool) {
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

		// Cross-job deduplication: check against previously indexed content
		if req.CrossJobDedup && req.ContentIndex != nil && pageSimHash != 0 {
			matches, err := req.ContentIndex.FindDuplicates(ctx, pageSimHash, req.CrossJobDedupThreshold)
			if err != nil {
				slog.Warn("failed to query cross-job duplicates", "url", apperrors.SanitizeURL(item.URL), "error", err)
			} else if len(matches) > 0 {
				// Filter out matches from the current job
				var crossJobMatches []CrossJobDuplicate
				for _, match := range matches {
					if match.JobID != req.RequestID {
						crossJobMatches = append(crossJobMatches, CrossJobDuplicate{
							JobID:     match.JobID,
							URL:       match.URL,
							Distance:  match.Distance,
							IndexedAt: match.IndexedAt.Format(time.RFC3339),
						})
					}
				}
				if len(crossJobMatches) > 0 {
					result.CrossJobDuplicates = crossJobMatches
					slog.Info("found cross-job duplicates", "url", apperrors.SanitizeURL(item.URL), "count", len(crossJobMatches))
				}
			}
		}

		// Index the content fingerprint for cross-job deduplication
		if req.ContentIndex != nil && pageSimHash != 0 {
			if err := req.ContentIndex.Index(ctx, req.RequestID, item.URL, pageSimHash); err != nil {
				slog.Warn("failed to index content fingerprint", "url", apperrors.SanitizeURL(item.URL), "error", err)
			}
		}

		finalResult, err := applyCrawlOutputPipeline(ctx, registry, baseCtx, result)
		if err != nil {
			slog.Error("crawl output pipeline failed", "url", apperrors.SanitizeURL(item.URL), "error", err)
			return PageResult{}, false, false
		}

		slog.Info("crawl page complete", "url", apperrors.SanitizeURL(item.URL), "status", res.Status, "title", result.Title)
		return finalResult, true, false
	}

	tasks := make(chan task, req.MaxPages)
	results := make(chan PageResult, req.MaxPages)
	var wg sync.WaitGroup
	var visitedMu sync.Mutex
	visited := map[string]bool{}
	var processed int32
	var pageSeqNum int32

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	enqueue := func(rawURL string, depth int) {
		if atomic.LoadInt32(&processed) >= int32(req.MaxPages) {
			return
		}

		// Parse URL to get path for pattern matching
		parsedURL, err := url.Parse(rawURL)
		if err != nil {
			slog.Debug("skipping invalid URL", "url", apperrors.SanitizeURL(rawURL), "error", err)
			return
		}

		// Check robots.txt if cache is provided
		if req.RobotsCache != nil {
			allowed, err := req.RobotsCache.IsAllowed(rawURL, req.UserAgent)
			if err != nil {
				slog.Debug("robots.txt check failed, allowing URL", "url", apperrors.SanitizeURL(rawURL), "error", err)
			} else if !allowed {
				slog.Info("skipping URL blocked by robots.txt", "url", apperrors.SanitizeURL(rawURL))
				return
			}
		}

		// Apply pattern filtering (skip root URL - it's always allowed)
		if parsedURL.Path != "" && parsedURL.Path != "/" {
			if !patternMatcher.Matches(parsedURL.Path) {
				slog.Debug("skipping URL due to pattern filter", "url", apperrors.SanitizeURL(rawURL), "path", parsedURL.Path)
				return
			}
		}

		norm := normalizeURL(rawURL)
		visitedMu.Lock()
		if visited[norm] {
			visitedMu.Unlock()
			return
		}
		visited[norm] = true
		visitedMu.Unlock()

		slog.Debug("enqueuing crawl task", "url", apperrors.SanitizeURL(rawURL), "depth", depth)
		select {
		case tasks <- task{URL: rawURL, Depth: depth}:
			wg.Add(1)
		default:
			slog.Warn("crawl task channel full", "url", apperrors.SanitizeURL(rawURL))
		case <-ctx.Done():
		}
	}

	// If sitemap URL provided, fetch and enqueue URLs from it
	if req.SitemapURL != "" {
		parser := NewSitemapParser(fetcher)
		sitemapURLs, err := parser.ParseSitemap(ctx, req.SitemapURL)
		if err != nil {
			slog.Error("failed to parse sitemap", "url", apperrors.SanitizeURL(req.SitemapURL), "error", err)
			// Don't fail the crawl - just log and continue
		} else {
			for _, u := range sitemapURLs {
				// Only enqueue same-host URLs
				if sameHost(startURL, u) {
					enqueue(u, 0) // Start sitemap URLs at depth 0
				}
			}
		}
	}

	// If SitemapOnly is true, don't crawl the root URL
	if !req.SitemapOnly {
		enqueue(req.URL, 0)
	} else if req.SitemapURL == "" {
		// If SitemapOnly but no sitemap URL, we have nothing to crawl
		return nil, apperrors.Validation("sitemapOnly requires sitemapURL")
	}

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

					seqNum := atomic.AddInt32(&pageSeqNum, 1)
					if atomic.AddInt32(&processed, 1) <= int32(req.MaxPages) {
						results <- res
						// Dispatch webhook event for this page
						req.dispatchPageEvent(ctx, res, item.Depth, int(seqNum))
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

// dispatchPageEvent sends a webhook notification for a crawled page.
func (req *Request) dispatchPageEvent(ctx context.Context, result PageResult, depth int, seqNum int) {
	if req.WebhookDispatcher == nil || req.WebhookConfig == nil {
		return
	}
	if !webhook.ShouldSendEvent(webhook.EventPageCrawled, "", req.WebhookConfig.Events) {
		return
	}

	payload := webhook.Payload{
		EventID:     fmt.Sprintf("%s-page-%d", req.RequestID, seqNum),
		EventType:   webhook.EventPageCrawled,
		Timestamp:   time.Now(),
		JobID:       req.RequestID,
		JobKind:     string(model.KindCrawl),
		PageURL:     result.URL,
		PageStatus:  result.Status,
		PageTitle:   result.Title,
		PageDepth:   depth,
		IsDuplicate: result.DuplicateOf != "",
		DuplicateOf: result.DuplicateOf,
		CrawlSeqNum: seqNum,
	}

	req.WebhookDispatcher.Dispatch(ctx, req.WebhookConfig.URL, payload, req.WebhookConfig.Secret)
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
		return PageResult{}, apperrors.Wrap(apperrors.KindInternal, "failed to marshal result", err)
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
		return PageResult{}, apperrors.Internal("pipeline output type mismatch for crawl")
	}
	return typed, nil
}
