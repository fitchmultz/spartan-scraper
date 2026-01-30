// Package scheduler provides test utilities for scheduler tests.
//
// This file is responsible for:
// - Providing setupTestManager helper for creating test job managers
// - Creating temporary data directories for test isolation
// - Initializing stores and managers with test defaults
//
// This file does NOT handle:
// - Production use (test-only utilities)
// - Test assertions or test cases
//
// Invariants:
// - Creates temp dir via t.TempDir() for test isolation
// - Cleanup function closes the store
// - Uses test-specific timeouts and limits for faster tests
package scheduler

import (
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func setupTestManager(t *testing.T) (*jobs.Manager, *store.Store, func()) {
	t.Helper()
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}

	m := jobs.NewManager(
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
	)

	cleanup := func() {
		st.Close()
	}

	return m, st, cleanup
}
