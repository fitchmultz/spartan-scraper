// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// ProxyPool manages a collection of proxies with rotation.
type ProxyPool struct {
	entries  []ProxyEntry
	stats    map[string]*ProxyStats
	strategy RotationStrategy
	mu       sync.RWMutex
	rrIndex  uint64

	healthChecker HealthChecker
	stopHealthCh  chan struct{}
}

// NewProxyPool creates a new proxy pool from configuration.
func NewProxyPool(config ProxyPoolConfig) (*ProxyPool, error) {
	if len(config.Proxies) == 0 {
		return nil, apperrors.Validation("proxy pool must contain at least one proxy")
	}

	// Validate all proxies have IDs
	seenIDs := make(map[string]bool)
	for i, proxy := range config.Proxies {
		if proxy.ID == "" {
			return nil, apperrors.Validation(fmt.Sprintf("proxy at index %d is missing required field: id", i))
		}
		if proxy.URL == "" {
			return nil, apperrors.Validation(fmt.Sprintf("proxy %s is missing required field: url", proxy.ID))
		}
		if seenIDs[proxy.ID] {
			return nil, apperrors.Validation(fmt.Sprintf("duplicate proxy ID: %s", proxy.ID))
		}
		seenIDs[proxy.ID] = true
	}

	// Initialize stats for each proxy
	stats := make(map[string]*ProxyStats)
	for _, proxy := range config.Proxies {
		stats[proxy.ID] = &ProxyStats{
			IsHealthy: true,
		}
	}

	// Set default health check config
	healthCheck := config.HealthCheck
	if healthCheck.TestURL == "" {
		healthCheck.TestURL = "http://httpbin.org/ip"
	}
	if healthCheck.IntervalSeconds <= 0 {
		healthCheck.IntervalSeconds = 60
	}
	if healthCheck.TimeoutSeconds <= 0 {
		healthCheck.TimeoutSeconds = 10
	}
	if healthCheck.MaxConsecutiveFails <= 0 {
		healthCheck.MaxConsecutiveFails = 3
	}
	if healthCheck.RecoveryAfterSeconds <= 0 {
		healthCheck.RecoveryAfterSeconds = 300
	}

	pool := &ProxyPool{
		entries:  config.Proxies,
		stats:    stats,
		strategy: ParseRotationStrategy(config.DefaultStrategy),
		healthChecker: &DefaultHealthChecker{
			TestURL: healthCheck.TestURL,
			Timeout: time.Duration(healthCheck.TimeoutSeconds) * time.Second,
		},
		stopHealthCh: make(chan struct{}),
	}

	// Start health checking if enabled
	if healthCheck.Enabled {
		go pool.runHealthChecks(healthCheck)
	}

	slog.Info("Proxy pool initialized",
		"proxies", len(config.Proxies),
		"strategy", pool.strategy.String(),
		"health_check", healthCheck.Enabled,
	)

	return pool, nil
}

// RecordSuccess updates stats for a successful proxy request.
func (p *ProxyPool) RecordSuccess(proxyID string, latencyMs int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	stats, ok := p.stats[proxyID]
	if !ok {
		return
	}

	stats.RequestCount++
	stats.SuccessCount++
	stats.LastUsed = time.Now()
	stats.ConsecutiveFails = 0

	// Update average latency using exponential moving average
	if stats.AvgLatencyMs == 0 {
		stats.AvgLatencyMs = latencyMs
	} else {
		stats.AvgLatencyMs = (stats.AvgLatencyMs*9 + latencyMs) / 10
	}
}

// RecordFailure updates stats for a failed proxy request.
func (p *ProxyPool) RecordFailure(proxyID string, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	stats, ok := p.stats[proxyID]
	if !ok {
		return
	}

	stats.RequestCount++
	stats.FailureCount++
	stats.LastUsed = time.Now()
	stats.LastFailed = time.Now()
	stats.ConsecutiveFails++
}

// GetStats returns current stats for all proxies.
func (p *ProxyPool) GetStats() map[string]ProxyStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]ProxyStats, len(p.stats))
	for id, stats := range p.stats {
		result[id] = *stats
	}
	return result
}

// GetProxyStats returns stats for a specific proxy.
func (p *ProxyPool) GetProxyStats(proxyID string) (ProxyStats, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats, ok := p.stats[proxyID]
	if !ok {
		return ProxyStats{}, false
	}
	return *stats, true
}

// GetHealthyProxyCount returns the number of healthy proxies.
func (p *ProxyPool) GetHealthyProxyCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, stats := range p.stats {
		if stats.IsHealthy {
			count++
		}
	}
	return count
}

// GetTotalProxyCount returns the total number of proxies.
func (p *ProxyPool) GetTotalProxyCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.entries)
}

// GetEntries returns a copy of all proxy entries.
func (p *ProxyPool) GetEntries() []ProxyEntry {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]ProxyEntry, len(p.entries))
	copy(result, p.entries)
	return result
}

// GetStrategy returns the current rotation strategy.
func (p *ProxyPool) GetStrategy() RotationStrategy {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.strategy
}

// SetStrategy changes the rotation strategy.
func (p *ProxyPool) SetStrategy(strategy RotationStrategy) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.strategy = strategy
}

// Stop stops the proxy pool and its background health checks.
func (p *ProxyPool) Stop() {
	close(p.stopHealthCh)
}

// runHealthChecks runs periodic health checks on all proxies.
func (p *ProxyPool) runHealthChecks(cfg HealthCheckConfig) {
	ticker := time.NewTicker(time.Duration(cfg.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	// Run initial health check
	p.checkAllProxies(cfg)

	for {
		select {
		case <-ticker.C:
			p.checkAllProxies(cfg)
		case <-p.stopHealthCh:
			return
		}
	}
}

// checkAllProxies performs health checks on all proxies.
func (p *ProxyPool) checkAllProxies(cfg HealthCheckConfig) {
	ctx := context.Background()

	for _, proxy := range p.entries {
		go p.checkProxy(ctx, proxy, cfg)
	}
}

// checkProxy performs a health check on a single proxy.
func (p *ProxyPool) checkProxy(ctx context.Context, proxy ProxyEntry, cfg HealthCheckConfig) {
	latencyMs, err := p.healthChecker.Check(ctx, proxy)

	p.mu.Lock()
	defer p.mu.Unlock()

	stats, ok := p.stats[proxy.ID]
	if !ok {
		return
	}

	if err != nil {
		stats.ConsecutiveFails++
		stats.LastFailed = time.Now()

		// Mark unhealthy after max consecutive failures
		if stats.ConsecutiveFails >= cfg.MaxConsecutiveFails {
			if stats.IsHealthy {
				stats.IsHealthy = false
				slog.Warn("Proxy marked unhealthy",
					"proxy_id", proxy.ID,
					"consecutive_fails", stats.ConsecutiveFails,
				)
			}
		}
	} else {
		// Update latency on success
		if stats.AvgLatencyMs == 0 {
			stats.AvgLatencyMs = latencyMs
		} else {
			stats.AvgLatencyMs = (stats.AvgLatencyMs*9 + latencyMs) / 10
		}

		// Check if we should recover
		if !stats.IsHealthy {
			recoveryThreshold := time.Duration(cfg.RecoveryAfterSeconds) * time.Second
			if time.Since(stats.LastFailed) >= recoveryThreshold {
				stats.IsHealthy = true
				stats.ConsecutiveFails = 0
				slog.Info("Proxy recovered",
					"proxy_id", proxy.ID,
					"latency_ms", latencyMs,
				)
			}
		} else {
			stats.ConsecutiveFails = 0
		}
	}
}
