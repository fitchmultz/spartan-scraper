// Package api provides the REST API server for Spartan Scraper.
// This file implements the metrics collection layer for performance monitoring
// and rate limiting visibility.
//
// Responsibilities:
// - Collecting request metrics (duration, success/failure, fetcher type, host)
// - Tracking fetcher usage breakdown (HTTP, Chromedp, Playwright)
// - Recording job duration histograms
// - Exposing rate limiter state for all known hosts
// - Providing thread-safe access to metrics snapshots
//
// This file does NOT:
// - Persist metrics to disk (all in-memory with circular buffers)
// - Expose raw URLs (only host-level aggregation)
// - Provide historical data beyond the configured retention window
//
// Invariants:
// - All metrics operations are thread-safe via RWMutex
// - Circular buffers have fixed capacity to cap memory usage
// - Host names are sanitized before storage (no query params, paths)
package api

import (
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

const (
	// DefaultMetricsBufferSize is the default size for circular buffers
	DefaultMetricsBufferSize = 1000
	// DefaultMetricsRetention is the time window for metrics calculations
	DefaultMetricsRetention = 5 * time.Minute
)

// RequestMetric represents a single request measurement
type RequestMetric struct {
	Timestamp   time.Time
	Duration    time.Duration
	Success     bool
	FetcherType string // "http", "chromedp", "playwright"
	Host        string // sanitized host only
}

// FetcherUsageBreakdown tracks usage counts per fetcher type
type FetcherUsageBreakdown struct {
	HTTP       uint64 `json:"http"`
	Chromedp   uint64 `json:"chromedp"`
	Playwright uint64 `json:"playwright"`
}

// RateLimitStatus represents the current state for a single host
type RateLimitStatus struct {
	Host        string  `json:"host"`
	QPS         float64 `json:"qps"`
	Burst       int     `json:"burst"`
	Tokens      float64 `json:"tokens"`      // Estimated current tokens
	LastRequest int64   `json:"lastRequest"` // Unix timestamp
}

// MetricsSnapshot is a point-in-time view of all metrics
type MetricsSnapshot struct {
	// Request metrics
	RequestsPerSec  float64 `json:"requestsPerSec"`
	SuccessRate     float64 `json:"successRate"`       // Percentage 0-100
	AvgResponseTime float64 `json:"avgResponseTimeMs"` // Milliseconds
	ActiveRequests  int     `json:"activeRequests"`
	TotalRequests   uint64  `json:"totalRequests"`

	// Fetcher breakdown
	FetcherUsage FetcherUsageBreakdown `json:"fetcherUsage"`

	// Rate limit status
	RateLimitStatus []RateLimitStatus `json:"rateLimitStatus"`

	// Job metrics
	JobThroughput  float64 `json:"jobThroughputPerMin"` // Jobs per minute
	AvgJobDuration float64 `json:"avgJobDurationMs"`    // Milliseconds

	// Timestamp
	Timestamp int64 `json:"timestamp"`
}

// RingBuffer is a thread-safe circular buffer for fixed-size data
type RingBuffer[T any] struct {
	data     []T
	capacity int
	head     int // Write position
	size     int // Current size
	mu       sync.RWMutex
}

// NewRingBuffer creates a new ring buffer with the specified capacity
func NewRingBuffer[T any](capacity int) *RingBuffer[T] {
	return &RingBuffer[T]{
		data:     make([]T, capacity),
		capacity: capacity,
	}
}

// Push adds an item to the buffer, overwriting old data if full
func (r *RingBuffer[T]) Push(item T) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[r.head] = item
	r.head = (r.head + 1) % r.capacity
	if r.size < r.capacity {
		r.size++
	}
}

// GetAll returns all items in the buffer (oldest first)
func (r *RingBuffer[T]) GetAll() []T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.size == 0 {
		return nil
	}

	result := make([]T, r.size)
	if r.size < r.capacity {
		// Buffer not full yet, copy from start
		copy(result, r.data[:r.size])
	} else {
		// Buffer full, need to rotate
		// head points to oldest item
		for i := 0; i < r.size; i++ {
			idx := (r.head + i) % r.capacity
			result[i] = r.data[idx]
		}
	}
	return result
}

// Size returns the current number of items in the buffer
func (r *RingBuffer[T]) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.size
}

// hostLimiterInfo tracks extended information about a host's rate limiter
type hostLimiterInfo struct {
	limiter     *rate.Limiter
	qps         float64
	burst       int
	lastRequest time.Time
}

// MetricsCollector holds all runtime metrics
type MetricsCollector struct {
	// Rate limiter tracking
	hostLimiters   map[string]*hostLimiterInfo
	hostLimitersMu sync.RWMutex
	defaultQPS     float64
	defaultBurst   int

	// Request metrics
	requestMetrics *RingBuffer[RequestMetric]

	// Job duration metrics
	jobDurations *RingBuffer[time.Duration]

	// Fetcher usage counters
	fetcherUsage FetcherUsageBreakdown

	// Current state
	activeRequests int64
	totalRequests  uint64
	successCount   uint64
	failureCount   uint64

	// Configuration
	retention time.Duration
}

