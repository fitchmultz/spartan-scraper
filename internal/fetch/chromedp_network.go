// Package fetch provides HTTP and headless browser content fetching capabilities.
//
// This file provides network idle detection and response tracking for chromedp.
// It tracks active network requests to determine when page loading is complete
// and captures HTTP response status codes from document requests.
package fetch

import (
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/cdproto/network"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// networkTracker tracks active network requests to detect idle state.
type networkTracker struct {
	inflight      int32         // Number of active network requests (atomic)
	mu            sync.Mutex    // Protects idleSince, done, and closed
	idleSince     time.Time     // When inflight first reached 0
	quietDuration time.Duration // How long to wait at 0 inflight before declaring idle
	done          chan struct{} // Closed when network idle is confirmed
	closed        int32         // 0 = open, 1 = closed (atomic for double-close protection)
	firstSeen     int32         // 0 = not seen, 1 = seen (atomic)
}

// responseTracker captures the main document response status.
type responseTracker struct {
	mu        sync.Mutex
	captured  bool
	status    int64
	targetURL string
}

func (t *networkTracker) onEvent(ev any) {
	switch ev := ev.(type) {
	case *network.EventRequestWillBeSent:
		atomic.StoreInt32(&t.firstSeen, 1)
		newCount := atomic.AddInt32(&t.inflight, 1)
		t.resetIdleSince()
		slog.Debug("request started", "requestId", ev.RequestID, "inflight", newCount)

	case *network.EventLoadingFinished:
		atomic.StoreInt32(&t.firstSeen, 1)
		newCount := atomic.AddInt32(&t.inflight, -1)
		if newCount < 0 {
			slog.Warn("inflight counter went negative", "count", newCount, "requestId", ev.RequestID)
			atomic.StoreInt32(&t.inflight, 0)
			newCount = 0
		}
		slog.Debug("request finished", "requestId", ev.RequestID, "inflight", newCount)
		t.checkIdle()

	case *network.EventLoadingFailed:
		atomic.StoreInt32(&t.firstSeen, 1)
		newCount := atomic.AddInt32(&t.inflight, -1)
		if newCount < 0 {
			slog.Warn("inflight counter went negative", "count", newCount, "requestId", ev.RequestID)
			atomic.StoreInt32(&t.inflight, 0)
			newCount = 0
		}
		slog.Debug("request failed", "requestId", ev.RequestID, "inflight", newCount)
		t.checkIdle()
	}
}

func (t *networkTracker) resetIdleSince() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.idleSince = time.Time{}
}

func (t *networkTracker) checkIdle() {
	if atomic.LoadInt32(&t.inflight) == 0 {
		t.mu.Lock()
		defer t.mu.Unlock()

		if atomic.LoadInt32(&t.closed) != 0 {
			return
		}

		if t.idleSince.IsZero() {
			t.idleSince = time.Now()
		} else if time.Since(t.idleSince) >= t.quietDuration {
			atomic.StoreInt32(&t.closed, 1)
			close(t.done)
		}
	} else {
		t.resetIdleSince()
	}
}

// onEvent captures the first document response status.
// Note: For redirect chains (e.g., 302 -> 200), this captures the redirect status (302).
// This is intentional as it represents the HTTP-level navigation result.
func (rt *responseTracker) onEvent(ev any) {
	rt.mu.Lock()
	// Double-check: if already captured, release lock and return
	if rt.captured {
		rt.mu.Unlock()
		return
	}

	evResp, ok := ev.(*network.EventResponseReceived)
	if !ok {
		rt.mu.Unlock()
		return
	}

	// Capture the status of the main document request
	if evResp.Type == network.ResourceTypeDocument {
		// Check if the URL matches our target (allowing for redirects)
		// We require that one URL is a prefix of the other AND they share
		// the same scheme and netloc (host:port) to avoid false matches.
		respURL := evResp.Response.URL
		if rt.urlsMatch(respURL, rt.targetURL) {
			rt.status = evResp.Response.Status
			rt.captured = true
			rt.mu.Unlock()
			slog.Debug("captured response status", "url", apperrors.SanitizeURL(respURL), "target", apperrors.SanitizeURL(rt.targetURL), "status", rt.status)
			return
		}
		slog.Debug("document response URL does not match target", "respURL", apperrors.SanitizeURL(respURL), "targetURL", apperrors.SanitizeURL(rt.targetURL))
	}
	rt.mu.Unlock()
}

func (rt *responseTracker) getStatus() int64 {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt.captured {
		return rt.status
	}
	return 0
}

// urlsMatch checks if two URLs match for the purpose of identifying the main navigation.
// It returns true if:
// 1. The URLs are exactly equal, OR
// 2. One is a prefix of the other AND they share the same scheme and host
//
// This allows for URL redirects (e.g., https://example.com -> https://example.com/welcome)
// while preventing false matches (e.g., https://example.com/a matching https://other.com/a).
//
// Examples:
//   - https://example.com matches https://example.com/path (same host, prefix)
//   - https://example.com:8080 matches https://example.com:8080/api (same host+port)
//   - https://example.com does NOT match https://other.com (different host)
//   - https://example.com does NOT match http://example.com (different scheme)
//   - https://example.com/api does NOT match https://example.com/app (no prefix relationship)
func (rt *responseTracker) urlsMatch(a, b string) bool {
	if a == b {
		return true
	}

	ua, errA := url.Parse(a)
	ub, errB := url.Parse(b)
	if errA != nil || errB != nil {
		return false
	}

	// Must match scheme and host (host includes port for non-standard ports)
	if ua.Scheme != ub.Scheme || ua.Host != ub.Host {
		return false
	}

	// One path must be a prefix of the other (allows for redirects)
	// Normalize paths to ensure /path and /path/ match consistently
	aPath := strings.TrimSuffix(ua.Path, "/")
	bPath := strings.TrimSuffix(ub.Path, "/")
	if aPath == "" {
		aPath = "/"
	}
	if bPath == "" {
		bPath = "/"
	}

	return strings.HasPrefix(aPath, bPath) || strings.HasPrefix(bPath, aPath)
}
