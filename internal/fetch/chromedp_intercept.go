// Package fetch provides HTTP and headless browser content fetching capabilities.
//
// This file handles network request/response interception for API scraping.
// It provides the networkInterceptor type for capturing network traffic based on
// configurable URL patterns and resource types. Does NOT handle request execution
// or browser lifecycle management.
package fetch

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// networkInterceptor captures network requests and responses for API scraping.
type networkInterceptor struct {
	config    NetworkInterceptConfig
	mu        sync.Mutex
	requests  map[network.RequestID]*InterceptedRequest
	responses map[network.RequestID]*InterceptedResponse
	entries   []InterceptedEntry
	ctx       context.Context
	cancel    context.CancelFunc
}

func newNetworkInterceptor(config NetworkInterceptConfig) *networkInterceptor {
	ctx, cancel := context.WithCancel(context.Background())
	return &networkInterceptor{
		config:    config,
		requests:  make(map[network.RequestID]*InterceptedRequest),
		responses: make(map[network.RequestID]*InterceptedResponse),
		entries:   make([]InterceptedEntry, 0, config.MaxEntries),
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (ni *networkInterceptor) shouldIntercept(url string, resourceType network.ResourceType) bool {
	// Check resource type (case-insensitive comparison)
	if len(ni.config.ResourceTypes) > 0 {
		matched := false
		rtStr := strings.ToLower(string(resourceType))
		for _, rt := range ni.config.ResourceTypes {
			if strings.ToLower(string(rt)) == rtStr {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check URL patterns
	if len(ni.config.URLPatterns) == 0 {
		return true
	}
	for _, pattern := range ni.config.URLPatterns {
		if matchGlob(pattern, url) {
			return true
		}
	}
	return false
}

func (ni *networkInterceptor) onEvent(ev any) {
	switch ev := ev.(type) {
	case *network.EventRequestWillBeSent:
		if !ni.shouldIntercept(ev.Request.URL, ev.Type) {
			return
		}
		ni.mu.Lock()
		defer ni.mu.Unlock()

		// Check max entries
		if len(ni.entries) >= ni.config.MaxEntries {
			slog.Warn("network interceptor max entries reached, dropping request", "requestId", ev.RequestID)
			return
		}

		req := &InterceptedRequest{
			RequestID:    string(ev.RequestID),
			URL:          ev.Request.URL,
			Method:       ev.Request.Method,
			Headers:      make(map[string]string),
			Timestamp:    time.Now(),
			ResourceType: InterceptedResourceType(ev.Type),
		}

		// Copy headers
		for k, v := range ev.Request.Headers {
			if str, ok := v.(string); ok {
				req.Headers[k] = str
			}
		}

		// Capture request body if enabled and present
		// Note: PostData may be available in the event depending on CDP version
		// For now, we capture what we can from headers and URL

		ni.requests[ev.RequestID] = req

	case *network.EventResponseReceived:
		ni.mu.Lock()
		interceptedReq, exists := ni.requests[ev.RequestID]
		ni.mu.Unlock()
		if !exists {
			return
		}

		resp := &InterceptedResponse{
			RequestID:  string(ev.RequestID),
			Status:     int(ev.Response.Status),
			StatusText: ev.Response.StatusText,
			Headers:    make(map[string]string),
			Timestamp:  time.Now(),
			BodySize:   int64(ev.Response.EncodedDataLength),
		}

		// Copy headers
		for k, v := range ev.Response.Headers {
			if str, ok := v.(string); ok {
				resp.Headers[k] = str
			}
		}

		ni.mu.Lock()
		ni.responses[ev.RequestID] = resp
		ni.mu.Unlock()

		// Fetch response body asynchronously
		if ni.config.CaptureResponseBody {
			go ni.fetchResponseBody(ev.RequestID, resp, interceptedReq)
		} else {
			// Create entry immediately if not capturing body
			ni.mu.Lock()
			entry := InterceptedEntry{
				Request:  *interceptedReq,
				Response: resp,
				Duration: resp.Timestamp.Sub(interceptedReq.Timestamp),
			}
			ni.entries = append(ni.entries, entry)
			delete(ni.requests, ev.RequestID)
			delete(ni.responses, ev.RequestID)
			ni.mu.Unlock()
		}

	case *network.EventLoadingFailed:
		ni.mu.Lock()
		req, exists := ni.requests[ev.RequestID]
		if exists {
			// Create entry with nil response for failed requests
			entry := InterceptedEntry{
				Request:  *req,
				Response: nil,
				Duration: time.Since(req.Timestamp),
			}
			ni.entries = append(ni.entries, entry)
			delete(ni.requests, ev.RequestID)
			delete(ni.responses, ev.RequestID)
		}
		ni.mu.Unlock()
	}
}

func (ni *networkInterceptor) fetchResponseBody(requestID network.RequestID, resp *InterceptedResponse, req *InterceptedRequest) {
	// Use chromedp to fetch the response body
	var body []byte
	err := chromedp.Run(ni.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		body, err = network.GetResponseBody(requestID).Do(ctx)
		return err
	}))

	ni.mu.Lock()
	defer ni.mu.Unlock()

	// Check if request still exists (might have been cleaned up)
	if _, exists := ni.requests[requestID]; !exists {
		return
	}

	if err == nil && len(body) > 0 {
		bodyStr := string(body)
		if int64(len(bodyStr)) > ni.config.MaxBodySize {
			bodyStr = bodyStr[:ni.config.MaxBodySize]
		}
		resp.Body = bodyStr
		resp.BodySize = int64(len(bodyStr))
	}

	// Create entry
	entry := InterceptedEntry{
		Request:  *req,
		Response: resp,
		Duration: resp.Timestamp.Sub(req.Timestamp),
	}
	ni.entries = append(ni.entries, entry)

	// Cleanup
	delete(ni.requests, requestID)
	delete(ni.responses, requestID)
}

func (ni *networkInterceptor) getEntries() []InterceptedEntry {
	ni.mu.Lock()
	defer ni.mu.Unlock()
	result := make([]InterceptedEntry, len(ni.entries))
	copy(result, ni.entries)
	return result
}

func (ni *networkInterceptor) stop() {
	ni.cancel()
}

// matchGlob performs simple glob matching for URL patterns.
// Supports * (matches any sequence) and ** (matches any path segments).
func matchGlob(pattern, s string) bool {
	// Handle exact match
	if pattern == s {
		return true
	}

	// Simple glob implementation - convert pattern to a basic matcher
	if pattern == "**/*" || pattern == "*" {
		return true
	}

	// Handle **/ prefix (matches any path segments)
	if strings.HasPrefix(pattern, "**/") {
		suffix := pattern[3:]
		// For patterns like **/api/**, we need to find the middle part anywhere in the string
		// and then match the rest
		if strings.Contains(suffix, "*") {
			// Split the suffix by * and find each part in the string
			parts := strings.Split(suffix, "*")
			currentPos := 0
			for _, part := range parts {
				if part == "" {
					continue
				}
				idx := strings.Index(s[currentPos:], part)
				if idx == -1 {
					return false
				}
				currentPos += idx + len(part)
			}
			return true
		}
		// Simple suffix match for **/api pattern
		return strings.Contains(s, suffix)
	}

	// Handle **/ suffix (matches any path segments at end)
	if strings.HasSuffix(pattern, "/**") {
		prefix := pattern[:len(pattern)-3]
		return strings.HasPrefix(s, prefix)
	}

	// Handle * wildcard in the middle
	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "*")
		// Remove empty strings from parts
		var nonEmptyParts []string
		for _, p := range parts {
			if p != "" {
				nonEmptyParts = append(nonEmptyParts, p)
			}
		}

		if len(nonEmptyParts) == 0 {
			return true
		}

		// Check that all parts appear in order
		currentPos := 0
		for i, part := range nonEmptyParts {
			idx := strings.Index(s[currentPos:], part)
			if idx == -1 {
				return false
			}
			// First part must be at the beginning if pattern doesn't start with *
			if i == 0 && !strings.HasPrefix(pattern, "*") && idx != 0 {
				return false
			}
			currentPos += idx + len(part)
		}

		// Last part must be at the end if pattern doesn't end with *
		if !strings.HasSuffix(pattern, "*") {
			lastPart := nonEmptyParts[len(nonEmptyParts)-1]
			if !strings.HasSuffix(s, lastPart) {
				return false
			}
		}

		return true
	}

	return strings.Contains(s, pattern)
}
