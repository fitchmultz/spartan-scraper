// Package watch verifies persisted watch history behavior.
//
// Purpose:
// - Prove watch checks persist inspectable history records and stable artifact snapshots.
//
// Responsibilities:
// - Verify record creation, list/get pagination, and status derivation.
// - Verify per-check artifact snapshots resolve independently from transient source files.
// - Verify deleting a watch also removes its history and persisted check artifacts.
//
// Scope:
// - WatchHistoryStore behavior only.
//
// Usage:
// - Run with `go test ./internal/watch`.
//
// Invariants/Assumptions:
// - History records are sorted newest-first when listed.
// - Artifact snapshots remain available after the original source file changes or disappears.
package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatchHistoryStoreRecordAndResolveArtifact(t *testing.T) {
	dataDir := t.TempDir()
	sourcePath := filepath.Join(dataDir, "source.png")
	if err := os.WriteFile(sourcePath, []byte("png-bits"), 0o600); err != nil {
		t.Fatalf("failed to seed source artifact: %v", err)
	}

	store := NewWatchHistoryStore(dataDir)
	recordedAt := time.Now().Add(-time.Minute).UTC()
	record, err := store.Record(WatchCheckResult{
		WatchID:   "watch-1",
		URL:       "https://example.com/watch",
		CheckedAt: recordedAt,
		Changed:   true,
		DiffText:  "-old\n+new",
		Artifacts: []Artifact{{
			Kind:        ArtifactKindCurrentScreenshot,
			Filename:    "source.png",
			ContentType: "image/png",
			ByteSize:    8,
			Path:        sourcePath,
		}},
	})
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	if record.ID == "" {
		t.Fatal("expected record id to be set")
	}
	if record.Status != CheckStatusChanged {
		t.Fatalf("expected changed status, got %q", record.Status)
	}
	if len(record.Artifacts) != 1 {
		t.Fatalf("expected one persisted artifact, got %d", len(record.Artifacts))
	}

	if err := os.WriteFile(sourcePath, []byte("new-bits"), 0o600); err != nil {
		t.Fatalf("failed to mutate source artifact: %v", err)
	}

	stored, err := store.GetByID("watch-1", record.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if stored.CheckedAt.Format(time.RFC3339Nano) != recordedAt.Format(time.RFC3339Nano) {
		t.Fatalf("unexpected checkedAt: %s", stored.CheckedAt)
	}

	resolved, err := store.ResolveArtifact("watch-1", record.ID, ArtifactKindCurrentScreenshot)
	if err != nil {
		t.Fatalf("ResolveArtifact failed: %v", err)
	}
	data, err := os.ReadFile(resolved.Path)
	if err != nil {
		t.Fatalf("failed to read resolved artifact: %v", err)
	}
	if string(data) != "png-bits" {
		t.Fatalf("expected persisted snapshot to preserve original bytes, got %q", string(data))
	}

	records, total, err := store.GetByWatch("watch-1", 10, 0)
	if err != nil {
		t.Fatalf("GetByWatch failed: %v", err)
	}
	if total != 1 || len(records) != 1 {
		t.Fatalf("expected one record, got total=%d len=%d", total, len(records))
	}
}

func TestWatchHistoryStoreDeleteWatchRemovesHistoryAndArtifacts(t *testing.T) {
	dataDir := t.TempDir()
	sourcePath := filepath.Join(dataDir, "artifact.bin")
	if err := os.WriteFile(sourcePath, []byte("artifact"), 0o600); err != nil {
		t.Fatalf("failed to seed artifact: %v", err)
	}

	store := NewWatchHistoryStore(dataDir)
	record, err := store.Record(WatchCheckResult{
		WatchID:   "watch-delete",
		URL:       "https://example.com/delete",
		CheckedAt: time.Now().UTC(),
		Baseline:  true,
		Artifacts: []Artifact{{
			Kind:        ArtifactKindCurrentScreenshot,
			Filename:    "artifact.bin",
			ContentType: "application/octet-stream",
			ByteSize:    8,
			Path:        sourcePath,
		}},
	})
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}

	if err := store.DeleteWatch("watch-delete"); err != nil {
		t.Fatalf("DeleteWatch failed: %v", err)
	}

	records, total, err := store.GetByWatch("watch-delete", 10, 0)
	if err != nil {
		t.Fatalf("GetByWatch after delete failed: %v", err)
	}
	if total != 0 || len(records) != 0 {
		t.Fatalf("expected no records after delete, got total=%d len=%d", total, len(records))
	}
	if _, err := store.ResolveArtifact("watch-delete", record.ID, ArtifactKindCurrentScreenshot); !os.IsNotExist(err) {
		t.Fatalf("expected artifact snapshot to be removed, got err=%v", err)
	}
}
