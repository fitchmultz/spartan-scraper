// Package jobs provides test helper functions for the jobs package.
// This file contains utilities for setting up test Manager instances
// with isolated storage and appropriate test defaults.
//
// Responsibilities:
// - Creating test Manager instances with temporary data directories
// - Providing cleanup functions for test isolation
// - Setting up test-appropriate configuration (timeouts, concurrency limits)
//
// This file does NOT:
// - Contain actual test cases (see *_test.go files)
// - Perform assertions or test validations
// - Modify production code behavior
//
// Invariants:
// - setupTestManager returns a cleanup function that must be called via defer
// - Test managers use t.TempDir() for automatic cleanup
// - Adaptive rate limiting is disabled in tests (nil config)

package jobs

import (
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func setupTestManager(t *testing.T) (*Manager, *store.Store, func()) {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	m := NewManager(
		st,
		dataDir,
		"TestAgent/1.0",
		30*time.Second,
		2,
		10,
		20,
		3,
		100*time.Millisecond,
		10*1024*1024,
		false,
		fetch.DefaultCircuitBreakerConfig(),
		nil, // no adaptive rate limiting in tests
	)

	cleanup := func() {
		st.Close()
	}

	return m, st, cleanup
}
