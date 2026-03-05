// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// HealthChecker defines the interface for proxy health checking.
type HealthChecker interface {
	Check(ctx context.Context, proxy ProxyEntry) (latencyMs int64, err error)
}

// DefaultHealthChecker makes HTTP request through proxy to test endpoint.
type DefaultHealthChecker struct {
	TestURL string
	Timeout time.Duration
}

// Check performs a health check on the given proxy.
func (c *DefaultHealthChecker) Check(ctx context.Context, proxy ProxyEntry) (latencyMs int64, err error) {
	testURL := c.TestURL
	if testURL == "" {
		testURL = "http://httpbin.org/ip"
	}

	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	proxyURL, err := url.Parse(proxy.URL)
	if err != nil {
		return 0, fmt.Errorf("invalid proxy URL: %w", err)
	}

	start := time.Now()

	// Create transport with proxy
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, testURL, nil)
	if err != nil {
		return 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	latency := time.Since(start).Milliseconds()
	return latency, nil
}
