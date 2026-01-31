// Package extract provides HTML content extraction using selectors, JSON-LD, and regex.
// This file defines types for template performance metrics and A/B testing.
//
// Template metrics track extraction success rates, field coverage, and timing
// to enable data-driven template optimization.
//
// A/B tests compare two templates (baseline vs variant) using statistical
// significance testing to determine which performs better.
package extract

import (
	"time"
)

// ABTestStatus represents the current state of an A/B test.
type ABTestStatus string

const (
	ABTestStatusPending   ABTestStatus = "pending"
	ABTestStatusRunning   ABTestStatus = "running"
	ABTestStatusPaused    ABTestStatus = "paused"
	ABTestStatusCompleted ABTestStatus = "completed"
)

// SuccessCriteria defines what constitutes a successful extraction for A/B testing.
type SuccessCriteria struct {
	Metric           string   `json:"metric"`                    // "success_rate", "field_coverage", "combined"
	MinImprovement   float64  `json:"min_improvement"`           // e.g., 0.05 for 5%
	RequiredFields   []string `json:"required_fields,omitempty"` // Fields that must be present
	MinFieldCoverage float64  `json:"min_field_coverage"`        // 0.0-1.0
}

// TemplateABTest represents an A/B test configuration comparing two templates.
type TemplateABTest struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	BaselineTemplate string          `json:"baseline_template"`
	VariantTemplate  string          `json:"variant_template"`
	Allocation       map[string]int  `json:"allocation"` // {"baseline": 50, "variant": 50}
	StartTime        time.Time       `json:"start_time"`
	EndTime          *time.Time      `json:"end_time,omitempty"`
	Status           ABTestStatus    `json:"status"`
	SuccessCriteria  SuccessCriteria `json:"success_criteria"`
	MinSampleSize    int             `json:"min_sample_size"`  // Per variant
	ConfidenceLevel  float64         `json:"confidence_level"` // e.g., 0.95
	Winner           *string         `json:"winner,omitempty"` // "baseline", "variant", or nil
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
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
	TestType           string     `json:"test_type"` // "chi_square", "t_test"
	PValue             float64    `json:"p_value"`
	IsSignificant      bool       `json:"is_significant"`
	ConfidenceInterval [2]float64 `json:"confidence_interval"` // [lower, upper]
	EffectSize         float64    `json:"effect_size"`         // Cohen's d or odds ratio
}

// TemplateComparison represents the full statistical comparison of two templates.
type TemplateComparison struct {
	TestID           string            `json:"test_id"`
	BaselineTemplate string            `json:"baseline_template"`
	VariantTemplate  string            `json:"variant_template"`
	BaselineMetrics  ComparisonMetrics `json:"baseline_metrics"`
	VariantMetrics   ComparisonMetrics `json:"variant_metrics"`
	StatisticalTest  StatisticalResult `json:"statistical_test"`
	Winner           *string           `json:"winner,omitempty"` // "baseline", "variant", or nil if inconclusive
	Recommendation   string            `json:"recommendation"`
	GeneratedAt      time.Time         `json:"generated_at"`
}

// TemplateExtractionRecord stores a single extraction event for detailed analysis.
type TemplateExtractionRecord struct {
	ID               string    `json:"id"`
	TestID           *string   `json:"test_id,omitempty"`
	TemplateName     string    `json:"template_name"`
	TargetURL        string    `json:"target_url"`
	Success          bool      `json:"success"`
	FieldCoverage    float64   `json:"field_coverage"`
	ExtractionTimeMs int64     `json:"extraction_time_ms"`
	ValidationErrors []string  `json:"validation_errors,omitempty"`
	ExtractedFields  []string  `json:"extracted_fields"`
	Timestamp        time.Time `json:"timestamp"`
}

// TemplateMetrics represents aggregated hourly metrics for a single template.
type TemplateMetrics struct {
	TemplateName          string    `json:"template_name"`
	Hour                  time.Time `json:"hour"`
	ExtractionsTotal      int64     `json:"extractions_total"`
	ExtractionsSuccess    int64     `json:"extractions_success"`
	FieldCoverageSum      float64   `json:"field_coverage_sum"` // For avg calculation
	FieldCoverageCount    int64     `json:"field_coverage_count"`
	TotalExtractionTimeMs int64     `json:"total_extraction_time_ms"`
}

// FieldCoverage calculates the average field coverage for this hourly bucket.
func (m *TemplateMetrics) FieldCoverage() float64 {
	if m.FieldCoverageCount == 0 {
		return 0
	}
	return m.FieldCoverageSum / float64(m.FieldCoverageCount)
}

// AvgExtractionTimeMs calculates the average extraction time for this hourly bucket.
func (m *TemplateMetrics) AvgExtractionTimeMs() int64 {
	if m.ExtractionsTotal == 0 {
		return 0
	}
	return m.TotalExtractionTimeMs / m.ExtractionsTotal
}

// SuccessRate calculates the success rate as a percentage.
func (m *TemplateMetrics) SuccessRate() float64 {
	if m.ExtractionsTotal == 0 {
		return 0
	}
	return float64(m.ExtractionsSuccess) * 100.0 / float64(m.ExtractionsTotal)
}

// IsValidAllocation checks if the allocation percentages sum to 100.
func IsValidAllocation(allocation map[string]int) bool {
	total := 0
	for _, v := range allocation {
		total += v
	}
	return total == 100
}

// GetEffectiveAllocation returns the allocation with defaults applied.
// Defaults to 50/50 if not specified or invalid.
func GetEffectiveAllocation(allocation map[string]int) map[string]int {
	if IsValidAllocation(allocation) {
		return allocation
	}
	return map[string]int{"baseline": 50, "variant": 50}
}
