// Package distributed provides distributed coordination primitives.
//
// This file contains tests for Redis-based worker registry.
package distributed

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// TestRedisRegistry_Register_Success verifies successful worker registration.
func TestRedisRegistry_Register_Success(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()
	rr := NewRedisRegistry(client, "test:worker:", 30*time.Second).(*RedisRegistry)

	worker := Worker{
		ID:            "worker-1",
		NodeID:        "node-1",
		StartedAt:     time.Now(),
		LastHeartbeat: time.Now(),
		Status:        WorkerStatusActive,
		Version:       "1.0.0",
	}

	err := rr.Register(ctx, worker)
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	// Verify worker data exists in Redis
	key := "test:worker:worker-1"
	data, err := mr.Get(key)
	if err != nil {
		t.Fatalf("expected worker data to exist in Redis: %v", err)
	}

	// Verify data can be unmarshaled and matches input
	var storedWorker Worker
	if err := json.Unmarshal([]byte(data), &storedWorker); err != nil {
		t.Fatalf("failed to unmarshal worker data: %v", err)
	}
	if storedWorker.ID != worker.ID {
		t.Errorf("expected worker ID %q, got %q", worker.ID, storedWorker.ID)
	}
	if storedWorker.NodeID != worker.NodeID {
		t.Errorf("expected node ID %q, got %q", worker.NodeID, storedWorker.NodeID)
	}
	if storedWorker.Status != worker.Status {
		t.Errorf("expected status %q, got %q", worker.Status, storedWorker.Status)
	}
}

// TestRedisRegistry_Register_Error verifies Register returns apperrors.KindInternal on Redis failure.
func TestRedisRegistry_Register_Error(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()

	rr := NewRedisRegistry(client, "test:worker:", 30*time.Second).(*RedisRegistry)

	// Close the connection to force an error
	mr.Close()

	worker := Worker{
		ID:            "worker-1",
		NodeID:        "node-1",
		StartedAt:     time.Now(),
		LastHeartbeat: time.Now(),
		Status:        WorkerStatusActive,
		Version:       "1.0.0",
	}

	err := rr.Register(ctx, worker)
	if err == nil {
		t.Fatal("expected error when Redis is unavailable")
	}

	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("expected internal error, got kind: %v", apperrors.KindOf(err))
	}

	safeMsg := apperrors.SafeMessage(err)
	if safeMsg != "failed to register worker" {
		t.Errorf("expected safe message 'failed to register worker', got %q", safeMsg)
	}
}

// TestRedisRegistry_Unregister_Success verifies successful worker unregistration.
func TestRedisRegistry_Unregister_Success(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()
	rr := NewRedisRegistry(client, "test:worker:", 30*time.Second).(*RedisRegistry)

	// Register worker
	worker := Worker{
		ID:            "worker-1",
		NodeID:        "node-1",
		StartedAt:     time.Now(),
		LastHeartbeat: time.Now(),
		Status:        WorkerStatusActive,
		Version:       "1.0.0",
	}
	if err := rr.Register(ctx, worker); err != nil {
		t.Fatalf("failed to register worker: %v", err)
	}

	// Verify worker exists
	key := "test:worker:worker-1"
	if !mr.Exists(key) {
		t.Fatal("expected worker to exist in Redis")
	}

	// Unregister worker
	err := rr.Unregister(ctx, "worker-1")
	if err != nil {
		t.Fatalf("Unregister returned error: %v", err)
	}

	// Verify worker no longer exists
	if mr.Exists(key) {
		t.Error("expected worker to be removed from Redis")
	}

	// Verify GetWorker returns not found
	_, err = rr.GetWorker(ctx, "worker-1")
	if err == nil {
		t.Error("expected error for unregistered worker")
	}
	if !apperrors.IsKind(err, apperrors.KindNotFound) {
		t.Errorf("expected not_found error, got kind: %v", apperrors.KindOf(err))
	}
}

// TestRedisRegistry_Unregister_Error verifies Unregister returns apperrors.KindInternal on Redis failure.
func TestRedisRegistry_Unregister_Error(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()

	rr := NewRedisRegistry(client, "test:worker:", 30*time.Second).(*RedisRegistry)

	// Close the connection to force an error
	mr.Close()

	err := rr.Unregister(ctx, "worker-1")
	if err == nil {
		t.Fatal("expected error when Redis is unavailable")
	}

	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("expected internal error, got kind: %v", apperrors.KindOf(err))
	}

	safeMsg := apperrors.SafeMessage(err)
	if safeMsg != "failed to unregister worker" {
		t.Errorf("expected safe message 'failed to unregister worker', got %q", safeMsg)
	}
}

