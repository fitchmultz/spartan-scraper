// Package store provides SQLite-backed persistent storage for analytics data.
//
// This file handles:
// - Analytics database schema initialization
// - Prepared statement management (creation and cleanup)
// - Core utility functions for analytics operations
//
// This file does NOT handle:
// - Business logic for recording or querying analytics data
// - Time-series aggregations or rollups
//
// Invariants:
// - All statements are prepared during store initialization
// - All statements are closed when the store is closed
// - Schema creation is idempotent (uses IF NOT EXISTS)
package store

import (
	"database/sql"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
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

	return nil
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
