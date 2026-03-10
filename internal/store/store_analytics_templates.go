// Package store provides SQLite-backed persistent storage for analytics data.
//
// This file handles:
// - Template extraction metrics storage and retrieval
// - A/B testing configuration and management
// - Per-extraction record storage for statistical analysis
//
// This file does NOT handle:
// - Time-series metrics (hourly/daily aggregations) - see store_analytics_metrics.go
// - Database schema initialization - see store_analytics_core.go
// - Real-time metrics collection
//
// Invariants:
// - All timestamps stored as RFC3339 in UTC
// - A/B test records have foreign key constraints on test_id in extraction records
// - Template metrics are aggregated hourly
package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

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
	if err != nil {
		if isNoRowsError(err) {
			return nil, apperrors.NotFound("AB test not found")
		}
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
