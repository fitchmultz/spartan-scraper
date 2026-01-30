// Package api provides HTTP handlers for scrape job endpoints.
// The scrape handler enqueues single-page scraping jobs with optional
// auth, extraction templates, and pipeline configurations.
package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
)

func (s *Server) handleScrape(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, r, apperrors.UnsupportedMediaType("content-type must be application/json"))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req ScrapeRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, r, apperrors.Validation(err.Error()))
		return
	}
	if req.URL == "" {
		writeError(w, r, apperrors.Validation("url is required"))
		return
	}
	opts := validate.JobValidationOpts{
		URL:         req.URL,
		Timeout:     req.TimeoutSeconds,
		AuthProfile: req.AuthProfile,
	}
	if err := validate.ValidateJob(opts, model.KindScrape); err != nil {
		writeError(w, r, err)
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
		writeError(w, r, err)
		return
	}

	// Determine HTTP method (default to GET if not specified)
	method := req.Method
	if method == "" {
		method = "GET"
	}

	requestID := contextRequestID(r.Context())
	spec := jobs.JobSpec{
		Kind:           model.KindScrape,
		URL:            req.URL,
		Method:         method,
		Body:           []byte(req.Body),
		ContentType:    req.ContentType,
		Headless:       req.Headless,
		UsePlaywright:  usePlaywright,
		Auth:           authOptions,
		TimeoutSeconds: timeout,
		Extract:        extractOpts,
		Pipeline:       pipelineOpts,
		Incremental:    incremental,
		RequestID:      requestID,
		Screenshot:     req.Screenshot,
		Device:         req.Device,
	}
	if req.Webhook != nil {
		spec.WebhookURL = req.Webhook.URL
		spec.WebhookEvents = req.Webhook.Events
		spec.WebhookSecret = req.Webhook.Secret
	}
	job, err := s.manager.CreateJob(r.Context(), spec)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if err := s.manager.Enqueue(job); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, model.SanitizeJob(job))
}
