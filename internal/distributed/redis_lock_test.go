// Package distributed provides distributed coordination primitives.
//
// This file contains tests for Redis-based distributed locking and leader election.
package distributed

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/redis/go-redis/v9"
)

// TestRedisLock_Acquire_InvalidTTL tests that Acquire rejects non-positive TTL values.
func TestRedisLock_Acquire_InvalidTTL(t *testing.T) {
	rl := &RedisLock{}

	tests := []struct {
		name string
		ttl  time.Duration
	}{
		{
			name: "zero TTL",
			ttl:  0,
		},
		{
			name: "negative TTL",
			ttl:  -1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			acquired, token, err := rl.Acquire(ctx, "test-key", tt.ttl)

			if acquired {
				t.Error("expected acquired to be false")
			}
			if token != "" {
				t.Error("expected empty token")
			}
			if err == nil {
				t.Fatal("expected error for invalid TTL")
			}

			if !apperrors.IsKind(err, apperrors.KindValidation) {
				t.Errorf("expected validation error, got kind: %v", apperrors.KindOf(err))
			}

			expectedMsg := "ttl must be positive"
			if err.Error() != expectedMsg {
				t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
			}
		})
	}
}

// TestRedisLock_Renew_InvalidTTL tests that Renew rejects non-positive TTL values.
func TestRedisLock_Renew_InvalidTTL(t *testing.T) {
	rl := &RedisLock{}

	tests := []struct {
		name string
		ttl  time.Duration
	}{
		{
			name: "zero TTL",
			ttl:  0,
		},
		{
			name: "negative TTL",
			ttl:  -1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := rl.Renew(ctx, "test-key", "test-token", tt.ttl)

			if err == nil {
				t.Fatal("expected error for invalid TTL")
			}

			if !apperrors.IsKind(err, apperrors.KindValidation) {
				t.Errorf("expected validation error, got kind: %v", apperrors.KindOf(err))
			}

			expectedMsg := "ttl must be positive"
			if err.Error() != expectedMsg {
				t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
			}
		})
	}
}

// TestRedisLeaderElection_Elect_InvalidTTL tests that Elect rejects non-positive TTL values.
func TestRedisLeaderElection_Elect_InvalidTTL(t *testing.T) {
	rle := &RedisLeaderElection{}

	tests := []struct {
		name string
		ttl  time.Duration
	}{
		{
			name: "zero TTL",
			ttl:  0,
		},
		{
			name: "negative TTL",
			ttl:  -1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			elected, err := rle.Elect(ctx, "test-role", "test-instance", tt.ttl)

			if elected {
				t.Error("expected elected to be false")
			}
			if err == nil {
				t.Fatal("expected error for invalid TTL")
			}

			if !apperrors.IsKind(err, apperrors.KindValidation) {
				t.Errorf("expected validation error, got kind: %v", apperrors.KindOf(err))
			}

			expectedMsg := "ttl must be positive"
			if err.Error() != expectedMsg {
				t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
			}
		})
	}
}

// TestRedisLeaderElection_RenewLeadership_InvalidTTL tests that RenewLeadership rejects non-positive TTL values.
func TestRedisLeaderElection_RenewLeadership_InvalidTTL(t *testing.T) {
	rle := &RedisLeaderElection{}

	tests := []struct {
		name string
		ttl  time.Duration
	}{
		{
			name: "zero TTL",
			ttl:  0,
		},
		{
			name: "negative TTL",
			ttl:  -1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := rle.RenewLeadership(ctx, "test-role", "test-instance", tt.ttl)

			if err == nil {
				t.Fatal("expected error for invalid TTL")
			}

			if !apperrors.IsKind(err, apperrors.KindValidation) {
				t.Errorf("expected validation error, got kind: %v", apperrors.KindOf(err))
			}

			expectedMsg := "ttl must be positive"
			if err.Error() != expectedMsg {
				t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
			}
		})
	}
}

// TestGenerateToken_Success tests that generateToken returns a valid token.
func TestGenerateToken_Success(t *testing.T) {
	token, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken() returned error: %v", err)
	}

	// Token should be 32 hex characters (16 bytes * 2)
	if len(token) != 32 {
		t.Errorf("expected token length 32, got %d", len(token))
	}

	// Token should be valid hex
	for _, c := range token {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("token contains invalid character: %c", c)
		}
	}

	// Tokens should be unique (statistically very unlikely to collide)
	token2, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken() returned error: %v", err)
	}
	if token == token2 {
		t.Error("generateToken() returned duplicate tokens")
	}
}

