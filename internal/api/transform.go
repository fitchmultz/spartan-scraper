// Package api provides HTTP handlers for job result transformation endpoints.
// This file handles transformation preview using JMESPath and JSONata expressions.
package api

import (
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

// TransformPreviewRequest requests a transformation preview.
type TransformPreviewRequest struct {
	JobID      string `json:"jobId"`           // Job to preview against
	Expression string `json:"expression"`      // JMESPath or JSONata expression
	Language   string `json:"language"`        // "jmespath" or "jsonata"
	Limit      int    `json:"limit,omitempty"` // Max results to return (default 10)
}

// TransformPreviewResponse returns transformed results.
type TransformPreviewResponse struct {
	Results     []any  `json:"results"`         // Transformed data
	Error       string `json:"error,omitempty"` // Error message if failed
	ResultCount int    `json:"resultCount"`     // Number of results
}

// TransformValidateRequest requests validation of a transformation expression.
type TransformValidateRequest struct {
	Expression string `json:"expression"` // JMESPath or JSONata expression
	Language   string `json:"language"`   // "jmespath" or "jsonata"
}

// TransformValidateResponse returns validation result.
type TransformValidateResponse struct {
	Valid   bool   `json:"valid"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// handlePreviewTransform handles POST /v1/jobs/{id}/preview-transform
func (s *Server) handlePreviewTransform(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	id, err := requireResourceID(r, "jobs", "job id")
	if err != nil {
		writeError(w, r, err)
		return
	}

	var req TransformPreviewRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	// Validate language
	if req.Language != "jmespath" && req.Language != "jsonata" {
		writeError(w, r, apperrors.Validation("language must be 'jmespath' or 'jsonata'"))
		return
	}

	// Validate expression
	if req.Expression == "" {
		writeError(w, r, apperrors.Validation("expression is required"))
		return
	}

	// Set default limit
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}

	// Get job
	job, err := s.store.Get(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if err := s.validateJobResultPath(job, "job has no results"); err != nil {
		writeError(w, r, err)
		return
	}

	// Load results
	results, err := s.loadJobResults(job, req.Limit)
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Apply transformation
	transformedResults, transformErr := exporter.ApplyTransformation(results, req.Expression, req.Language)

	resp := TransformPreviewResponse{
		Results:     transformedResults,
		ResultCount: len(transformedResults),
	}

	if transformErr != nil {
		resp.Error = apperrors.SafeMessage(transformErr)
	}

	writeJSON(w, resp)
}

// handleValidateTransform handles POST /v1/transform/validate
func (s *Server) handleValidateTransform(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req TransformValidateRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	// Validate language
	if req.Language != "jmespath" && req.Language != "jsonata" {
		writeError(w, r, apperrors.Validation("language must be 'jmespath' or 'jsonata'"))
		return
	}

	// Validate expression
	if req.Expression == "" {
		writeError(w, r, apperrors.Validation("expression is required"))
		return
	}

	var validationErr error
	if req.Language == "jmespath" {
		validationErr = pipeline.CompileJMESPath(req.Expression)
	} else {
		validationErr = pipeline.CompileJSONata(req.Expression)
	}

	resp := TransformValidateResponse{
		Valid: validationErr == nil,
	}

	if validationErr != nil {
		resp.Error = apperrors.SafeMessage(validationErr)
		resp.Message = "Invalid expression"
	} else {
		resp.Message = "Expression is valid"
	}

	writeJSON(w, resp)
}

// loadJobResults loads results from a job file.
// Reads JSONL format (one JSON object per line) from job.ResultPath.
// Returns up to limit results.
func (s *Server) loadJobResults(job model.Job, limit int) ([]any, error) {
	if job.ResultPath == "" {
		return []any{}, nil
	}

	file, err := s.openJobResultFile(job, "job has no results")
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return decodeJobResultItems(file, limit)
}

// ApplyTransformation applies a transformation expression to data.
// This wrapper preserves existing api package call sites and tests.
func ApplyTransformation(data []any, expression, language string) ([]any, error) {
	return exporter.ApplyTransformation(data, expression, language)
}
