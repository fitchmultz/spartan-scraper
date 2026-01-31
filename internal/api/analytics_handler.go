// Package api provides HTTP handlers for analytics endpoints.
//
// This file handles:
// - Historical metrics queries
// - Host performance analytics
// - Job trend analysis
// - Dashboard data aggregation
//
// This file does NOT handle:
// - Real-time metrics (metrics_handler.go handles that)
// - Data collection (analytics/collector.go handles that)
// - Report generation (analytics_reports.go handles that)
package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/analytics"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// handleAnalyticsMetrics returns historical metrics for a time range.
func (s *Server) handleAnalyticsMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	// Parse query parameters
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	granularity := r.URL.Query().Get("granularity")

	if startStr == "" || endStr == "" {
		writeError(w, r, apperrors.Validation("start and end parameters are required"))
		return
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		writeError(w, r, apperrors.Validation("invalid start time format, expected RFC3339"))
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		writeError(w, r, apperrors.Validation("invalid end time format, expected RFC3339"))
		return
	}

	// Default to hourly granularity
	g := analytics.GranularityHourly
	if granularity == "daily" {
		g = analytics.GranularityDaily
	}

	metrics, err := s.analyticsService.GetMetrics(r.Context(), start, end, g)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, map[string]interface{}{
		"granularity": g,
		"data":        metrics,
	})
}

// handleAnalyticsHosts returns per-host performance metrics.
func (s *Server) handleAnalyticsHosts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	// Parse query parameters
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	host := r.URL.Query().Get("host")

	if startStr == "" || endStr == "" {
		writeError(w, r, apperrors.Validation("start and end parameters are required"))
		return
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		writeError(w, r, apperrors.Validation("invalid start time format, expected RFC3339"))
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		writeError(w, r, apperrors.Validation("invalid end time format, expected RFC3339"))
		return
	}

	var result interface{}

	if host != "" {
		// Get metrics for specific host
		result, err = s.analyticsService.GetHostMetrics(r.Context(), host, start, end)
	} else {
		// Get top hosts
		limitStr := r.URL.Query().Get("limit")
		limit := 10
		if limitStr != "" {
			if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
				limit = l
			}
		}
		result, err = s.analyticsService.GetTopHosts(r.Context(), start, end, limit)
	}

	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, map[string]interface{}{
		"hosts": result,
	})
}

// handleAnalyticsTrends returns job success/failure trends.
func (s *Server) handleAnalyticsTrends(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	// Parse query parameters
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")

	if startStr == "" || endStr == "" {
		writeError(w, r, apperrors.Validation("start and end parameters are required"))
		return
	}

	start, err := time.Parse(time.RFC3339, startStr)
	if err != nil {
		writeError(w, r, apperrors.Validation("invalid start time format, expected RFC3339"))
		return
	}

	end, err := time.Parse(time.RFC3339, endStr)
	if err != nil {
		writeError(w, r, apperrors.Validation("invalid end time format, expected RFC3339"))
		return
	}

	trends, err := s.analyticsService.GetJobTrends(r.Context(), start, end)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, map[string]interface{}{
		"trends": trends,
	})
}

// handleAnalyticsDashboard returns pre-computed dashboard data.
func (s *Server) handleAnalyticsDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	// Parse period parameter
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "24h"
	}

	// Validate period
	validPeriods := map[string]bool{"24h": true, "7d": true, "30d": true, "90d": true}
	if !validPeriods[period] {
		writeError(w, r, apperrors.Validation("invalid period, expected 24h, 7d, 30d, or 90d"))
		return
	}

	dashboard, err := s.analyticsService.GetDashboardData(r.Context(), period)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, dashboard)
}
