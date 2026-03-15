// Package watch verifies deterministic watch artifact persistence.
//
// Purpose:
// - Cover rotation, resolution, and cleanup for watch-owned artifacts.
//
// Responsibilities:
// - Ensure current screenshots rotate into previous screenshots.
// - Confirm resolved artifact metadata exposes stable filenames and MIME types.
// - Verify watch artifact directories are removed on cleanup.
//
// Scope:
// - ArtifactStore behavior only.
//
// Usage:
// - Run with `go test ./internal/watch`.
//
// Invariants/Assumptions:
// - Test fixtures use small fake image headers sufficient for MIME sniffing.
package watch

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArtifactStoreReplaceCurrentRotatesPrevious(t *testing.T) {
	tempDir := t.TempDir()
	store := NewArtifactStore(tempDir)
	watchID := "watch-123"

	firstSource := filepath.Join(tempDir, "first.png")
	if err := os.WriteFile(firstSource, append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, []byte("first")...), 0o600); err != nil {
		t.Fatalf("write first source: %v", err)
	}
	firstArtifact, previousArtifact, err := store.ReplaceCurrent(watchID, firstSource)
	if err != nil {
		t.Fatalf("replace current (first): %v", err)
	}
	if previousArtifact != nil {
		t.Fatalf("expected no previous artifact on first write, got %#v", previousArtifact)
	}
	if firstArtifact.Kind != ArtifactKindCurrentScreenshot {
		t.Fatalf("expected current artifact kind, got %s", firstArtifact.Kind)
	}
	if firstArtifact.ContentType != "image/png" {
		t.Fatalf("expected image/png, got %s", firstArtifact.ContentType)
	}

	secondSource := filepath.Join(tempDir, "second.png")
	if err := os.WriteFile(secondSource, append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, []byte("second")...), 0o600); err != nil {
		t.Fatalf("write second source: %v", err)
	}
	currentArtifact, rotatedPrevious, err := store.ReplaceCurrent(watchID, secondSource)
	if err != nil {
		t.Fatalf("replace current (second): %v", err)
	}
	if rotatedPrevious == nil {
		t.Fatal("expected previous artifact after rotating current screenshot")
	}
	if rotatedPrevious.Kind != ArtifactKindPreviousScreenshot {
		t.Fatalf("expected previous artifact kind, got %s", rotatedPrevious.Kind)
	}
	if currentArtifact.Filename == rotatedPrevious.Filename {
		t.Fatalf("expected current and previous filenames to differ, got %s", currentArtifact.Filename)
	}

	currentBytes, err := os.ReadFile(currentArtifact.Path)
	if err != nil {
		t.Fatalf("read current artifact: %v", err)
	}
	if string(currentBytes[len(currentBytes)-6:]) != "second" {
		t.Fatalf("expected current artifact to contain second payload, got %q", string(currentBytes))
	}
	previousBytes, err := os.ReadFile(rotatedPrevious.Path)
	if err != nil {
		t.Fatalf("read previous artifact: %v", err)
	}
	if string(previousBytes[len(previousBytes)-5:]) != "first" {
		t.Fatalf("expected previous artifact to contain first payload, got %q", string(previousBytes))
	}
}

func TestArtifactStoreVisualDiffAndCleanup(t *testing.T) {
	tempDir := t.TempDir()
	store := NewArtifactStore(tempDir)
	watchID := "watch-456"

	sourcePath := filepath.Join(tempDir, "diff.png")
	if err := os.WriteFile(sourcePath, append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, []byte("diff")...), 0o600); err != nil {
		t.Fatalf("write diff source: %v", err)
	}
	artifact, err := store.ReplaceVisualDiff(watchID, sourcePath)
	if err != nil {
		t.Fatalf("replace visual diff: %v", err)
	}
	if artifact.Kind != ArtifactKindVisualDiff {
		t.Fatalf("expected visual diff kind, got %s", artifact.Kind)
	}

	artifacts, err := store.List(watchID)
	if err != nil {
		t.Fatalf("list artifacts: %v", err)
	}
	if len(artifacts) != 1 {
		t.Fatalf("expected one artifact, got %d", len(artifacts))
	}
	if err := store.ClearVisualDiff(watchID); err != nil {
		t.Fatalf("clear visual diff: %v", err)
	}
	if _, err := store.Resolve(watchID, ArtifactKindVisualDiff); !os.IsNotExist(err) {
		t.Fatalf("expected visual diff to be removed, got err=%v", err)
	}
	if err := store.RemoveAll(watchID); err != nil {
		t.Fatalf("remove all: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tempDir, "watch_artifacts", watchID)); !os.IsNotExist(err) {
		t.Fatalf("expected artifact directory removal, got err=%v", err)
	}
}
