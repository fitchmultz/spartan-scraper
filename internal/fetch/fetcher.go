// Package fetch provides abstractions and implementations for fetching web content.
// It includes support for standard HTTP requests, headless browser rendering
// (via Chromedp or Playwright), rate limiting, and automatic retries with backoff.
package fetch

import "context"

type Fetcher interface {
	Fetch(ctx context.Context, req Request) (Result, error)
}

func NewFetcher() Fetcher {
	return NewAdaptiveFetcher()
}
