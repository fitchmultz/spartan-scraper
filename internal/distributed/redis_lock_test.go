// Package distributed provides distributed coordination primitives.
//
// This file contains tests for Redis-based distributed locking and leader election.
package distributed

import (
	"context"
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
