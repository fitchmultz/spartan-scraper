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

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
)

type jobRequestOptions struct {
	authURL        string
	authProfile    string
	auth           *fetch.AuthOptions
	extract        *extract.ExtractOptions
	pipeline       *pipeline.Options
	webhook        *WebhookConfig
	screenshot     *fetch.ScreenshotConfig
	device         *fetch.DeviceEmulation
	incremental    *bool
	playwright     *bool
	timeoutSeconds int
	requestID      string
}

func (s *Server) applySingleJobDefaults(spec *jobs.JobSpec, opts jobRequestOptions) error {
	spec.TimeoutSeconds = opts.timeoutSeconds
	if spec.TimeoutSeconds <= 0 {
		spec.TimeoutSeconds = s.manager.DefaultTimeoutSeconds()
	}
	spec.UsePlaywright = valueOr(opts.playwright, s.manager.DefaultUsePlaywright())
	spec.Extract = valueOr(opts.extract, extract.ExtractOptions{})
	spec.Pipeline = valueOr(opts.pipeline, pipeline.Options{})
	spec.Incremental = valueOr(opts.incremental, false)
	spec.RequestID = opts.requestID
	spec.Screenshot = opts.screenshot
	spec.Device = opts.device
	applyWebhookConfig(spec, opts.webhook)

	authOptions, err := resolveAuthForRequest(s.cfg, opts.authURL, opts.authProfile, opts.auth)
	if err != nil {
		return err
	}
	spec.Auth = authOptions
	return nil
}

func (s *Server) applyBatchJobDefaults(spec *jobs.JobSpec, opts jobRequestOptions) error {
	spec.TimeoutSeconds = opts.timeoutSeconds
	if spec.TimeoutSeconds <= 0 {
		spec.TimeoutSeconds = s.manager.DefaultTimeoutSeconds()
	}
	spec.UsePlaywright = valueOr(opts.playwright, s.manager.DefaultUsePlaywright())
	spec.Extract = valueOr(opts.extract, extract.ExtractOptions{})
	spec.Pipeline = valueOr(opts.pipeline, pipeline.Options{})
	spec.Incremental = valueOr(opts.incremental, false)
	spec.RequestID = opts.requestID
	spec.Screenshot = opts.screenshot
	spec.Device = opts.device
	applyWebhookConfig(spec, opts.webhook)

	if opts.auth != nil || opts.authProfile != "" {
		authOptions, err := resolveAuthForRequest(s.cfg, opts.authURL, opts.authProfile, opts.auth)
		if err != nil {
			return err
		}
		spec.Auth = authOptions
	}

	return nil
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
