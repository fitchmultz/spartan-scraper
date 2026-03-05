// Package model defines shared domain types for batch job operations.
//
// This file is responsible for:
// - Defining Batch and BatchStatus types for batch job collections
// - Providing status constants and validation methods
//
// This file does NOT handle:
// - Batch persistence or storage operations (see store package)
// - Batch execution or job management (see jobs package)
//
// Invariants:
// - BatchStatus values are validated via IsValid()
// - BatchStatus transitions are managed by the jobs package
package model

import "time"

// BatchStatus represents the aggregate status of a batch of jobs.
type BatchStatus string

const (
	// BatchStatusPending indicates all jobs are queued and waiting to start.
	BatchStatusPending BatchStatus = "pending"
	// BatchStatusProcessing indicates at least one job is running.
	BatchStatusProcessing BatchStatus = "processing"
	// BatchStatusCompleted indicates all jobs have reached a terminal state.
	BatchStatusCompleted BatchStatus = "completed"
	// BatchStatusFailed indicates all jobs failed or were canceled (none succeeded).
	BatchStatusFailed BatchStatus = "failed"
	// BatchStatusPartial indicates a mix of succeeded and failed/canceled jobs.
	BatchStatusPartial BatchStatus = "partial"
	// BatchStatusCanceled indicates the batch was manually canceled.
	BatchStatusCanceled BatchStatus = "canceled"
)

var validBatchStatuses = map[BatchStatus]bool{
	BatchStatusPending:    true,
	BatchStatusProcessing: true,
	BatchStatusCompleted:  true,
	BatchStatusFailed:     true,
	BatchStatusPartial:    true,
	BatchStatusCanceled:   true,
}

// IsValid returns true if the batch status is a recognized value.
func (s BatchStatus) IsValid() bool {
	return validBatchStatuses[s]
}

// IsTerminal returns true if the batch has reached a terminal state.
// Terminal states are: completed, failed, partial, canceled.
func (s BatchStatus) IsTerminal() bool {
	return s == BatchStatusCompleted ||
		s == BatchStatusFailed ||
		s == BatchStatusPartial ||
		s == BatchStatusCanceled
}

// ValidBatchStatuses returns all valid batch status values.
func ValidBatchStatuses() []BatchStatus {
	return []BatchStatus{
		BatchStatusPending,
		BatchStatusProcessing,
		BatchStatusCompleted,
		BatchStatusFailed,
		BatchStatusPartial,
		BatchStatusCanceled,
	}
}

// Batch represents a collection of related jobs submitted together.
type Batch struct {
	ID        string      `json:"id"`
	Kind      Kind        `json:"kind"`
	Status    BatchStatus `json:"status"`
	JobCount  int         `json:"job_count"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// BatchJob represents the association between a batch and its jobs.
type BatchJob struct {
	BatchID string `json:"batch_id"`
	JobID   string `json:"job_id"`
	Index   int    `json:"index"` // Position in original request
}

// BatchJobStats contains aggregated job status counts for a batch.
type BatchJobStats struct {
	Queued    int `json:"queued"`
	Running   int `json:"running"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	Canceled  int `json:"canceled"`
}

// CalculateBatchStatus determines the aggregate batch status from job counts.
// total is the total number of jobs in the batch.
// stats contains counts of jobs in each status.
func CalculateBatchStatus(stats BatchJobStats, total int) BatchStatus {
	completed := stats.Succeeded + stats.Failed + stats.Canceled

	// If nothing has completed yet
	if completed == 0 {
		if stats.Running > 0 {
			return BatchStatusProcessing
		}
		return BatchStatusPending
	}

	// If all jobs are complete
	if completed == total {
		if stats.Succeeded == total {
			return BatchStatusCompleted
		}
		if stats.Succeeded == 0 {
			return BatchStatusFailed
		}
		return BatchStatusPartial
	}

	// Some jobs still running/pending
	return BatchStatusProcessing
}
