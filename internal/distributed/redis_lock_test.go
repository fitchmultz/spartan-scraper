// Package distributed provides distributed coordination primitives.
//
// This file contains tests for Redis-based distributed locking and leader election.
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
