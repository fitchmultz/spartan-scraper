// Package manage tests OAuth CLI management flows.
//
// Purpose:
// - Provide shared helpers for scenario-focused OAuth CLI test suites.
//
// Responsibilities:
// - Create temp data dirs and seed auth profiles for config, discovery, initiate, and token tests.
//
// Scope:
// - Test-only helpers for OAuth CLI command coverage.
//
// Usage:
// - Used by auth_oauth_*_test.go files in this package.
//
// Invariants/Assumptions:
// - Helpers must remain side-effect free outside their temp data dir.
package manage

import (
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
)

func setupTestDataDir(t *testing.T) string {
	t.Helper()
	return t.TempDir()
}

func createTestProfile(t *testing.T, dataDir, name string, oauth2 *auth.OAuth2Config) {
	t.Helper()
	profile := auth.Profile{
		Name:   name,
		OAuth2: oauth2,
	}
	if err := auth.UpsertProfile(dataDir, profile); err != nil {
		t.Fatalf("failed to create test profile: %v", err)
	}
}
