// Package fsutil provides focused regression coverage for output-path containment helpers.
//
// Purpose:
// - Verify output-path validation handles traversal rejection and symlink-normalized roots correctly.
//
// Responsibilities:
// - Confirm paths inside the allowed root resolve successfully.
// - Confirm escaping the allowed root is rejected.
// - Confirm symlinked roots and canonical targets are treated as the same containment boundary.
//
// Scope:
// - `ResolvePathWithinRoot` behavior only.
//
// Usage:
// - Run with `go test ./internal/fsutil`.
//
// Invariants/Assumptions:
// - Tests use temp directories and do not rely on platform-global symlink layouts.
// - Symlink-specific coverage skips when the platform cannot create test symlinks.
package fsutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolvePathWithinRoot(t *testing.T) {
	t.Run("accepts relative path within root", func(t *testing.T) {
		root := t.TempDir()
		resolved, err := ResolvePathWithinRoot(root, filepath.Join("exports", "result.json"))
		if err != nil {
			t.Fatalf("ResolvePathWithinRoot() error = %v", err)
		}
		want, err := resolvePathForContainment(filepath.Join(root, "exports", "result.json"))
		if err != nil {
			t.Fatalf("resolvePathForContainment() error = %v", err)
		}
		if resolved != want {
			t.Fatalf("ResolvePathWithinRoot() = %q, want %q", resolved, want)
		}
	})

	t.Run("rejects traversal outside root", func(t *testing.T) {
		root := t.TempDir()
		_, err := ResolvePathWithinRoot(root, filepath.Join("..", "escape.json"))
		if err == nil || !strings.Contains(err.Error(), "output path must stay within") {
			t.Fatalf("expected containment error, got %v", err)
		}
	})

	t.Run("accepts canonical target under symlinked root", func(t *testing.T) {
		realRoot := filepath.Join(t.TempDir(), "real-root")
		if err := os.MkdirAll(realRoot, 0o755); err != nil {
			t.Fatalf("MkdirAll(realRoot): %v", err)
		}
		aliasParent := t.TempDir()
		aliasRoot := filepath.Join(aliasParent, "alias-root")
		if err := os.Symlink(realRoot, aliasRoot); err != nil {
			if os.IsPermission(err) || strings.Contains(strings.ToLower(err.Error()), "not permitted") {
				t.Skipf("symlink creation unavailable: %v", err)
			}
			t.Fatalf("Symlink(aliasRoot): %v", err)
		}

		resolved, err := ResolvePathWithinRoot(aliasRoot, filepath.Join(realRoot, "exports", "result.json"))
		if err != nil {
			t.Fatalf("ResolvePathWithinRoot() error = %v", err)
		}
		want, err := resolvePathForContainment(filepath.Join(realRoot, "exports", "result.json"))
		if err != nil {
			t.Fatalf("resolvePathForContainment() error = %v", err)
		}
		if resolved != want {
			t.Fatalf("ResolvePathWithinRoot() = %q, want %q", resolved, want)
		}
	})
}
