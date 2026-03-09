// Package api implements the REST API server for Spartan Scraper.
// This file handles A/B testing endpoints for template comparison.
package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

// handleABTests handles requests to /v1/template-ab-tests
func (s *Server) handleABTests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListABTests(w, r)
	case http.MethodPost:
		s.handleCreateABTest(w, r)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

// handleABTest handles requests to /v1/template-ab-tests/{id}
func (s *Server) handleABTest(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/template-ab-tests/")
	parts := strings.Split(path, "/")
	testID := parts[0]

	if testID == "" {
		writeError(w, r, apperrors.Validation("test ID is required"))
		return
	}

	// Check for sub-paths
	if len(parts) > 1 {
		switch parts[1] {
		case "start":
			if r.Method == http.MethodPost {
				s.handleStartABTest(w, r, testID)
				return
			}
		case "stop":
			if r.Method == http.MethodPost {
				s.handleStopABTest(w, r, testID)
				return
			}
		case "results":
			if r.Method == http.MethodGet {
				s.handleGetABTestResults(w, r, testID)
				return
			}
		case "auto-select":
			if r.Method == http.MethodPost {
				s.handleAutoSelectWinner(w, r, testID)
				return
			}
		}
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetABTest(w, r, testID)
	case http.MethodPatch:
		s.handleUpdateABTest(w, r, testID)
	case http.MethodDelete:
		s.handleDeleteABTest(w, r, testID)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

// handleListABTests handles GET /v1/template-ab-tests
func (s *Server) handleListABTests(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

	records, err := s.store.ListABTests(r.Context(), status)
	if err != nil {
		writeError(w, r, err)
		return
	}

	tests := make([]*extract.TemplateABTest, 0, len(records))
	for i := range records {
		tests = append(tests, templateABRecordToResponse(&records[i]))
	}

	writeJSON(w, map[string]interface{}{
		"tests": tests,
	})
}

// handleCreateABTest handles POST /v1/template-ab-tests
func (s *Server) handleCreateABTest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name             string                  `json:"name"`
		Description      string                  `json:"description"`
		BaselineTemplate string                  `json:"baseline_template"`
		VariantTemplate  string                  `json:"variant_template"`
		Allocation       map[string]int          `json:"allocation"`
		SuccessCriteria  extract.SuccessCriteria `json:"success_criteria"`
		MinSampleSize    int                     `json:"min_sample_size"`
		ConfidenceLevel  float64                 `json:"confidence_level"`
	}

	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	// Validate required fields
	if req.Name == "" {
		writeError(w, r, apperrors.Validation("name is required"))
		return
	}
	if req.BaselineTemplate == "" || req.VariantTemplate == "" {
		writeError(w, r, apperrors.Validation("baseline_template and variant_template are required"))
		return
	}

	test := &extract.TemplateABTest{
		Name:             req.Name,
		Description:      req.Description,
		BaselineTemplate: req.BaselineTemplate,
		VariantTemplate:  req.VariantTemplate,
		Allocation:       req.Allocation,
		Status:           extract.ABTestStatusPending,
		SuccessCriteria:  req.SuccessCriteria,
		MinSampleSize:    req.MinSampleSize,
		ConfidenceLevel:  req.ConfidenceLevel,
		StartTime:        time.Now().UTC(),
	}

	// Set defaults
	if test.MinSampleSize == 0 {
		test.MinSampleSize = 100
	}
	if test.ConfidenceLevel == 0 {
		test.ConfidenceLevel = 0.95
	}
	if !extract.IsValidAllocation(test.Allocation) {
		test.Allocation = map[string]int{"baseline": 50, "variant": 50}
	}

	// Create test using AB test service
	// Note: In a real implementation, we'd inject the ABTestService into the Server
	// For now, we'll create it directly
	registry, _ := extract.LoadTemplateRegistry(s.cfg.DataDir)
	collector := extract.NewTemplateMetricsCollector(s.store)
	service := extract.NewABTestService(s.store, registry, collector)

	createdTest, err := service.CreateTest(r.Context(), test)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeCreatedJSON(w, createdTest)
}

// handleGetABTest handles GET /v1/template-ab-tests/{id}
func (s *Server) handleGetABTest(w http.ResponseWriter, r *http.Request, testID string) {
	record, err := s.store.GetABTest(r.Context(), testID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, templateABRecordToResponse(record))
}

