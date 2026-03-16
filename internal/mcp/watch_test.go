// Package mcp provides tests for watch management MCP tools.
//
// Purpose:
//   - Prove MCP exposes the stored-watch workflow already available on the other
//     primary operator surfaces.
//
// Responsibilities:
//   - Verify watch tools are registered in the published MCP tool list.
//   - Verify watch CRUD flows work end to end.
//   - Verify manual watch checks can establish baselines, detect changes, and
//     trigger follow-on jobs.
//
// Scope:
// - MCP watch tool behavior only.
//
// Usage:
// - Run with `go test ./internal/mcp`.
//
// Invariants/Assumptions:
// - Watch MCP tools should preserve API-aligned defaults and not-found behavior.
package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/watch"
)

func TestWatchToolsInToolsList(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	toolMap := make(map[string]tool)
	for _, item := range srv.toolsList() {
		toolMap[item.Name] = item
	}

	for _, name := range []string{
		"watch_list",
		"watch_get",
		"watch_create",
		"watch_update",
		"watch_delete",
		"watch_check",
	} {
		if _, ok := toolMap[name]; !ok {
			t.Fatalf("expected tool %s in toolsList", name)
		}
	}

	createRequired := toolMap["watch_create"].InputSchema["required"].([]string)
	if len(createRequired) != 1 || createRequired[0] != "url" {
		t.Fatalf("unexpected required fields for watch_create: %#v", createRequired)
	}
	updateRequired := toolMap["watch_update"].InputSchema["required"].([]string)
	if len(updateRequired) != 1 || updateRequired[0] != "id" {
		t.Fatalf("unexpected required fields for watch_update: %#v", updateRequired)
	}
}

