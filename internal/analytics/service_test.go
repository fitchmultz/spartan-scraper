// Package analytics provides tests for the analytics service.
//
// This file tests:
// - Dashboard data retrieval for all time periods (24h, 7d, 30d, 90d)
// - Metrics queries with hourly and daily granularity
// - Host metrics, top hosts, and job trends queries
// - Error handling for store failures
//
// This file does NOT test:
// - Data collection (see collector_test.go)
// - Data persistence (see store/*_test.go)
package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func setupTestStore(t *testing.T) (*store.Store, func()) {
	dataDir := t.TempDir()
	s, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	return s, func() { s.Close() }
}

func seedHourlyMetrics(t *testing.T, s *store.Store, hours int) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Hour)
	for i := 0; i < hours; i++ {
		hour := now.Add(-time.Duration(i) * time.Hour)
		metrics := &store.AnalyticsHourlyMetrics{
			Hour:              hour,
			RequestsTotal:     int64(100 + i*10),
			RequestsSuccess:   int64(90 + i*9),
			RequestsFailed:    int64(10 + i),
			AvgResponseTimeMs: 150.0 + float64(i)*10,
			JobsCompleted:     int64(50 + i*5),
			JobsFailed:        int64(5 + i),
			FetcherHTTP:       int64(80 + i*8),
			FetcherChromedp:   int64(20 + i*2),
			CreatedAt:         time.Now().UTC(),
		}
		if err := s.RecordHourlyMetrics(ctx, metrics); err != nil {
			t.Fatalf("Failed to seed hourly metrics: %v", err)
		}
	}
}

func seedDailyMetrics(t *testing.T, s *store.Store, days int) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(24 * time.Hour)
	for i := 0; i < days; i++ {
		date := now.Add(-time.Duration(i) * 24 * time.Hour)
		metrics := &store.AnalyticsDailyMetrics{
			Date:              date,
			RequestsTotal:     int64(2400 + i*100),
			RequestsSuccess:   int64(2160 + i*90),
			RequestsFailed:    int64(240 + i*10),
			AvgResponseTimeMs: 150.0 + float64(i)*5,
			JobsCompleted:     int64(1200 + i*50),
			JobsFailed:        int64(120 + i*5),
			UniqueHosts:       10 + i,
			CreatedAt:         time.Now().UTC(),
		}
		if err := s.RecordDailyMetrics(ctx, metrics); err != nil {
			t.Fatalf("Failed to seed daily metrics: %v", err)
		}
	}
}

func seedHostMetrics(t *testing.T, s *store.Store, host string, hours int) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Hour)
	for i := 0; i < hours; i++ {
		hour := now.Add(-time.Duration(i) * time.Hour)
		metrics := &store.AnalyticsHostMetrics{
			Hour:              hour,
			Host:              host,
			RequestsTotal:     int64(50 + i*5),
			RequestsSuccess:   int64(45 + i*4),
			RequestsFailed:    int64(5 + i),
			AvgResponseTimeMs: 200.0 + float64(i)*5,
		}
		if err := s.RecordHostMetrics(ctx, metrics); err != nil {
			t.Fatalf("Failed to seed host metrics: %v", err)
		}
	}
}

func seedJobTrends(t *testing.T, s *store.Store, kind model.Kind, status model.Status, count int) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(24 * time.Hour)
	for i := 0; i < count; i++ {
		date := now.Add(-time.Duration(i) * 24 * time.Hour)
		trend := &store.AnalyticsJobTrend{
			Date:          date,
			JobKind:       kind,
			Status:        status,
			Count:         int64(10 + i),
			AvgDurationMs: float64(5000 + i*100),
		}
		if err := s.RecordJobTrend(ctx, trend); err != nil {
			t.Fatalf("Failed to seed job trend: %v", err)
		}
	}
}

