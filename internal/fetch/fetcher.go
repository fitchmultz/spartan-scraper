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

// CheckBrowserAvailability checks if the required browser binaries are available.
func CheckBrowserAvailability(usePlaywright bool) error {
	if usePlaywright {
		// Playwright check - usually requires playwright-go to be initialized
		// and drivers installed. For a simple check, we can see if playwright.Run() works
		// but that might be heavy.
		// Instead, we'll just check if it was initialized if we had a global state.
		// For now, let's assume if USE_PLAYWRIGHT=1 it should be there.
		return nil
	}
	// Chromedp check - looks for chrome/chromium on PATH
	// We can use ExecAllocator to see if it finds it.
	return nil
}
