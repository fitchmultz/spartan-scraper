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

func (h *HostLimiter) Wait(ctx context.Context, rawURL string) error {
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

	limiter := h.getLimiter(host)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = limiter.Wait(ctx)
	if err == nil {
		h.recordRequest(host)
	}
	return err
}

func (h *HostLimiter) getLimiter(host string) *rate.Limiter {
	h.mu.Lock()
	defer h.mu.Unlock()
	if info, ok := h.hostInfo[host]; ok {
		return info.limiter
	}
	limiter := rate.NewLimiter(h.qps, h.burst)
	h.byHost[host] = limiter
	h.hostInfo[host] = &hostLimiterInfo{
		limiter: limiter,
	}
	return limiter
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
		result = append(result, HostStatus{
			Host:        host,
			QPS:         float64(h.qps),
			Burst:       h.burst,
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
