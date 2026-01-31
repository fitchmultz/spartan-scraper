// Package api implements the REST API server for Spartan Scraper.
// This file handles template performance metrics and comparison endpoints.
package api

import (
	"net/http"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

// handleTemplateMetrics handles GET /v1/template-metrics
// Query params: template (optional), from, to
func (s *Server) handleTemplateMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	// Parse query parameters
	templateName := r.URL.Query().Get("template")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	if fromStr == "" || toStr == "" {
		writeError(w, r, apperrors.Validation("from and to parameters are required"))
		return
	}

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		writeError(w, r, apperrors.Validation("invalid from time format, expected RFC3339"))
		return
	}

	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		writeError(w, r, apperrors.Validation("invalid to time format, expected RFC3339"))
		return
	}

	var metrics interface{}
	if templateName != "" {
		// Get metrics for specific template
		m, err := s.store.GetTemplateMetrics(r.Context(), templateName, from, to)
		if err != nil {
			writeError(w, r, err)
			return
		}
		metrics = m
	} else {
		// Get all template metrics
		m, err := s.store.GetAllTemplateMetrics(r.Context(), from, to)
		if err != nil {
			writeError(w, r, err)
			return
		}
		metrics = m
	}

	writeJSON(w, map[string]interface{}{
		"metrics": metrics,
	})
}

// handleTemplateComparison handles GET /v1/template-comparison
// Query params: template_a, template_b, from, to
func (s *Server) handleTemplateComparison(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	// Parse query parameters
	templateA := r.URL.Query().Get("template_a")
	templateB := r.URL.Query().Get("template_b")
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	if templateA == "" || templateB == "" {
		writeError(w, r, apperrors.Validation("template_a and template_b parameters are required"))
		return
	}

	if fromStr == "" || toStr == "" {
		writeError(w, r, apperrors.Validation("from and to parameters are required"))
		return
	}

	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		writeError(w, r, apperrors.Validation("invalid from time format, expected RFC3339"))
		return
	}

	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		writeError(w, r, apperrors.Validation("invalid to time format, expected RFC3339"))
		return
	}

	// Get extraction records for both templates
	recordsA, err := s.store.GetExtractionRecords(r.Context(), "", templateA, from, to)
	if err != nil {
		writeError(w, r, err)
		return
	}

	recordsB, err := s.store.GetExtractionRecords(r.Context(), "", templateB, from, to)
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Calculate metrics
	metricsA := calculateComparisonMetrics(recordsA)
	metricsB := calculateComparisonMetrics(recordsB)

	// Calculate statistical significance
	statisticalResult := calculateChiSquareTest(metricsA, metricsB, 0.95)

	// Determine winner
	var winner *string
	recommendation := "Insufficient data for conclusion"

	if statisticalResult.IsSignificant {
		if metricsB.SuccessRate > metricsA.SuccessRate {
			w := "template_b"
			winner = &w
			recommendation = "Template B shows statistically significant improvement"
		} else if metricsA.SuccessRate > metricsB.SuccessRate {
			w := "template_a"
			winner = &w
			recommendation = "Template A performs better"
		} else {
			recommendation = "No significant difference between templates"
		}
	} else {
		if metricsA.SampleSize < 100 || metricsB.SampleSize < 100 {
			recommendation = "Need more samples to reach statistical significance"
		} else {
			recommendation = "No statistically significant difference detected"
		}
	}

	comparison := map[string]interface{}{
		"template_a":         templateA,
		"template_b":         templateB,
		"template_a_metrics": metricsA,
		"template_b_metrics": metricsB,
		"statistical_test":   statisticalResult,
		"winner":             winner,
		"recommendation":     recommendation,
		"generated_at":       time.Now().UTC(),
	}

	writeJSON(w, comparison)
}

// ComparisonMetrics holds aggregated metrics for template comparison.
type ComparisonMetrics struct {
	SampleSize          int64   `json:"sample_size"`
	SuccessRate         float64 `json:"success_rate"`
	FieldCoverage       float64 `json:"field_coverage"`
	AvgExtractionTimeMs int64   `json:"avg_extraction_time_ms"`
}