// handleUpdateABTest handles PATCH /v1/template-ab-tests/{id}
func (s *Server) handleUpdateABTest(w http.ResponseWriter, r *http.Request, testID string) {
	// Get existing test
	record, err := s.store.GetABTest(r.Context(), testID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Can only update pending or paused tests
	if record.Status != string(extract.ABTestStatusPending) && record.Status != string(extract.ABTestStatusPaused) {
		writeError(w, r, apperrors.Validation("can only update pending or paused tests"))
		return
	}

	var req struct {
		Name            string                   `json:"name,omitempty"`
		Description     string                   `json:"description,omitempty"`
		Allocation      map[string]int           `json:"allocation,omitempty"`
		SuccessCriteria *extract.SuccessCriteria `json:"success_criteria,omitempty"`
		MinSampleSize   *int                     `json:"min_sample_size,omitempty"`
		ConfidenceLevel *float64                 `json:"confidence_level,omitempty"`
	}

	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	// Update fields
	if req.Name != "" {
		record.Name = req.Name
	}
	if req.Description != "" {
		record.Description = req.Description
	}
	if req.Allocation != nil && extract.IsValidAllocation(req.Allocation) {
		allocationJSON, _ := json.Marshal(req.Allocation)
		record.AllocationJSON = string(allocationJSON)
	}
	if req.SuccessCriteria != nil {
		successCriteriaJSON, _ := json.Marshal(req.SuccessCriteria)
		record.SuccessCriteriaJSON = string(successCriteriaJSON)
	}
	if req.MinSampleSize != nil {
		record.MinSampleSize = *req.MinSampleSize
	}
	if req.ConfidenceLevel != nil {
		record.ConfidenceLevel = *req.ConfidenceLevel
	}

	record.UpdatedAt = time.Now().UTC()

	if err := s.store.UpdateABTest(r.Context(), record); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, templateABRecordToResponse(record))
}

// handleDeleteABTest handles DELETE /v1/template-ab-tests/{id}
func (s *Server) handleDeleteABTest(w http.ResponseWriter, r *http.Request, testID string) {
	if err := s.store.DeleteABTest(r.Context(), testID); err != nil {
		writeError(w, r, err)
		return
	}

	writeNoContent(w)
}

// handleStartABTest handles POST /v1/template-ab-tests/{id}/start
func (s *Server) handleStartABTest(w http.ResponseWriter, r *http.Request, testID string) {
	registry, _ := extract.LoadTemplateRegistry(s.cfg.DataDir)
	collector := extract.NewTemplateMetricsCollector(s.store)
	service := extract.NewABTestService(s.store, registry, collector)

	if err := service.StartTest(r.Context(), testID); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, map[string]interface{}{
		"status":  "started",
		"test_id": testID,
	})
}

// handleStopABTest handles POST /v1/template-ab-tests/{id}/stop
func (s *Server) handleStopABTest(w http.ResponseWriter, r *http.Request, testID string) {
	registry, _ := extract.LoadTemplateRegistry(s.cfg.DataDir)
	collector := extract.NewTemplateMetricsCollector(s.store)
	service := extract.NewABTestService(s.store, registry, collector)

	if err := service.StopTest(r.Context(), testID); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, map[string]interface{}{
		"status":  "stopped",
		"test_id": testID,
	})
}

// handleGetABTestResults handles GET /v1/template-ab-tests/{id}/results
func (s *Server) handleGetABTestResults(w http.ResponseWriter, r *http.Request, testID string) {
	registry, _ := extract.LoadTemplateRegistry(s.cfg.DataDir)
	collector := extract.NewTemplateMetricsCollector(s.store)
	service := extract.NewABTestService(s.store, registry, collector)

	comparison, err := service.GetTestResults(r.Context(), testID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, comparison)
}

// handleAutoSelectWinner handles POST /v1/template-ab-tests/{id}/auto-select
func (s *Server) handleAutoSelectWinner(w http.ResponseWriter, r *http.Request, testID string) {
	registry, _ := extract.LoadTemplateRegistry(s.cfg.DataDir)
	collector := extract.NewTemplateMetricsCollector(s.store)
	service := extract.NewABTestService(s.store, registry, collector)

	comparison, err := service.AutoSelectWinner(r.Context(), testID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, comparison)
}

func templateABRecordToResponse(record *store.TemplateABTestRecord) *extract.TemplateABTest {
	var allocation map[string]int
	if record.AllocationJSON != "" {
		_ = json.Unmarshal([]byte(record.AllocationJSON), &allocation)
	}

	var successCriteria extract.SuccessCriteria
	if record.SuccessCriteriaJSON != "" {
		_ = json.Unmarshal([]byte(record.SuccessCriteriaJSON), &successCriteria)
	}

	return &extract.TemplateABTest{
		ID:               record.ID,
		Name:             record.Name,
		Description:      record.Description,
		BaselineTemplate: record.BaselineTemplate,
		VariantTemplate:  record.VariantTemplate,
		Allocation:       allocation,
		StartTime:        record.StartTime,
		EndTime:          record.EndTime,
		Status:           extract.ABTestStatus(record.Status),
		SuccessCriteria:  successCriteria,
		MinSampleSize:    record.MinSampleSize,
		ConfidenceLevel:  record.ConfidenceLevel,
		Winner:           record.Winner,
		CreatedAt:        record.CreatedAt,
		UpdatedAt:        record.UpdatedAt,
	}
}