// Callback adapts the collector to the fetch-layer metrics callback contract.
// A zero duration is treated as a request start marker; non-zero durations
// record completed request outcomes.
func (m *MetricsCollector) Callback() func(duration time.Duration, success bool, fetcherType, rawURL string) {
	return func(duration time.Duration, success bool, fetcherType, rawURL string) {
		if duration <= 0 {
			m.StartRequest()
			return
		}
		m.RecordRequest(duration, success, fetcherType, rawURL)
	}
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		hostLimiters:   make(map[string]*hostLimiterInfo),
		requestMetrics: NewRingBuffer[RequestMetric](DefaultMetricsBufferSize),
		jobDurations:   NewRingBuffer[time.Duration](DefaultMetricsBufferSize),
		retention:      DefaultMetricsRetention,
		defaultQPS:     10, // Default from config
		defaultBurst:   5,
	}
}

// SetDefaultRateLimit sets the default rate limit parameters for new hosts
func (m *MetricsCollector) SetDefaultRateLimit(qps float64, burst int) {
	m.defaultQPS = qps
	m.defaultBurst = burst
}

// RegisterHostLimiter registers a host with its rate limiter for tracking
func (m *MetricsCollector) RegisterHostLimiter(host string, limiter *rate.Limiter, qps float64, burst int) {
	m.hostLimitersMu.Lock()
	defer m.hostLimitersMu.Unlock()

	if _, exists := m.hostLimiters[host]; !exists {
		m.hostLimiters[host] = &hostLimiterInfo{
			limiter:     limiter,
			qps:         qps,
			burst:       burst,
			lastRequest: time.Time{},
		}
	}
}

// RecordRequest records a request metric
func (m *MetricsCollector) RecordRequest(duration time.Duration, success bool, fetcherType, rawURL string) {
	// Sanitize URL to get just the host
	host := ""
	if u, err := url.Parse(rawURL); err == nil && u.Host != "" {
		host = u.Host
	}

	// Update counters
	m.finishRequest()
	atomic.AddUint64(&m.totalRequests, 1)
	if success {
		atomic.AddUint64(&m.successCount, 1)
	} else {
		atomic.AddUint64(&m.failureCount, 1)
	}

	// Update fetcher usage
	switch fetcherType {
	case "http":
		atomic.AddUint64(&m.fetcherUsage.HTTP, 1)
	case "chromedp":
		atomic.AddUint64(&m.fetcherUsage.Chromedp, 1)
	case "playwright":
		atomic.AddUint64(&m.fetcherUsage.Playwright, 1)
	}

	// Record to ring buffer
	m.requestMetrics.Push(RequestMetric{
		Timestamp:   time.Now(),
		Duration:    duration,
		Success:     success,
		FetcherType: fetcherType,
		Host:        host,
	})

	// Update last request time for host
	if host != "" {
		m.hostLimitersMu.Lock()
		if info, exists := m.hostLimiters[host]; exists {
			info.lastRequest = time.Now()
		}
		m.hostLimitersMu.Unlock()
	}
}

// StartRequest marks the beginning of a request (for active request counting)
func (m *MetricsCollector) StartRequest() {
	atomic.AddInt64(&m.activeRequests, 1)
}

func (m *MetricsCollector) finishRequest() {
	for {
		current := atomic.LoadInt64(&m.activeRequests)
		if current <= 0 {
			return
		}
		if atomic.CompareAndSwapInt64(&m.activeRequests, current, current-1) {
			return
		}
	}
}

// RecordJobDuration records a job duration measurement
func (m *MetricsCollector) RecordJobDuration(duration time.Duration) {
	m.jobDurations.Push(duration)
}

