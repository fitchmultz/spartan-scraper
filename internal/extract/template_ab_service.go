// Package extract provides HTML content extraction using selectors, JSON-LD, and regex.
// This file implements A/B testing service for template comparison.
//
// The service manages A/B test lifecycle, variant selection, and statistical
// significance testing to determine winning templates.
package extract

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"math"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/google/uuid"
)

// ABTestService manages A/B tests and variant selection.
type ABTestService struct {
	store       *store.Store
	registry    *TemplateRegistry
	collector   *TemplateMetricsCollector
	activeTests map[string]*TemplateABTest // cache of active tests
	mu          sync.RWMutex
}

// NewABTestService creates a new A/B test service.
func NewABTestService(store *store.Store, registry *TemplateRegistry, collector *TemplateMetricsCollector) *ABTestService {
	return &ABTestService{
		store:       store,
		registry:    registry,
		collector:   collector,
		activeTests: make(map[string]*TemplateABTest),
	}
}

// CreateTest creates a new A/B test.
func (s *ABTestService) CreateTest(ctx context.Context, test *TemplateABTest) (*TemplateABTest, error) {
	// Validate templates exist
	if _, err := s.registry.GetTemplate(test.BaselineTemplate); err != nil {
		return nil, apperrors.Validation("baseline template not found: " + test.BaselineTemplate)
	}
	if _, err := s.registry.GetTemplate(test.VariantTemplate); err != nil {
		return nil, apperrors.Validation("variant template not found: " + test.VariantTemplate)
	}

	// Validate allocation
	if !IsValidAllocation(test.Allocation) {
		test.Allocation = GetEffectiveAllocation(test.Allocation)
	}

	// Set defaults
	if test.ID == "" {
		test.ID = uuid.New().String()
	}
	if test.MinSampleSize == 0 {
		test.MinSampleSize = 100
	}
	if test.ConfidenceLevel == 0 {
		test.ConfidenceLevel = 0.95
	}
	if test.Status == "" {
		test.Status = ABTestStatusPending
	}

	now := time.Now().UTC()
	test.CreatedAt = now
	test.UpdatedAt = now

	// Store in database
	allocationJSON, _ := json.Marshal(test.Allocation)
	successCriteriaJSON, _ := json.Marshal(test.SuccessCriteria)

	record := &store.TemplateABTestRecord{
		ID:                  test.ID,
		Name:                test.Name,
		Description:         test.Description,
		BaselineTemplate:    test.BaselineTemplate,
		VariantTemplate:     test.VariantTemplate,
		AllocationJSON:      string(allocationJSON),
		StartTime:           test.StartTime,
		Status:              string(test.Status),
		SuccessCriteriaJSON: string(successCriteriaJSON),
		MinSampleSize:       test.MinSampleSize,
		ConfidenceLevel:     test.ConfidenceLevel,
		CreatedAt:           test.CreatedAt,
		UpdatedAt:           test.UpdatedAt,
	}

	if test.EndTime != nil {
		record.EndTime = test.EndTime
	}

	if err := s.store.CreateABTest(ctx, record); err != nil {
		return nil, err
	}

	return test, nil
}

// StartTest begins an A/B test.
func (s *ABTestService) StartTest(ctx context.Context, testID string) error {
	test, err := s.store.GetABTest(ctx, testID)
	if err != nil {
		return err
	}

	if test.Status != string(ABTestStatusPending) && test.Status != string(ABTestStatusPaused) {
		return apperrors.Validation("test can only be started from pending or paused state")
	}

	now := time.Now().UTC()
	if err := s.store.UpdateABTestStatus(ctx, testID, string(ABTestStatusRunning)); err != nil {
		return err
	}

	// Update start time if not already set
	if test.StartTime.IsZero() || test.StartTime.After(now) {
		record, _ := s.store.GetABTest(ctx, testID)
		record.StartTime = now
		record.UpdatedAt = now
		s.store.UpdateABTest(ctx, record)
	}

	// Add to active tests cache
	s.mu.Lock()
	s.activeTests[testID] = s.recordToTest(test)
	s.mu.Unlock()

	return nil
}

