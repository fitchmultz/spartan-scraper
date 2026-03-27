// Package store verifies per-store dedup lifecycle ownership.
//
// Purpose:
// - Prove each Store instance owns its own dedup content index and can reopen cleanly after shutdown.
//
// Responsibilities:
// - Verify separate stores do not share indexed dedup state.
// - Verify closing one store does not poison a later store opened on the same data dir.
//
// Scope:
// - Store-level dedup lifecycle only.
//
// Usage:
// - Run with `go test ./internal/store`.
//
// Invariants/Assumptions:
// - Dedup index initialization is lazy via `GetContentIndex()`.
// - Store shutdown is responsible for releasing dedup resources.
package store

import (
	"context"
	"testing"
)

func TestContentIndexLifecycleIsPerStore(t *testing.T) {
	ctx := context.Background()
	sharedDir := t.TempDir()

	firstStore, err := Open(sharedDir)
	if err != nil {
		t.Fatalf("open first store: %v", err)
	}
	firstIndex := firstStore.GetContentIndex()
	if firstIndex == nil {
		t.Fatal("expected first store content index")
	}
	if err := firstIndex.Index(ctx, "11111111-1111-1111-1111-111111111111", "https://example.com/a", 11); err != nil {
		t.Fatalf("seed first store dedup entry: %v", err)
	}

	isolatedStore, err := Open(t.TempDir())
	if err != nil {
		t.Fatalf("open isolated store: %v", err)
	}
	isolatedIndex := isolatedStore.GetContentIndex()
	if isolatedIndex == nil {
		t.Fatal("expected isolated store content index")
	}
	isolatedStats, err := isolatedIndex.Stats(ctx)
	if err != nil {
		t.Fatalf("read isolated store stats: %v", err)
	}
	if isolatedStats.TotalIndexed != 0 {
		t.Fatalf("expected isolated store stats to start empty, got %#v", isolatedStats)
	}

	if err := firstStore.Close(); err != nil {
		t.Fatalf("close first store: %v", err)
	}
	if err := isolatedStore.Close(); err != nil {
		t.Fatalf("close isolated store: %v", err)
	}

	reopenedStore, err := Open(sharedDir)
	if err != nil {
		t.Fatalf("reopen first store data dir: %v", err)
	}
	defer func() {
		if err := reopenedStore.Close(); err != nil {
			t.Fatalf("close reopened store: %v", err)
		}
	}()

	reopenedIndex := reopenedStore.GetContentIndex()
	if reopenedIndex == nil {
		t.Fatal("expected reopened store content index")
	}
	history, err := reopenedIndex.GetContentHistory(ctx, "https://example.com/a")
	if err != nil {
		t.Fatalf("read reopened store history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected reopened store history to preserve 1 entry, got %#v", history)
	}
}
