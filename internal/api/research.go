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

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func (s *Server) handleResearch(w http.ResponseWriter, r *http.Request) {
	handleSingleJobSubmission(s, w, r, singleJobSubmission[ResearchRequest]{
		kind:      model.KindResearch,
		validate:  validateResearchRequest,
		buildSpec: researchJobSpecFromRequest,
		requestOptions: func(r *http.Request, req ResearchRequest) jobRequestOptions {
			return researchJobRequestOptions(contextRequestID(r.Context()), req)
		},
	})
}
