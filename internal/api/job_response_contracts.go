// Package api centralizes sanitized job and batch response envelope construction.
//
// Purpose:
// - Provide one canonical response-shaping path for job and batch surfaces.
//
// Responsibilities:
// - Wrap sanitized jobs in stable single-item and collection envelopes.
// - Shape batch metadata, aggregated stats, and optional included job pages.
// - Keep REST and MCP outputs aligned so clients do not branch on transport-specific payloads.
//
// Scope:
// - Response construction only; persistence and execution remain in other packages.
//
// Usage:
// - Called by REST handlers and MCP tool handlers before encoding responses.
//
// Invariants/Assumptions:
// - Jobs are always sanitized before they leave trusted server boundaries.
// - Collection responses always emit arrays, never null.
// - Batch envelopes always carry aggregate stats, even when individual jobs are omitted.
package api

import (
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func normalizeSanitizedJobs(jobs []model.Job) []model.Job {
	sanitized := model.SanitizeJobs(jobs)
	if sanitized == nil {
		return []model.Job{}
	}
	return sanitized
}

// BuildJobResponse returns the canonical single-job response envelope.
func BuildJobResponse(job model.Job) JobResponse {
	return JobResponse{Job: model.SanitizeJob(job)}
}

// BuildJobListResponse returns the canonical paginated job collection envelope.
func BuildJobListResponse(jobs []model.Job, total int, limit int, offset int) JobListResponse {
	return JobListResponse{
		Jobs:   normalizeSanitizedJobs(jobs),
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
}

// BuildBatchSummary returns the canonical aggregate batch summary.
func BuildBatchSummary(batch model.Batch, stats model.BatchJobStats) BatchSummary {
	return BatchSummary{
		ID:        batch.ID,
		Kind:      string(batch.Kind),
		Status:    string(batch.Status),
		JobCount:  batch.JobCount,
		Stats:     stats,
		CreatedAt: batch.CreatedAt,
		UpdatedAt: batch.UpdatedAt,
	}
}

// BuildBatchResponse returns the canonical batch response envelope.
func BuildBatchResponse(batch model.Batch, stats model.BatchJobStats, jobs []model.Job, total int, limit int, offset int) BatchResponse {
	return BatchResponse{
		Batch:  BuildBatchSummary(batch, stats),
		Jobs:   normalizeSanitizedJobs(jobs),
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
}

// BuildBatchListResponse returns the canonical paginated batch-summary collection envelope.
func BuildBatchListResponse(batches []model.Batch, stats []model.BatchJobStats, total int, limit int, offset int) BatchListResponse {
	summaries := make([]BatchSummary, 0, len(batches))
	for i, batch := range batches {
		var batchStats model.BatchJobStats
		if i < len(stats) {
			batchStats = stats[i]
		}
		summaries = append(summaries, BuildBatchSummary(batch, batchStats))
	}
	if summaries == nil {
		summaries = []BatchSummary{}
	}
	return BatchListResponse{
		Batches: summaries,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}
}

// BuildCreatedBatchResponse returns the canonical batch envelope immediately after submission.
func BuildCreatedBatchResponse(batchID string, kind model.Kind, createdJobs []model.Job) BatchResponse {
	jobCount := len(createdJobs)
	createdAt := time.Time{}
	updatedAt := time.Time{}
	if jobCount > 0 {
		createdAt = createdJobs[0].CreatedAt
		updatedAt = createdJobs[0].UpdatedAt
	}
	stats := model.BatchJobStats{Queued: jobCount}
	batch := model.Batch{
		ID:        batchID,
		Kind:      kind,
		Status:    model.BatchStatusPending,
		JobCount:  jobCount,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
	return BuildBatchResponse(batch, stats, createdJobs, jobCount, jobCount, 0)
}
