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

type HostLimiter struct {
	qps    rate.Limit
	burst  int
	mu     sync.Mutex
	byHost map[string]*rate.Limiter
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
		qps:    limit,
		burst:  burst,
		byHost: map[string]*rate.Limiter{},
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
	return limiter.Wait(ctx)
}

func (h *HostLimiter) getLimiter(host string) *rate.Limiter {
	h.mu.Lock()
	defer h.mu.Unlock()
	if limiter, ok := h.byHost[host]; ok {
		return limiter
	}
	limiter := rate.NewLimiter(h.qps, h.burst)
	h.byHost[host] = limiter
	return limiter
}
