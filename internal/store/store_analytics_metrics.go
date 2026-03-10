// Package store provides SQLite-backed persistent storage for analytics data.
//
// This file handles:
// - Time-series metrics storage and retrieval (hourly, daily)
// - Per-host performance tracking and aggregation
// - Job trend analysis by kind and status
// - Data retention and daily rollup calculations
//
// This file does NOT handle:
// - Template metrics or A/B testing (see store_analytics_templates.go)
// - Database schema initialization (see store_analytics_core.go)
// - Real-time metrics collection
//
// Invariants:
// - All timestamps stored as RFC3339 in UTC
// - Hourly aggregations are immutable after the hour passes
// - Daily rollups computed from hourly data
// - Automatic retention purges data older than configured retention period
package store

import (
	"context"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// RecordHourlyMetrics records or updates hourly metrics.
func (s *Store) RecordHourlyMetrics(ctx context.Context, metrics *AnalyticsHourlyMetrics) error {
	hourStr := metrics.Hour.UTC().Format(time.RFC3339)
	createdAtStr := metrics.CreatedAt.UTC().Format(time.RFC3339)

	_, err := s.stmtRecordHourlyMetrics.ExecContext(ctx,
		hourStr,
		metrics.RequestsTotal,
		metrics.RequestsSuccess,
		metrics.RequestsFailed,
		metrics.AvgResponseTimeMs,
		metrics.TotalResponseTime.Milliseconds(),
		metrics.JobsCompleted,
		metrics.JobsFailed,
		metrics.AvgJobDurationMs,
		metrics.TotalJobDuration.Milliseconds(),
		metrics.FetcherHTTP,
		metrics.FetcherChromedp,
		metrics.FetcherPlaywright,
		createdAtStr,
	)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to record hourly metrics", err)
	}
	return nil
}

// GetHourlyMetrics retrieves hourly metrics for a time range.
func (s *Store) GetHourlyMetrics(ctx context.Context, start, end time.Time) ([]AnalyticsHourlyMetrics, error) {
	startStr := start.UTC().Format(time.RFC3339)
	endStr := end.UTC().Format(time.RFC3339)

	rows, err := s.stmtGetHourlyMetrics.QueryContext(ctx, startStr, endStr)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to query hourly metrics", err)
	}
	defer rows.Close()

	var results []AnalyticsHourlyMetrics
	for rows.Next() {
		var m AnalyticsHourlyMetrics
		var hourStr, createdAtStr string
		var totalResponseTimeMs, totalJobDurationMs int64

		err := rows.Scan(
			&hourStr,
			&m.RequestsTotal,
			&m.RequestsSuccess,
			&m.RequestsFailed,
			&m.AvgResponseTimeMs,
			&totalResponseTimeMs,
			&m.JobsCompleted,
			&m.JobsFailed,
			&m.AvgJobDurationMs,
			&totalJobDurationMs,
			&m.FetcherHTTP,
			&m.FetcherChromedp,
			&m.FetcherPlaywright,
			&createdAtStr,
		)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to scan hourly metrics row", err)
		}

		m.Hour, _ = time.Parse(time.RFC3339, hourStr)
		m.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		m.TotalResponseTime = time.Duration(totalResponseTimeMs) * time.Millisecond
		m.TotalJobDuration = time.Duration(totalJobDurationMs) * time.Millisecond

		results = append(results, m)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "error iterating hourly metrics rows", err)
	}

	return results, nil
}

// RecordHostMetrics records or updates host metrics for an hour.
func (s *Store) RecordHostMetrics(ctx context.Context, metrics *AnalyticsHostMetrics) error {
	hourStr := metrics.Hour.UTC().Format(time.RFC3339)

	_, err := s.stmtRecordHostMetrics.ExecContext(ctx,
		hourStr,
		metrics.Host,
		metrics.RequestsTotal,
		metrics.RequestsSuccess,
		metrics.RequestsFailed,
		metrics.AvgResponseTimeMs,
		metrics.TotalResponseTime.Milliseconds(),
		metrics.RateLimitHits,
	)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to record host metrics", err)
	}
	return nil
}

// GetHostMetrics retrieves host metrics for a specific host and time range.
func (s *Store) GetHostMetrics(ctx context.Context, host string, start, end time.Time) ([]AnalyticsHostMetrics, error) {
	startStr := start.UTC().Format(time.RFC3339)
	endStr := end.UTC().Format(time.RFC3339)

	rows, err := s.stmtGetHostMetrics.QueryContext(ctx, host, startStr, endStr)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to query host metrics", err)
	}
	defer rows.Close()

	var results []AnalyticsHostMetrics
	for rows.Next() {
		var m AnalyticsHostMetrics
		var hourStr string
		var totalResponseTimeMs int64

		err := rows.Scan(
			&hourStr,
			&m.Host,
			&m.RequestsTotal,
			&m.RequestsSuccess,
			&m.RequestsFailed,
			&m.AvgResponseTimeMs,
			&totalResponseTimeMs,
			&m.RateLimitHits,
		)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to scan host metrics row", err)
		}

		m.Hour, _ = time.Parse(time.RFC3339, hourStr)
		m.TotalResponseTime = time.Duration(totalResponseTimeMs) * time.Millisecond

		results = append(results, m)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "error iterating host metrics rows", err)
	}

	return results, nil
}