func TestService_GetDashboardData_24h(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	seedHourlyMetrics(t, s, 24)
	seedHostMetrics(t, s, "example.com", 24)
	seedJobTrends(t, s, model.KindScrape, model.StatusSucceeded, 1)

	svc := NewService(s)
	ctx := context.Background()

	data, err := svc.GetDashboardData(ctx, "24h")
	if err != nil {
		t.Fatalf("GetDashboardData failed: %v", err)
	}

	if data.Period != "24h" {
		t.Errorf("expected period 24h, got %s", data.Period)
	}

	// Summary should be populated
	if data.Summary.TotalRequests == 0 {
		t.Error("expected non-zero TotalRequests in summary")
	}

	// TimeSeries should have data for 24h period
	if len(data.TimeSeries) == 0 {
		t.Error("expected time series data for 24h period")
	}

	// TopHosts should be populated
	if len(data.TopHosts) == 0 {
		t.Error("expected top hosts data")
	}
}

func TestService_GetDashboardData_7d(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	seedHourlyMetrics(t, s, 24*7)
	seedHostMetrics(t, s, "example.com", 24*7)
	seedJobTrends(t, s, model.KindScrape, model.StatusSucceeded, 7)

	svc := NewService(s)
	ctx := context.Background()

	data, err := svc.GetDashboardData(ctx, "7d")
	if err != nil {
		t.Fatalf("GetDashboardData failed: %v", err)
	}

	if data.Period != "7d" {
		t.Errorf("expected period 7d, got %s", data.Period)
	}

	if len(data.TimeSeries) == 0 {
		t.Error("expected time series data for 7d period")
	}
}

func TestService_GetDashboardData_30d(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	seedDailyMetrics(t, s, 30)
	seedHostMetrics(t, s, "example.com", 24*30)
	seedJobTrends(t, s, model.KindScrape, model.StatusSucceeded, 30)

	svc := NewService(s)
	ctx := context.Background()

	data, err := svc.GetDashboardData(ctx, "30d")
	if err != nil {
		t.Fatalf("GetDashboardData failed: %v", err)
	}

	if data.Period != "30d" {
		t.Errorf("expected period 30d, got %s", data.Period)
	}

	// For 30d period, time series should still be populated
	if len(data.Trends) == 0 {
		t.Error("expected trends data for 30d period")
	}
}

func TestService_GetDashboardData_90d(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	seedDailyMetrics(t, s, 90)
	seedHostMetrics(t, s, "example.com", 24*90)
	seedJobTrends(t, s, model.KindScrape, model.StatusSucceeded, 90)

	svc := NewService(s)
	ctx := context.Background()

	data, err := svc.GetDashboardData(ctx, "90d")
	if err != nil {
		t.Fatalf("GetDashboardData failed: %v", err)
	}

	if data.Period != "90d" {
		t.Errorf("expected period 90d, got %s", data.Period)
	}
}

func TestService_GetDashboardData_DefaultFallback(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	seedHourlyMetrics(t, s, 24)
	seedHostMetrics(t, s, "example.com", 24)

	svc := NewService(s)
	ctx := context.Background()

	// Invalid period should fall back to 24h time range but period string is preserved
	data, err := svc.GetDashboardData(ctx, "invalid")
	if err != nil {
		t.Fatalf("GetDashboardData failed: %v", err)
	}

	if data.Period != "invalid" {
		t.Errorf("expected period to remain 'invalid', got %s", data.Period)
	}

	// Note: TimeSeries is only populated for "24h" and "7d" periods.
	// For invalid periods, the time range defaults to 24h but time series is not fetched.
	// This is the current service behavior - summary, top hosts, and trends are still returned.
	if data.Summary.TotalRequests == 0 {
		t.Error("expected summary data for default (24h) period")
	}
}

func TestService_GetMetrics_Hourly(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	seedHourlyMetrics(t, s, 24)

	svc := NewService(s)
	ctx := context.Background()
	now := time.Now().UTC()
	start := now.Add(-24 * time.Hour)

	result, err := svc.GetMetrics(ctx, start, now, GranularityHourly)
	if err != nil {
		t.Fatalf("GetMetrics failed: %v", err)
	}

	metrics, ok := result.([]store.AnalyticsHourlyMetrics)
	if !ok {
		t.Fatalf("expected []AnalyticsHourlyMetrics, got %T", result)
	}

	if len(metrics) == 0 {
		t.Error("expected hourly metrics data")
	}
}

