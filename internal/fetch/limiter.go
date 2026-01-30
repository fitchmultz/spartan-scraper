// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"context"
	"net/url"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// HostStatus represents the current rate limit state for a single host
type HostStatus struct {
	Host        string
	QPS         float64
	Burst       int
	LastRequest time.Time
}

// HostLimiter manages per-host rate limiters
type HostLimiter struct {
	qps    rate.Limit
	burst  int
	mu     sync.Mutex
	byHost map[string]*rate.Limiter

	// Extended tracking for metrics
	hostInfo map[string]*hostLimiterInfo
}

type hostLimiterInfo struct {
	limiter     *rate.Limiter
	lastRequest time.Time
	qps         rate.Limit // per-host QPS (may differ from global)
	burst       int        // per-host burst (may differ from global)
}

func NewHostLimiter(qps int, burst int) *HostLimiter {
	limit := rate.Limit(qps)
	if qps <= 0 {
		limit = rate.Inf
	}
	if burst <= 0 {
		burst = 1
	}
	return &HostLimiter{
		qps:      limit,
		burst:    burst,
		byHost:   map[string]*rate.Limiter{},
		hostInfo: map[string]*hostLimiterInfo{},
	}
}

// Wait waits for the rate limiter for the given URL using global default rates.
// For per-host rate configuration, use WaitWithRates instead.
func (h *HostLimiter) Wait(ctx context.Context, rawURL string) error {
	return h.WaitWithRates(ctx, rawURL, 0, 0)
}

// WaitWithRates waits for the rate limiter for the given URL with optional per-host rates.
// If profileQPS or profileBurst are 0, the global defaults are used.
func (h *HostLimiter) WaitWithRates(ctx context.Context, rawURL string, profileQPS int, profileBurst int) error {
	if h == nil || h.qps == rate.Inf {
		return nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil
	}
	host := parsed.Host
	if host == "" {
		return nil
	}

	limiter := h.getLimiterWithRates(host, profileQPS, profileBurst)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = limiter.Wait(ctx)
	if err == nil {
		h.recordRequest(host)
	}
	return err
}

// getLimiterWithRates returns a limiter for the host, creating one if needed.
// If the host already has a limiter, it will be reused (rate changes only take
// effect for new hosts or after restart).
func (h *HostLimiter) getLimiterWithRates(host string, qps int, burst int) *rate.Limiter {
	h.mu.Lock()
	defer h.mu.Unlock()

	if info, ok := h.hostInfo[host]; ok {
		return info.limiter
	}

	// Use provided rates or fall back to global defaults
	limit := rate.Limit(qps)
	if qps <= 0 {
		limit = h.qps
	}
	if burst <= 0 {
		burst = h.burst
	}
	if burst <= 0 {
		burst = 1
	}

	limiter := rate.NewLimiter(limit, burst)
	h.byHost[host] = limiter
	h.hostInfo[host] = &hostLimiterInfo{
		limiter: limiter,
		qps:     limit,
		burst:   burst,
	}
	return limiter
}

// getLimiter returns a limiter for the host using global default rates.
// Deprecated: Use getLimiterWithRates for per-host rate configuration.
func (h *HostLimiter) getLimiter(host string) *rate.Limiter {
	return h.getLimiterWithRates(host, 0, 0)
}

// recordRequest updates the last request time for a host
func (h *HostLimiter) recordRequest(host string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if info, ok := h.hostInfo[host]; ok {
		info.lastRequest = time.Now()
	}
}

// GetHostStatus returns rate limit status for all known hosts
func (h *HostLimiter) GetHostStatus() []HostStatus {
	h.mu.Lock()
	defer h.mu.Unlock()

	result := make([]HostStatus, 0, len(h.hostInfo))
	for host, info := range h.hostInfo {
		qps := float64(info.qps)
		if info.qps == rate.Inf {
			qps = 0 // 0 indicates unlimited
		}
		result = append(result, HostStatus{
			Host:        host,
			QPS:         qps,
			Burst:       info.burst,
			LastRequest: info.lastRequest,
		})
	}
	return result
}

// GetLimiter returns the rate limiter for a specific host (for metrics registration)
func (h *HostLimiter) GetLimiter(host string) *rate.Limiter {
	h.mu.Lock()
	defer h.mu.Unlock()
	if info, ok := h.hostInfo[host]; ok {
		return info.limiter
	}
	return nil
}

// GetQPS returns the configured QPS
func (h *HostLimiter) GetQPS() float64 {
	return float64(h.qps)
}

// GetBurst returns the configured burst
func (h *HostLimiter) GetBurst() int {
	return h.burst
}
