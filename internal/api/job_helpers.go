// Package api provides shared job-construction helpers for Spartan Scraper's API.
//
// Purpose:
// - Centralize repeated job-spec defaults and batch creation behavior.
//
// Responsibilities:
// - Apply shared timeout, Playwright, auth, webhook, extraction, and device settings.
// - Validate repeated batch URL constraints.
// - Create and enqueue batch jobs with a single response-shaping path.
//
// Scope:
// - Shared helper logic for API handlers that submit scrape, crawl, and research jobs.
//
// Usage:
//   - Handlers build the kind-specific fields, then call applySingleJobDefaults or
//     applyBatchJobDefaults before creating or enqueuing jobs.
//
// Invariants/Assumptions:
// - Auth profile resolution should behave the same for single-job and batch endpoints.
// - Empty optional request sections resolve to zero-value options rather than nil-dependent branches.
package api

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
)

type jobRequestOptions struct {
	authURL          string
	authProfile      string
	auth             *fetch.AuthOptions
	extract          *extract.ExtractOptions
	pipeline         *pipeline.Options
	webhook          *WebhookConfig
	screenshot       *fetch.ScreenshotConfig
	device           *fetch.DeviceEmulation
	networkIntercept *fetch.NetworkInterceptConfig
	incremental      *bool
	playwright       *bool
	timeoutSeconds   int
	requestID        string
}

type singleJobSubmission[T any] struct {
	kind           model.Kind
	validate       func(T) error
	buildSpec      func(T) jobs.JobSpec
	requestOptions func(*http.Request, T) jobRequestOptions
}

type batchJobSubmission[T any] struct {
	kind       model.Kind
	validate   func(T) error
	buildSpecs func(*http.Request, T) ([]jobs.JobSpec, error)
}

func applyJobDefaultsWithConfig(cfg config.Config, defaultTimeoutSeconds int, defaultUsePlaywright bool, spec *jobs.JobSpec, opts jobRequestOptions, resolveAuth bool) error {
	spec.TimeoutSeconds = opts.timeoutSeconds
	if spec.TimeoutSeconds <= 0 {
		spec.TimeoutSeconds = defaultTimeoutSeconds
	}
	spec.UsePlaywright = valueOr(opts.playwright, defaultUsePlaywright)
	spec.AuthProfile = opts.authProfile
	spec.Extract = valueOr(opts.extract, extract.ExtractOptions{})
	spec.Pipeline = valueOr(opts.pipeline, pipeline.Options{})
	spec.Incremental = valueOr(opts.incremental, false)
	spec.RequestID = opts.requestID
	spec.Screenshot = opts.screenshot
	spec.Device = opts.device
	spec.NetworkIntercept = opts.networkIntercept
	applyWebhookConfig(spec, opts.webhook)

	if !resolveAuth {
		spec.Auth = valueOr(opts.auth, fetch.AuthOptions{})
		spec.Auth.NormalizeTransport()
		if err := spec.Auth.ValidateTransport(); err != nil {
			return err
		}
		return nil
	}

	authOptions, err := resolveAuthForRequest(cfg, opts.authURL, opts.authProfile, opts.auth)
	if err != nil {
		return err
	}
	spec.Auth = authOptions
	return nil
}

func (s *Server) applyJobDefaults(spec *jobs.JobSpec, opts jobRequestOptions, resolveAuth bool) error {
	return applyJobDefaultsWithConfig(s.cfg, s.manager.DefaultTimeoutSeconds(), s.manager.DefaultUsePlaywright(), spec, opts, resolveAuth)
}

func applyWebhookConfig(spec *jobs.JobSpec, webhook *WebhookConfig) {
	if webhook == nil {
		return
	}
	spec.WebhookURL = webhook.URL
	spec.WebhookEvents = webhook.Events
	spec.WebhookSecret = webhook.Secret
}

func validateBatchURLs(items []BatchJobRequest) error {
	for i, job := range items {
		if err := validate.ValidateURL(job.URL); err != nil {
			return apperrors.Validation(fmt.Sprintf("invalid URL at index %d: %v", i, err))
		}
	}
	return nil
}

func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method == method {
		return true
	}
	writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	return false
}

func decodeJSONRequest[T any](w http.ResponseWriter, r *http.Request) (T, bool) {
	var req T
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return req, false
	}
	return req, true
}

func handleSingleJobSubmission[T any](s *Server, w http.ResponseWriter, r *http.Request, submission singleJobSubmission[T]) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := decodeJSONRequest[T](w, r)
	if !ok {
		return
	}
	if err := submission.validate(req); err != nil {
		writeError(w, r, err)
		return
	}

	spec := submission.buildSpec(req)
	if err := s.applyJobDefaults(&spec, submission.requestOptions(r, req), true); err != nil {
		writeError(w, r, err)
		return
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

func handleBatchJobSubmission[T any](s *Server, w http.ResponseWriter, r *http.Request, submission batchJobSubmission[T]) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := decodeJSONRequest[T](w, r)
	if !ok {
		return
	}
	if err := submission.validate(req); err != nil {
		writeError(w, r, err)
		return
	}

	specs, err := submission.buildSpecs(r, req)
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp, err := s.createAndEnqueueBatch(r.Context(), submission.kind, specs)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeCreatedJSON(w, resp)
}

func (s *Server) createAndEnqueueBatch(ctx context.Context, kind model.Kind, specs []jobs.JobSpec) (BatchResponse, error) {
	batchID := jobs.GenerateBatchID()
	createdJobs, err := s.manager.CreateBatchJobs(ctx, kind, specs, batchID)
	if err != nil {
		return BatchResponse{}, err
	}
	if err := s.manager.EnqueueBatch(createdJobs); err != nil {
		return BatchResponse{}, err
	}

	return BatchResponse{
		ID:        batchID,
		Kind:      string(kind),
		Status:    string(model.BatchStatusPending),
		JobCount:  len(createdJobs),
		Jobs:      model.SanitizeJobs(createdJobs),
		CreatedAt: createdJobs[0].CreatedAt,
	}, nil
}