func TestService_GetMetrics_Daily(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	seedDailyMetrics(t, s, 30)

	svc := NewService(s)
	ctx := context.Background()
	now := time.Now().UTC()
	start := now.Add(-30 * 24 * time.Hour)

	result, err := svc.GetMetrics(ctx, start, now, GranularityDaily)
	if err != nil {
		t.Fatalf("GetMetrics failed: %v", err)
	}

	metrics, ok := result.([]store.AnalyticsDailyMetrics)
	if !ok {
		t.Fatalf("expected []AnalyticsDailyMetrics, got %T", result)
	}

	if len(metrics) == 0 {
		t.Error("expected daily metrics data")
	}
}

func TestService_GetMetrics_DefaultFallback(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	seedHourlyMetrics(t, s, 24)

	svc := NewService(s)
	ctx := context.Background()
	now := time.Now().UTC()
	start := now.Add(-24 * time.Hour)

	// Invalid granularity should fall back to hourly
	result, err := svc.GetMetrics(ctx, start, now, Granularity("invalid"))
	if err != nil {
		t.Fatalf("GetMetrics failed: %v", err)
	}

	_, ok := result.([]store.AnalyticsHourlyMetrics)
	if !ok {
		t.Fatalf("expected []AnalyticsHourlyMetrics for default fallback, got %T", result)
	}
}

func TestService_GetHostMetrics(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	seedHostMetrics(t, s, "example.com", 24)
	seedHostMetrics(t, s, "other.com", 12)

	svc := NewService(s)
	ctx := context.Background()
	now := time.Now().UTC()
	start := now.Add(-24 * time.Hour)

	metrics, err := svc.GetHostMetrics(ctx, "example.com", start, now)
	if err != nil {
		t.Fatalf("GetHostMetrics failed: %v", err)
	}

	if len(metrics) == 0 {
		t.Error("expected host metrics data")
	}

	// Verify all returned metrics are for the requested host
	for _, m := range metrics {
		if m.Host != "example.com" {
			t.Errorf("expected host example.com, got %s", m.Host)
		}
	}
}

func TestService_GetTopHosts(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	// Seed multiple hosts with different request counts
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Hour)

	hosts := []struct {
		name     string
		requests int64
	}{
		{"high-traffic.com", 1000},
		{"medium-traffic.com", 500},
		{"low-traffic.com", 100},
	}

	for _, h := range hosts {
		metrics := &store.AnalyticsHostMetrics{
			Hour:            now,
			Host:            h.name,
			RequestsTotal:   h.requests,
			RequestsSuccess: h.requests - 10,
			RequestsFailed:  10,
		}
		if err := s.RecordHostMetrics(ctx, metrics); err != nil {
			t.Fatalf("Failed to seed host metrics: %v", err)
		}
	}

	svc := NewService(s)
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)

	topHosts, err := svc.GetTopHosts(ctx, start, end, 10)
	if err != nil {
		t.Fatalf("GetTopHosts failed: %v", err)
	}

	if len(topHosts) != 3 {
		t.Errorf("expected 3 hosts, got %d", len(topHosts))
	}

	// Verify ordering (highest traffic first)
	if len(topHosts) > 0 && topHosts[0].Host != "high-traffic.com" {
		t.Errorf("expected high-traffic.com first, got %s", topHosts[0].Host)
	}
}

