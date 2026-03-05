// Package store provides SQLite-backed persistent storage for analytics data.
//
// This file defines the types used for analytics data in the store package.
// These types are separate from the analytics package to avoid import cycles.
package store

import (
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// AnalyticsHourlyMetrics represents aggregated metrics for a single hour.
type AnalyticsHourlyMetrics struct {
	Hour              time.Time
	RequestsTotal     int64
	RequestsSuccess   int64
	RequestsFailed    int64
	AvgResponseTimeMs float64
	TotalResponseTime time.Duration
	JobsCompleted     int64
	JobsFailed        int64
	AvgJobDurationMs  float64
	TotalJobDuration  time.Duration
	FetcherHTTP       int64
	FetcherChromedp   int64
	FetcherPlaywright int64
	CreatedAt         time.Time
}

// AnalyticsHostMetrics represents aggregated metrics for a single host in an hour.
type AnalyticsHostMetrics struct {
	Hour              time.Time
	Host              string
	RequestsTotal     int64
	RequestsSuccess   int64
	RequestsFailed    int64
	AvgResponseTimeMs float64
	TotalResponseTime time.Duration
	RateLimitHits     int64
	LastRequest       int64 // Unix timestamp
}

// AnalyticsHostSummary represents aggregated host metrics over a time range.
type AnalyticsHostSummary struct {
	Host              string
	RequestsTotal     int64
	AvgResponseTimeMs float64
	SuccessRate       float64
	RateLimitHits     int64
}

// AnalyticsDailyMetrics represents aggregated metrics for a single day.
type AnalyticsDailyMetrics struct {
	Date              time.Time
	RequestsTotal     int64
	RequestsSuccess   int64
	RequestsFailed    int64
	AvgResponseTimeMs float64
	JobsCompleted     int64
	JobsFailed        int64
	AvgJobDurationMs  float64
	UniqueHosts       int
	CreatedAt         time.Time
}

// AnalyticsJobTrend represents job completion trends by kind and status.
type AnalyticsJobTrend struct {
	Date          time.Time
	JobKind       model.Kind
	Status        model.Status
	Count         int64
	AvgDurationMs float64
	TotalDuration time.Duration
}

// AnalyticsSummary provides a high-level overview for a time period.
type AnalyticsSummary struct {
	TotalRequests     int64
	TotalJobs         int64
	AvgSuccessRate    float64
	AvgResponseTimeMs float64
	UniqueHosts       int
}

// AnalyticsTemplateMetrics represents aggregated metrics for a single template in an hour.
type AnalyticsTemplateMetrics struct {
	Hour                time.Time
	TemplateName        string
	ExtractionsTotal    int64
	ExtractionsSuccess  int64
	FieldCoverageAvg    float64
	AvgExtractionTimeMs float64
}

// TemplateABTestRecord represents an A/B test configuration in storage.
type TemplateABTestRecord struct {
	ID                  string
	Name                string
	Description         string
	BaselineTemplate    string
	VariantTemplate     string
	AllocationJSON      string
	StartTime           time.Time
	EndTime             *time.Time
	Status              string
	SuccessCriteriaJSON string
	MinSampleSize       int
	ConfidenceLevel     float64
	Winner              *string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// TemplateExtractionRecord represents a single extraction event in storage.
type TemplateExtractionRecord struct {
	ID               string
	TestID           *string
	TemplateName     string
	TargetURL        string
	Success          bool
	FieldCoverage    float64
	ExtractionTimeMs int64
	ValidationErrors string // JSON array
	ExtractedFields  string // JSON array
	Timestamp        time.Time
}