// TestRedisRegistry_Heartbeat_Success verifies successful heartbeat update.
func TestRedisRegistry_Heartbeat_Success(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()
	rr := NewRedisRegistry(client, "test:worker:", 30*time.Second).(*RedisRegistry)

	// Register worker
	worker := Worker{
		ID:            "worker-1",
		NodeID:        "node-1",
		StartedAt:     time.Now(),
		LastHeartbeat: time.Now(),
		Status:        WorkerStatusActive,
		Version:       "1.0.0",
	}
	if err := rr.Register(ctx, worker); err != nil {
		t.Fatalf("failed to register worker: %v", err)
	}

	// Get initial data
	key := "test:worker:worker-1"
	initialData, err := mr.Get(key)
	if err != nil {
		t.Fatalf("failed to get worker data: %v", err)
	}
	var initialWorker Worker
	if err := json.Unmarshal([]byte(initialData), &initialWorker); err != nil {
		t.Fatalf("failed to unmarshal worker data: %v", err)
	}

	// Fast forward a bit
	mr.FastForward(100 * time.Millisecond)

	// Send heartbeat
	if err := rr.Heartbeat(ctx, "worker-1"); err != nil {
		t.Fatalf("Heartbeat returned error: %v", err)
	}

	// Verify LastHeartbeat is updated
	newData, err := mr.Get(key)
	if err != nil {
		t.Fatalf("failed to get worker data: %v", err)
	}
	var newWorker Worker
	if err := json.Unmarshal([]byte(newData), &newWorker); err != nil {
		t.Fatalf("failed to unmarshal worker data: %v", err)
	}
	if !newWorker.LastHeartbeat.After(initialWorker.LastHeartbeat) {
		t.Error("expected LastHeartbeat to be updated")
	}

	// Verify TTL is extended
	ttl := mr.TTL(key)
	if ttl <= 0 {
		t.Errorf("expected positive TTL, got %v", ttl)
	}
}

// TestRedisRegistry_Heartbeat_NotFound verifies heartbeat fails for unregistered worker.
func TestRedisRegistry_Heartbeat_NotFound(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	rr := NewRedisRegistry(client, "test:worker:", 30*time.Second).(*RedisRegistry)

	err := rr.Heartbeat(ctx, "nonexistent-worker")
	if err == nil {
		t.Fatal("expected error for unregistered worker")
	}

	if !apperrors.IsKind(err, apperrors.KindNotFound) {
		t.Errorf("expected not_found error, got kind: %v", apperrors.KindOf(err))
	}

	expectedMsg := "worker not registered"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

// TestRedisRegistry_Heartbeat_Error verifies Heartbeat returns apperrors.KindInternal on Redis failure.
// Note: Heartbeat first reads the worker data, so the error will be from the Get operation.
func TestRedisRegistry_Heartbeat_Error(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()

	rr := NewRedisRegistry(client, "test:worker:", 30*time.Second).(*RedisRegistry)

	// Register a worker first
	worker := Worker{
		ID:            "worker-1",
		NodeID:        "node-1",
		StartedAt:     time.Now(),
		LastHeartbeat: time.Now(),
		Status:        WorkerStatusActive,
		Version:       "1.0.0",
	}
	if err := rr.Register(ctx, worker); err != nil {
		t.Fatalf("failed to register worker: %v", err)
	}

	// Close the connection to force an error
	mr.Close()

	err := rr.Heartbeat(ctx, "worker-1")
	if err == nil {
		t.Fatal("expected error when Redis is unavailable")
	}

	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("expected internal error, got kind: %v", apperrors.KindOf(err))
	}

	// Heartbeat reads first, so the error will be from Get, not Set
	safeMsg := apperrors.SafeMessage(err)
	if safeMsg != "failed to get worker" {
		t.Errorf("expected safe message 'failed to get worker', got %q", safeMsg)
	}
}

// TestRedisRegistry_UpdateStatus_Success verifies successful status update.
func TestRedisRegistry_UpdateStatus_Success(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()
	rr := NewRedisRegistry(client, "test:worker:", 30*time.Second).(*RedisRegistry)

	// Register worker with Active status
	worker := Worker{
		ID:            "worker-1",
		NodeID:        "node-1",
		StartedAt:     time.Now(),
		LastHeartbeat: time.Now(),
		Status:        WorkerStatusActive,
		Version:       "1.0.0",
	}
	if err := rr.Register(ctx, worker); err != nil {
		t.Fatalf("failed to register worker: %v", err)
	}

	// Fast forward a bit
	mr.FastForward(100 * time.Millisecond)

	// Update status to Draining
	err := rr.UpdateStatus(ctx, "worker-1", WorkerStatusDraining)
	if err != nil {
		t.Fatalf("UpdateStatus returned error: %v", err)
	}

	// Verify status is updated
	key := "test:worker:worker-1"
	data, err := mr.Get(key)
	if err != nil {
		t.Fatalf("failed to get worker data: %v", err)
	}
	var updatedWorker Worker
	if err := json.Unmarshal([]byte(data), &updatedWorker); err != nil {
		t.Fatalf("failed to unmarshal worker data: %v", err)
	}
	if updatedWorker.Status != WorkerStatusDraining {
		t.Errorf("expected status %q, got %q", WorkerStatusDraining, updatedWorker.Status)
	}

	// Verify LastHeartbeat is also updated
	if !updatedWorker.LastHeartbeat.After(worker.LastHeartbeat) {
		t.Error("expected LastHeartbeat to be updated")
	}
}

