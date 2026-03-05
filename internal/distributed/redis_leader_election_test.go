// Package distributed provides distributed coordination primitives.
//
// This file contains tests for Redis-based leader election.
package distributed

import (
	"context"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

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
