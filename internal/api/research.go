// Package api provides HTTP handlers for research job endpoints.
//
// Purpose:
// - Accept multi-source research submissions over the API.
//
// Responsibilities:
// - Validate research requests.
// - Build a research JobSpec with shared request defaults.
// - Create and enqueue the job, then return the sanitized record.
//
// Scope:
// - Research request handling only; job execution lives in internal/jobs.
//
// Usage:
// - Registered for POST /v1/research.
//
// Invariants/Assumptions:
// - Requests must be JSON and include both a query and at least one URL.
// - Auth resolution for research uses the first target URL as the host context.
package api

import (
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/validate"
)

func (s *Server) handleResearch(w http.ResponseWriter, r *http.Request) {
	handleSingleJobSubmission(s, w, r, singleJobSubmission[ResearchRequest]{
		kind: model.KindResearch,
		validate: func(req ResearchRequest) error {
			if req.Query == "" || len(req.URLs) == 0 {
				return apperrors.Validation("query and urls are required")
			}
			if err := model.ValidateResearchAgenticConfig(req.Agentic); err != nil {
				return err
			}
			return validate.ValidateJob(validate.JobValidationOpts{
				Query:       req.Query,
				URLs:        req.URLs,
				MaxDepth:    req.MaxDepth,
				MaxPages:    req.MaxPages,
				Timeout:     req.TimeoutSeconds,
				AuthProfile: req.AuthProfile,
			}, model.KindResearch)
		},
		buildSpec: func(req ResearchRequest) jobs.JobSpec {
			return jobs.JobSpec{
				Kind:     model.KindResearch,
				Query:    req.Query,
				URLs:     req.URLs,
				MaxDepth: req.MaxDepth,
				MaxPages: req.MaxPages,
				Headless: req.Headless,
				Agentic:  req.Agentic,
			}
		},
		requestOptions: func(r *http.Request, req ResearchRequest) jobRequestOptions {
			return jobRequestOptions{
				authURL:          req.URLs[0],
				authProfile:      req.AuthProfile,
				auth:             req.Auth,
				extract:          req.Extract,
				pipeline:         req.Pipeline,
				webhook:          req.Webhook,
				screenshot:       req.Screenshot,
				device:           req.Device,
				networkIntercept: req.NetworkIntercept,
				playwright:       req.Playwright,
				timeoutSeconds:   req.TimeoutSeconds,
				requestID:        contextRequestID(r.Context()),
			}
		},
	})
}
