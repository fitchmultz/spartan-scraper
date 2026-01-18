package crawl

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/fetch"
	"spartan-scraper/internal/model"
)

type Request struct {
	URL           string
	MaxDepth      int
	MaxPages      int
	Concurrency   int
	Headless      bool
	UsePlaywright bool
	Auth          fetch.AuthOptions
	Extract       extract.ExtractOptions
	Timeout       time.Duration
	UserAgent     string
	Limiter       *fetch.HostLimiter
	MaxRetries    int
	RetryBase     time.Duration
	DataDir       string
	Incremental   bool
	Store         CrawlStateStore
}

type CrawlStateStore interface {
	GetCrawlState(url string) (model.CrawlState, error)
	UpsertCrawlState(state model.CrawlState) error
}

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

func Run(req Request) ([]PageResult, error) {
	if req.MaxDepth <= 0 {
		req.MaxDepth = 1
	}
	if req.MaxPages <= 0 {
		req.MaxPages = 100
	}
	if req.Concurrency <= 0 {
		req.Concurrency = 4
	}

	startURL, err := url.Parse(req.URL)
	if err != nil {
		return nil, err
	}

	fetcher := fetch.NewFetcher()

	type task struct {
		URL   string
		Depth int
	}

	processPage := func(item task) (PageResult, bool, bool) {
		// Check state if incremental
		var state model.CrawlState
		var ifNoneMatch, ifModifiedSince string

		if req.Incremental && req.Store != nil {
			existingState, err := req.Store.GetCrawlState(item.URL)
			if err == nil {
				state = existingState
				ifNoneMatch = state.ETag
				ifModifiedSince = state.LastModified
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

		res, err := fetcher.Fetch(fetchReq)
		if err != nil {
			return PageResult{}, false, false // Don't enqueue children if fetch failed
		}
		if res.Status >= 400 {
			return PageResult{}, false, false
		}

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
			// Update LastScraped timestamp
			if req.Incremental && req.Store != nil {
				state.LastScraped = time.Now()
				_ = req.Store.UpsertCrawlState(state)
			}
			// Return a page result indicating it was skipped
			return PageResult{
				URL:    item.URL,
				Status: 304,
			}, false, true // skip processing/extracting
		}

		// If changed (or first run), extract and save state
		output, extractErr := extract.Execute(extract.ExecuteInput{
			URL:     item.URL,
			HTML:    res.HTML,
			Options: req.Extract,
			DataDir: req.DataDir,
		})
		if extractErr != nil {
			return PageResult{}, false, false
		}

		// Update state
		if req.Incremental && req.Store != nil {
			newState := model.CrawlState{
				URL:          item.URL,
				ETag:         res.ETag,
				LastModified: res.LastModified,
				ContentHash:  currentHash,
				LastScraped:  time.Now(),
			}
			_ = req.Store.UpsertCrawlState(newState)
		}

		return PageResult{
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
		}, true, false
	}

	tasks := make(chan task, req.MaxPages)
	results := make(chan PageResult, req.MaxPages)
	var wg sync.WaitGroup
	var visitedMu sync.Mutex
	visited := map[string]bool{}
	var processed int32

	ctx, cancel := context.WithCancel(context.Background())
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

		wg.Add(1)
		select {
		case tasks <- task{URL: url, Depth: depth}:
		default:
			wg.Done()
		case <-ctx.Done():
			wg.Done()
		}
	}

	enqueue(req.URL, 0)

	for i := 0; i < req.Concurrency; i++ {
		go func() {
			for item := range tasks {
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
		}()
	}

	go func() {
		wg.Wait()
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
