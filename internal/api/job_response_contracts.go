// Package api centralizes sanitized job and batch response envelope construction.
//
// Purpose:
// - Provide one canonical response-shaping path for job and batch surfaces.
//
// Responsibilities:
// - Wrap sanitized jobs in stable single-item and collection envelopes.
// - Derive run-history, queue-progression, and failure-context fields from persisted jobs and batches.
// - Shape batch metadata, aggregated stats, progress, and optional included job pages.
// - Keep REST, CLI direct-mode, and MCP outputs aligned so clients do not branch on transport-specific payloads.
//
// Scope:
// - Response construction only; persistence and execution remain in other packages.
//
// Usage:
// - Called by REST handlers, CLI direct-mode helpers, and MCP tool handlers before encoding responses.
//
// Invariants/Assumptions:
// - Jobs are always sanitized before they leave trusted server boundaries.
// - Collection responses always emit arrays, never null.
// - Batch envelopes always carry aggregate stats and explicit progress, even when individual jobs are omitted.
package api

import (
	"context"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func normalizeInspectableJobs(jobs []InspectableJob) []InspectableJob {
	if jobs == nil {
		return []InspectableJob{}
	}
	return jobs
}

func collectJobIDs(jobs []model.Job) []string {
	ids := make([]string, 0, len(jobs))
	for _, job := range jobs {
		jobID := strings.TrimSpace(job.ID)
		if jobID != "" {
			ids = append(ids, jobID)
		}
	}
	return ids
}

func buildInspectableJob(job model.Job, meta store.JobBatchMeta) InspectableJob {
	safeJob := model.SanitizeJob(job)
	return InspectableJob{
		Job: safeJob,
		Run: buildJobRunSummary(safeJob, meta),
	}
}

func buildInspectableJobs(jobs []model.Job) []InspectableJob {
	result := make([]InspectableJob, 0, len(jobs))
	for _, job := range jobs {
		result = append(result, buildInspectableJob(job, store.JobBatchMeta{}))
	}
	return normalizeInspectableJobs(result)
}

func buildStoreBackedInspectableJobs(ctx context.Context, st *store.Store, jobs []model.Job) ([]InspectableJob, error) {
	if st == nil {
		return buildInspectableJobs(jobs), nil
	}

	metaByJobID, err := st.LoadJobBatchMeta(ctx, collectJobIDs(jobs))
	if err != nil {
		return nil, err
	}

	result := make([]InspectableJob, 0, len(jobs))
	for _, job := range jobs {
		result = append(result, buildInspectableJob(job, metaByJobID[job.ID]))
	}
	return normalizeInspectableJobs(result), nil
}

// BuildJobResponse returns the canonical single-job response envelope.
func BuildJobResponse(job model.Job) JobResponse {
	return JobResponse{Job: buildInspectableJob(job, store.JobBatchMeta{})}
}

// BuildStoreBackedJobResponse returns the canonical single-job response envelope with batch metadata enrichment.
func BuildStoreBackedJobResponse(ctx context.Context, st *store.Store, job model.Job) (JobResponse, error) {
	jobs, err := buildStoreBackedInspectableJobs(ctx, st, []model.Job{job})
	if err != nil {
		return JobResponse{}, err
	}
	if len(jobs) == 0 {
		return JobResponse{}, nil
	}
	return JobResponse{Job: jobs[0]}, nil
}

// BuildJobListResponse returns the canonical paginated job collection envelope.
func BuildJobListResponse(jobs []model.Job, total int, limit int, offset int) JobListResponse {
	return JobListResponse{
		Jobs:   buildInspectableJobs(jobs),
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
}

// BuildStoreBackedJobListResponse returns the canonical paginated job collection envelope with batch metadata enrichment.
func BuildStoreBackedJobListResponse(ctx context.Context, st *store.Store, jobs []model.Job, total int, limit int, offset int) (JobListResponse, error) {
	items, err := buildStoreBackedInspectableJobs(ctx, st, jobs)
	if err != nil {
		return JobListResponse{}, err
	}
	return JobListResponse{
		Jobs:   items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func buildBatchProgress(stats model.BatchJobStats, total int) BatchProgress {
	completed := stats.Succeeded + stats.Failed + stats.Canceled
	remaining := total - completed
	if remaining < 0 {
		remaining = 0
	}
	percent := 0
	if total > 0 {
		percent = (completed * 100) / total
	}
	return BatchProgress{
		Completed: completed,
		Remaining: remaining,
		Percent:   percent,
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
		Progress:  buildBatchProgress(stats, batch.JobCount),
		CreatedAt: batch.CreatedAt,
		UpdatedAt: batch.UpdatedAt,
	}
}

// BuildBatchResponse returns the canonical batch response envelope.
func BuildBatchResponse(batch model.Batch, stats model.BatchJobStats, jobs []model.Job, total int, limit int, offset int) BatchResponse {
	return BatchResponse{
		Batch:  BuildBatchSummary(batch, stats),
		Jobs:   buildInspectableJobs(jobs),
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}
}

// BuildStoreBackedBatchResponse returns the canonical batch response envelope with batch-aware job enrichment.
func BuildStoreBackedBatchResponse(ctx context.Context, st *store.Store, batch model.Batch, stats model.BatchJobStats, jobs []model.Job, total int, limit int, offset int) (BatchResponse, error) {
	items, err := buildStoreBackedInspectableJobs(ctx, st, jobs)
	if err != nil {
		return BatchResponse{}, err
	}
	return BatchResponse{
		Batch:  BuildBatchSummary(batch, stats),
		Jobs:   items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
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

func buildJobRunSummary(job model.Job, meta store.JobBatchMeta) JobRunSummary {
	waitEnd := time.Now()
	if job.StartedAt != nil {
		waitEnd = *job.StartedAt
	} else if job.FinishedAt != nil {
		waitEnd = *job.FinishedAt
	}

	totalEnd := time.Now()
	if job.FinishedAt != nil {
		totalEnd = *job.FinishedAt
	}

	runMs := int64(0)
	if job.StartedAt != nil {
		runEnd := time.Now()
		if job.FinishedAt != nil {
			runEnd = *job.FinishedAt
		}
		runMs = durationMs(*job.StartedAt, runEnd)
	}

	summary := JobRunSummary{
		WaitMs:  durationMs(job.CreatedAt, waitEnd),
		RunMs:   runMs,
		TotalMs: durationMs(job.CreatedAt, totalEnd),
	}

	if meta.BatchID != "" {
		summary.Queue = buildJobQueueProgress(meta)
	}
	if failure := buildJobFailureContext(job); failure != nil {
		summary.Failure = failure
	}

	return summary
}

func buildJobQueueProgress(meta store.JobBatchMeta) *JobQueueProgress {
	completed := meta.Stats.Succeeded + meta.Stats.Failed + meta.Stats.Canceled
	remaining := meta.BatchTotal - completed
	if remaining < 0 {
		remaining = 0
	}
	percent := 0
	if meta.BatchTotal > 0 {
		percent = (completed * 100) / meta.BatchTotal
	}
	return &JobQueueProgress{
		BatchID:   meta.BatchID,
		Index:     meta.BatchIndex + 1,
		Total:     meta.BatchTotal,
		Completed: completed,
		Remaining: remaining,
		Queued:    meta.Stats.Queued,
		Running:   meta.Stats.Running,
		Percent:   percent,
	}
}

func buildJobFailureContext(job model.Job) *JobFailureContext {
	if job.Status != model.StatusFailed && job.Status != model.StatusCanceled {
		return nil
	}

	summary := strings.TrimSpace(job.Error)
	if summary == "" && job.Status == model.StatusCanceled {
		summary = "canceled by user"
	}
	if summary == "" {
		summary = "job ended without a recorded error message"
	}
	if len(summary) > 240 {
		summary = summary[:239] + "…"
	}

	category, retryable := classifyJobFailure(summary, job.Status)
	return &JobFailureContext{
		Category:  category,
		Summary:   summary,
		Retryable: retryable,
		Terminal:  true,
	}
}

func classifyJobFailure(summary string, status model.Status) (string, bool) {
	if status == model.StatusCanceled {
		return "canceled", false
	}

	lower := strings.ToLower(summary)
	switch {
	case strings.Contains(lower, "deadline exceeded"), strings.Contains(lower, "timeout"):
		return "timeout", true
	case strings.Contains(lower, "unauthorized"), strings.Contains(lower, "forbidden"), strings.Contains(lower, "401"), strings.Contains(lower, "403"), strings.Contains(lower, "auth"):
		return "auth", false
	case strings.Contains(lower, "dial tcp"), strings.Contains(lower, "connection refused"), strings.Contains(lower, "no such host"), strings.Contains(lower, "tls"), strings.Contains(lower, "eof"), strings.Contains(lower, "network"):
		return "network", true
	case strings.Contains(lower, "playwright"), strings.Contains(lower, "chromedp"), strings.Contains(lower, "browser"):
		return "browser", true
	case strings.Contains(lower, "invalid"), strings.Contains(lower, "validation"):
		return "validation", false
	default:
		return "unknown", false
	}
}

func durationMs(start, end time.Time) int64 {
	if end.Before(start) {
		return 0
	}
	return end.Sub(start).Milliseconds()
}
