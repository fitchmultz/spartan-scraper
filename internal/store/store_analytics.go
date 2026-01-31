// Package store provides SQLite-backed persistent storage for analytics data.
//
// This file handles:
// - Time-series metrics storage (hourly/daily aggregations)
// - Per-host performance tracking
// - Job trend analysis
// - Data retention and rollup calculations
//
// This file does NOT handle:
// - Real-time metrics collection (api/metrics.go handles that)
// - Analytics queries (analytics/service.go handles that)
// - Report generation (analytics/reports.go handles that)
//
// Invariants:
// - All timestamps stored as RFC3339 in UTC
// - Hourly aggregations are immutable after the hour passes
// - Daily rollups computed from hourly data
// - Automatic retention purges data older than configured retention period
package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// initAnalyticsTables creates the analytics tables if they don't exist.
// This is called during store initialization.
func (s *Store) initAnalyticsTables() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS analytics_hourly (
			hour TEXT PRIMARY KEY,
			requests_total INTEGER NOT NULL DEFAULT 0,
			requests_success INTEGER NOT NULL DEFAULT 0,
			requests_failed INTEGER NOT NULL DEFAULT 0,
			avg_response_time_ms REAL NOT NULL DEFAULT 0,
			total_response_time_ms INTEGER NOT NULL DEFAULT 0,
			jobs_completed INTEGER NOT NULL DEFAULT 0,
			jobs_failed INTEGER NOT NULL DEFAULT 0,
			avg_job_duration_ms REAL NOT NULL DEFAULT 0,
			total_job_duration_ms INTEGER NOT NULL DEFAULT 0,
			fetcher_http INTEGER NOT NULL DEFAULT 0,
			fetcher_chromedp INTEGER NOT NULL DEFAULT 0,
			fetcher_playwright INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_analytics_hourly_created ON analytics_hourly(created_at DESC);

		CREATE TABLE IF NOT EXISTS analytics_host_hourly (
			hour TEXT NOT NULL,
			host TEXT NOT NULL,
			requests_total INTEGER NOT NULL DEFAULT 0,
			requests_success INTEGER NOT NULL DEFAULT 0,
			requests_failed INTEGER NOT NULL DEFAULT 0,
			avg_response_time_ms REAL NOT NULL DEFAULT 0,
			total_response_time_ms INTEGER NOT NULL DEFAULT 0,
			rate_limit_hits INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (hour, host)
		);
		CREATE INDEX IF NOT EXISTS idx_analytics_host_hourly_hour ON analytics_host_hourly(hour DESC);
		CREATE INDEX IF NOT EXISTS idx_analytics_host_hourly_host ON analytics_host_hourly(host);

		CREATE TABLE IF NOT EXISTS analytics_daily (
			date TEXT PRIMARY KEY,
			requests_total INTEGER NOT NULL DEFAULT 0,
			requests_success INTEGER NOT NULL DEFAULT 0,
			requests_failed INTEGER NOT NULL DEFAULT 0,
			avg_response_time_ms REAL NOT NULL DEFAULT 0,
			jobs_completed INTEGER NOT NULL DEFAULT 0,
			jobs_failed INTEGER NOT NULL DEFAULT 0,
			avg_job_duration_ms REAL NOT NULL DEFAULT 0,
			unique_hosts INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_analytics_daily_created ON analytics_daily(created_at DESC);

		CREATE TABLE IF NOT EXISTS analytics_job_trends (
			date TEXT NOT NULL,
			job_kind TEXT NOT NULL,
			status TEXT NOT NULL,
			count INTEGER NOT NULL DEFAULT 0,
			avg_duration_ms REAL NOT NULL DEFAULT 0,
			total_duration_ms INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (date, job_kind, status)
		);
		CREATE INDEX IF NOT EXISTS idx_analytics_job_trends_date ON analytics_job_trends(date DESC);

		-- Template metrics table
		CREATE TABLE IF NOT EXISTS analytics_template_hourly (
			hour TEXT NOT NULL,
			template_name TEXT NOT NULL,
			extractions_total INTEGER DEFAULT 0,
			extractions_success INTEGER DEFAULT 0,
			field_coverage_sum REAL DEFAULT 0,
			field_coverage_count INTEGER DEFAULT 0,
			total_extraction_time_ms INTEGER DEFAULT 0,
			PRIMARY KEY (hour, template_name)
		);
		CREATE INDEX IF NOT EXISTS idx_analytics_template_hourly_hour ON analytics_template_hourly(hour DESC);
		CREATE INDEX IF NOT EXISTS idx_analytics_template_hourly_name ON analytics_template_hourly(template_name);

		-- A/B tests table
		CREATE TABLE IF NOT EXISTS template_ab_tests (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			baseline_template TEXT NOT NULL,
			variant_template TEXT NOT NULL,
			allocation_json TEXT NOT NULL,
			start_time TEXT NOT NULL,
			end_time TEXT,
			status TEXT NOT NULL,
			success_criteria_json TEXT NOT NULL,
			min_sample_size INTEGER DEFAULT 100,
			confidence_level REAL DEFAULT 0.95,
			winner TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_template_ab_tests_status ON template_ab_tests(status);

		-- Per-extraction records for statistical analysis
		CREATE TABLE IF NOT EXISTS template_extraction_records (
			id TEXT PRIMARY KEY,
			test_id TEXT,
			template_name TEXT NOT NULL,
			target_url TEXT NOT NULL,
			success INTEGER NOT NULL,
			field_coverage REAL NOT NULL,
			extraction_time_ms INTEGER NOT NULL,
			validation_errors_json TEXT,
			extracted_fields_json TEXT,
			timestamp TEXT NOT NULL,
			FOREIGN KEY (test_id) REFERENCES template_ab_tests(id)
		);
		CREATE INDEX IF NOT EXISTS idx_template_records_test ON template_extraction_records(test_id);
		CREATE INDEX IF NOT EXISTS idx_template_records_template ON template_extraction_records(template_name);
		CREATE INDEX IF NOT EXISTS idx_template_records_timestamp ON template_extraction_records(timestamp DESC);
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create analytics tables", err)
	}

	// Prepare analytics statements
	if err := s.prepareAnalyticsStatements(); err != nil {
		return err
	}

	return nil
}

// prepareAnalyticsStatements prepares SQL statements for analytics operations.
func (s *Store) prepareAnalyticsStatements() error {
	var err error

	s.stmtRecordHourlyMetrics, err = s.db.Prepare(`
		INSERT INTO analytics_hourly (
			hour, requests_total, requests_success, requests_failed,
			avg_response_time_ms, total_response_time_ms,
			jobs_completed, jobs_failed, avg_job_duration_ms, total_job_duration_ms,
			fetcher_http, fetcher_chromedp, fetcher_playwright, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(hour) DO UPDATE SET
			requests_total = excluded.requests_total,
			requests_success = excluded.requests_success,
			requests_failed = excluded.requests_failed,
			avg_response_time_ms = excluded.avg_response_time_ms,
			total_response_time_ms = excluded.total_response_time_ms,
			jobs_completed = excluded.jobs_completed,
			jobs_failed = excluded.jobs_failed,
			avg_job_duration_ms = excluded.avg_job_duration_ms,
			total_job_duration_ms = excluded.total_job_duration_ms,
			fetcher_http = excluded.fetcher_http,
			fetcher_chromedp = excluded.fetcher_chromedp,
			fetcher_playwright = excluded.fetcher_playwright
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare record hourly metrics statement", err)
	}

	s.stmtGetHourlyMetrics, err = s.db.Prepare(`
		SELECT hour, requests_total, requests_success, requests_failed,
			avg_response_time_ms, total_response_time_ms,
			jobs_completed, jobs_failed, avg_job_duration_ms, total_job_duration_ms,
			fetcher_http, fetcher_chromedp, fetcher_playwright, created_at
		FROM analytics_hourly
		WHERE hour >= ? AND hour <= ?
		ORDER BY hour ASC
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare get hourly metrics statement", err)
	}

	s.stmtRecordHostMetrics, err = s.db.Prepare(`
		INSERT INTO analytics_host_hourly (
			hour, host, requests_total, requests_success, requests_failed,
			avg_response_time_ms, total_response_time_ms, rate_limit_hits
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(hour, host) DO UPDATE SET
			requests_total = excluded.requests_total,
			requests_success = excluded.requests_success,
			requests_failed = excluded.requests_failed,
			avg_response_time_ms = excluded.avg_response_time_ms,
			total_response_time_ms = excluded.total_response_time_ms,
			rate_limit_hits = excluded.rate_limit_hits
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare record host metrics statement", err)
	}

	s.stmtGetHostMetrics, err = s.db.Prepare(`
		SELECT hour, host, requests_total, requests_success, requests_failed,
			avg_response_time_ms, total_response_time_ms, rate_limit_hits
		FROM analytics_host_hourly
		WHERE host = ? AND hour >= ? AND hour <= ?
		ORDER BY hour ASC
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare get host metrics statement", err)
	}

	s.stmtGetTopHosts, err = s.db.Prepare(`
		SELECT host,
			SUM(requests_total) as total_requests,
			AVG(avg_response_time_ms) as avg_response_time,
			CASE WHEN SUM(requests_total) > 0
				THEN CAST(SUM(requests_success) AS REAL) * 100.0 / SUM(requests_total)
				ELSE 100.0
			END as success_rate,
			SUM(rate_limit_hits) as total_rate_limit_hits
		FROM analytics_host_hourly
		WHERE hour >= ? AND hour <= ?
		GROUP BY host
		ORDER BY total_requests DESC
		LIMIT ?
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare get top hosts statement", err)
	}

	s.stmtRecordDailyMetrics, err = s.db.Prepare(`
		INSERT INTO analytics_daily (
			date, requests_total, requests_success, requests_failed,
			avg_response_time_ms, jobs_completed, jobs_failed,
			avg_job_duration_ms, unique_hosts, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(date) DO UPDATE SET
			requests_total = excluded.requests_total,
			requests_success = excluded.requests_success,
			requests_failed = excluded.requests_failed,
			avg_response_time_ms = excluded.avg_response_time_ms,
			jobs_completed = excluded.jobs_completed,
			jobs_failed = excluded.jobs_failed,
			avg_job_duration_ms = excluded.avg_job_duration_ms,
			unique_hosts = excluded.unique_hosts
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare record daily metrics statement", err)
	}

	s.stmtGetDailyMetrics, err = s.db.Prepare(`
		SELECT date, requests_total, requests_success, requests_failed,
			avg_response_time_ms, jobs_completed, jobs_failed,
			avg_job_duration_ms, unique_hosts, created_at
		FROM analytics_daily
		WHERE date >= ? AND date <= ?
		ORDER BY date ASC
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare get daily metrics statement", err)
	}

	s.stmtRecordJobTrend, err = s.db.Prepare(`
		INSERT INTO analytics_job_trends (
			date, job_kind, status, count, avg_duration_ms, total_duration_ms
		) VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(date, job_kind, status) DO UPDATE SET
			count = excluded.count,
			avg_duration_ms = excluded.avg_duration_ms,
			total_duration_ms = excluded.total_duration_ms
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare record job trend statement", err)
	}

	s.stmtGetJobTrends, err = s.db.Prepare(`
		SELECT date, job_kind, status, count, avg_duration_ms, total_duration_ms
		FROM analytics_job_trends
		WHERE date >= ? AND date <= ?
		ORDER BY date ASC, job_kind, status
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare get job trends statement", err)
	}

	s.stmtPurgeOldAnalytics, err = s.db.Prepare(`
		DELETE FROM analytics_hourly WHERE hour < ?;
		DELETE FROM analytics_host_hourly WHERE hour < ?;
		DELETE FROM analytics_daily WHERE date < ?;
		DELETE FROM analytics_job_trends WHERE date < ?;
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare purge old analytics statement", err)
	}

	// Template metrics statements
	s.stmtRecordTemplateMetrics, err = s.db.Prepare(`
		INSERT INTO analytics_template_hourly (
			hour, template_name, extractions_total, extractions_success,
			field_coverage_sum, field_coverage_count, total_extraction_time_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(hour, template_name) DO UPDATE SET
			extractions_total = excluded.extractions_total,
			extractions_success = excluded.extractions_success,
			field_coverage_sum = excluded.field_coverage_sum,
			field_coverage_count = excluded.field_coverage_count,
			total_extraction_time_ms = excluded.total_extraction_time_ms
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare record template metrics statement", err)
	}

	s.stmtGetTemplateMetrics, err = s.db.Prepare(`
		SELECT hour, template_name, extractions_total, extractions_success,
			field_coverage_sum, field_coverage_count, total_extraction_time_ms
		FROM analytics_template_hourly
		WHERE template_name = ? AND hour >= ? AND hour <= ?
		ORDER BY hour ASC
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare get template metrics statement", err)
	}

	s.stmtGetAllTemplateMetrics, err = s.db.Prepare(`
		SELECT hour, template_name, extractions_total, extractions_success,
			field_coverage_sum, field_coverage_count, total_extraction_time_ms
		FROM analytics_template_hourly
		WHERE hour >= ? AND hour <= ?
		ORDER BY hour ASC
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare get all template metrics statement", err)
	}

	// A/B test statements
	s.stmtCreateABTest, err = s.db.Prepare(`
		INSERT INTO template_ab_tests (
			id, name, description, baseline_template, variant_template,
			allocation_json, start_time, end_time, status, success_criteria_json,
			min_sample_size, confidence_level, winner, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare create AB test statement", err)
	}

	s.stmtGetABTest, err = s.db.Prepare(`
		SELECT id, name, description, baseline_template, variant_template,
			allocation_json, start_time, end_time, status, success_criteria_json,
			min_sample_size, confidence_level, winner, created_at, updated_at
		FROM template_ab_tests
		WHERE id = ?
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare get AB test statement", err)
	}

	s.stmtListABTests, err = s.db.Prepare(`
		SELECT id, name, description, baseline_template, variant_template,
			allocation_json, start_time, end_time, status, success_criteria_json,
			min_sample_size, confidence_level, winner, created_at, updated_at
		FROM template_ab_tests
		WHERE (? = '' OR status = ?)
		ORDER BY created_at DESC
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare list AB tests statement", err)
	}

	s.stmtUpdateABTest, err = s.db.Prepare(`
		UPDATE template_ab_tests SET
			name = ?, description = ?, baseline_template = ?, variant_template = ?,
			allocation_json = ?, start_time = ?, end_time = ?, status = ?,
			success_criteria_json = ?, min_sample_size = ?, confidence_level = ?,
			winner = ?, updated_at = ?
		WHERE id = ?
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare update AB test statement", err)
	}

	s.stmtUpdateABTestStatus, err = s.db.Prepare(`
		UPDATE template_ab_tests SET status = ?, updated_at = ? WHERE id = ?
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare update AB test status statement", err)
	}

	s.stmtDeleteABTest, err = s.db.Prepare(`
		DELETE FROM template_ab_tests WHERE id = ?
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare delete AB test statement", err)
	}

	// Extraction record statements
	s.stmtRecordExtraction, err = s.db.Prepare(`
		INSERT INTO template_extraction_records (
			id, test_id, template_name, target_url, success, field_coverage,
			extraction_time_ms, validation_errors_json, extracted_fields_json, timestamp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare record extraction statement", err)
	}

	s.stmtGetExtractionRecords, err = s.db.Prepare(`
		SELECT id, test_id, template_name, target_url, success, field_coverage,
			extraction_time_ms, validation_errors_json, extracted_fields_json, timestamp
		FROM template_extraction_records
		WHERE (? = '' OR test_id = ?) AND (? = '' OR template_name = ?)
			AND timestamp >= ? AND timestamp <= ?
		ORDER BY timestamp DESC
	`)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to prepare get extraction records statement", err)
	}

	return nil
}

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
	if err != nil && err != sql.ErrNoRows {
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

// RecordTemplateMetrics records or updates hourly template metrics.
func (s *Store) RecordTemplateMetrics(ctx context.Context, metrics *AnalyticsTemplateMetrics) error {
	hourStr := metrics.Hour.UTC().Format(time.RFC3339)

	_, err := s.stmtRecordTemplateMetrics.ExecContext(ctx,
		hourStr,
		metrics.TemplateName,
		metrics.ExtractionsTotal,
		metrics.ExtractionsSuccess,
		metrics.FieldCoverageAvg*float64(metrics.ExtractionsTotal),  // Convert avg back to sum
		metrics.ExtractionsTotal,                                    // coverage count = total for avg calculation
		int64(metrics.AvgExtractionTimeMs)*metrics.ExtractionsTotal, // Convert avg back to total
	)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to record template metrics", err)
	}
	return nil
}

// GetTemplateMetrics retrieves template metrics for a specific template and time range.
func (s *Store) GetTemplateMetrics(ctx context.Context, templateName string, start, end time.Time) ([]AnalyticsTemplateMetrics, error) {
	startStr := start.UTC().Format(time.RFC3339)
	endStr := end.UTC().Format(time.RFC3339)

	rows, err := s.stmtGetTemplateMetrics.QueryContext(ctx, templateName, startStr, endStr)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to query template metrics", err)
	}
	defer rows.Close()

	return s.scanTemplateMetrics(rows)
}

// GetAllTemplateMetrics retrieves all template metrics for a time range.
func (s *Store) GetAllTemplateMetrics(ctx context.Context, start, end time.Time) ([]AnalyticsTemplateMetrics, error) {
	startStr := start.UTC().Format(time.RFC3339)
	endStr := end.UTC().Format(time.RFC3339)

	rows, err := s.stmtGetAllTemplateMetrics.QueryContext(ctx, startStr, endStr)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to query all template metrics", err)
	}
	defer rows.Close()

	return s.scanTemplateMetrics(rows)
}

// scanTemplateMetrics scans template metrics rows into structs.
func (s *Store) scanTemplateMetrics(rows *sql.Rows) ([]AnalyticsTemplateMetrics, error) {
	var results []AnalyticsTemplateMetrics
	for rows.Next() {
		var m AnalyticsTemplateMetrics
		var hourStr string
		var fieldCoverageSum float64
		var fieldCoverageCount int64
		var totalExtractionTimeMs int64

		err := rows.Scan(
			&hourStr,
			&m.TemplateName,
			&m.ExtractionsTotal,
			&m.ExtractionsSuccess,
			&fieldCoverageSum,
			&fieldCoverageCount,
			&totalExtractionTimeMs,
		)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to scan template metrics row", err)
		}

		m.Hour, _ = time.Parse(time.RFC3339, hourStr)

		// Calculate averages
		if fieldCoverageCount > 0 {
			m.FieldCoverageAvg = fieldCoverageSum / float64(fieldCoverageCount)
		}
		if m.ExtractionsTotal > 0 {
			m.AvgExtractionTimeMs = float64(totalExtractionTimeMs) / float64(m.ExtractionsTotal)
		}

		results = append(results, m)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "error iterating template metrics rows", err)
	}

	return results, nil
}

