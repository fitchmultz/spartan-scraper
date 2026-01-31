// Package distributed provides distributed coordination primitives.
//
// This package defines:
// - Lock interface for distributed locking
// - Worker registry for worker discovery and heartbeats
// - Leader election for scheduler coordination
//
// This package does NOT handle:
// - Queue operations (queue package handles this)
// - Job execution (jobs package handles this)
// - Crawl state storage (store package handles this)
//
// Invariants:
// - Locks have TTL and must be renewed by the holder
// - Lock tokens verify ownership before release
// - Failed lock acquisitions return (false, nil) not an error
package distributed

import (
	"context"
	"time"
)

// Lock provides distributed locking for scheduler and singleton operations.
// Implementations must handle lock TTLs, ownership verification, and
// automatic expiration for safety.
type Lock interface {
	// Acquire attempts to acquire the lock with given TTL.
	// Returns true if lock was acquired, false if already held by another.
	// Returns an error only for communication/storage failures.
	// The token is used to verify ownership for Renew and Release.
	Acquire(ctx context.Context, key string, ttl time.Duration) (acquired bool, token string, err error)

	// Release explicitly releases the lock.
	// The token must match the one returned by Acquire.
	// Returns error if token doesn't match (lock held by another).
	Release(ctx context.Context, key string, token string) error

	// Renew extends the lock TTL. Must be called periodically by holder.
	// The token must match the one returned by Acquire.
	// Returns error if token doesn't match (lock lost or stolen).
	Renew(ctx context.Context, key string, token string, ttl time.Duration) error
}

// WorkerStatus represents the current state of a worker.
type WorkerStatus string

const (
	// WorkerStatusActive means the worker is processing jobs.
	WorkerStatusActive WorkerStatus = "active"
	// WorkerStatusDraining means the worker is finishing in-flight jobs
	// and will not accept new ones.
	WorkerStatusDraining WorkerStatus = "draining"
	// WorkerStatusStopped means the worker has shut down.
	WorkerStatusStopped WorkerStatus = "stopped"
)

// Worker represents a registered worker instance.
type Worker struct {
	ID            string       `json:"id"`
	NodeID        string       `json:"nodeId"`
	StartedAt     time.Time    `json:"startedAt"`
	LastHeartbeat time.Time    `json:"lastHeartbeat"`
	Status        WorkerStatus `json:"status"`
	Version       string       `json:"version"`
	Capabilities  []string     `json:"capabilities,omitempty"`
}

// Registry provides worker registration and discovery.
type Registry interface {
	// Register registers a new worker with the given ID.
	// The worker must call Heartbeat periodically to stay registered.
	Register(ctx context.Context, worker Worker) error

	// Heartbeat updates the worker's last seen timestamp.
	// Must be called periodically (e.g., every 10 seconds).
	Heartbeat(ctx context.Context, workerID string) error

	// UpdateStatus updates the worker's status.
	UpdateStatus(ctx context.Context, workerID string, status WorkerStatus) error

	// Unregister removes a worker from the registry.
	Unregister(ctx context.Context, workerID string) error

	// ListWorkers returns all currently registered workers.
	// Workers that haven't heartbeated recently may be excluded.
	ListWorkers(ctx context.Context) ([]Worker, error)

	// GetWorker returns a specific worker by ID.
	GetWorker(ctx context.Context, workerID string) (Worker, error)
}

// LeaderElection provides leader election for distributed coordination.
type LeaderElection interface {
	// Elect attempts to become the leader for the given role.
	// Returns true if this instance is now the leader.
	// The leadership has a TTL and must be renewed via RenewLeadership.
	Elect(ctx context.Context, role string, instanceID string, ttl time.Duration) (bool, error)

	// RenewLeadership extends the leadership TTL.
	// Must be called periodically by the leader.
	RenewLeadership(ctx context.Context, role string, instanceID string, ttl time.Duration) error

	// Resign releases leadership.
	Resign(ctx context.Context, role string, instanceID string) error

	// IsLeader checks if this instance is currently the leader.
	IsLeader(ctx context.Context, role string, instanceID string) (bool, error)

	// GetLeader returns the current leader for a role.
	GetLeader(ctx context.Context, role string) (string, error)
}
