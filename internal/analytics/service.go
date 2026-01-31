// Package analytics provides analytics query services and data aggregation.
//
// This file handles:
// - Querying analytics data from the store
// - Aggregating data for dashboard views
// - Time-series data retrieval
//
// This file does NOT handle:
// - Data collection (collector.go handles that)
// - Data persistence (store/store_analytics.go handles that)
// - Report generation (reports.go handles that)
package analytics

import (
	"context"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/store"
)

// Granularity represents the time bucket size for analytics queries.
type Granularity string

const (
	GranularityHourly Granularity = "hourly"
	GranularityDaily  Granularity = "daily"
)

// DashboardData contains all data needed for the analytics dashboard.
type DashboardData struct {
	Period     string                         `json:"period"`
	Summary    store.AnalyticsSummary         `json:"summary"`
	TimeSeries []store.AnalyticsHourlyMetrics `json:"timeSeries"`
	TopHosts   []store.AnalyticsHostSummary   `json:"topHosts"`
	Trends     []store.AnalyticsJobTrend      `json:"trends"`
}

// Service provides analytics query capabilities.
type Service struct {
	store *store.Store
}

// NewService creates a new analytics service.
func NewService(store *store.Store) *Service {
	return &Service{store: store}
}

// GetDashboardData retrieves pre-computed dashboard data for a time period.
func (s *Service) GetDashboardData(ctx context.Context, period string) (*DashboardData, error) {
	now := time.Now().UTC()
	var start time.Time

	switch period {
	case "24h":
		start = now.Add(-24 * time.Hour)
	case "7d":
		start = now.Add(-7 * 24 * time.Hour)
	case "30d":
		start = now.Add(-30 * 24 * time.Hour)
	case "90d":
		start = now.Add(-90 * 24 * time.Hour)
	default:
		start = now.Add(-24 * time.Hour)
	}

	end := now

	// Get summary
	summary, err := s.store.GetAnalyticsSummary(ctx, start, end)
	if err != nil {
		return nil, err
	}

	// Get time series data (hourly for shorter periods, daily for longer)
	var timeSeries []store.AnalyticsHourlyMetrics
	if period == "24h" || period == "7d" {
		timeSeries, err = s.store.GetHourlyMetrics(ctx, start, end)
		if err != nil {
			return nil, err
		}
	}

	// Get top hosts
	topHosts, err := s.store.GetTopHosts(ctx, start, end, 10)
	if err != nil {
		return nil, err
	}

	// Get job trends
	trends, err := s.store.GetJobTrends(ctx, start, end)
	if err != nil {
		return nil, err
	}

	return &DashboardData{
		Period:     period,
		Summary:    *summary,
		TimeSeries: timeSeries,
		TopHosts:   topHosts,
		Trends:     trends,
	}, nil
}

// GetMetrics retrieves metrics for a time range with specified granularity.
func (s *Service) GetMetrics(ctx context.Context, start, end time.Time, granularity Granularity) (interface{}, error) {
	switch granularity {
	case GranularityDaily:
		return s.store.GetDailyMetrics(ctx, start, end)
	case GranularityHourly:
		return s.store.GetHourlyMetrics(ctx, start, end)
	default:
		return s.store.GetHourlyMetrics(ctx, start, end)
	}
}

// GetHostMetrics retrieves metrics for a specific host.
func (s *Service) GetHostMetrics(ctx context.Context, host string, start, end time.Time) ([]store.AnalyticsHostMetrics, error) {
	return s.store.GetHostMetrics(ctx, host, start, end)
}

// GetTopHosts retrieves the top N hosts by request count.
func (s *Service) GetTopHosts(ctx context.Context, start, end time.Time, limit int) ([]store.AnalyticsHostSummary, error) {
	return s.store.GetTopHosts(ctx, start, end, limit)
}

// GetJobTrends retrieves job trends for a date range.
func (s *Service) GetJobTrends(ctx context.Context, start, end time.Time) ([]store.AnalyticsJobTrend, error) {
	return s.store.GetJobTrends(ctx, start, end)
}
