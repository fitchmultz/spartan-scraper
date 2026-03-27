// Package system provides deterministic PR-safe coverage for the API-owned dedup workflow.
//
// Purpose:
// - Verify the running server exposes dedup lookup, history, stats, and cleanup behavior against seeded fixture data.
//
// Responsibilities:
// - Seed a fixture store's content index with deterministic fixture URLs and hashes.
// - Exercise the HTTP dedup endpoints through a live server process.
// - Confirm delete cleanup updates stats and history as expected.
//
// Scope:
// - Dedup API behavior only; broader crawl/job coverage lives in sibling system tests.
//
// Usage:
// - Runs automatically via `go test ./internal/system`.
//
// Invariants/Assumptions:
// - Dedup remains an API-owned workflow.
// - Seeded job IDs use valid UUID format because delete validation enforces it.
// - Fixture URLs are deterministic and stable for assertions.
package system

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/dedup"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/testsite"
)

func TestDedupAPIFlow(t *testing.T) {
	const (
		jobA = "11111111-1111-1111-1111-111111111111"
		jobB = "22222222-2222-2222-2222-222222222222"
		jobC = "33333333-3333-3333-3333-333333333333"
	)

	dataDir := t.TempDir()
	site := testsite.Start(t)
	seedDedupIndex(t, dataDir, []dedupSeedEntry{
		{jobID: jobA, url: site.ScrapeURL(), simhash: 42},
		{jobID: jobB, url: site.ScrapeURL(), simhash: 42},
		{jobID: jobC, url: site.ArticleURL(), simhash: 99},
	})

	port := freePort(t)
	env := append(baseEnv(dataDir), "PORT="+strconv.Itoa(port))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverCmd, cleanup := startProcess(ctx, t, env, t.TempDir(), systemBinaryPath, "server")
	defer cleanup()

	client := &http.Client{Timeout: 5 * time.Second}
	waitForHealth(t, client, port)

	historyURL := fmt.Sprintf("http://127.0.0.1:%d/v1/dedup/history?url=%s", port, url.QueryEscape(site.ScrapeURL()))
	var history []dedup.ContentEntry
	getJSON(t, client, historyURL, &history)
	if len(history) != 2 {
		t.Fatalf("expected 2 history entries, got %d", len(history))
	}
	assertHistoryHasJobs(t, history, jobA, jobB)

	var matches []dedup.DuplicateMatch
	getJSON(t, client, fmt.Sprintf("http://127.0.0.1:%d/v1/dedup/duplicates?simhash=42&threshold=0", port), &matches)
	if len(matches) != 2 {
		t.Fatalf("expected 2 duplicate matches, got %d", len(matches))
	}
	assertDuplicateMatchesHaveJobs(t, matches, jobA, jobB)

	var stats dedup.Stats
	getJSON(t, client, fmt.Sprintf("http://127.0.0.1:%d/v1/dedup/stats", port), &stats)
	if stats.TotalIndexed != 3 || stats.UniqueURLs != 2 || stats.UniqueJobs != 3 || stats.DuplicatePairs != 1 {
		t.Fatalf("unexpected stats before delete: %#v", stats)
	}

	deleteReq, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("http://127.0.0.1:%d/v1/dedup/job/%s", port, jobB), nil)
	if err != nil {
		t.Fatalf("build delete request: %v", err)
	}
	deleteResp, err := client.Do(deleteReq)
	if err != nil {
		t.Fatalf("delete dedup job entries: %v", err)
	}
	_ = deleteResp.Body.Close()
	if deleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete status 204, got %d", deleteResp.StatusCode)
	}

	getJSON(t, client, fmt.Sprintf("http://127.0.0.1:%d/v1/dedup/stats", port), &stats)
	if stats.TotalIndexed != 2 || stats.UniqueURLs != 2 || stats.UniqueJobs != 2 || stats.DuplicatePairs != 0 {
		t.Fatalf("unexpected stats after delete: %#v", stats)
	}

	getJSON(t, client, historyURL, &history)
	if len(history) != 1 || history[0].JobID != jobA {
		t.Fatalf("unexpected history after delete: %#v", history)
	}

	cancel()
	_ = serverCmd.Wait()
}

type dedupSeedEntry struct {
	jobID   string
	url     string
	simhash uint64
}

func seedDedupIndex(t *testing.T, dataDir string, entries []dedupSeedEntry) {
	t.Helper()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("open store for dedup seeding: %v", err)
	}
	index := st.GetContentIndex()
	if index == nil {
		t.Fatal("expected content index to be initialized during seeding")
	}
	for _, entry := range entries {
		if err := index.Index(context.Background(), entry.jobID, entry.url, entry.simhash); err != nil {
			t.Fatalf("seed dedup entry: %v", err)
		}
	}
	if err := st.Close(); err != nil {
		t.Fatalf("close seeded store: %v", err)
	}
}

func getJSON(t *testing.T, client *http.Client, endpoint string, target any) {
	t.Helper()
	resp, err := client.Get(endpoint)
	if err != nil {
		t.Fatalf("GET %s: %v", endpoint, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s returned %d", endpoint, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		t.Fatalf("decode %s: %v", endpoint, err)
	}
}

func assertHistoryHasJobs(t *testing.T, history []dedup.ContentEntry, wantJobIDs ...string) {
	t.Helper()
	seen := map[string]bool{}
	for _, entry := range history {
		seen[entry.JobID] = true
	}
	for _, jobID := range wantJobIDs {
		if !seen[jobID] {
			t.Fatalf("expected history to include job %s, got %#v", jobID, history)
		}
	}
}

func assertDuplicateMatchesHaveJobs(t *testing.T, matches []dedup.DuplicateMatch, wantJobIDs ...string) {
	t.Helper()
	seen := map[string]bool{}
	for _, match := range matches {
		seen[match.JobID] = true
	}
	for _, jobID := range wantJobIDs {
		if !seen[jobID] {
			t.Fatalf("expected duplicate matches to include job %s, got %#v", jobID, matches)
		}
	}
}