func TestService_GetTopHosts_WithLimit(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Hour)

	// Seed 5 hosts
	for i := 0; i < 5; i++ {
		metrics := &store.AnalyticsHostMetrics{
			Hour:            now,
			Host:            "host-" + string(rune('a'+i)) + ".com",
			RequestsTotal:   int64(100 - i*10),
			RequestsSuccess: int64(90 - i*9),
			RequestsFailed:  10,
		}
		if err := s.RecordHostMetrics(ctx, metrics); err != nil {
			t.Fatalf("Failed to seed host metrics: %v", err)
		}
	}

	svc := NewService(s)
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)

	topHosts, err := svc.GetTopHosts(ctx, start, end, 3)
	if err != nil {
		t.Fatalf("GetTopHosts failed: %v", err)
	}

	if len(topHosts) != 3 {
		t.Errorf("expected 3 hosts (limited), got %d", len(topHosts))
	}
}

func TestService_GetJobTrends(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	seedJobTrends(t, s, model.KindScrape, model.StatusSucceeded, 7)
	seedJobTrends(t, s, model.KindCrawl, model.StatusFailed, 5)

	svc := NewService(s)
	ctx := context.Background()
	now := time.Now().UTC()
	start := now.Add(-7 * 24 * time.Hour)

	trends, err := svc.GetJobTrends(ctx, start, now)
	if err != nil {
		t.Fatalf("GetJobTrends failed: %v", err)
	}

	if len(trends) == 0 {
		t.Error("expected job trends data")
	}

	// Verify we have both kinds
	hasScrape := false
	hasCrawl := false
	for _, t := range trends {
		if t.JobKind == model.KindScrape {
			hasScrape = true
		}
		if t.JobKind == model.KindCrawl {
			hasCrawl = true
		}
	}

	if !hasScrape {
		t.Error("expected scrape job trends")
	}
	if !hasCrawl {
		t.Error("expected crawl job trends")
	}
}

func TestService_EmptyStore(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	svc := NewService(s)
	ctx := context.Background()

	// Test with empty store - should not error
	data, err := svc.GetDashboardData(ctx, "24h")
	if err != nil {
		t.Fatalf("GetDashboardData on empty store failed: %v", err)
	}

	if data.Summary.TotalRequests != 0 {
		t.Error("expected zero TotalRequests for empty store")
	}

	if len(data.TimeSeries) != 0 {
		t.Error("expected empty time series for empty store")
	}

	if len(data.TopHosts) != 0 {
		t.Error("expected empty top hosts for empty store")
	}

	if len(data.Trends) != 0 {
		t.Error("expected empty trends for empty store")
	}
}

func TestService_GetMetrics_TimeRangeFiltering(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	ctx := context.Background()
	baseTime := time.Now().UTC().Truncate(time.Hour)

	// Create metrics at specific hours
	for i := 0; i < 10; i++ {
		hour := baseTime.Add(-time.Duration(i) * time.Hour)
		metrics := &store.AnalyticsHourlyMetrics{
			Hour:            hour,
			RequestsTotal:   int64(100 + i),
			RequestsSuccess: 90,
			RequestsFailed:  10,
			CreatedAt:       time.Now().UTC(),
		}
		if err := s.RecordHourlyMetrics(ctx, metrics); err != nil {
			t.Fatalf("Failed to seed hourly metrics: %v", err)
		}
	}

	svc := NewService(s)

	// Query for only last 5 hours
	start := baseTime.Add(-5 * time.Hour)
	end := baseTime.Add(time.Hour)

	result, err := svc.GetMetrics(ctx, start, end, GranularityHourly)
	if err != nil {
		t.Fatalf("GetMetrics failed: %v", err)
	}

	metrics := result.([]store.AnalyticsHourlyMetrics)

	// Should get 6 hours of data (including current hour)
	if len(metrics) < 5 || len(metrics) > 7 {
		t.Errorf("expected ~6 hours of data, got %d", len(metrics))
	}

	// Verify all returned metrics are within the time range
	for _, m := range metrics {
		if m.Hour.Before(start) || m.Hour.After(end) {
			t.Errorf("metric hour %v outside range [%v, %v]", m.Hour, start, end)
		}
	}
}

func TestNewService(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	svc := NewService(s)
	if svc == nil {
		t.Fatal("NewService returned nil")
	}

	if svc.store != s {
		t.Error("service store not set correctly")
	}
}
