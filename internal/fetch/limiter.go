// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"context"
	"log/slog"
	"net/url"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"golang.org/x/time/rate"
)

// HostStatus represents the current rate limit state for a single host
type HostStatus struct {
	Host        string
	QPS         float64
	Burst       int
	LastRequest time.Time
	// Adaptive rate limiting fields
	CurrentQPS           float64 // actual QPS after adaptation
	AdaptiveEnabled      bool
	ConsecutiveSuccesses int
	Consecutive429s      int
	InCooldown           bool
	CooldownUntil        time.Time
	// Circuit breaker fields
	CircuitBreakerState    string    // closed, open, half-open
	CircuitBreakerFailures int       // Current failure count
	CircuitBreakerLastFail time.Time // Last failure timestamp
}

// AdaptiveConfig controls the behavior of adaptive rate limiting.
// When enabled, the limiter dynamically adjusts QPS per host based on
// server responses (429 status codes, Retry-After headers) and successful
// request patterns using an additive increase/multiplicative decrease algorithm.
type AdaptiveConfig struct {
	Enabled                bool
	MinQPS                 rate.Limit    // floor (e.g., 0.1 = 1 req per 10s)
	MaxQPS                 rate.Limit    // ceiling (initial QPS)
	AdditiveIncrease       rate.Limit    // QPS to add on success (e.g., 0.5)
	MultiplicativeDecrease float64       // factor to multiply on 429 (e.g., 0.5 = halve)
	SuccessThreshold       int           // consecutive successes before increase
	CooldownPeriod         time.Duration // minimum time between adjustments
}

// HostLimiter manages per-host rate limiters
type HostLimiter struct {
	qps    rate.Limit
	burst  int
	mu     sync.Mutex
	byHost map[string]*rate.Limiter

	// Extended tracking for metrics
	hostInfo map[string]*hostLimiterInfo

	// Adaptive rate limiting configuration (nil if disabled)
	adaptive *AdaptiveConfig

	// Circuit breaker for per-host failure isolation (nil if disabled)
	circuitBreaker *CircuitBreaker
}

type hostLimiterInfo struct {
	limiter     *rate.Limiter
	lastRequest time.Time
	qps         rate.Limit // per-host QPS (may differ from global)
	burst       int        // per-host burst (may differ from global)
	// Adaptive rate limiting state
	currentQPS           rate.Limit // dynamically adjusted QPS
	consecutiveSuccesses int        // for additive increase
	consecutive429s      int        // for multiplicative decrease
	lastRateAdjustment   time.Time  // prevent thrashing
	cooldownUntil        time.Time  // Retry-After cooldown
}

func NewHostLimiter(qps int, burst int) *HostLimiter {
	return newHostLimiterWithAdaptiveAndCircuitBreaker(qps, burst, nil, nil)
}

// NewAdaptiveHostLimiter creates a HostLimiter with adaptive rate limiting enabled.
// The limiter will dynamically adjust QPS per host based on server responses.
func NewAdaptiveHostLimiter(qps int, burst int, cfg *AdaptiveConfig) *HostLimiter {
	return newHostLimiterWithAdaptiveAndCircuitBreaker(qps, burst, cfg, nil)
}

// NewHostLimiterWithCircuitBreaker creates a HostLimiter with circuit breaker enabled.
// The circuit breaker will isolate failing hosts to prevent cascading failures.
func NewHostLimiterWithCircuitBreaker(qps int, burst int, cb *CircuitBreaker) *HostLimiter {
	return newHostLimiterWithAdaptiveAndCircuitBreaker(qps, burst, nil, cb)
}

// NewAdaptiveHostLimiterWithCircuitBreaker creates a HostLimiter with both adaptive
// rate limiting and circuit breaker enabled.
func NewAdaptiveHostLimiterWithCircuitBreaker(qps int, burst int, adaptiveCfg *AdaptiveConfig, cb *CircuitBreaker) *HostLimiter {
	return newHostLimiterWithAdaptiveAndCircuitBreaker(qps, burst, adaptiveCfg, cb)
}