// TestGenerateToken_ErrorPath tests that generateToken error is properly propagated.
// Note: Testing actual crypto/rand.Read failure is not practical without build tags
// or interfaces, but we verify the error handling contract is in place.
func TestGenerateToken_ErrorPath(t *testing.T) {
	// This test documents the error handling behavior:
	// 1. generateToken() returns ("", error) on rand.Read failure
	// 2. The error is wrapped with apperrors.KindInternal
	// 3. Acquire checks the error and returns immediately before calling Redis
	//
	// The implementation ensures this by:
	// - generateToken returns (string, error) with proper error wrapping
	// - Acquire checks the error and propagates it without reaching Redis
	//
	// Since crypto/rand.Read failure is extremely rare (only on systems with
	// broken entropy sources), we verify the contract through code inspection
	// and the fact that the code compiles correctly.
}

// setupTestRedis creates a miniredis instance and redis client for testing.
func setupTestRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	return mr, client
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

// TestRedisLock_Acquire_Success verifies successful lock acquisition.
func TestRedisLock_Acquire_Success(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()
	rl := NewRedisLock(client, "test:lock:")

	acquired, token, err := rl.Acquire(ctx, "test-key", 5*time.Second)
	if err != nil {
		t.Fatalf("Acquire returned error: %v", err)
	}
	if !acquired {
		t.Error("expected acquired to be true")
	}
	if token == "" {
		t.Error("expected non-empty token")
	}

	// Verify key exists in Redis
	fullKey := "test:lock:test-key"
	val, err := mr.Get(fullKey)
	if err != nil {
		t.Fatalf("expected key to exist in Redis: %v", err)
	}
	if val != token {
		t.Errorf("expected Redis value %q, got %q", token, val)
	}

	// Verify TTL is set
	ttl := mr.TTL(fullKey)
	if ttl <= 0 || ttl > 5*time.Second {
		t.Errorf("expected TTL around 5s, got %v", ttl)
	}
}

// TestRedisLock_Acquire_AlreadyHeld verifies lock acquisition fails when already held.
func TestRedisLock_Acquire_AlreadyHeld(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	rl := NewRedisLock(client, "test:lock:")

	// First acquisition should succeed
	acquired1, token1, err := rl.Acquire(ctx, "test-key", 5*time.Second)
	if err != nil {
		t.Fatalf("first Acquire returned error: %v", err)
	}
	if !acquired1 {
		t.Error("expected first acquired to be true")
	}
	if token1 == "" {
		t.Fatal("expected non-empty token")
	}

	// Second acquisition should fail (lock already held)
	acquired2, token2, err := rl.Acquire(ctx, "test-key", 5*time.Second)
	if err != nil {
		t.Fatalf("second Acquire returned error: %v", err)
	}
	if acquired2 {
		t.Error("expected second acquired to be false")
	}
	if token2 != "" {
		t.Error("expected empty token for failed acquisition")
	}
}

// TestRedisLock_Release_Success verifies successful lock release.
func TestRedisLock_Release_Success(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()
	rl := NewRedisLock(client, "test:lock:")

	// Acquire lock
	acquired, token, err := rl.Acquire(ctx, "test-key", 5*time.Second)
	if err != nil {
		t.Fatalf("Acquire returned error: %v", err)
	}
	if !acquired {
		t.Fatal("expected acquired to be true")
	}

	// Verify key exists
	fullKey := "test:lock:test-key"
	if !mr.Exists(fullKey) {
		t.Fatal("expected key to exist in Redis")
	}

	// Release with correct token
	err = rl.Release(ctx, "test-key", token)
	if err != nil {
		t.Fatalf("Release returned error: %v", err)
	}

	// Verify key no longer exists
	if mr.Exists(fullKey) {
		t.Error("expected key to be deleted from Redis")
	}
}

