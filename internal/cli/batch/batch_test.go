// Package batch contains shared helpers for scenario-focused batch CLI tests.
//
// Purpose:
// - Keep batch CLI test suites small while preserving shared store helpers.
//
// Responsibilities:
// - Provide common batch store inspection helpers reused by parsing, output, and command suites.
//
// Scope:
// - Test-only helpers for batch CLI coverage.
//
// Usage:
// - Used by batch_*_test.go files in this package.
//
// Invariants/Assumptions:
// - Helpers operate on temp-data stores owned by the calling test.
package batch

import (
	"context"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func latestBatchJobSpec(t *testing.T, dataDir string) map[string]interface{} {
	t.Helper()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer st.Close()

	jobs, err := st.List(context.Background())
	if err != nil {
		t.Fatalf("failed to list jobs: %v", err)
	}
	if len(jobs) == 0 {
		t.Fatal("expected at least one job")
	}
	return jobs[0].SpecMap()
}
