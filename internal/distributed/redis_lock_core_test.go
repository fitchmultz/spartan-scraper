// Package distributed provides distributed coordination primitives.
//
// This file contains tests for Redis-based distributed locking.
package distributed

import (
	"context"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
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