func TestHandleWatchLifecycle(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	var (
		mu   sync.RWMutex
		body = "<html><body><h1>alpha</h1></body></html>"
	)
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(body))
	}))
	defer site.Close()

	ctx := context.Background()

	createResult, err := srv.handleToolCall(ctx, map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "watch_create",
			"arguments": map[string]interface{}{
				"url":             site.URL,
				"intervalSeconds": 120,
				"enabled":         true,
				"diffFormat":      "unified",
			},
		}),
	})
	if err != nil {
		t.Fatalf("watch_create failed: %v", err)
	}
	createdWatch, ok := createResult.(*watch.Watch)
	if !ok {
		t.Fatalf("expected created watch, got %#v", createResult)
	}
	if createdWatch.ID == "" {
		t.Fatal("expected created watch to have an id")
	}
	if createdWatch.URL != site.URL {
		t.Fatalf("unexpected created watch url: %q", createdWatch.URL)
	}

	listResult, err := srv.handleToolCall(ctx, map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name":      "watch_list",
			"arguments": map[string]interface{}{},
		}),
	})
	if err != nil {
		t.Fatalf("watch_list failed: %v", err)
	}
	listPayload, ok := listResult.(map[string]interface{})
	if !ok {
		t.Fatalf("expected list payload, got %#v", listResult)
	}
	watches, ok := listPayload["watches"].([]watch.Watch)
	if !ok {
		t.Fatalf("expected watches slice, got %#v", listPayload["watches"])
	}
	if len(watches) != 1 {
		t.Fatalf("expected 1 watch, got %d", len(watches))
	}

	updateResult, err := srv.handleToolCall(ctx, map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "watch_update",
			"arguments": map[string]interface{}{
				"id":       createdWatch.ID,
				"selector": "h1",
				"enabled":  false,
				"jobTrigger": map[string]interface{}{
					"kind": "scrape",
					"request": map[string]interface{}{
						"url": site.URL,
					},
				},
			},
		}),
	})
	if err != nil {
		t.Fatalf("watch_update failed: %v", err)
	}
	updatedWatch, ok := updateResult.(*watch.Watch)
	if !ok {
		t.Fatalf("expected updated watch, got %#v", updateResult)
	}
	if updatedWatch.URL != site.URL {
		t.Fatalf("expected url to be preserved, got %q", updatedWatch.URL)
	}
	if updatedWatch.Selector != "h1" {
		t.Fatalf("unexpected selector: %q", updatedWatch.Selector)
	}
	if updatedWatch.Enabled {
		t.Fatal("expected updated watch to be disabled")
	}
	if updatedWatch.JobTrigger == nil {
		t.Fatal("expected job trigger to be set")
	}
	if updatedWatch.JobTrigger.Kind != model.KindScrape {
		t.Fatalf("unexpected job trigger kind: %q", updatedWatch.JobTrigger.Kind)
	}

	getResult, err := srv.handleToolCall(ctx, map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "watch_get",
			"arguments": map[string]interface{}{
				"id": createdWatch.ID,
			},
		}),
	})
	if err != nil {
		t.Fatalf("watch_get failed: %v", err)
	}
	gotWatch, ok := getResult.(*watch.Watch)
	if !ok {
		t.Fatalf("expected get watch result, got %#v", getResult)
	}
	if gotWatch.ID != createdWatch.ID {
		t.Fatalf("unexpected watch id: %q", gotWatch.ID)
	}

	_, err = srv.handleToolCall(ctx, map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "watch_update",
			"arguments": map[string]interface{}{
				"id":      createdWatch.ID,
				"enabled": true,
			},
		}),
	})
	if err != nil {
		t.Fatalf("watch_update re-enable failed: %v", err)
	}

	firstCheckResult, err := srv.handleToolCall(ctx, map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "watch_check",
			"arguments": map[string]interface{}{
				"id": createdWatch.ID,
			},
		}),
	})
	if err != nil {
		t.Fatalf("first watch_check failed: %v", err)
	}
	firstCheck, ok := firstCheckResult.(*watch.WatchCheckResult)
	if !ok {
		t.Fatalf("expected first check result, got %#v", firstCheckResult)
	}
	if firstCheck.WatchID != createdWatch.ID {
		t.Fatalf("unexpected watch id in first check: %q", firstCheck.WatchID)
	}
	if firstCheck.CurrentHash == "" {
		t.Fatal("expected first check to include current hash")
	}
	if firstCheck.Changed {
		t.Fatal("expected first check to establish baseline without reporting changed")
	}

	mu.Lock()
	body = "<html><body><h1>beta</h1></body></html>"
	mu.Unlock()

	secondCheckResult, err := srv.handleToolCall(ctx, map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "watch_check",
			"arguments": map[string]interface{}{
				"id": createdWatch.ID,
			},
		}),
	})
	if err != nil {
		t.Fatalf("second watch_check failed: %v", err)
	}
	secondCheck, ok := secondCheckResult.(*watch.WatchCheckResult)
	if !ok {
		t.Fatalf("expected second check result, got %#v", secondCheckResult)
	}
	if !secondCheck.Changed {
		t.Fatal("expected second check to detect a change")
	}
	if secondCheck.DiffText == "" {
		t.Fatal("expected second check to include a diff")
	}
	if len(secondCheck.TriggeredJobs) != 1 {
		t.Fatalf("expected exactly one triggered job, got %#v", secondCheck.TriggeredJobs)
	}

	deleteResult, err := srv.handleToolCall(ctx, map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "watch_delete",
			"arguments": map[string]interface{}{
				"id": createdWatch.ID,
			},
		}),
	})
	if err != nil {
		t.Fatalf("watch_delete failed: %v", err)
	}
	deletePayload, ok := deleteResult.(map[string]interface{})
	if !ok {
		t.Fatalf("expected delete payload, got %#v", deleteResult)
	}
	deleted, ok := deletePayload["deleted"].(bool)
	if !ok || !deleted {
		t.Fatalf("unexpected delete payload: %#v", deletePayload)
	}

	_, err = srv.handleToolCall(ctx, map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name": "watch_get",
			"arguments": map[string]interface{}{
				"id": createdWatch.ID,
			},
		}),
	})
	if err == nil {
		t.Fatal("expected watch_get after delete to fail")
	}
	if !apperrors.IsKind(err, apperrors.KindNotFound) {
		t.Fatalf("expected not found error, got %v", err)
	}
}