// GetTopHosts retrieves the top N hosts by request count for a time range.
func (s *Store) GetTopHosts(ctx context.Context, start, end time.Time, limit int) ([]AnalyticsHostSummary, error) {
	startStr := start.UTC().Format(time.RFC3339)
	endStr := end.UTC().Format(time.RFC3339)

	rows, err := s.stmtGetTopHosts.QueryContext(ctx, startStr, endStr, limit)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to query top hosts", err)
	}
	defer rows.Close()

	var results []AnalyticsHostSummary
	for rows.Next() {
		var h AnalyticsHostSummary

		err := rows.Scan(
			&h.Host,
			&h.RequestsTotal,
			&h.AvgResponseTimeMs,
			&h.SuccessRate,
			&h.RateLimitHits,
		)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to scan top hosts row", err)
		}

		results = append(results, h)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "error iterating top hosts rows", err)
	}

	return results, nil
}

// RecordDailyMetrics records or updates daily metrics.
func (s *Store) RecordDailyMetrics(ctx context.Context, metrics *AnalyticsDailyMetrics) error {
	dateStr := metrics.Date.UTC().Format("2006-01-02")
	createdAtStr := metrics.CreatedAt.UTC().Format(time.RFC3339)

	_, err := s.stmtRecordDailyMetrics.ExecContext(ctx,
		dateStr,
		metrics.RequestsTotal,
		metrics.RequestsSuccess,
		metrics.RequestsFailed,
		metrics.AvgResponseTimeMs,
		metrics.JobsCompleted,
		metrics.JobsFailed,
		metrics.AvgJobDurationMs,
		metrics.UniqueHosts,
		createdAtStr,
	)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to record daily metrics", err)
	}
	return nil
}

// GetDailyMetrics retrieves daily metrics for a date range.
func (s *Store) GetDailyMetrics(ctx context.Context, start, end time.Time) ([]AnalyticsDailyMetrics, error) {
	startStr := start.UTC().Format("2006-01-02")
	endStr := end.UTC().Format("2006-01-02")

	rows, err := s.stmtGetDailyMetrics.QueryContext(ctx, startStr, endStr)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to query daily metrics", err)
	}
	defer rows.Close()

	var results []AnalyticsDailyMetrics
	for rows.Next() {
		var m AnalyticsDailyMetrics
		var dateStr, createdAtStr string

		err := rows.Scan(
			&dateStr,
			&m.RequestsTotal,
			&m.RequestsSuccess,
			&m.RequestsFailed,
			&m.AvgResponseTimeMs,
			&m.JobsCompleted,
			&m.JobsFailed,
			&m.AvgJobDurationMs,
			&m.UniqueHosts,
			&createdAtStr,
		)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to scan daily metrics row", err)
		}

		m.Date, _ = time.Parse("2006-01-02", dateStr)
		m.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)

		results = append(results, m)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "error iterating daily metrics rows", err)
	}

	return results, nil
}

// RecordJobTrend records or updates a job trend entry.
func (s *Store) RecordJobTrend(ctx context.Context, trend *AnalyticsJobTrend) error {
	dateStr := trend.Date.UTC().Format("2006-01-02")

	_, err := s.stmtRecordJobTrend.ExecContext(ctx,
		dateStr,
		string(trend.JobKind),
		string(trend.Status),
		trend.Count,
		trend.AvgDurationMs,
		trend.TotalDuration.Milliseconds(),
	)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to record job trend", err)
	}
	return nil
}

// GetJobTrends retrieves job trends for a date range.
func (s *Store) GetJobTrends(ctx context.Context, start, end time.Time) ([]AnalyticsJobTrend, error) {
	startStr := start.UTC().Format("2006-01-02")
	endStr := end.UTC().Format("2006-01-02")

	rows, err := s.stmtGetJobTrends.QueryContext(ctx, startStr, endStr)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to query job trends", err)
	}
	defer rows.Close()

	var results []AnalyticsJobTrend
	for rows.Next() {
		var t AnalyticsJobTrend
		var dateStr, kindStr, statusStr string
		var totalDurationMs int64

		err := rows.Scan(
			&dateStr,
			&kindStr,
			&statusStr,
			&t.Count,
			&t.AvgDurationMs,
			&totalDurationMs,
		)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to scan job trend row", err)
		}

		t.Date, _ = time.Parse("2006-01-02", dateStr)
		t.JobKind = model.Kind(kindStr)
		t.Status = model.Status(statusStr)
		t.TotalDuration = time.Duration(totalDurationMs) * time.Millisecond

		results = append(results, t)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "error iterating job trends rows", err)
	}

	return results, nil
}