// StopTest ends an A/B test.
func (s *ABTestService) StopTest(ctx context.Context, testID string) error {
	test, err := s.store.GetABTest(ctx, testID)
	if err != nil {
		return err
	}

	if test.Status != string(ABTestStatusRunning) {
		return apperrors.Validation("test can only be stopped from running state")
	}

	now := time.Now().UTC()

	// Update record
	record, _ := s.store.GetABTest(ctx, testID)
	record.Status = string(ABTestStatusCompleted)
	record.EndTime = &now
	record.UpdatedAt = now

	if err := s.store.UpdateABTest(ctx, record); err != nil {
		return err
	}

	// Remove from active tests cache
	s.mu.Lock()
	delete(s.activeTests, testID)
	s.mu.Unlock()

	return nil
}

// SelectTemplate chooses which template to use for a given URL.
// Returns: template name, test ID (if part of test), isVariant bool, error
func (s *ABTestService) SelectTemplate(ctx context.Context, targetURL string, requestedTemplate string) (string, *string, bool, error) {
	s.mu.RLock()
	activeTests := make(map[string]*TemplateABTest, len(s.activeTests))
	for k, v := range s.activeTests {
		activeTests[k] = v
	}
	s.mu.RUnlock()

	// Find an active test that includes the requested template
	for _, test := range activeTests {
		if test.BaselineTemplate == requestedTemplate || test.VariantTemplate == requestedTemplate {
			// Use hash-based allocation for consistent variant assignment
			hash := s.hashURL(targetURL, test.ID)
			allocation := GetEffectiveAllocation(test.Allocation)

			baselineWeight := allocation["baseline"]
			variantWeight := allocation["variant"]

			// Normalize weights
			totalWeight := baselineWeight + variantWeight
			if totalWeight == 0 {
				totalWeight = 100
			}

			// Determine variant based on hash
			hashValue := hash % uint32(totalWeight)
			isVariant := hashValue >= uint32(baselineWeight)

			if isVariant {
				return test.VariantTemplate, &test.ID, true, nil
			}
			return test.BaselineTemplate, &test.ID, false, nil
		}
	}

	// No active test found, use requested template
	return requestedTemplate, nil, false, nil
}

// GetTestResults generates comparison report with statistical significance.
func (s *ABTestService) GetTestResults(ctx context.Context, testID string) (*TemplateComparison, error) {
	test, err := s.store.GetABTest(ctx, testID)
	if err != nil {
		return nil, err
	}

	// Get extraction records for both templates
	startTime := test.StartTime
	endTime := time.Now().UTC()
	if test.EndTime != nil {
		endTime = *test.EndTime
	}

	baselineRecords, err := s.store.GetExtractionRecords(ctx, testID, test.BaselineTemplate, startTime, endTime)
	if err != nil {
		return nil, err
	}

	variantRecords, err := s.store.GetExtractionRecords(ctx, testID, test.VariantTemplate, startTime, endTime)
	if err != nil {
		return nil, err
	}

	// Calculate metrics
	baselineMetrics := s.calculateMetrics(baselineRecords)
	variantMetrics := s.calculateMetrics(variantRecords)

	// Perform statistical test
	statisticalResult := s.calculateStatisticalSignificance(
		baselineMetrics,
		variantMetrics,
		test.ConfidenceLevel,
	)

	// Determine winner
	var winner *string
	recommendation := "Insufficient data for conclusion"

	if statisticalResult.IsSignificant {
		if variantMetrics.SuccessRate > baselineMetrics.SuccessRate {
			w := "variant"
			winner = &w
			recommendation = "Variant shows statistically significant improvement"
		} else if baselineMetrics.SuccessRate > variantMetrics.SuccessRate {
			w := "baseline"
			winner = &w
			recommendation = "Baseline performs better, no change recommended"
		} else {
			recommendation = "No significant difference between templates"
		}
	} else {
		if baselineMetrics.SampleSize < int64(test.MinSampleSize) ||
			variantMetrics.SampleSize < int64(test.MinSampleSize) {
			recommendation = "Need more samples to reach statistical significance"
		} else {
			recommendation = "No statistically significant difference detected"
		}
	}

	return &TemplateComparison{
		TestID:           testID,
		BaselineTemplate: test.BaselineTemplate,
		VariantTemplate:  test.VariantTemplate,
		BaselineMetrics:  baselineMetrics,
		VariantMetrics:   variantMetrics,
		StatisticalTest:  statisticalResult,
		Winner:           winner,
		Recommendation:   recommendation,
		GeneratedAt:      time.Now().UTC(),
	}, nil
}

