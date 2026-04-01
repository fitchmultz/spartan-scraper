// Package extract tests file-backed AI cache behavior.
//
// Purpose:
// - Verify file-backed AI cache persistence remains deterministic for callers and tests.
//
// Responsibilities:
// - Cover synchronous cache persistence and reload behavior.
//
// Scope:
// - FileAICache persistence behavior only.
//
// Usage:
// - Run with the extract package test suite.
//
// Invariants/Assumptions:
// - `Set` must persist cache state before returning so later readers and temp-dir cleanup see stable filesystem state.
package extract

import "testing"

func TestFileAICacheSetPersistsBeforeReturn(t *testing.T) {
	dataDir := t.TempDir()
	cache := NewFileAICache(dataDir, DefaultAICacheTTL)

	cache.Set("test-key", &AIExtractResult{
		Fields: map[string]FieldValue{
			"pricing_model": {
				Values: []string{"usage-based"},
				Source: FieldSourceLLM,
			},
		},
		Confidence: 0.91,
	})

	reloaded := NewFileAICache(dataDir, DefaultAICacheTTL)
	cached, ok := reloaded.Get("test-key")
	if !ok {
		t.Fatal("expected cache entry to be readable immediately after Set")
	}
	if cached == nil {
		t.Fatal("expected cached result, got nil")
	}
	if len(cached.Fields["pricing_model"].Values) != 1 || cached.Fields["pricing_model"].Values[0] != "usage-based" {
		t.Fatalf("unexpected cached values: %#v", cached.Fields["pricing_model"].Values)
	}
}
