// Package api provides HTTP handlers for crawl job endpoints.
// The crawl handler enqueues website crawling jobs with configurable
// depth, page limits, auth, extraction templates, and pipeline configurations.
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

func (s *Server) handleCrawl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeError(w, r, apperrors.UnsupportedMediaType("content-type must be application/json"))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req CrawlRequest
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
		MaxDepth:    req.MaxDepth,
		MaxPages:    req.MaxPages,
		Timeout:     req.TimeoutSeconds,
		AuthProfile: req.AuthProfile,
	}
	if err := validate.ValidateJob(opts, model.KindCrawl); err != nil {
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
	requestID := contextRequestID(r.Context())
	sitemapOnly := false
	if req.SitemapOnly != nil {
		sitemapOnly = *req.SitemapOnly
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
		RequestID:      requestID,
		SitemapURL:     req.SitemapURL,
		SitemapOnly:    sitemapOnly,
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