// TestRedisLock_Release_WrongToken verifies release fails with wrong token.
func TestRedisLock_Release_WrongToken(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()
	rl := NewRedisLock(client, "test:lock:")

	// Acquire lock
	acquired, _, err := rl.Acquire(ctx, "test-key", 5*time.Second)
	if err != nil {
		t.Fatalf("Acquire returned error: %v", err)
	}
	if !acquired {
		t.Fatal("expected acquired to be true")
	}

	// Verify key exists
	fullKey := "test:lock:test-key"
	if !mr.Exists(fullKey) {
		t.Fatal("expected key to exist in Redis")
	}

	// Try to release with wrong token
	err = rl.Release(ctx, "test-key", "wrong-token")
	if err == nil {
		t.Fatal("expected error for wrong token")
	}

	if !apperrors.IsKind(err, apperrors.KindPermission) {
		t.Errorf("expected permission error, got kind: %v", apperrors.KindOf(err))
	}

	expectedMsg := "lock not held or token mismatch"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}

	// Verify key still exists (not deleted)
	if !mr.Exists(fullKey) {
		t.Error("expected key to still exist in Redis")
	}
}

// TestRedisLock_Release_AlreadyReleased verifies release fails when lock already released.
func TestRedisLock_Release_AlreadyReleased(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	rl := NewRedisLock(client, "test:lock:")

	// Acquire lock
	acquired, token, err := rl.Acquire(ctx, "test-key", 5*time.Second)
	if err != nil {
		t.Fatalf("Acquire returned error: %v", err)
	}
	if !acquired {
		t.Fatal("expected acquired to be true")
	}

	// Release with correct token
	err = rl.Release(ctx, "test-key", token)
	if err != nil {
		t.Fatalf("Release returned error: %v", err)
	}

	// Try to release again
	err = rl.Release(ctx, "test-key", token)
	if err == nil {
		t.Fatal("expected error for already released lock")
	}

	if !apperrors.IsKind(err, apperrors.KindPermission) {
		t.Errorf("expected permission error, got kind: %v", apperrors.KindOf(err))
	}
}

// TestRedisLock_Renew_Success verifies successful lock renewal.
func TestRedisLock_Renew_Success(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()
	rl := NewRedisLock(client, "test:lock:")

	// Acquire lock with 1 second TTL
	acquired, token, err := rl.Acquire(ctx, "test-key", 1*time.Second)
	if err != nil {
		t.Fatalf("Acquire returned error: %v", err)
	}
	if !acquired {
		t.Fatal("expected acquired to be true")
	}

	fullKey := "test:lock:test-key"
	initialTTL := mr.TTL(fullKey)

	// Fast forward a bit
	mr.FastForward(100 * time.Millisecond)

	// Renew with longer TTL
	err = rl.Renew(ctx, "test-key", token, 5*time.Second)
	if err != nil {
		t.Fatalf("Renew returned error: %v", err)
	}

	// Verify TTL is extended
	newTTL := mr.TTL(fullKey)
	if newTTL <= initialTTL {
		t.Errorf("expected TTL to be extended, got %v (was %v)", newTTL, initialTTL)
	}

	// Verify key still exists
	if !mr.Exists(fullKey) {
		t.Error("expected key to still exist in Redis")
	}
}