func newHostLimiterWithAdaptiveAndCircuitBreaker(qps int, burst int, adaptiveCfg *AdaptiveConfig, cb *CircuitBreaker) *HostLimiter {
	limit := rate.Limit(qps)
	if qps <= 0 {
		limit = rate.Inf
	}
	if burst <= 0 {
		burst = 1
	}

	// Set up adaptive config defaults if provided
	if adaptiveCfg != nil {
		if adaptiveCfg.MinQPS <= 0 {
			adaptiveCfg.MinQPS = 0.1 // 1 req per 10 seconds minimum
		}
		if adaptiveCfg.MaxQPS <= 0 {
			adaptiveCfg.MaxQPS = limit
		}
		if adaptiveCfg.AdditiveIncrease <= 0 {
			adaptiveCfg.AdditiveIncrease = 0.5 // add 0.5 QPS per increase
		}
		if adaptiveCfg.MultiplicativeDecrease <= 0 {
			adaptiveCfg.MultiplicativeDecrease = 0.5 // halve the rate
		}
		if adaptiveCfg.SuccessThreshold <= 0 {
			adaptiveCfg.SuccessThreshold = 5 // 5 consecutive successes before increase
		}
		if adaptiveCfg.CooldownPeriod <= 0 {
			adaptiveCfg.CooldownPeriod = time.Second // 1 second between adjustments
		}
	}

	return &HostLimiter{
		qps:            limit,
		burst:          burst,
		byHost:         map[string]*rate.Limiter{},
		hostInfo:       map[string]*hostLimiterInfo{},
		adaptive:       adaptiveCfg,
		circuitBreaker: cb,
	}
}

// Wait waits for the rate limiter for the given URL using global default rates.
// For per-host rate configuration, use WaitWithRates instead.
func (h *HostLimiter) Wait(ctx context.Context, rawURL string) error {
	return h.WaitWithRates(ctx, rawURL, 0, 0)
}

