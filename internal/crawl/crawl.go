package crawl

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/fetch"
)

type Request struct {
	URL           string
	MaxDepth      int
	MaxPages      int
	Concurrency   int
	Headless      bool
	UsePlaywright bool
	Auth          fetch.AuthOptions
	Timeout       time.Duration
	UserAgent     string
	Limiter       *fetch.HostLimiter
	MaxRetries    int
	RetryBase     time.Duration
	DataDir       string
}

type PageResult struct {
	URL      string         `json:"url"`
	Status   int            `json:"status"`
	Title    string         `json:"title"`
	Text     string         `json:"text"`
	Links    []string       `json:"links"`
	Metadata extract.Result `json:"metadata"`
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

				res, err := fetcher.Fetch(fetch.Request{
					URL:            item.URL,
					Timeout:        req.Timeout,
					UserAgent:      req.UserAgent,
					Headless:       req.Headless,
					UsePlaywright:  req.UsePlaywright,
					Auth:           req.Auth,
					Limiter:        req.Limiter,
					MaxRetries:     req.MaxRetries,
					RetryBaseDelay: req.RetryBase,
					DataDir:        req.DataDir,
				})
				if err == nil {
					if res.Status >= 400 {
						wg.Done()
						continue
					}
					extracted, extractErr := extract.FromHTML(res.HTML)
					if extractErr == nil {
						if atomic.AddInt32(&processed, 1) <= int32(req.MaxPages) {
							results <- PageResult{
								URL:      item.URL,
								Status:   res.Status,
								Title:    extracted.Title,
								Text:     extracted.Text,
								Links:    extracted.Links,
								Metadata: extracted,
							}
						}

						if item.Depth < req.MaxDepth {
							for _, href := range extracted.Links {
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
