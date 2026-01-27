// Package api provides HTTP handlers for crawl job endpoints.
// The crawl handler enqueues website crawling jobs with configurable
// depth, page limits, auth, extraction templates, and pipeline configurations.
package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/jobs"
	"spartan-scraper/internal/model"
	"spartan-scraper/internal/pipeline"
	"spartan-scraper/internal/validate"
)

func (s *Server) handleCrawl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeJSONError(w, http.StatusUnsupportedMediaType, "content-type must be application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req CrawlRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.URL == "" {
		writeJSONError(w, http.StatusBadRequest, "url is required")
		return
	}
	validator := validate.CrawlRequestValidator{
		URL:         req.URL,
		MaxDepth:    req.MaxDepth,
		MaxPages:    req.MaxPages,
		Timeout:     req.TimeoutSeconds,
		AuthProfile: req.AuthProfile,
	}
	if err := validator.Validate(); err != nil {
		writeError(w, err)
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
		writeError(w, err)
		return
	}
	spec := jobs.JobSpec{
		Kind:           model.KindCrawl,
		URL:            req.URL,
		MaxDepth:       req.MaxDepth,
		MaxPages:       req.MaxPages,
		Headless:       req.Headless,
		UsePlaywright:  usePlaywright,
		Auth:           authOptions,
		TimeoutSeconds: timeout,
		Extract:        extractOpts,
		Pipeline:       pipelineOpts,
		Incremental:    incremental,
	}
	job, err := s.manager.CreateJob(r.Context(), spec)
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.manager.Enqueue(job); err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, job)
}
