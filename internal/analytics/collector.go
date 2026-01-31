// Package analytics provides historical metrics collection and aggregation.
//
// This file handles:
// - Periodic aggregation of real-time metrics into hourly buckets
// - Background rollup of hourly data to daily summaries
// - Integration with the metrics collector for data capture
//
// This file does NOT handle:
// - Real-time metrics (api/metrics.go handles that)
// - Querying analytics data (service.go handles that)
// - Report generation (reports.go handles that)
//
// Invariants:
// - Aggregations run every 5 minutes to capture metrics
// - Hourly buckets are closed and immutable after the hour passes
// - Uses background goroutine with graceful shutdown
package analytics

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

// MetricsSnapshot represents a point-in-time view of metrics.
// This is a copy of api.MetricsSnapshot to avoid import cycles.
type MetricsSnapshot struct {
	RequestsPerSec  float64
	SuccessRate     float64
	AvgResponseTime float64
	ActiveRequests  int
	TotalRequests   uint64
	FetcherUsage    struct {
		HTTP       uint64
		Chromedp   uint64
		Playwright uint64
	}
	RateLimitStatus []struct {
		Host        string
		QPS         float64
		Burst       int
		Tokens      float64
		LastRequest int64
	}
	JobThroughput  float64
	AvgJobDuration float64
	Timestamp      int64
}

// MetricsCollector defines the interface for collecting metrics.
// This interface is implemented by api.MetricsCollector.
type MetricsCollector interface {
	GetSnapshot() MetricsSnapshot
}

// Collector periodically aggregates real-time metrics into historical data.
type Collector struct {
	store            *store.Store
	metricsCollector MetricsCollector
	ticker           *time.Ticker
	stopCh           chan struct{}
	wg               sync.WaitGroup
	lastAggregation  time.Time
	mu               sync.Mutex

	// Current hour buffer (accumulating data)
	currentHour   time.Time
	hourlyMetrics *store.AnalyticsHourlyMetrics
	hostMetrics   map[string]*store.AnalyticsHostMetrics
}

// NewCollector creates a new analytics collector.
func NewCollector(s *store.Store, metricsCollector MetricsCollector) *Collector {
	return &Collector{
		store:            s,
		metricsCollector: metricsCollector,
		stopCh:           make(chan struct{}),
		hostMetrics:      make(map[string]*store.AnalyticsHostMetrics),
	}
}

// Start begins the periodic aggregation process.
func (c *Collector) Start() {
	c.ticker = time.NewTicker(5 * time.Minute)
	c.wg.Add(1)
	go c.run()
	slog.Info("analytics collector started")
}

// Stop gracefully shuts down the collector.
func (c *Collector) Stop() {
	close(c.stopCh)
	c.ticker.Stop()
	c.wg.Wait()

	// Flush any remaining data
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c.flushCurrentHour(ctx)

	slog.Info("analytics collector stopped")
}

// run is the main event loop for the collector.
func (c *Collector) run() {
	defer c.wg.Done()

	// Do an initial aggregation immediately
	c.aggregate()

	for {
		select {
		case <-c.ticker.C:
			c.aggregate()
		case <-c.stopCh:
			return
		}
	}
}

// aggregate captures current metrics and stores them.
func (c *Collector) aggregate() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	now := time.Now().UTC()
	currentHour := truncateToHour(now)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we've moved to a new hour
	if !c.currentHour.Equal(currentHour) {
		// Flush the previous hour's data
		if !c.currentHour.IsZero() && c.hourlyMetrics != nil {
			if err := c.store.RecordHourlyMetrics(ctx, c.hourlyMetrics); err != nil {
				slog.Error("failed to record hourly metrics", "error", err, "hour", c.currentHour)
			}

			// Flush host metrics for the previous hour
			for _, hm := range c.hostMetrics {
				if err := c.store.RecordHostMetrics(ctx, hm); err != nil {
					slog.Error("failed to record host metrics", "error", err, "host", hm.Host)
				}
			}

			// Clear host metrics for new hour
			c.hostMetrics = make(map[string]*store.AnalyticsHostMetrics)

			// Check if we should do a daily rollup (previous hour was the last of the day)
			if c.currentHour.Hour() == 23 {
				if _, err := c.store.RollupDaily(ctx, c.currentHour); err != nil {
					slog.Error("failed to rollup daily metrics", "error", err, "date", c.currentHour)
				}
			}
		}

		// Start a new hour
		c.currentHour = currentHour
		c.hourlyMetrics = &store.AnalyticsHourlyMetrics{
			Hour:      currentHour,
			CreatedAt: now,
		}
	}

	// Get current snapshot from metrics collector
	snapshot := c.metricsCollector.GetSnapshot()

	// Update hourly metrics with current snapshot data
	c.updateHourlyMetrics(snapshot, now)

	c.lastAggregation = now
	slog.Debug("analytics aggregation completed", "hour", currentHour)
}

