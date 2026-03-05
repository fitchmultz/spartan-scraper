// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"time"
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