// TestRedisRegistry_UpdateStatus_NotFound verifies status update fails for unregistered worker.
func TestRedisRegistry_UpdateStatus_NotFound(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	rr := NewRedisRegistry(client, "test:worker:", 30*time.Second).(*RedisRegistry)

	err := rr.UpdateStatus(ctx, "nonexistent-worker", WorkerStatusStopped)
	if err == nil {
		t.Fatal("expected error for unregistered worker")
	}

	if !apperrors.IsKind(err, apperrors.KindNotFound) {
		t.Errorf("expected not_found error, got kind: %v", apperrors.KindOf(err))
	}
}

// TestRedisRegistry_UpdateStatus_Error verifies UpdateStatus returns apperrors.KindInternal on Redis failure.
// Note: UpdateStatus first reads the worker data, so the error will be from the Get operation.
func TestRedisRegistry_UpdateStatus_Error(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()

	rr := NewRedisRegistry(client, "test:worker:", 30*time.Second).(*RedisRegistry)

	// Register a worker first
	worker := Worker{
		ID:            "worker-1",
		NodeID:        "node-1",
		StartedAt:     time.Now(),
		LastHeartbeat: time.Now(),
		Status:        WorkerStatusActive,
		Version:       "1.0.0",
	}
	if err := rr.Register(ctx, worker); err != nil {
		t.Fatalf("failed to register worker: %v", err)
	}

	// Close the connection to force an error
	mr.Close()

	err := rr.UpdateStatus(ctx, "worker-1", WorkerStatusStopped)
	if err == nil {
		t.Fatal("expected error when Redis is unavailable")
	}

	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("expected internal error, got kind: %v", apperrors.KindOf(err))
	}

	// UpdateStatus reads first, so the error will be from Get, not Set
	safeMsg := apperrors.SafeMessage(err)
	if safeMsg != "failed to get worker" {
		t.Errorf("expected safe message 'failed to get worker', got %q", safeMsg)
	}
}

// TestRedisRegistry_GetWorker_Success verifies successful worker retrieval.
func TestRedisRegistry_GetWorker_Success(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	rr := NewRedisRegistry(client, "test:worker:", 30*time.Second).(*RedisRegistry)

	// Register worker
	worker := Worker{
		ID:            "worker-1",
		NodeID:        "node-1",
		StartedAt:     time.Now(),
		LastHeartbeat: time.Now(),
		Status:        WorkerStatusActive,
		Version:       "1.0.0",
	}
	if err := rr.Register(ctx, worker); err != nil {
		t.Fatalf("failed to register worker: %v", err)
	}

	// Get worker
	retrieved, err := rr.GetWorker(ctx, "worker-1")
	if err != nil {
		t.Fatalf("GetWorker returned error: %v", err)
	}

	if retrieved.ID != worker.ID {
		t.Errorf("expected worker ID %q, got %q", worker.ID, retrieved.ID)
	}
	if retrieved.NodeID != worker.NodeID {
		t.Errorf("expected node ID %q, got %q", worker.NodeID, retrieved.NodeID)
	}
	if retrieved.Status != worker.Status {
		t.Errorf("expected status %q, got %q", worker.Status, retrieved.Status)
	}
	if retrieved.Version != worker.Version {
		t.Errorf("expected version %q, got %q", worker.Version, retrieved.Version)
	}
}

// TestRedisRegistry_GetWorker_NotFound verifies GetWorker returns not found for missing worker.
func TestRedisRegistry_GetWorker_NotFound(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	rr := NewRedisRegistry(client, "test:worker:", 30*time.Second).(*RedisRegistry)

	_, err := rr.GetWorker(ctx, "nonexistent-worker")
	if err == nil {
		t.Fatal("expected error for missing worker")
	}

	if !apperrors.IsKind(err, apperrors.KindNotFound) {
		t.Errorf("expected not_found error, got kind: %v", apperrors.KindOf(err))
	}

	expectedMsg := "worker not found"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

// TestRedisRegistry_GetWorker_Error verifies GetWorker returns internal error on Redis failure.
func TestRedisRegistry_GetWorker_Error(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()
	rr := NewRedisRegistry(client, "test:worker:", 30*time.Second).(*RedisRegistry)

	// Register a worker first
	worker := Worker{
		ID:            "worker-1",
		NodeID:        "node-1",
		StartedAt:     time.Now(),
		LastHeartbeat: time.Now(),
		Status:        WorkerStatusActive,
		Version:       "1.0.0",
	}
	if err := rr.Register(ctx, worker); err != nil {
		t.Fatalf("failed to register worker: %v", err)
	}

	// Close Redis connection
	mr.Close()

	_, err := rr.GetWorker(ctx, "worker-1")
	if err == nil {
		t.Fatal("expected error when Redis is unavailable")
	}

	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("expected internal error, got kind: %v", apperrors.KindOf(err))
	}
}