// CreateABTest creates a new A/B test.
func (s *Store) CreateABTest(ctx context.Context, test *TemplateABTestRecord) error {
	startStr := test.StartTime.UTC().Format(time.RFC3339)
	var endStr *string
	if test.EndTime != nil {
		e := test.EndTime.UTC().Format(time.RFC3339)
		endStr = &e
	}
	createdAtStr := test.CreatedAt.UTC().Format(time.RFC3339)
	updatedAtStr := test.UpdatedAt.UTC().Format(time.RFC3339)

	_, err := s.stmtCreateABTest.ExecContext(ctx,
		test.ID,
		test.Name,
		test.Description,
		test.BaselineTemplate,
		test.VariantTemplate,
		test.AllocationJSON,
		startStr,
		endStr,
		test.Status,
		test.SuccessCriteriaJSON,
		test.MinSampleSize,
		test.ConfidenceLevel,
		test.Winner,
		createdAtStr,
		updatedAtStr,
	)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create AB test", err)
	}
	return nil
}

// GetABTest retrieves an A/B test by ID.
func (s *Store) GetABTest(ctx context.Context, id string) (*TemplateABTestRecord, error) {
	row := s.stmtGetABTest.QueryRowContext(ctx, id)

	var test TemplateABTestRecord
	var startStr, createdAtStr, updatedAtStr string
	var endStr *string

	err := row.Scan(
		&test.ID,
		&test.Name,
		&test.Description,
		&test.BaselineTemplate,
		&test.VariantTemplate,
		&test.AllocationJSON,
		&startStr,
		&endStr,
		&test.Status,
		&test.SuccessCriteriaJSON,
		&test.MinSampleSize,
		&test.ConfidenceLevel,
		&test.Winner,
		&createdAtStr,
		&updatedAtStr,
	)
	if err == sql.ErrNoRows {
		return nil, apperrors.NotFound("AB test not found")
	}
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to get AB test", err)
	}

	test.StartTime, _ = time.Parse(time.RFC3339, startStr)
	if endStr != nil {
		t, _ := time.Parse(time.RFC3339, *endStr)
		test.EndTime = &t
	}
	test.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	test.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	return &test, nil
}

