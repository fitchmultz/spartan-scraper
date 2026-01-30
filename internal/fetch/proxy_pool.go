// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// ProxyEntry represents a single proxy in the pool.
type ProxyEntry struct {
	ID          string   `json:"id"`
	URL         string   `json:"url"`
	Username    string   `json:"username,omitempty"`
	Password    string   `json:"password,omitempty"`
	Region      string   `json:"region,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Weight      int      `json:"weight,omitempty"`
	MaxRequests int      `json:"max_requests,omitempty"`
}

// ToProxyConfig converts ProxyEntry to ProxyConfig for use with existing fetchers.
func (e ProxyEntry) ToProxyConfig() ProxyConfig {
	return ProxyConfig{
		URL:      e.URL,
		Username: e.Username,
		Password: e.Password,
	}
}

// ProxyStats tracks usage and health for a proxy.
type ProxyStats struct {
	RequestCount     uint64    `json:"request_count"`
	SuccessCount     uint64    `json:"success_count"`
	FailureCount     uint64    `json:"failure_count"`
	LastUsed         time.Time `json:"last_used"`
	LastFailed       time.Time `json:"last_failed,omitempty"`
	AvgLatencyMs     int64     `json:"avg_latency_ms"`
	ConsecutiveFails int       `json:"consecutive_fails"`
	IsHealthy        bool      `json:"is_healthy"`
}

// SuccessRate returns the success rate as a percentage (0-100).
func (s ProxyStats) SuccessRate() float64 {
	total := s.SuccessCount + s.FailureCount
	if total == 0 {
		return 100.0
	}
	return float64(s.SuccessCount) / float64(total) * 100
}

// RotationStrategy defines proxy selection algorithm.
type RotationStrategy int

const (
	RotationRoundRobin RotationStrategy = iota
	RotationRandom
	RotationLeastUsed
	RotationWeighted
	RotationLeastLatency
)

// String returns the string representation of the rotation strategy.
func (r RotationStrategy) String() string {
	switch r {
	case RotationRoundRobin:
		return "round_robin"
	case RotationRandom:
		return "random"
	case RotationLeastUsed:
		return "least_used"
	case RotationWeighted:
		return "weighted"
	case RotationLeastLatency:
		return "least_latency"
	default:
		return "unknown"
	}
}

// ParseRotationStrategy parses a rotation strategy from string.
func ParseRotationStrategy(s string) RotationStrategy {
	switch s {
	case "round_robin":
		return RotationRoundRobin
	case "random":
		return RotationRandom
	case "least_used":
		return RotationLeastUsed
	case "weighted":
		return RotationWeighted
	case "least_latency":
		return RotationLeastLatency
	default:
		return RotationRoundRobin
	}
}

// ProxySelectionHints provides hints for proxy selection.
type ProxySelectionHints struct {
	PreferredRegion string   `json:"preferred_region,omitempty"`
	RequiredTags    []string `json:"required_tags,omitempty"`
	ExcludeProxyIDs []string `json:"exclude_proxy_ids,omitempty"`
}

// HealthCheckConfig configures proxy health checking.
type HealthCheckConfig struct {
	Enabled              bool   `json:"enabled"`
	IntervalSeconds      int    `json:"interval_seconds"`
	TimeoutSeconds       int    `json:"timeout_seconds"`
	MaxConsecutiveFails  int    `json:"max_consecutive_fails"`
	RecoveryAfterSeconds int    `json:"recovery_after_seconds"`
	TestURL              string `json:"test_url,omitempty"`
}

// DefaultHealthCheckConfig returns sensible defaults for health checking.
func DefaultHealthCheckConfig() HealthCheckConfig {
	return HealthCheckConfig{
		Enabled:              true,
		IntervalSeconds:      60,
		TimeoutSeconds:       10,
		MaxConsecutiveFails:  3,
		RecoveryAfterSeconds: 300,
		TestURL:              "http://httpbin.org/ip",
	}
}

// ProxyPoolConfig is the configuration file format for proxy pools.
type ProxyPoolConfig struct {
	DefaultStrategy string            `json:"default_strategy"`
	HealthCheck     HealthCheckConfig `json:"health_check,omitempty"`
	Proxies         []ProxyEntry      `json:"proxies"`
}

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

// LoadProxyPoolFromFile loads a proxy pool from a JSON configuration file.
func LoadProxyPoolFromFile(path string) (*ProxyPool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, apperrors.NotFound(fmt.Sprintf("proxy pool config file not found: %s", path))
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to read proxy pool config", err)
	}

	var config ProxyPoolConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, apperrors.Wrap(apperrors.KindValidation, "invalid proxy pool config JSON", err)
	}

	return NewProxyPool(config)
}

// Select returns a proxy based on the configured rotation strategy.
// Returns an error if no healthy proxies are available.
func (p *ProxyPool) Select(hints ProxySelectionHints) (ProxyEntry, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Filter proxies based on hints
	candidates := p.filterProxies(hints)
	if len(candidates) == 0 {
		return ProxyEntry{}, apperrors.NotFound("no healthy proxies available matching selection criteria")
	}

	// Select based on strategy
	var selected ProxyEntry
	switch p.strategy {
	case RotationRoundRobin:
		selected = p.selectRoundRobin(candidates)
	case RotationRandom:
		selected = p.selectRandom(candidates)
	case RotationLeastUsed:
		selected = p.selectLeastUsed(candidates)
	case RotationWeighted:
		selected = p.selectWeighted(candidates)
	case RotationLeastLatency:
		selected = p.selectLeastLatency(candidates)
	default:
		selected = p.selectRoundRobin(candidates)
	}

	return selected, nil
}

// filterProxies returns proxies matching the selection hints.
func (p *ProxyPool) filterProxies(hints ProxySelectionHints) []ProxyEntry {
	var candidates []ProxyEntry

	for _, proxy := range p.entries {
		stats := p.stats[proxy.ID]

		// Skip unhealthy proxies
		if stats != nil && !stats.IsHealthy {
			continue
		}

		// Skip excluded proxies
		if containsString(hints.ExcludeProxyIDs, proxy.ID) {
			continue
		}

		// Check region preference
		if hints.PreferredRegion != "" && proxy.Region != hints.PreferredRegion {
			continue
		}

		// Check required tags
		if len(hints.RequiredTags) > 0 {
			if !hasAllTags(proxy.Tags, hints.RequiredTags) {
				continue
			}
		}

		candidates = append(candidates, proxy)
	}

	return candidates
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

// selectRoundRobin selects the next proxy in round-robin order.
func (p *ProxyPool) selectRoundRobin(candidates []ProxyEntry) ProxyEntry {
	if len(candidates) == 1 {
		return candidates[0]
	}

	idx := atomic.AddUint64(&p.rrIndex, 1) % uint64(len(candidates))
	return candidates[idx]
}

// selectRandom selects a random proxy from candidates.
func (p *ProxyPool) selectRandom(candidates []ProxyEntry) ProxyEntry {
	if len(candidates) == 1 {
		return candidates[0]
	}

	// Use a simple pseudo-random selection based on time
	// In production, consider using crypto/rand for better randomness
	idx := time.Now().UnixNano() % int64(len(candidates))
	return candidates[idx]
}

// selectLeastUsed selects the proxy with the lowest request count.
func (p *ProxyPool) selectLeastUsed(candidates []ProxyEntry) ProxyEntry {
	if len(candidates) == 1 {
		return candidates[0]
	}

	var selected ProxyEntry
	minRequests := ^uint64(0) // Max uint64

	for _, proxy := range candidates {
		stats := p.stats[proxy.ID]
		if stats.RequestCount < minRequests {
			minRequests = stats.RequestCount
			selected = proxy
		}
	}

	return selected
}

// selectWeighted performs weighted random selection.
func (p *ProxyPool) selectWeighted(candidates []ProxyEntry) ProxyEntry {
	if len(candidates) == 1 {
		return candidates[0]
	}

	// Calculate total weight
	totalWeight := 0
	for _, proxy := range candidates {
		weight := proxy.Weight
		if weight <= 0 {
			weight = 1 // Default weight
		}
		totalWeight += weight
	}

	if totalWeight <= 0 {
		// Fallback to random if no weights set
		return p.selectRandom(candidates)
	}

	// Select based on weight
	// Use time-based pseudo-random for simplicity
	r := int(time.Now().UnixNano() % int64(totalWeight))
	cumulativeWeight := 0

	for _, proxy := range candidates {
		weight := proxy.Weight
		if weight <= 0 {
			weight = 1
		}
		cumulativeWeight += weight
		if r < cumulativeWeight {
			return proxy
		}
	}

	// Fallback to last candidate
	return candidates[len(candidates)-1]
}

// selectLeastLatency selects the proxy with the lowest average latency.
func (p *ProxyPool) selectLeastLatency(candidates []ProxyEntry) ProxyEntry {
	if len(candidates) == 1 {
		return candidates[0]
	}

	if len(candidates) == 0 {
		return ProxyEntry{}
	}

	var selected ProxyEntry
	minLatency := int64(math.MaxInt64) // Max int64

	for _, proxy := range candidates {
		stats, ok := p.stats[proxy.ID]
		if !ok {
			continue
		}
		// If no latency data yet, treat as high latency to prefer measured proxies
		latency := stats.AvgLatencyMs
		if latency == 0 {
			latency = 10000 // 10 seconds default for unmeasured
		}
		if latency < minLatency {
			minLatency = latency
			selected = proxy
		}
	}

	return selected
}

// Helper functions

func containsString(slice []string, s string) bool {
	return slices.Contains(slice, s)
}

func hasAllTags(proxyTags, requiredTags []string) bool {
	for _, required := range requiredTags {
		if !slices.Contains(proxyTags, required) {
			return false
		}
	}
	return true
}

// ProxyPoolFromConfig creates a proxy pool from the global config.
// Returns nil if no proxy pool is configured.
func ProxyPoolFromConfig(dataDir string) (*ProxyPool, error) {
	if dataDir == "" {
		dataDir = ".data"
	}

	path := filepath.Join(dataDir, "proxy_pool.json")

	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	return LoadProxyPoolFromFile(path)
}