// TestRedisRegistry_ListWorkers_MultipleBatches verifies SCAN iterates through multiple batches.
func TestRedisRegistry_ListWorkers_MultipleBatches(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()

	rr := NewRedisRegistry(client, "test:worker:", 30*time.Second).(*RedisRegistry)

	// Register more workers than the batch size (100) to test multiple SCAN iterations
	numWorkers := 250
	for i := 0; i < numWorkers; i++ {
		worker := Worker{
			ID:            fmt.Sprintf("worker-%03d", i),
			NodeID:        fmt.Sprintf("node-%d", i%10),
			StartedAt:     time.Now(),
			LastHeartbeat: time.Now(),
			Status:        WorkerStatusActive,
			Version:       "1.0.0",
		}
		if err := rr.Register(ctx, worker); err != nil {
			t.Fatalf("failed to register worker %d: %v", i, err)
		}
	}

	workers, err := rr.ListWorkers(ctx)
	if err != nil {
		t.Fatalf("ListWorkers failed: %v", err)
	}

	if len(workers) != numWorkers {
		t.Errorf("expected %d workers, got %d", numWorkers, len(workers))
	}
}

// TestRedisRegistry_ListWorkers_ContextCancellation verifies context cancellation is respected.
func TestRedisRegistry_ListWorkers_ContextCancellation(t *testing.T) {
	_, client := setupTestRedis(t)

	rr := NewRedisRegistry(client, "test:worker:", 30*time.Second).(*RedisRegistry)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := rr.ListWorkers(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}

	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("expected internal error, got kind: %v", apperrors.KindOf(err))
	}
}

// TestRedisRegistry_ListWorkers_ExpiredWorkers verifies expired workers are skipped.
func TestRedisRegistry_ListWorkers_ExpiredWorkers(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()

	// Use a very short TTL
	rr := NewRedisRegistry(client, "test:worker:", 100*time.Millisecond).(*RedisRegistry)

	// Register a worker
	worker := Worker{
		ID:            "worker-1",
		NodeID:        "node-1",
		StartedAt:     time.Now(),
		LastHeartbeat: time.Now(),
		Status:        WorkerStatusActive,
		Version:       "1.0.0",
	}
	if err := rr.Register(ctx, worker); err != nil {
		t.Fatalf("failed to register worker: %v", err)
	}

	// Verify worker is listed
	workers, err := rr.ListWorkers(ctx)
	if err != nil {
		t.Fatalf("ListWorkers failed: %v", err)
	}
	if len(workers) != 1 {
		t.Errorf("expected 1 worker, got %d", len(workers))
	}

	// Fast-forward past TTL
	mr.FastForward(200 * time.Millisecond)

	// Worker should be expired now
	workers, err = rr.ListWorkers(ctx)
	if err != nil {
		t.Fatalf("ListWorkers failed: %v", err)
	}
	if len(workers) != 0 {
		t.Errorf("expected 0 workers after expiry, got %d", len(workers))
	}
}

// TestRedisRegistry_ListWorkers_InvalidJSON verifies unmarshal errors are handled gracefully.
func TestRedisRegistry_ListWorkers_InvalidJSON(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()

	rr := NewRedisRegistry(client, "test:worker:", 30*time.Second).(*RedisRegistry)

	// Register a valid worker
	validWorker := Worker{
		ID:            "worker-valid",
		NodeID:        "node-1",
		StartedAt:     time.Now(),
		LastHeartbeat: time.Now(),
		Status:        WorkerStatusActive,
		Version:       "1.0.0",
	}
	if err := rr.Register(ctx, validWorker); err != nil {
		t.Fatalf("failed to register valid worker: %v", err)
	}

	// Manually insert invalid JSON
	mr.Set("test:worker:worker-invalid", "not-valid-json")

	// List should return only the valid worker, skipping invalid ones
	workers, err := rr.ListWorkers(ctx)
	if err != nil {
		t.Fatalf("ListWorkers failed: %v", err)
	}
	if len(workers) != 1 {
		t.Errorf("expected 1 valid worker, got %d", len(workers))
	}
	if workers[0].ID != "worker-valid" {
		t.Errorf("expected worker-valid, got %s", workers[0].ID)
	}
}
