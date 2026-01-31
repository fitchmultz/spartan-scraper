// Package fetch provides HTTP and headless browser content fetching capabilities.
//
// This file handles network request/response interception for Playwright-based API scraping.
// It provides the playwrightInterceptor type for capturing network traffic based on
// configurable URL patterns and resource types. Does NOT handle request execution
// or browser lifecycle management.
package fetch

import (
	"log/slog"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"
)

// playwrightInterceptor captures network requests and responses for API scraping.
type playwrightInterceptor struct {
	config  NetworkInterceptConfig
	mu      sync.Mutex
	entries []InterceptedEntry
	pending map[string]*InterceptedRequest // URL -> request (for matching)
}

func newPlaywrightInterceptor(config NetworkInterceptConfig) *playwrightInterceptor {
	return &playwrightInterceptor{
		config:  config,
		entries: make([]InterceptedEntry, 0, config.MaxEntries),
		pending: make(map[string]*InterceptedRequest),
	}
}

func (pi *playwrightInterceptor) shouldIntercept(url string, resourceType string) bool {
	// Check resource type
	if len(pi.config.ResourceTypes) > 0 {
		matched := false
		for _, rt := range pi.config.ResourceTypes {
			if string(rt) == resourceType {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check URL patterns
	if len(pi.config.URLPatterns) == 0 {
		return true
	}
	for _, pattern := range pi.config.URLPatterns {
		if matchGlob(pattern, url) {
			return true
		}
	}
	return false
}

func (pi *playwrightInterceptor) handleRoute(route playwright.Route) {
	req := route.Request()
	url := req.URL()
	resourceType := req.ResourceType()

	if !pi.shouldIntercept(url, resourceType) {
		_ = route.Continue()
		return
	}

	pi.mu.Lock()
	// Check max entries
	if len(pi.entries) >= pi.config.MaxEntries {
		pi.mu.Unlock()
		slog.Warn("playwright interceptor max entries reached, dropping request", "url", url)
		_ = route.Continue()
		return
	}

	interceptedReq := &InterceptedRequest{
		RequestID:    req.URL(), // Use URL as ID since Playwright doesn't expose request ID
		URL:          url,
		Method:       req.Method(),
		Headers:      make(map[string]string),
		Timestamp:    time.Now(),
		ResourceType: InterceptedResourceType(resourceType),
	}

	// Copy headers
	for k, v := range req.Headers() {
		interceptedReq.Headers[k] = v
	}

	// Capture request body if enabled
	if pi.config.CaptureRequestBody {
		if postData, err := req.PostData(); err == nil && postData != "" {
			body := postData
			if int64(len(body)) > pi.config.MaxBodySize {
				body = body[:pi.config.MaxBodySize]
			}
			interceptedReq.Body = body
			interceptedReq.BodySize = int64(len(postData))
		}
	}

	pi.pending[url] = interceptedReq
	pi.mu.Unlock()

	// Continue the request and capture response
	_ = route.Continue()

	// Note: Response capture happens via page event listeners
}

func (pi *playwrightInterceptor) onResponse(resp playwright.Response) {
	req := resp.Request()
	url := req.URL()

	pi.mu.Lock()
	interceptedReq, exists := pi.pending[url]
	if !exists {
		pi.mu.Unlock()
		return
	}
	delete(pi.pending, url)

	// Check max entries again
	if len(pi.entries) >= pi.config.MaxEntries {
		pi.mu.Unlock()
		return
	}

	interceptedResp := &InterceptedResponse{
		RequestID:  interceptedReq.RequestID,
		Status:     resp.Status(),
		StatusText: resp.StatusText(),
		Headers:    make(map[string]string),
		Timestamp:  time.Now(),
	}

	// Copy headers
	for k, v := range resp.Headers() {
		interceptedResp.Headers[k] = v
	}

	// Capture response body if enabled
	if pi.config.CaptureResponseBody {
		if body, err := resp.Body(); err == nil && len(body) > 0 {
			bodyStr := string(body)
			if int64(len(bodyStr)) > pi.config.MaxBodySize {
				bodyStr = bodyStr[:pi.config.MaxBodySize]
			}
			interceptedResp.Body = bodyStr
			interceptedResp.BodySize = int64(len(bodyStr))
		}
	}

	entry := InterceptedEntry{
		Request:  *interceptedReq,
		Response: interceptedResp,
		Duration: interceptedResp.Timestamp.Sub(interceptedReq.Timestamp),
	}
	pi.entries = append(pi.entries, entry)
	pi.mu.Unlock()
}

func (pi *playwrightInterceptor) onRequestFailed(req playwright.Request) {
	url := req.URL()

	pi.mu.Lock()
	interceptedReq, exists := pi.pending[url]
	if !exists {
		pi.mu.Unlock()
		return
	}
	delete(pi.pending, url)

	entry := InterceptedEntry{
		Request:  *interceptedReq,
		Response: nil,
		Duration: time.Since(interceptedReq.Timestamp),
	}
	pi.entries = append(pi.entries, entry)
	pi.mu.Unlock()
}

func (pi *playwrightInterceptor) getEntries() []InterceptedEntry {
	pi.mu.Lock()
	defer pi.mu.Unlock()
	result := make([]InterceptedEntry, len(pi.entries))
	copy(result, pi.entries)
	return result
}
