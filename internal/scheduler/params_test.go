// Package scheduler provides tests for typed schedule helper functions.
//
// Purpose:
// - Verify schedule helper behavior after the params-to-spec cutover.
//
// Responsibilities:
// - Cover execution config extraction.
// - Cover target URL derivation.
// - Cover auth resolution with and without auth profiles.
//
// Scope:
// - Helper-level tests for typed schedule decoding only.
//
// Usage:
// - Run with `go test ./internal/scheduler`.
//
// Invariants/Assumptions:
// - Typed schedules reuse the same V1 specs as persisted jobs.
// - Auth profiles are resolved at execution time for schedules.
package scheduler

import (
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestExecutionSpecForSchedule(t *testing.T) {
	schedule := testScrapeSchedule("https://example.com")
	spec := schedule.Spec.(model.ScrapeSpecV1)
	spec.Execution.TimeoutSeconds = 45
	spec.Execution.AuthProfile = "profile-a"
	schedule.Spec = spec

	exec, err := executionSpecForSchedule(schedule)
	if err != nil {
		t.Fatalf("executionSpecForSchedule() error = %v", err)
	}
	if exec.TimeoutSeconds != 45 {
		t.Fatalf("timeout = %d, want 45", exec.TimeoutSeconds)
	}
	if exec.AuthProfile != "profile-a" {
		t.Fatalf("authProfile = %q, want profile-a", exec.AuthProfile)
	}
}

func TestTargetURLForSchedule(t *testing.T) {
	research := testResearchSchedule("pricing", []string{"https://example.com", "https://example.com/docs"}, 2, 100)
	if got := targetURLForSchedule(research); got != "https://example.com" {
		t.Fatalf("targetURLForSchedule() = %q, want https://example.com", got)
	}
}

func TestResolveScheduleAuthWithoutProfile(t *testing.T) {
	schedule := testScrapeSchedule("https://example.com")
	spec := schedule.Spec.(model.ScrapeSpecV1)
	spec.Execution.Auth.Headers = map[string]string{"X-Test": "value"}
	schedule.Spec = spec

	authOptions, err := resolveScheduleAuth(schedule, t.TempDir(), auth.EnvOverrides{})
	if err != nil {
		t.Fatalf("resolveScheduleAuth() error = %v", err)
	}
	if authOptions.Headers["X-Test"] != "value" {
		t.Fatalf("header X-Test = %q, want value", authOptions.Headers["X-Test"])
	}
}

func TestResolveScheduleAuthWithProfile(t *testing.T) {
	dataDir := t.TempDir()
	if err := auth.UpsertProfile(dataDir, auth.Profile{
		Name:    "base",
		Headers: []auth.HeaderKV{{Key: "Authorization", Value: "Bearer base-token"}},
	}); err != nil {
		t.Fatalf("UpsertProfile() error = %v", err)
	}

	schedule := testScrapeSchedule("https://example.com")
	spec := schedule.Spec.(model.ScrapeSpecV1)
	spec.Execution.AuthProfile = "base"
	spec.Execution.Auth.Headers = map[string]string{"X-Override": "present"}
	schedule.Spec = spec

	authOptions, err := resolveScheduleAuth(schedule, dataDir, auth.EnvOverrides{})
	if err != nil {
		t.Fatalf("resolveScheduleAuth() error = %v", err)
	}
	if authOptions.Headers["Authorization"] != "Bearer base-token" {
		t.Fatalf("Authorization = %q, want Bearer base-token", authOptions.Headers["Authorization"])
	}
	if authOptions.Headers["X-Override"] != "present" {
		t.Fatalf("X-Override = %q, want present", authOptions.Headers["X-Override"])
	}
}