// StatisticalResult contains the outcome of a statistical significance test.
type StatisticalResult struct {
	TestType           string     `json:"test_type"`
	PValue             float64    `json:"p_value"`
	IsSignificant      bool       `json:"is_significant"`
	ConfidenceInterval [2]float64 `json:"confidence_interval"`
	EffectSize         float64    `json:"effect_size"`
}

// calculateComparisonMetrics aggregates extraction records into comparison metrics.
func calculateComparisonMetrics(records []store.TemplateExtractionRecord) ComparisonMetrics {
	if len(records) == 0 {
		return ComparisonMetrics{}
	}

	var successCount int64
	var totalFieldCoverage float64
	var totalExtractionTime int64

	for _, record := range records {
		if record.Success {
			successCount++
		}
		totalFieldCoverage += record.FieldCoverage
		totalExtractionTime += record.ExtractionTimeMs
	}

	sampleSize := int64(len(records))
	return ComparisonMetrics{
		SampleSize:          sampleSize,
		SuccessRate:         float64(successCount) * 100.0 / float64(sampleSize),
		FieldCoverage:       totalFieldCoverage / float64(sampleSize),
		AvgExtractionTimeMs: totalExtractionTime / sampleSize,
	}
}

// calculateChiSquareTest performs chi-square test for success rates.
func calculateChiSquareTest(baseline, variant ComparisonMetrics, confidenceLevel float64) StatisticalResult {
	// Contingency table:
	//           | Success | Failure
	// Baseline  |    a    |    b
	// Variant   |    c    |    d

	baselineSuccess := int64(baseline.SuccessRate * float64(baseline.SampleSize) / 100.0)
	variantSuccess := int64(variant.SuccessRate * float64(variant.SampleSize) / 100.0)

	a := float64(baselineSuccess)
	b := float64(baseline.SampleSize - baselineSuccess)
	c := float64(variantSuccess)
	d := float64(variant.SampleSize - variantSuccess)

	n := a + b + c + d

	// Chi-square statistic with Yates' continuity correction
	var chi2 float64
	if a+b > 0 && c+d > 0 && a+c > 0 && b+d > 0 {
		chi2 = (n * pow(abs(a*d-b*c)-n/2, 2)) / ((a + b) * (c + d) * (a + c) * (b + d))
	}

	// p-value approximation
	pValue := chiSquarePValue(chi2, 1)

	// Significant if p < alpha (1 - confidence level)
	alpha := 1.0 - confidenceLevel
	isSignificant := pValue < alpha

	// Calculate confidence interval for difference in proportions
	p1 := a / (a + b)
	p2 := c / (c + d)
	diff := p2 - p1

	se := sqrt(p1*(1-p1)/(a+b) + p2*(1-p2)/(c+d))
	z := 1.96 // 95% confidence
	if confidenceLevel == 0.99 {
		z = 2.576
	}

	margin := z * se
	ciLower := diff - margin
	ciUpper := diff + margin

	return StatisticalResult{
		TestType:           "chi_square",
		PValue:             pValue,
		IsSignificant:      isSignificant,
		ConfidenceInterval: [2]float64{ciLower, ciUpper},
		EffectSize:         diff,
	}
}

// Helper math functions
func pow(x, y float64) float64 {
	result := 1.0
	for i := 0; i < int(y); i++ {
		result *= x
	}
	return result
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// Newton's method
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

// chiSquarePValue approximates the p-value from chi-square statistic.
func chiSquarePValue(chi2 float64, df int) float64 {
	if chi2 <= 0 {
		return 1.0
	}

	// For df = 1, use direct approximation
	if df == 1 {
		return exp(-chi2/2) / sqrt(3.14159265359*chi2)
	}

	// Simplified approximation for other degrees of freedom
	return exp(-chi2 / 2)
}

func exp(x float64) float64 {
	// Taylor series approximation for e^x
	result := 1.0
	term := 1.0
	for i := 1; i < 20; i++ {
		term *= x / float64(i)
		result += term
	}
	return result
}