// ListABTests retrieves A/B tests filtered by status (empty string for all).
func (s *Store) ListABTests(ctx context.Context, status string) ([]TemplateABTestRecord, error) {
	rows, err := s.stmtListABTests.QueryContext(ctx, status, status)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to list AB tests", err)
	}
	defer rows.Close()

	var results []TemplateABTestRecord
	for rows.Next() {
		var test TemplateABTestRecord
		var startStr, createdAtStr, updatedAtStr string
		var endStr *string

		err := rows.Scan(
			&test.ID,
			&test.Name,
			&test.Description,
			&test.BaselineTemplate,
			&test.VariantTemplate,
			&test.AllocationJSON,
			&startStr,
			&endStr,
			&test.Status,
			&test.SuccessCriteriaJSON,
			&test.MinSampleSize,
			&test.ConfidenceLevel,
			&test.Winner,
			&createdAtStr,
			&updatedAtStr,
		)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to scan AB test row", err)
		}

		test.StartTime, _ = time.Parse(time.RFC3339, startStr)
		if endStr != nil {
			t, _ := time.Parse(time.RFC3339, *endStr)
			test.EndTime = &t
		}
		test.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		test.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

		results = append(results, test)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "error iterating AB test rows", err)
	}

	return results, nil
}

// UpdateABTest updates an existing A/B test.
func (s *Store) UpdateABTest(ctx context.Context, test *TemplateABTestRecord) error {
	startStr := test.StartTime.UTC().Format(time.RFC3339)
	var endStr *string
	if test.EndTime != nil {
		e := test.EndTime.UTC().Format(time.RFC3339)
		endStr = &e
	}
	updatedAtStr := test.UpdatedAt.UTC().Format(time.RFC3339)

	_, err := s.stmtUpdateABTest.ExecContext(ctx,
		test.Name,
		test.Description,
		test.BaselineTemplate,
		test.VariantTemplate,
		test.AllocationJSON,
		startStr,
		endStr,
		test.Status,
		test.SuccessCriteriaJSON,
		test.MinSampleSize,
		test.ConfidenceLevel,
		test.Winner,
		updatedAtStr,
		test.ID,
	)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to update AB test", err)
	}
	return nil
}