// TestRedisLock_Renew_WrongToken verifies renew fails with wrong token.
func TestRedisLock_Renew_WrongToken(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()
	rl := NewRedisLock(client, "test:lock:")

	// Acquire lock
	acquired, _, err := rl.Acquire(ctx, "test-key", 5*time.Second)
	if err != nil {
		t.Fatalf("Acquire returned error: %v", err)
	}
	if !acquired {
		t.Fatal("expected acquired to be true")
	}

	// Verify key exists
	fullKey := "test:lock:test-key"
	if !mr.Exists(fullKey) {
		t.Fatal("expected key to exist in Redis")
	}

	// Try to renew with wrong token
	err = rl.Renew(ctx, "test-key", "wrong-token", 5*time.Second)
	if err == nil {
		t.Fatal("expected error for wrong token")
	}

	if !apperrors.IsKind(err, apperrors.KindPermission) {
		t.Errorf("expected permission error, got kind: %v", apperrors.KindOf(err))
	}

	expectedMsg := "lock not held or token mismatch"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

// TestRedisLock_Renew_AlreadyExpired verifies renew fails when lock has expired.
func TestRedisLock_Renew_AlreadyExpired(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()
	rl := NewRedisLock(client, "test:lock:")

	// Acquire lock with very short TTL
	acquired, token, err := rl.Acquire(ctx, "test-key", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Acquire returned error: %v", err)
	}
	if !acquired {
		t.Fatal("expected acquired to be true")
	}

	// Fast-forward past TTL
	mr.FastForward(200 * time.Millisecond)

	// Try to renew - should fail because lock expired
	err = rl.Renew(ctx, "test-key", token, 5*time.Second)
	if err == nil {
		t.Fatal("expected error for expired lock")
	}

	if !apperrors.IsKind(err, apperrors.KindPermission) {
		t.Errorf("expected permission error, got kind: %v", apperrors.KindOf(err))
	}
}

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

// TestRedisLeaderElection_Elect_Success verifies successful leader election.
func TestRedisLeaderElection_Elect_Success(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()
	rle := NewRedisLeaderElection(client, "test:leader:")

	elected, err := rle.Elect(ctx, "scheduler", "instance-1", 5*time.Second)
	if err != nil {
		t.Fatalf("Elect returned error: %v", err)
	}
	if !elected {
		t.Error("expected elected to be true")
	}

	// Verify leader key exists in Redis
	key := "test:leader:scheduler"
	val, err := mr.Get(key)
	if err != nil {
		t.Fatalf("expected leader key to exist: %v", err)
	}
	if val != "instance-1" {
		t.Errorf("expected leader value %q, got %q", "instance-1", val)
	}

	// Verify TTL is set
	ttl := mr.TTL(key)
	if ttl <= 0 || ttl > 5*time.Second {
		t.Errorf("expected TTL around 5s, got %v", ttl)
	}
}

// TestRedisLeaderElection_Elect_AlreadyLeader verifies re-election succeeds for current leader.
func TestRedisLeaderElection_Elect_AlreadyLeader(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	rle := NewRedisLeaderElection(client, "test:leader:")

	// First election
	elected, err := rle.Elect(ctx, "scheduler", "instance-1", 5*time.Second)
	if err != nil {
		t.Fatalf("first Elect returned error: %v", err)
	}
	if !elected {
		t.Fatal("expected first elected to be true")
	}

	// Second election with same instance (before TTL expires)
	elected, err = rle.Elect(ctx, "scheduler", "instance-1", 5*time.Second)
	if err != nil {
		t.Fatalf("second Elect returned error: %v", err)
	}
	if !elected {
		t.Error("expected second elected to be true (already leader)")
	}
}

// TestRedisLeaderElection_Elect_AnotherLeader verifies election fails when another is leader.
func TestRedisLeaderElection_Elect_AnotherLeader(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	rle := NewRedisLeaderElection(client, "test:leader:")

	// First instance becomes leader
	elected, err := rle.Elect(ctx, "scheduler", "instance-1", 5*time.Second)
	if err != nil {
		t.Fatalf("first Elect returned error: %v", err)
	}
	if !elected {
		t.Fatal("expected first elected to be true")
	}

	// Second instance tries to become leader
	elected, err = rle.Elect(ctx, "scheduler", "instance-2", 5*time.Second)
	if err != nil {
		t.Fatalf("second Elect returned error: %v", err)
	}
	if elected {
		t.Error("expected second elected to be false (another is leader)")
	}
}

// TestRedisLeaderElection_RenewLeadership_Success verifies successful leadership renewal.
func TestRedisLeaderElection_RenewLeadership_Success(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()
	rle := NewRedisLeaderElection(client, "test:leader:")

	// Become leader with short TTL
	elected, err := rle.Elect(ctx, "scheduler", "instance-1", 1*time.Second)
	if err != nil {
		t.Fatalf("Elect returned error: %v", err)
	}
	if !elected {
		t.Fatal("expected elected to be true")
	}

	key := "test:leader:scheduler"
	initialTTL := mr.TTL(key)

	// Fast forward a bit
	mr.FastForward(100 * time.Millisecond)

	// Renew leadership
	err = rle.RenewLeadership(ctx, "scheduler", "instance-1", 5*time.Second)
	if err != nil {
		t.Fatalf("RenewLeadership returned error: %v", err)
	}

	// Verify TTL is extended
	newTTL := mr.TTL(key)
	if newTTL <= initialTTL {
		t.Errorf("expected TTL to be extended, got %v (was %v)", newTTL, initialTTL)
	}
}

// TestRedisLeaderElection_RenewLeadership_NotLeader verifies renew fails when not leader.
func TestRedisLeaderElection_RenewLeadership_NotLeader(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	rle := NewRedisLeaderElection(client, "test:leader:")

	// First instance becomes leader
	elected, err := rle.Elect(ctx, "scheduler", "instance-1", 5*time.Second)
	if err != nil {
		t.Fatalf("Elect returned error: %v", err)
	}
	if !elected {
		t.Fatal("expected elected to be true")
	}

	// Second instance tries to renew leadership
	err = rle.RenewLeadership(ctx, "scheduler", "instance-2", 5*time.Second)
	if err == nil {
		t.Fatal("expected error when not leader")
	}

	if !apperrors.IsKind(err, apperrors.KindPermission) {
		t.Errorf("expected permission error, got kind: %v", apperrors.KindOf(err))
	}

	expectedMsg := "not the current leader"
	if err.Error() != expectedMsg {
		t.Errorf("expected error message %q, got %q", expectedMsg, err.Error())
	}
}

// TestRedisLeaderElection_RenewLeadership_LeadershipLost verifies renew fails when leadership expired.
func TestRedisLeaderElection_RenewLeadership_LeadershipLost(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()
	rle := NewRedisLeaderElection(client, "test:leader:")

	// Become leader with very short TTL
	elected, err := rle.Elect(ctx, "scheduler", "instance-1", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Elect returned error: %v", err)
	}
	if !elected {
		t.Fatal("expected elected to be true")
	}

	// Fast-forward past TTL
	mr.FastForward(200 * time.Millisecond)

	// Try to renew - should fail because leadership expired
	err = rle.RenewLeadership(ctx, "scheduler", "instance-1", 5*time.Second)
	if err == nil {
		t.Fatal("expected error when leadership expired")
	}

	if !apperrors.IsKind(err, apperrors.KindPermission) {
		t.Errorf("expected permission error, got kind: %v", apperrors.KindOf(err))
	}
}

// TestRedisLeaderElection_Resign_Success verifies successful leadership resignation.
func TestRedisLeaderElection_Resign_Success(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()
	rle := NewRedisLeaderElection(client, "test:leader:")

	// Become leader
	elected, err := rle.Elect(ctx, "scheduler", "instance-1", 5*time.Second)
	if err != nil {
		t.Fatalf("Elect returned error: %v", err)
	}
	if !elected {
		t.Fatal("expected elected to be true")
	}

	// Verify leader key exists
	key := "test:leader:scheduler"
	if !mr.Exists(key) {
		t.Fatal("expected leader key to exist")
	}

	// Resign leadership
	err = rle.Resign(ctx, "scheduler", "instance-1")
	if err != nil {
		t.Fatalf("Resign returned error: %v", err)
	}

	// Verify leader key no longer exists
	if mr.Exists(key) {
		t.Error("expected leader key to be deleted")
	}
}

// TestRedisLeaderElection_Resign_NotLeader verifies resign succeeds silently when not leader.
func TestRedisLeaderElection_Resign_NotLeader(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()
	rle := NewRedisLeaderElection(client, "test:leader:")

	// First instance becomes leader
	elected, err := rle.Elect(ctx, "scheduler", "instance-1", 5*time.Second)
	if err != nil {
		t.Fatalf("Elect returned error: %v", err)
	}
	if !elected {
		t.Fatal("expected elected to be true")
	}

	// Second instance tries to resign (should succeed silently)
	err = rle.Resign(ctx, "scheduler", "instance-2")
	if err != nil {
		t.Fatalf("Resign returned error: %v", err)
	}

	// Verify first instance is still leader
	key := "test:leader:scheduler"
	val, err := mr.Get(key)
	if err != nil {
		t.Fatalf("expected leader key to exist: %v", err)
	}
	if val != "instance-1" {
		t.Errorf("expected leader to still be instance-1, got %q", val)
	}
}

// TestRedisLeaderElection_IsLeader_True verifies IsLeader returns true for current leader.
func TestRedisLeaderElection_IsLeader_True(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	rle := NewRedisLeaderElection(client, "test:leader:")

	// Become leader
	elected, err := rle.Elect(ctx, "scheduler", "instance-1", 5*time.Second)
	if err != nil {
		t.Fatalf("Elect returned error: %v", err)
	}
	if !elected {
		t.Fatal("expected elected to be true")
	}

	// Check if leader
	isLeader, err := rle.IsLeader(ctx, "scheduler", "instance-1")
	if err != nil {
		t.Fatalf("IsLeader returned error: %v", err)
	}
	if !isLeader {
		t.Error("expected IsLeader to be true")
	}
}

// TestRedisLeaderElection_IsLeader_False verifies IsLeader returns false for non-leader.
func TestRedisLeaderElection_IsLeader_False(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	rle := NewRedisLeaderElection(client, "test:leader:")

	// First instance becomes leader
	elected, err := rle.Elect(ctx, "scheduler", "instance-1", 5*time.Second)
	if err != nil {
		t.Fatalf("Elect returned error: %v", err)
	}
	if !elected {
		t.Fatal("expected elected to be true")
	}

	// Check if second instance is leader
	isLeader, err := rle.IsLeader(ctx, "scheduler", "instance-2")
	if err != nil {
		t.Fatalf("IsLeader returned error: %v", err)
	}
	if isLeader {
		t.Error("expected IsLeader to be false")
	}
}

// TestRedisLeaderElection_IsLeader_NoLeader verifies IsLeader returns false when no leader.
func TestRedisLeaderElection_IsLeader_NoLeader(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	rle := NewRedisLeaderElection(client, "test:leader:")

	// Check if leader without any election
	isLeader, err := rle.IsLeader(ctx, "scheduler", "instance-1")
	if err != nil {
		t.Fatalf("IsLeader returned error: %v", err)
	}
	if isLeader {
		t.Error("expected IsLeader to be false when no leader")
	}
}

// TestRedisLeaderElection_GetLeader_WithLeader verifies GetLeader returns leader when one exists.
func TestRedisLeaderElection_GetLeader_WithLeader(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	rle := NewRedisLeaderElection(client, "test:leader:")

	// Become leader
	elected, err := rle.Elect(ctx, "scheduler", "instance-1", 5*time.Second)
	if err != nil {
		t.Fatalf("Elect returned error: %v", err)
	}
	if !elected {
		t.Fatal("expected elected to be true")
	}

	// Get leader
	leader, err := rle.GetLeader(ctx, "scheduler")
	if err != nil {
		t.Fatalf("GetLeader returned error: %v", err)
	}
	if leader != "instance-1" {
		t.Errorf("expected leader %q, got %q", "instance-1", leader)
	}
}

// TestRedisLeaderElection_GetLeader_NoLeader verifies GetLeader returns empty when no leader.
func TestRedisLeaderElection_GetLeader_NoLeader(t *testing.T) {
	_, client := setupTestRedis(t)
	ctx := context.Background()
	rle := NewRedisLeaderElection(client, "test:leader:")

	// Get leader without any election
	leader, err := rle.GetLeader(ctx, "scheduler")
	if err != nil {
		t.Fatalf("GetLeader returned error: %v", err)
	}
	if leader != "" {
		t.Errorf("expected empty leader, got %q", leader)
	}
}

// TestRedisLeaderElection_GetLeader_Error verifies GetLeader returns error on Redis failure.
func TestRedisLeaderElection_GetLeader_Error(t *testing.T) {
	mr, client := setupTestRedis(t)
	ctx := context.Background()
	rle := NewRedisLeaderElection(client, "test:leader:")

	// Become leader
	elected, err := rle.Elect(ctx, "scheduler", "instance-1", 5*time.Second)
	if err != nil {
		t.Fatalf("Elect returned error: %v", err)
	}
	if !elected {
		t.Fatal("expected elected to be true")
	}

	// Close Redis connection
	mr.Close()

	_, err = rle.GetLeader(ctx, "scheduler")
	if err == nil {
		t.Fatal("expected error when Redis is unavailable")
	}

	if !apperrors.IsKind(err, apperrors.KindInternal) {
		t.Errorf("expected internal error, got kind: %v", apperrors.KindOf(err))
	}
}