// truncateToHour returns the time truncated to the hour.
func truncateToHour(t time.Time) time.Time {
	return t.UTC().Truncate(time.Hour)
}

// updateHourlyMetrics updates the current hour's metrics from a snapshot.
func (c *Collector) updateHourlyMetrics(snapshot MetricsSnapshot, now time.Time) {
	// Note: We accumulate metrics over the hour
	// The snapshot contains cumulative totals, so we need to track deltas
	// For simplicity, we'll store the current values and let the query layer handle deltas

	// Update request counts (these are cumulative, so we store the current total)
	// The hourly record will contain the max values seen during the hour
	currentTotal := int64(snapshot.TotalRequests)
	if currentTotal > c.hourlyMetrics.RequestsTotal {
		delta := currentTotal - c.hourlyMetrics.RequestsTotal
		c.hourlyMetrics.RequestsTotal = currentTotal

		// Estimate success/failure based on current success rate
		successRate := snapshot.SuccessRate / 100.0
		c.hourlyMetrics.RequestsSuccess += int64(float64(delta) * successRate)
		c.hourlyMetrics.RequestsFailed += int64(float64(delta) * (1.0 - successRate))
	}

	// Update response time (weighted average)
	if snapshot.AvgResponseTime > 0 {
		// Simple moving average
		if c.hourlyMetrics.RequestsTotal > 0 {
			c.hourlyMetrics.AvgResponseTimeMs = (c.hourlyMetrics.AvgResponseTimeMs + snapshot.AvgResponseTime) / 2
		} else {
			c.hourlyMetrics.AvgResponseTimeMs = snapshot.AvgResponseTime
		}
	}

	// Update fetcher usage
	c.hourlyMetrics.FetcherHTTP = int64(snapshot.FetcherUsage.HTTP)
	c.hourlyMetrics.FetcherChromedp = int64(snapshot.FetcherUsage.Chromedp)
	c.hourlyMetrics.FetcherPlaywright = int64(snapshot.FetcherUsage.Playwright)

	// Update rate limit status for host metrics
	for _, status := range snapshot.RateLimitStatus {
		hm, exists := c.hostMetrics[status.Host]
		if !exists {
			hm = &store.AnalyticsHostMetrics{
				Hour: now.UTC().Truncate(time.Hour),
				Host: status.Host,
			}
			c.hostMetrics[status.Host] = hm
		}

		// Update host metrics from rate limit status
		// Note: We don't have per-host request counts from the snapshot,
		// but we track that the host was active
		hm.LastRequest = now.Unix()
	}
}

// flushCurrentHour persists the current hour's data.
func (c *Collector) flushCurrentHour(ctx context.Context) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.hourlyMetrics != nil && !c.currentHour.IsZero() {
		if err := c.store.RecordHourlyMetrics(ctx, c.hourlyMetrics); err != nil {
			slog.Error("failed to flush hourly metrics", "error", err)
		}

		for _, hm := range c.hostMetrics {
			if err := c.store.RecordHostMetrics(ctx, hm); err != nil {
				slog.Error("failed to flush host metrics", "error", err, "host", hm.Host)
			}
		}
	}
}

// RecordJobCompletion records a completed job for trend analysis.
func (c *Collector) RecordJobCompletion(ctx context.Context, kind string, status string, duration time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().UTC()
	date := truncateToDay(now)

	// Convert kind and status to model types
	jobKind := model.Kind(kind)
	jobStatus := model.Status(status)

	trend := &store.AnalyticsJobTrend{
		Date:          date,
		JobKind:       jobKind,
		Status:        jobStatus,
		Count:         1,
		AvgDurationMs: float64(duration.Milliseconds()),
		TotalDuration: duration,
	}

	if err := c.store.RecordJobTrend(ctx, trend); err != nil {
		slog.Error("failed to record job trend", "error", err)
	}
}

// truncateToDay returns the time truncated to the day.
func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// GetLastAggregation returns the time of the last successful aggregation.
func (c *Collector) GetLastAggregation() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastAggregation
}
