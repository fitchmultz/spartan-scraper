package crawl

import (
	"net/url"
	"strings"
	"time"

	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/fetch"
)

type Request struct {
	URL       string
	MaxDepth  int
	MaxPages  int
	Headless  bool
	Auth      fetch.AuthOptions
	Timeout   time.Duration
	UserAgent string
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

	startURL, err := url.Parse(req.URL)
	if err != nil {
		return nil, err
	}

	fetcher := fetch.NewFetcher(req.Headless)

	queue := []struct {
		URL   string
		Depth int
	}{{URL: req.URL, Depth: 0}}
	visited := map[string]bool{}
	results := make([]PageResult, 0)

	for len(queue) > 0 && len(results) < req.MaxPages {
		item := queue[0]
		queue = queue[1:]

		norm := normalizeURL(item.URL)
		if visited[norm] {
			continue
		}
		visited[norm] = true

		res, err := fetcher.Fetch(fetch.Request{
			URL:       item.URL,
			Timeout:   req.Timeout,
			UserAgent: req.UserAgent,
			Headless:  req.Headless,
			Auth:      req.Auth,
		})
		if err != nil {
			continue
		}

		extracted, err := extract.FromHTML(res.HTML)
		if err != nil {
			continue
		}

		results = append(results, PageResult{
			URL:      item.URL,
			Status:   res.Status,
			Title:    extracted.Title,
			Text:     extracted.Text,
			Links:    extracted.Links,
			Metadata: extracted,
		})

		if item.Depth >= req.MaxDepth {
			continue
		}

		for _, href := range extracted.Links {
			resolved := resolveURL(startURL, href)
			if resolved == "" {
				continue
			}
			if !sameHost(startURL, resolved) {
				continue
			}
			queue = append(queue, struct {
				URL   string
				Depth int
			}{URL: resolved, Depth: item.Depth + 1})
		}
	}

	return results, nil
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
