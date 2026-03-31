// Package mcp verifies MCP tool registry integrity.
//
// Purpose:
// - Ensure the typed tool registries stay aligned with the advertised MCP tool list.
//
// Responsibilities:
// - Compare runtime registry contents against tools/list output.
// - Confirm setup-mode exposes only the diagnostic registry.
// - Catch dispatch/list drift when new tools are added.
//
// Scope:
// - Registry integrity only; individual tool behavior is covered by other tests.
//
// Usage:
// - Run with `go test ./internal/mcp` or the repo CI targets.
//
// Invariants/Assumptions:
// - Every listed runtime tool has exactly one registered handler.
// - Setup mode lists and dispatches the same diagnostic subset.
// - Registry maps are static package-level source of truth for dispatch.
package mcp

import (
	"os"
	"sort"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func TestRuntimeToolRegistryMatchesToolsList(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	got := sortedToolRegistryNames(srv.activeToolRegistry())
	want := sortedToolNames(srv.toolsList())
	if len(got) != len(want) {
		t.Fatalf("runtime registry size = %d, want %d\ngot=%v\nwant=%v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("runtime registry mismatch\ngot=%v\nwant=%v", got, want)
		}
	}
}

func TestSetupToolRegistryMatchesToolsList(t *testing.T) {
	srv := NewSetupServer(testConfig(config.Config{}), api.SetupStatus{})
	defer srv.Close()

	got := sortedToolRegistryNames(srv.activeToolRegistry())
	want := sortedToolNames(srv.toolsList())
	if len(got) != len(want) {
		t.Fatalf("setup registry size = %d, want %d\ngot=%v\nwant=%v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("setup registry mismatch\ngot=%v\nwant=%v", got, want)
		}
	}
}

func sortedToolRegistryNames(registry toolRegistry) []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedToolNames(tools []tool) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
	}
	sort.Strings(names)
	return names
}
