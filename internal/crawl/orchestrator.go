package crawl

import (
	"context"
	"log/slog"
	"net/url"
	"sync"
	"sync/atomic"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

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

	// Simhash index for duplicate detection
	var simhashMu sync.Mutex
	seenSimHashes := make(map[uint64]string) // simhash -> URL mapping

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

					res, enqueueChildren, skipped := processPage(ctx, item, req, fetcher, registry, jsRegistry, seenSimHashes, &simhashMu)
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
