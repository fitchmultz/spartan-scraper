// Package api provides HTTP handlers for scrape job endpoints.
// The scrape handler enqueues single-page scraping jobs with optional
// auth, extraction templates, and pipeline configurations.
package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/pipeline"
	"spartan-scraper/internal/validate"
)

func (s *Server) handleScrape(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeJSONError(w, http.StatusUnsupportedMediaType, "content-type must be application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req ScrapeRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if req.URL == "" {
		writeJSONError(w, http.StatusBadRequest, "url is required")
		return
	}
	validator := validate.ScrapeRequestValidator{
		URL:         req.URL,
		Timeout:     req.TimeoutSeconds,
		AuthProfile: req.AuthProfile,
	}
	if err := validator.Validate(); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	incremental := false
	if req.Incremental != nil {
		incremental = *req.Incremental
	}

	extractOpts := extract.ExtractOptions{}
	if req.Extract != nil {
		extractOpts = *req.Extract
	}

	pipelineOpts := pipeline.Options{}
	if req.Pipeline != nil {
		pipelineOpts = *req.Pipeline
	}

	timeout := req.TimeoutSeconds
	if timeout <= 0 {
		timeout = s.manager.DefaultTimeoutSeconds()
	}
	usePlaywright := s.manager.DefaultUsePlaywright()
	if req.Playwright != nil {
		usePlaywright = *req.Playwright
	}

	authOptions, err := resolveAuthForRequest(s.cfg, req.URL, req.AuthProfile, req.Auth)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	job, err := s.manager.CreateScrapeJob(r.Context(), req.URL, req.Headless, usePlaywright, authOptions, timeout, extractOpts, pipelineOpts, incremental)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.manager.Enqueue(job); err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, "failed to enqueue job: "+err.Error())
		return
	}

	writeJSON(w, job)
}
