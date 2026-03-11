// Package manage provides tests for schedule CLI contract cutover behavior.
//
// Purpose:
// - Verify schedule CLI creation writes typed schedule specs instead of params bags.
//
// Responsibilities:
// - Exercise `spartan schedule add` path against local scheduler storage.
// - Assert the stored schedule uses specVersion + typed spec.
//
// Scope:
// - CLI schedule add coverage only.
//
// Usage:
// - Run with `go test ./internal/cli/manage`.
//
// Invariants/Assumptions:
// - Schedule add performs a hard cutover to the typed schedule contract.
// - List output behavior is unchanged and covered elsewhere.
package manage

import (
	"context"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
)

func TestRunScheduleAddPersistsTypedSpec(t *testing.T) {
	dataDir := t.TempDir()
	cfg := config.Config{
		DataDir:            dataDir,
		RequestTimeoutSecs: 30,
	}

	code := RunSchedule(context.Background(), cfg, []string{
		"add",
		"--kind", "scrape",
		"--interval", "60",
		"--url", "https://example.com",
		"--timeout", "30",
	})
	if code != 0 {
		t.Fatalf("RunSchedule() = %d, want 0", code)
	}

	schedules, err := scheduler.LoadAll(dataDir)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if len(schedules) != 1 {
		t.Fatalf("len(schedules) = %d, want 1", len(schedules))
	}
	if schedules[0].SpecVersion != model.JobSpecVersion1 {
		t.Fatalf("SpecVersion = %d, want %d", schedules[0].SpecVersion, model.JobSpecVersion1)
	}
	if _, ok := schedules[0].Spec.(model.ScrapeSpecV1); !ok {
		t.Fatalf("Spec type = %T, want model.ScrapeSpecV1", schedules[0].Spec)
	}
}
