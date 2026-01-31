// Package users provides user management, workspace membership, and RBAC.
//
// This file contains shared test helpers for the users package.
package users

import (
	"os"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/store"
)

// setupTestStore creates a temporary store for testing.
// It returns the store and a cleanup function that must be called after use.
func setupTestStore(t *testing.T) (*store.Store, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "users_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	s, err := store.Open(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to open store: %v", err)
	}

	cleanup := func() {
		s.Close()
		os.RemoveAll(tmpDir)
	}

	return s, cleanup
}