// GetSnapshot returns a point-in-time snapshot of all metrics
func (m *MetricsCollector) GetSnapshot() MetricsSnapshot {
	snapshot := MetricsSnapshot{
		Timestamp:       time.Now().Unix(),
		ActiveRequests:  int(atomic.LoadInt64(&m.activeRequests)),
		TotalRequests:   atomic.LoadUint64(&m.totalRequests),
		FetcherUsage:    m.getFetcherUsage(),
		RateLimitStatus: m.GetRateLimitStatus(),
	}

	// Calculate request metrics from ring buffer
	cutoff := time.Now().Add(-m.retention)
	requests := m.requestMetrics.GetAll()

	var recentRequests []RequestMetric
	for _, req := range requests {
		if req.Timestamp.After(cutoff) {
			recentRequests = append(recentRequests, req)
		}
	}

	// Calculate requests per second
	if len(recentRequests) > 1 {
		duration := recentRequests[len(recentRequests)-1].Timestamp.Sub(recentRequests[0].Timestamp)
		if duration > 0 {
			snapshot.RequestsPerSec = float64(len(recentRequests)) / duration.Seconds()
		}
	}

	// Calculate success rate
	total := len(recentRequests)
	if total > 0 {
		var successes int
		for _, req := range recentRequests {
			if req.Success {
				successes++
			}
		}
		snapshot.SuccessRate = float64(successes) * 100.0 / float64(total)
	} else {
		snapshot.SuccessRate = 100.0 // Default to 100% if no data
	}

	// Calculate average response time
	if total > 0 {
		var totalDuration time.Duration
		for _, req := range recentRequests {
			totalDuration += req.Duration
		}
		snapshot.AvgResponseTime = float64(totalDuration.Milliseconds()) / float64(total)
	}

	// Calculate job metrics
	durations := m.jobDurations.GetAll()
	if len(durations) > 0 {
		var totalDuration time.Duration
		for _, d := range durations {
			totalDuration += d
		}
		snapshot.AvgJobDuration = float64(totalDuration.Milliseconds()) / float64(len(durations))

		// Calculate throughput (jobs per minute)
		if len(durations) > 1 {
			// Estimate based on average duration and active jobs
			avgDuration := totalDuration / time.Duration(len(durations))
			if avgDuration > 0 {
				snapshot.JobThroughput = float64(time.Minute) / float64(avgDuration)
			}
		}
	}

	return snapshot
}

// getFetcherUsage returns a copy of the fetcher usage counters
func (m *MetricsCollector) getFetcherUsage() FetcherUsageBreakdown {
	return FetcherUsageBreakdown{
		HTTP:       atomic.LoadUint64(&m.fetcherUsage.HTTP),
		Chromedp:   atomic.LoadUint64(&m.fetcherUsage.Chromedp),
		Playwright: atomic.LoadUint64(&m.fetcherUsage.Playwright),
	}
}

// GetRateLimitStatus returns the current rate limit status for all known hosts
func (m *MetricsCollector) GetRateLimitStatus() []RateLimitStatus {
	m.hostLimitersMu.RLock()
	defer m.hostLimitersMu.RUnlock()

	result := make([]RateLimitStatus, 0, len(m.hostLimiters))
	now := time.Now()

	for host, info := range m.hostLimiters {
		status := RateLimitStatus{
			Host:  host,
			QPS:   info.qps,
			Burst: info.burst,
		}

		// Estimate tokens based on time elapsed since last request
		// rate.Limiter doesn't expose tokens directly, so we estimate
		if !info.lastRequest.IsZero() {
			status.LastRequest = info.lastRequest.Unix()
			elapsed := now.Sub(info.lastRequest).Seconds()
			// Tokens regenerate at QPS rate, capped at burst
			estimatedTokens := info.qps * elapsed
			if estimatedTokens > float64(info.burst) {
				estimatedTokens = float64(info.burst)
			}
			status.Tokens = estimatedTokens
		} else {
			// No requests yet, assume full bucket
			status.Tokens = float64(info.burst)
		}

		result = append(result, status)
	}

	return result
}

// SyncHostLimiters synchronizes host limiters from the HostLimiter to the metrics collector.
// This should be called periodically to ensure new hosts are registered for metrics tracking.
func (m *MetricsCollector) SyncHostLimiters(hostLimiter interface {
	GetHostStatus() []struct {
		Host        string
		QPS         float64
		Burst       int
		LastRequest time.Time
	}
	GetLimiter(host string) *rate.Limiter
}) {
	statuses := hostLimiter.GetHostStatus()
	for _, status := range statuses {
		limiter := hostLimiter.GetLimiter(status.Host)
		if limiter != nil {
			m.RegisterHostLimiter(status.Host, limiter, status.QPS, status.Burst)
		}
	}
}

// Reset clears all metrics (useful for testing)
func (m *MetricsCollector) Reset() {
	m.hostLimitersMu.Lock()
	m.hostLimiters = make(map[string]*hostLimiterInfo)
	m.hostLimitersMu.Unlock()

	m.requestMetrics = NewRingBuffer[RequestMetric](DefaultMetricsBufferSize)
	m.jobDurations = NewRingBuffer[time.Duration](DefaultMetricsBufferSize)

	atomic.StoreInt64(&m.activeRequests, 0)
	atomic.StoreUint64(&m.totalRequests, 0)
	atomic.StoreUint64(&m.successCount, 0)
	atomic.StoreUint64(&m.failureCount, 0)
	atomic.StoreUint64(&m.fetcherUsage.HTTP, 0)
	atomic.StoreUint64(&m.fetcherUsage.Chromedp, 0)
	atomic.StoreUint64(&m.fetcherUsage.Playwright, 0)
}