// UpdateABTestStatus updates only the status of an A/B test.
func (s *Store) UpdateABTestStatus(ctx context.Context, id string, status string) error {
	updatedAtStr := time.Now().UTC().Format(time.RFC3339)

	_, err := s.stmtUpdateABTestStatus.ExecContext(ctx, status, updatedAtStr, id)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to update AB test status", err)
	}
	return nil
}

// DeleteABTest deletes an A/B test by ID.
func (s *Store) DeleteABTest(ctx context.Context, id string) error {
	_, err := s.stmtDeleteABTest.ExecContext(ctx, id)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to delete AB test", err)
	}
	return nil
}

// RecordExtraction records a single extraction event.
func (s *Store) RecordExtraction(ctx context.Context, record *TemplateExtractionRecord) error {
	timestampStr := record.Timestamp.UTC().Format(time.RFC3339)

	_, err := s.stmtRecordExtraction.ExecContext(ctx,
		record.ID,
		record.TestID,
		record.TemplateName,
		record.TargetURL,
		record.Success,
		record.FieldCoverage,
		record.ExtractionTimeMs,
		record.ValidationErrors,
		record.ExtractedFields,
		timestampStr,
	)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to record extraction", err)
	}
	return nil
}

// GetExtractionRecords retrieves extraction records filtered by test ID and/or template name.
func (s *Store) GetExtractionRecords(ctx context.Context, testID, templateName string, start, end time.Time) ([]TemplateExtractionRecord, error) {
	startStr := start.UTC().Format(time.RFC3339)
	endStr := end.UTC().Format(time.RFC3339)

	// Handle empty string filters
	testFilter := testID
	if testFilter == "" {
		testFilter = ""
	}
	templateFilter := templateName
	if templateFilter == "" {
		templateFilter = ""
	}

	rows, err := s.stmtGetExtractionRecords.QueryContext(ctx, testFilter, testFilter, templateFilter, templateFilter, startStr, endStr)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to get extraction records", err)
	}
	defer rows.Close()

	var results []TemplateExtractionRecord
	for rows.Next() {
		var record TemplateExtractionRecord
		var timestampStr string

		err := rows.Scan(
			&record.ID,
			&record.TestID,
			&record.TemplateName,
			&record.TargetURL,
			&record.Success,
			&record.FieldCoverage,
			&record.ExtractionTimeMs,
			&record.ValidationErrors,
			&record.ExtractedFields,
			&timestampStr,
		)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to scan extraction record row", err)
		}

		record.Timestamp, _ = time.Parse(time.RFC3339, timestampStr)
		results = append(results, record)
	}

	if err := rows.Err(); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "error iterating extraction record rows", err)
	}

	return results, nil
}

// closeAnalyticsStatements closes all analytics prepared statements.
func (s *Store) closeAnalyticsStatements() error {
	stmts := []*sql.Stmt{
		s.stmtRecordHourlyMetrics,
		s.stmtGetHourlyMetrics,
		s.stmtRecordHostMetrics,
		s.stmtGetHostMetrics,
		s.stmtGetTopHosts,
		s.stmtRecordDailyMetrics,
		s.stmtGetDailyMetrics,
		s.stmtRecordJobTrend,
		s.stmtGetJobTrends,
		s.stmtPurgeOldAnalytics,
		s.stmtRecordTemplateMetrics,
		s.stmtGetTemplateMetrics,
		s.stmtGetAllTemplateMetrics,
		s.stmtCreateABTest,
		s.stmtGetABTest,
		s.stmtListABTests,
		s.stmtUpdateABTest,
		s.stmtUpdateABTestStatus,
		s.stmtDeleteABTest,
		s.stmtRecordExtraction,
		s.stmtGetExtractionRecords,
	}

	for _, stmt := range stmts {
		if stmt != nil {
			if err := stmt.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}