// ListTests returns all tests filtered by status.
func (s *ABTestService) ListTests(ctx context.Context, status string) ([]TemplateABTest, error) {
	records, err := s.store.ListABTests(ctx, status)
	if err != nil {
		return nil, err
	}

	tests := make([]TemplateABTest, len(records))
	for i, record := range records {
		tests[i] = *s.recordToTest(&record)
	}

	return tests, nil
}

// AutoSelectWinner automatically selects winner if significance threshold met.
func (s *ABTestService) AutoSelectWinner(ctx context.Context, testID string) (*TemplateComparison, error) {
	comparison, err := s.GetTestResults(ctx, testID)
	if err != nil {
		return nil, err
	}

	if !comparison.StatisticalTest.IsSignificant {
		return comparison, nil
	}

	// Update test with winner
	record, err := s.store.GetABTest(ctx, testID)
	if err != nil {
		return nil, err
	}

	record.Winner = comparison.Winner
	record.Status = string(ABTestStatusCompleted)
	now := time.Now().UTC()
	record.EndTime = &now
	record.UpdatedAt = now

	if err := s.store.UpdateABTest(ctx, record); err != nil {
		return nil, err
	}

	// Remove from active tests
	s.mu.Lock()
	delete(s.activeTests, testID)
	s.mu.Unlock()

	return comparison, nil
}

// LoadActiveTests loads all running tests into the cache.
// Should be called on service startup.
func (s *ABTestService) LoadActiveTests(ctx context.Context) error {
	records, err := s.store.ListABTests(ctx, string(ABTestStatusRunning))
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.activeTests = make(map[string]*TemplateABTest)
	for _, record := range records {
		test := s.recordToTest(&record)
		s.activeTests[test.ID] = test
	}

	return nil
}

// calculateMetrics aggregates extraction records into comparison metrics.
func (s *ABTestService) calculateMetrics(records []store.TemplateExtractionRecord) ComparisonMetrics {
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

// calculateStatisticalSignificance performs chi-square test for success rates.
func (s *ABTestService) calculateStatisticalSignificance(
	baseline, variant ComparisonMetrics,
	confidenceLevel float64,
) StatisticalResult {
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
		chi2 = (n * math.Pow(math.Abs(a*d-b*c)-n/2, 2)) / ((a + b) * (c + d) * (a + c) * (b + d))
	}

	// Degrees of freedom = 1 for 2x2 table
	// p-value from chi-square distribution (simplified approximation)
	pValue := chiSquarePValue(chi2, 1)

	// Significant if p < alpha (1 - confidence level)
	alpha := 1.0 - confidenceLevel
	isSignificant := pValue < alpha

	// Calculate confidence interval for difference in proportions
	p1 := a / (a + b)
	p2 := c / (c + d)
	diff := p2 - p1

	se := math.Sqrt(p1*(1-p1)/(a+b) + p2*(1-p2)/(c+d))
	z := 1.96 // 95% confidence
	if confidenceLevel == 0.99 {
		z = 2.576
	}

	margin := z * se
	ciLower := diff - margin
	ciUpper := diff + margin

	// Effect size (Cohen's h for proportions)
	var effectSize float64
	if p1 > 0 && p1 < 1 && p2 > 0 && p2 < 1 {
		effectSize = 2 * (math.Asin(math.Sqrt(p2)) - math.Asin(math.Sqrt(p1)))
	}

	return StatisticalResult{
		TestType:           "chi_square",
		PValue:             pValue,
		IsSignificant:      isSignificant,
		ConfidenceInterval: [2]float64{ciLower, ciUpper},
		EffectSize:         effectSize,
	}
}

