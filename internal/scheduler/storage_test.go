// Package scheduler provides tests for schedule storage operations.
// Tests cover Add, List, Delete, and LoadAll operations.
// Does NOT test schedule validation or complex persistence scenarios.
package scheduler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestSchedulerStorage(t *testing.T) {
	dataDir := t.TempDir()

	schedules, err := LoadAll(dataDir)
	if err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}
	if len(schedules) != 0 {
		t.Errorf("expected 0 schedules, got %d", len(schedules))
	}

	s1 := testScrapeSchedule("http://example.com")

	if _, err := Add(dataDir, s1); err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	list, _ := List(dataDir)
	if len(list) != 1 {
		t.Errorf("expected 1 schedule, got %d", len(list))
	}
	if list[0].Kind != model.KindScrape {
		t.Errorf("expected kind scrape, got %v", list[0].Kind)
	}

	id := list[0].ID
	if err := Delete(dataDir, id); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	list, _ = List(dataDir)
	if len(list) != 0 {
		t.Errorf("expected 0 schedules after delete, got %d", len(list))
	}
}

func TestLoadAllRejectsLegacyParamsScheduleStore(t *testing.T) {
	dataDir := t.TempDir()
	payload := `{
  "schedules": [
    {
      "id": "legacy",
      "kind": "scrape",
      "intervalSeconds": 60,
      "nextRun": "2026-03-11T12:00:00Z",
      "params": {
        "url": "https://example.com"
      }
    }
  ]
}`
	if err := os.WriteFile(filepath.Join(dataDir, "schedules.json"), []byte(payload), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := LoadAll(dataDir)
	if err == nil {
		t.Fatal("LoadAll() error = nil, want legacy schedule failure")
	}
	if !strings.Contains(err.Error(), "removed params contract") {
		t.Fatalf("LoadAll() error = %v, want removed params contract message", err)
	}
}