// WaitWithRates waits for the rate limiter for the given URL with optional per-host rates.
// If profileQPS or profileBurst are 0, the global defaults are used.
// This method also checks the circuit breaker before allowing the request.
func (h *HostLimiter) WaitWithRates(ctx context.Context, rawURL string, profileQPS int, profileBurst int) error {
	if h == nil {
		return nil
	}

	// When global QPS is unlimited but adaptive mode is enabled, we still need to
	// enforce adaptive rate limits. Skip early return if adaptive is enabled.
	// Also need to check circuit breaker even with unlimited QPS.
	if h.qps == rate.Inf && !h.IsAdaptiveEnabled() && !h.IsCircuitBreakerEnabled() {
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

	// Check circuit breaker first
	if h.circuitBreaker != nil && !h.circuitBreaker.Allow(host) {
		return apperrors.Wrap(apperrors.KindInternal,
			"circuit breaker open for host "+host,
			ErrCircuitBreakerOpen)
	}

	// Check for cooldown period (adaptive rate limiting)
	if waitTime := h.getCooldownWaitTime(host); waitTime > 0 {
		select {
		case <-time.After(waitTime):
			// Continue after cooldown
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	limiter := h.getLimiterWithRates(host, profileQPS, profileBurst)

	err = limiter.Wait(ctx)
	if err == nil {
		h.recordRequest(host)
	}
	return err
}

// getCooldownWaitTime returns the remaining cooldown time for a host, or 0 if no cooldown.
func (h *HostLimiter) getCooldownWaitTime(host string) time.Duration {
	if h == nil || h.adaptive == nil || !h.adaptive.Enabled {
		return 0
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	info, ok := h.hostInfo[host]
	if !ok {
		return 0
	}

	now := time.Now()
	if now.Before(info.cooldownUntil) {
		return info.cooldownUntil.Sub(now)
	}
	return 0
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
	info := &hostLimiterInfo{
		limiter: limiter,
		qps:     limit,
		burst:   burst,
	}

	// Initialize adaptive state if adaptive mode is enabled
	if h.adaptive != nil && h.adaptive.Enabled {
		info.currentQPS = limit
		if limit == rate.Inf {
			info.currentQPS = h.adaptive.MaxQPS
		}
	}

	h.byHost[host] = limiter
	h.hostInfo[host] = info
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

	// Get circuit breaker status if enabled
	var cbStatuses map[string]CircuitBreakerHostStatus
	if h.circuitBreaker != nil {
		cbStatuses = make(map[string]CircuitBreakerHostStatus)
		for _, s := range h.circuitBreaker.GetHostStatus() {
			cbStatuses[s.Host] = s
		}
	}

	result := make([]HostStatus, 0, len(h.hostInfo))
	for host, info := range h.hostInfo {
		qps := float64(info.qps)
		if info.qps == rate.Inf {
			qps = 0 // 0 indicates unlimited
		}

		status := HostStatus{
			Host:        host,
			QPS:         qps,
			Burst:       info.burst,
			LastRequest: info.lastRequest,
		}

		// Add adaptive fields if adaptive mode is enabled
		if h.adaptive != nil && h.adaptive.Enabled {
			status.AdaptiveEnabled = true
			status.CurrentQPS = float64(info.currentQPS)
			status.ConsecutiveSuccesses = info.consecutiveSuccesses
			status.Consecutive429s = info.consecutive429s
			status.InCooldown = time.Now().Before(info.cooldownUntil)
			status.CooldownUntil = info.cooldownUntil
		}

		// Add circuit breaker fields if enabled
		if cbStatus, ok := cbStatuses[host]; ok {
			status.CircuitBreakerState = cbStatus.State
			status.CircuitBreakerFailures = cbStatus.FailureCount
			status.CircuitBreakerLastFail = cbStatus.LastFailureTime
		}

		result = append(result, status)
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

// RecordSuccess reports a successful request (2xx/3xx status) for the given host.
// When adaptive rate limiting is enabled, this may increase the QPS for the host
// after a threshold of consecutive successes is reached.
func (h *HostLimiter) RecordSuccess(host string) {
	if h == nil || h.adaptive == nil || !h.adaptive.Enabled {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	info, ok := h.hostInfo[host]
	if !ok {
		return
	}

	// Check cooldown period
	if time.Since(info.lastRateAdjustment) < h.adaptive.CooldownPeriod {
		return
	}

	info.consecutiveSuccesses++
	info.consecutive429s = 0 // reset 429 counter on success

	// Check if we've reached the threshold for increasing rate
	if info.consecutiveSuccesses >= h.adaptive.SuccessThreshold {
		oldQPS := info.currentQPS
		newQPS := oldQPS + h.adaptive.AdditiveIncrease

		// Cap at max QPS
		if newQPS > h.adaptive.MaxQPS {
			newQPS = h.adaptive.MaxQPS
		}

		if newQPS != oldQPS {
			h.adjustQPSLocked(host, info, newQPS)
			slog.Info("Rate limit increased", "host", host, "oldQPS", oldQPS, "newQPS", newQPS, "consecutiveSuccesses", info.consecutiveSuccesses)
		}
		info.consecutiveSuccesses = 0
	}
}

// RecordRateLimit reports a 429 response for the given host with an optional
// Retry-After duration. When adaptive rate limiting is enabled, this will
// decrease the QPS for the host and optionally set a cooldown period.
func (h *HostLimiter) RecordRateLimit(host string, retryAfter time.Duration) {
	if h == nil || h.adaptive == nil || !h.adaptive.Enabled {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	info, ok := h.hostInfo[host]
	if !ok {
		return
	}

	// ALWAYS update counters on 429, even during cooldown (critical for AIMD correctness)
	info.consecutive429s++
	info.consecutiveSuccesses = 0 // reset success counter on 429

	// Set cooldown from Retry-After header (always respected, even during adjustment cooldown)
	if retryAfter > 0 {
		info.cooldownUntil = time.Now().Add(retryAfter)
		slog.Info("Host entering cooldown", "host", host, "duration", retryAfter, "source", "Retry-After")
	}

	// Check cooldown period for rate adjustments (but counters are already updated above)
	if time.Since(info.lastRateAdjustment) < h.adaptive.CooldownPeriod {
		return
	}

	oldQPS := info.currentQPS
	newQPS := rate.Limit(float64(oldQPS) * h.adaptive.MultiplicativeDecrease)

	// Ensure we don't go below minimum
	if newQPS < h.adaptive.MinQPS {
		newQPS = h.adaptive.MinQPS
	}

	if newQPS != oldQPS {
		h.adjustQPSLocked(host, info, newQPS)
		slog.Info("Rate limit decreased", "host", host, "oldQPS", oldQPS, "newQPS", newQPS, "consecutive429s", info.consecutive429s)
	}
}

// adjustQPSLocked updates the rate limiter with a new QPS.
// Must be called with h.mu held.
func (h *HostLimiter) adjustQPSLocked(_ string, info *hostLimiterInfo, newQPS rate.Limit) {
	info.currentQPS = newQPS
	info.lastRateAdjustment = time.Now()

	// Use SetLimitAt to dynamically adjust the rate without resetting the token bucket.
	// This preserves existing tokens and prevents "free burst" after rate decreases.
	info.limiter.SetLimitAt(time.Now(), newQPS)
}

// UpdateRateLimitInfo updates the limiter based on server-provided rate limit headers.
// This allows the limiter to respect server-provided rate limits instead of
// relying solely on adaptive AIMD behavior.
//
// Behavior:
//   - If info.Limit > 0, adjusts currentQPS to respect the server's limit
//   - If info.Remaining is low (< 10%), enters cooldown until reset time
//   - If info.Reset is in the future and remaining is low, respects that reset time
func (h *HostLimiter) UpdateRateLimitInfo(host string, info RateLimitInfo) {
	if h == nil || h.adaptive == nil || !h.adaptive.Enabled {
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	hostInfo, ok := h.hostInfo[host]
	if !ok {
		return
	}

	now := time.Now()

	// If remaining is very low (< 10%), enter cooldown until reset
	if info.Limit > 0 && info.Remaining >= 0 {
		usagePercent := float64(info.Limit-info.Remaining) / float64(info.Limit) * 100
		if usagePercent >= 90 && !info.Reset.IsZero() && info.Reset.After(now) {
			if info.Reset.After(hostInfo.cooldownUntil) {
				hostInfo.cooldownUntil = info.Reset
				slog.Info("Rate limit nearly exhausted, entering cooldown", "host", host, "remaining", info.Remaining, "limit", info.Limit)
			}
		}
	}

	// Adjust QPS based on server-provided limit (with some buffer)
	// We use 80% of the server's limit to provide a safety margin
	if info.Limit > 0 {
		targetQPS := rate.Limit(float64(info.Limit) * 0.8)

		// Clamp target to adaptive bounds
		if targetQPS < h.adaptive.MinQPS {
			targetQPS = h.adaptive.MinQPS
		}
		if targetQPS > h.adaptive.MaxQPS {
			targetQPS = h.adaptive.MaxQPS
		}

		// Only adjust downward aggressively; adjust upward conservatively
		currentQPS := hostInfo.currentQPS
		if targetQPS < currentQPS {
			// Server limit is lower than our current rate - respect it immediately
			h.adjustQPSLocked(host, hostInfo, targetQPS)
			slog.Info("Rate limit adjusted to server limit", "host", host, "oldQPS", currentQPS, "newQPS", targetQPS, "serverLimit", info.Limit)
		} else if targetQPS > currentQPS*1.5 {
			// Server limit is significantly higher - allow gradual increase
			// but don't jump immediately (let AIMD handle gradual increase)
			slog.Debug("Server rate limit higher than current", "host", host, "currentQPS", currentQPS, "serverLimit", info.Limit)
		}
	}
}

// IsAdaptiveEnabled returns true if adaptive rate limiting is enabled.
func (h *HostLimiter) IsAdaptiveEnabled() bool {
	return h != nil && h.adaptive != nil && h.adaptive.Enabled
}

// GetAdaptiveConfig returns a copy of the adaptive configuration, or nil if not enabled.
func (h *HostLimiter) GetAdaptiveConfig() *AdaptiveConfig {
	if h == nil || h.adaptive == nil {
		return nil
	}
	cfg := *h.adaptive
	return &cfg
}

// IsCircuitBreakerEnabled returns true if circuit breaker is enabled.
func (h *HostLimiter) IsCircuitBreakerEnabled() bool {
	return h != nil && h.circuitBreaker != nil && h.circuitBreaker.IsEnabled()
}

// GetCircuitBreaker returns the circuit breaker instance, or nil if not enabled.
func (h *HostLimiter) GetCircuitBreaker() *CircuitBreaker {
	if h == nil {
		return nil
	}
	return h.circuitBreaker
}

// RecordResult records the result of a request for both adaptive rate limiting
// and circuit breaker tracking.
func (h *HostLimiter) RecordResult(host string, err error, status int) {
	if h == nil {
		return
	}

	// Record for circuit breaker if enabled
	if h.circuitBreaker != nil && h.circuitBreaker.IsEnabled() {
		if err != nil || status >= 500 {
			h.circuitBreaker.RecordFailure(host)
		} else if status >= 200 && status < 400 {
			h.circuitBreaker.RecordSuccess(host)
		}
	}
}
