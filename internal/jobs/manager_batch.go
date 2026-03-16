// Package jobs provides batch job creation and management for the job manager.
//
// Purpose:
// - Coordinate batch creation, enqueueing, listing, cancellation, and aggregate status inspection.
//
// Responsibilities:
// - Create batches of scrape, crawl, and research jobs.
// - Enqueue all jobs belonging to a batch.
// - List persisted batches with computed aggregate status and pagination metadata.
// - Aggregate current batch status from constituent job statuses.
// - Cancel all jobs belonging to a batch.
//
// Scope:
// - Batch orchestration only; individual job execution lives in other files in this package.
//
// Usage:
// - Called by REST handlers, CLI direct-mode helpers, and MCP tool handlers.
//
// Invariants/Assumptions:
// - Batch size is validated against MaxBatchSize before creation.
// - All jobs in a batch are created before the batch record is persisted.
// - Batch status exposed to callers is derived from current job counts.
package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

// MaxBatchSize is the default maximum number of jobs allowed in a single batch.
const DefaultMaxBatchSize = 100

// CreateBatchJobs creates multiple jobs as a batch and returns the created jobs.
// This method:
// 1. Validates the batch size
// 2. Creates the batch record in the store
// 3. Creates all jobs in the batch
// 4. Associates jobs with the batch
//
// The specs slice must not be empty and must not exceed MaxBatchSize.
func (m *Manager) CreateBatchJobs(ctx context.Context, kind model.Kind, specs []JobSpec, batchID string) ([]model.Job, error) {
	if len(specs) == 0 {
		return nil, apperrors.Validation("batch must contain at least one job")
	}

	// Create all jobs
	jobs := make([]model.Job, 0, len(specs))
	jobIDs := make([]string, 0, len(specs))

	for i, spec := range specs {
		// Ensure each spec has the correct kind
		spec.Kind = kind

		// Create the job
		job, err := m.CreateJob(ctx, spec)
		if err != nil {
			return nil, fmt.Errorf("failed to create job at index %d: %w", i, err)
		}

		jobs = append(jobs, job)
		jobIDs = append(jobIDs, job.ID)
	}

	// Create the batch record
	batch := model.Batch{
		ID:        batchID,
		Kind:      kind,
		Status:    model.BatchStatusPending,
		JobCount:  len(jobs),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := m.store.CreateBatch(ctx, batch, jobIDs); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to create batch record", err)
	}

	slog.Info("created batch",
		"batchID", batchID,
		"kind", kind,
		"jobCount", len(jobs),
	)

	return jobs, nil
}

// EnqueueBatch enqueues all jobs in a batch.
// Returns an error if any job fails to enqueue.
// If the queue is full, returns apperrors.ErrQueueFull.
func (m *Manager) EnqueueBatch(jobs []model.Job) error {
	for _, job := range jobs {
		if err := m.Enqueue(job); err != nil {
			return fmt.Errorf("failed to enqueue job %s: %w", job.ID, err)
		}
	}
	return nil
}

// ListBatchStatuses returns a page of batches with their current aggregate stats and total count.
func (m *Manager) ListBatchStatuses(ctx context.Context, opts store.ListOptions) ([]model.Batch, []model.BatchJobStats, int, error) {
	batches, stats, err := m.store.ListBatchesWithStats(ctx, opts)
	if err != nil {
		return nil, nil, 0, err
	}
	total, err := m.store.CountBatches(ctx)
	if err != nil {
		return nil, nil, 0, err
	}
	return batches, stats, total, nil
}

// GetBatchStatus retrieves the current status of a batch including job statistics.
func (m *Manager) GetBatchStatus(ctx context.Context, batchID string) (model.Batch, model.BatchJobStats, error) {
	// Get the batch
	batch, err := m.store.GetBatch(ctx, batchID)
	if err != nil {
		return model.Batch{}, model.BatchJobStats{}, err
	}

	// Get job statistics
	stats, err := m.store.CountJobsByBatchAndStatus(ctx, batchID)
	if err != nil {
		return model.Batch{}, model.BatchJobStats{}, err
	}

	// Calculate current batch status
	batch.Status = model.CalculateBatchStatus(stats, batch.JobCount)

	return batch, stats, nil
}

// CancelBatch cancels all non-terminal jobs in a batch.
// Returns the number of jobs canceled.
func (m *Manager) CancelBatch(ctx context.Context, batchID string) (int, error) {
	// Get all job IDs in the batch
	jobIDs, err := m.store.GetBatchJobIDs(ctx, batchID)
	if err != nil {
		return 0, err
	}

	canceledCount := 0
	for _, jobID := range jobIDs {
		if err := m.CancelJob(ctx, jobID); err != nil {
			// Log but continue - job might already be terminal
			slog.Warn("failed to cancel job in batch",
				"batchID", batchID,
				"jobID", jobID,
				"error", err,
			)
		} else {
			canceledCount++
		}
	}

	// Update batch status to canceled
	if err := m.store.UpdateBatchStatus(ctx, batchID, model.BatchStatusCanceled); err != nil {
		slog.Warn("failed to update batch status after cancellation",
			"batchID", batchID,
			"error", err,
		)
	}

	slog.Info("canceled batch",
		"batchID", batchID,
		"jobsCanceled", canceledCount,
		"totalJobs", len(jobIDs),
	)

	return canceledCount, nil
}

// GenerateBatchID generates a new unique batch ID.
func GenerateBatchID() string {
	return uuid.NewString()
}
