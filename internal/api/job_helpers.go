// Package api provides shared HTTP submission helpers for job-creation endpoints.
//
// Purpose:
//   - Keep REST handlers thin while delegating operator-facing request conversion to
//     the canonical internal/submission package.
//
// Responsibilities:
// - Decode JSON requests and route them through shared submission builders.
// - Provide consistent request-default envelopes for live REST job creation.
// - Create and enqueue single jobs and batches with one response-shaping path.
//
// Scope:
//   - API transport glue only; request validation and request-to-spec conversion live in
//     internal/submission.
//
// Usage:
// - Used by scrape, crawl, research, and batch API handlers.
//
// Invariants/Assumptions:
// - REST creation endpoints should use the same operator-facing conversion rules as other surfaces.
// - Live REST submissions always resolve auth before persisting jobs.
package api

import (
	"context"
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/submission"
)

type singleJobSubmission[T any] struct {
	buildSpec func(*http.Request, T) (jobs.JobSpec, error)
}

type batchJobSubmission[T any] struct {
	kind       model.Kind
	buildSpecs func(*http.Request, T) ([]jobs.JobSpec, error)
}

func (s *Server) requestSubmissionDefaults(r *http.Request) submission.Defaults {
	return submission.Defaults{
		DefaultTimeoutSeconds: s.manager.DefaultTimeoutSeconds(),
		DefaultUsePlaywright:  s.manager.DefaultUsePlaywright(),
		RequestID:             contextRequestID(r.Context()),
		ResolveAuth:           true,
	}
}

func (s *Server) nonResolvingSubmissionDefaults() submission.Defaults {
	return submission.Defaults{
		DefaultTimeoutSeconds: s.manager.DefaultTimeoutSeconds(),
		DefaultUsePlaywright:  s.manager.DefaultUsePlaywright(),
		ResolveAuth:           false,
	}
}

func (s *Server) requestBatchDefaults(r *http.Request) submission.BatchDefaults {
	return submission.BatchDefaults{
		Defaults:     s.requestSubmissionDefaults(r),
		MaxBatchSize: s.cfg.MaxBatchSize,
	}
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

	spec, err := submission.buildSpec(r, req)
	if err != nil {
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

	resp, err := BuildStoreBackedJobResponse(r.Context(), s.store, job)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeCreatedJSON(w, resp)
}

func handleBatchJobSubmission[T any](s *Server, w http.ResponseWriter, r *http.Request, submission batchJobSubmission[T]) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	req, ok := decodeJSONRequest[T](w, r)
	if !ok {
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

	batch, stats, err := s.manager.GetBatchStatus(ctx, batchID)
	if err != nil {
		return BatchResponse{}, err
	}
	return BuildStoreBackedBatchResponse(ctx, s.store, batch, stats, createdJobs, len(createdJobs), len(createdJobs), 0)
}