// PurgeOldAnalytics removes analytics data older than the specified time.
func (s *Store) PurgeOldAnalytics(ctx context.Context, before time.Time) error {
	hourStr := before.UTC().Format(time.RFC3339)
	dateStr := before.UTC().Format("2006-01-02")

	_, err := s.stmtPurgeOldAnalytics.ExecContext(ctx, hourStr, hourStr, dateStr, dateStr)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to purge old analytics data", err)
	}
	return nil
}

// truncateToDay returns the time truncated to the day.
func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// RollupDaily aggregates hourly metrics into daily metrics for a specific date.
func (s *Store) RollupDaily(ctx context.Context, date time.Time) (*AnalyticsDailyMetrics, error) {
	startOfDay := truncateToDay(date)
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Get all hourly metrics for the day
	hourlyMetrics, err := s.GetHourlyMetrics(ctx, startOfDay, endOfDay)
	if err != nil {
		return nil, err
	}

	if len(hourlyMetrics) == 0 {
		// No data for this day
		return &AnalyticsDailyMetrics{
			Date:      startOfDay,
			CreatedAt: time.Now().UTC(),
		}, nil
	}

	// Aggregate the hourly data
	daily := &AnalyticsDailyMetrics{
		Date:      startOfDay,
		CreatedAt: time.Now().UTC(),
	}

	hostSet := make(map[string]bool)
	var totalResponseTime time.Duration
	var totalJobDuration time.Duration

	for _, h := range hourlyMetrics {
		daily.RequestsTotal += h.RequestsTotal
		daily.RequestsSuccess += h.RequestsSuccess
		daily.RequestsFailed += h.RequestsFailed
		daily.JobsCompleted += h.JobsCompleted
		daily.JobsFailed += h.JobsFailed
		totalResponseTime += h.TotalResponseTime
		totalJobDuration += h.TotalJobDuration
	}

	// Calculate averages
	if daily.RequestsTotal > 0 {
		daily.AvgResponseTimeMs = float64(totalResponseTime.Milliseconds()) / float64(daily.RequestsTotal)
	}
	if daily.JobsCompleted+daily.JobsFailed > 0 {
		daily.AvgJobDurationMs = float64(totalJobDuration.Milliseconds()) / float64(daily.JobsCompleted+daily.JobsFailed)
	}

	// Count unique hosts for the day
	hostRows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT host FROM analytics_host_hourly
		WHERE hour >= ? AND hour < ?
	`, startOfDay.Format(time.RFC3339), endOfDay.Format(time.RFC3339))
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to count unique hosts", err)
	}
	defer hostRows.Close()

	for hostRows.Next() {
		var host string
		if err := hostRows.Scan(&host); err != nil {
			continue
		}
		hostSet[host] = true
	}
	daily.UniqueHosts = len(hostSet)

	// Save the daily rollup
	if err := s.RecordDailyMetrics(ctx, daily); err != nil {
		return nil, err
	}

	return daily, nil
}

// GetAnalyticsSummary calculates a summary for a time range.
func (s *Store) GetAnalyticsSummary(ctx context.Context, start, end time.Time) (*AnalyticsSummary, error) {
	startStr := start.UTC().Format(time.RFC3339)
	endStr := end.UTC().Format(time.RFC3339)

	var summary AnalyticsSummary
	var totalResponseTimeMs int64
	var totalRequests int64

	// Get aggregate from hourly metrics
	row := s.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(requests_total), 0),
			COALESCE(SUM(requests_success), 0),
			COALESCE(SUM(jobs_completed) + SUM(jobs_failed), 0),
			COALESCE(SUM(total_response_time_ms), 0)
		FROM analytics_hourly
		WHERE hour >= ? AND hour <= ?
	`, startStr, endStr)

	err := row.Scan(&totalRequests, &summary.TotalRequests, &summary.TotalJobs, &totalResponseTimeMs)
	if err != nil && !isNoRowsError(err) {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to calculate analytics summary", err)
	}

	// Calculate average success rate
	if totalRequests > 0 {
		summary.AvgSuccessRate = float64(summary.TotalRequests) * 100.0 / float64(totalRequests)
	} else {
		summary.AvgSuccessRate = 100.0
	}

	// Calculate average response time
	if totalRequests > 0 {
		summary.AvgResponseTimeMs = float64(totalResponseTimeMs) / float64(totalRequests)
	}

	// Count unique hosts
	hostRow := s.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT host)
		FROM analytics_host_hourly
		WHERE hour >= ? AND hour <= ?
	`, startStr, endStr)
	hostRow.Scan(&summary.UniqueHosts)

	return &summary, nil
}