// chiSquarePValue approximates the p-value from chi-square statistic.
// Uses a simplified approximation of the chi-square CDF.
func chiSquarePValue(chi2 float64, df int) float64 {
	// Wilson-Hilferty approximation for chi-square CDF
	if chi2 <= 0 {
		return 1.0
	}

	// For df = 1, use direct approximation
	if df == 1 {
		// Approximation of the upper tail probability
		return math.Exp(-chi2/2) / math.Sqrt(math.Pi*chi2)
	}

	// General case using incomplete gamma function approximation
	// This is a simplified version - in production, use a proper stats library
	x := chi2 / 2.0
	k := float64(df) / 2.0

	// Approximation using regularized gamma
	lgamma, _ := math.Lgamma(k)
	return math.Exp(-x + (k-1)*math.Log(x) - lgamma)
}

// hashURL deterministically hashes URL for consistent variant assignment.
func (s *ABTestService) hashURL(url string, salt string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(url + salt))
	return h.Sum32()
}

// recordToTest converts a store record to a TemplateABTest.
func (s *ABTestService) recordToTest(record *store.TemplateABTestRecord) *TemplateABTest {
	var allocation map[string]int
	json.Unmarshal([]byte(record.AllocationJSON), &allocation)

	var successCriteria SuccessCriteria
	json.Unmarshal([]byte(record.SuccessCriteriaJSON), &successCriteria)

	return &TemplateABTest{
		ID:               record.ID,
		Name:             record.Name,
		Description:      record.Description,
		BaselineTemplate: record.BaselineTemplate,
		VariantTemplate:  record.VariantTemplate,
		Allocation:       allocation,
		StartTime:        record.StartTime,
		EndTime:          record.EndTime,
		Status:           ABTestStatus(record.Status),
		SuccessCriteria:  successCriteria,
		MinSampleSize:    record.MinSampleSize,
		ConfidenceLevel:  record.ConfidenceLevel,
		Winner:           record.Winner,
		CreatedAt:        record.CreatedAt,
		UpdatedAt:        record.UpdatedAt,
	}
}

// GetTest retrieves a single A/B test by ID.
func (s *ABTestService) GetTest(ctx context.Context, testID string) (*TemplateABTest, error) {
	record, err := s.store.GetABTest(ctx, testID)
	if err != nil {
		return nil, err
	}
	return s.recordToTest(record), nil
}

// DeleteTest deletes an A/B test.
func (s *ABTestService) DeleteTest(ctx context.Context, testID string) error {
	// Remove from active tests cache if present
	s.mu.Lock()
	delete(s.activeTests, testID)
	s.mu.Unlock()

	return s.store.DeleteABTest(ctx, testID)
}

// UpdateTest updates an existing A/B test.
func (s *ABTestService) UpdateTest(ctx context.Context, test *TemplateABTest) error {
	test.UpdatedAt = time.Now().UTC()

	allocationJSON, _ := json.Marshal(test.Allocation)
	successCriteriaJSON, _ := json.Marshal(test.SuccessCriteria)

	record := &store.TemplateABTestRecord{
		ID:                  test.ID,
		Name:                test.Name,
		Description:         test.Description,
		BaselineTemplate:    test.BaselineTemplate,
		VariantTemplate:     test.VariantTemplate,
		AllocationJSON:      string(allocationJSON),
		StartTime:           test.StartTime,
		Status:              string(test.Status),
		SuccessCriteriaJSON: string(successCriteriaJSON),
		MinSampleSize:       test.MinSampleSize,
		ConfidenceLevel:     test.ConfidenceLevel,
		Winner:              test.Winner,
		CreatedAt:           test.CreatedAt,
		UpdatedAt:           test.UpdatedAt,
	}

	if test.EndTime != nil {
		record.EndTime = test.EndTime
	}

	return s.store.UpdateABTest(ctx, record)
}
