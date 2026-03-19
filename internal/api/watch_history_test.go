// Package api provides HTTP handler tests for persisted watch history endpoints.
//
// Purpose:
// - Verify watch history inspection works through the REST API.
//
// Responsibilities:
// - Confirm manual checks persist history records that can be listed and inspected.
// - Confirm check-scoped artifact downloads resolve from persisted history snapshots.
//
// Scope:
// - `/v1/watch/{id}/history`, `/v1/watch/{id}/history/{checkId}`, and nested artifact routes only.
//
// Usage:
// - Run with `go test ./internal/api`.
//
// Invariants/Assumptions:
// - Manual checks always persist a history record before returning.
// - History artifact downloads remain watch- and check-scoped.
package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/watch"
)

func TestHandleWatchHistoryLifecycle(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := "<html><body><main>alpha</main></body></html>"
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(body))
	}))
	defer site.Close()

	createBody, _ := json.Marshal(WatchRequest{URL: site.URL, IntervalSeconds: 1800})
	createReq := httptest.NewRequest(http.MethodPost, "/v1/watch", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	srv.Routes().ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d: %s", createRR.Code, createRR.Body.String())
	}

	var created WatchResponse
	if err := json.Unmarshal(createRR.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}

	firstCheckReq := httptest.NewRequest(http.MethodPost, "/v1/watch/"+created.ID+"/check", nil)
	firstCheckRR := httptest.NewRecorder()
	srv.Routes().ServeHTTP(firstCheckRR, firstCheckReq)
	if firstCheckRR.Code != http.StatusOK {
		t.Fatalf("expected first check 200, got %d: %s", firstCheckRR.Code, firstCheckRR.Body.String())
	}
	var firstCheck WatchCheckResponse
	if err := json.Unmarshal(firstCheckRR.Body.Bytes(), &firstCheck); err != nil {
		t.Fatalf("failed to decode first check: %v", err)
	}
	if firstCheck.CheckID == "" {
		t.Fatal("expected first check to include checkId")
	}
	if !firstCheck.Baseline {
		t.Fatal("expected first check to establish a baseline")
	}

	body = "<html><body><main>beta</main></body></html>"
	secondCheckReq := httptest.NewRequest(http.MethodPost, "/v1/watch/"+created.ID+"/check", nil)
	secondCheckRR := httptest.NewRecorder()
	srv.Routes().ServeHTTP(secondCheckRR, secondCheckReq)
	if secondCheckRR.Code != http.StatusOK {
		t.Fatalf("expected second check 200, got %d: %s", secondCheckRR.Code, secondCheckRR.Body.String())
	}
	var secondCheck WatchCheckResponse
	if err := json.Unmarshal(secondCheckRR.Body.Bytes(), &secondCheck); err != nil {
		t.Fatalf("failed to decode second check: %v", err)
	}
	if secondCheck.CheckID == "" {
		t.Fatal("expected second check to include checkId")
	}
	if !secondCheck.Changed {
		t.Fatal("expected second check to detect a change")
	}

	historyReq := httptest.NewRequest(http.MethodGet, "/v1/watch/"+created.ID+"/history?limit=10", nil)
	historyRR := httptest.NewRecorder()
	srv.Routes().ServeHTTP(historyRR, historyReq)
	if historyRR.Code != http.StatusOK {
		t.Fatalf("expected history 200, got %d: %s", historyRR.Code, historyRR.Body.String())
	}
	var historyResp WatchCheckHistoryResponse
	if err := json.Unmarshal(historyRR.Body.Bytes(), &historyResp); err != nil {
		t.Fatalf("failed to decode history response: %v", err)
	}
	if historyResp.Total != 2 || len(historyResp.Checks) != 2 {
		t.Fatalf("expected 2 history items, got total=%d len=%d", historyResp.Total, len(historyResp.Checks))
	}
	if historyResp.Checks[0].ID != secondCheck.CheckID {
		t.Fatalf("expected latest history record first, got %q want %q", historyResp.Checks[0].ID, secondCheck.CheckID)
	}
	if historyResp.Checks[0].Status != "changed" {
		t.Fatalf("expected changed status, got %q", historyResp.Checks[0].Status)
	}

	detailReq := httptest.NewRequest(http.MethodGet, "/v1/watch/"+created.ID+"/history/"+firstCheck.CheckID, nil)
	detailRR := httptest.NewRecorder()
	srv.Routes().ServeHTTP(detailRR, detailReq)
	if detailRR.Code != http.StatusOK {
		t.Fatalf("expected detail 200, got %d: %s", detailRR.Code, detailRR.Body.String())
	}
	var detailResp WatchCheckInspectionResponse
	if err := json.Unmarshal(detailRR.Body.Bytes(), &detailResp); err != nil {
		t.Fatalf("failed to decode detail response: %v", err)
	}
	if detailResp.Check.ID != firstCheck.CheckID {
		t.Fatalf("unexpected detail id: %q", detailResp.Check.ID)
	}
	if detailResp.Check.Status != "baseline" {
		t.Fatalf("expected baseline status, got %q", detailResp.Check.Status)
	}
}

func TestHandleWatchHistoryArtifact(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	storage := watch.NewFileStorage(srv.cfg.DataDir)
	created, err := storage.Add(&watch.Watch{URL: "https://example.com/watch", IntervalSeconds: 1800, Enabled: true})
	if err != nil {
		t.Fatalf("failed to create watch: %v", err)
	}

	sourcePath := filepath.Join(srv.cfg.DataDir, "history-artifact.png")
	if err := os.WriteFile(sourcePath, []byte("artifact-bytes"), 0o600); err != nil {
		t.Fatalf("failed to write source artifact: %v", err)
	}

	record, err := watch.NewWatchHistoryStore(srv.cfg.DataDir).Record(watch.WatchCheckResult{
		WatchID:   created.ID,
		URL:       created.URL,
		CheckedAt: created.CreatedAt,
		Baseline:  true,
		Artifacts: []watch.Artifact{{
			Kind:        watch.ArtifactKindCurrentScreenshot,
			Filename:    "history-artifact.png",
			ContentType: "image/png",
			ByteSize:    int64(len("artifact-bytes")),
			Path:        sourcePath,
		}},
	})
	if err != nil {
		t.Fatalf("failed to seed watch history: %v", err)
	}

	artifactReq := httptest.NewRequest(http.MethodGet, "/v1/watch/"+created.ID+"/history/"+record.ID+"/artifacts/current-screenshot", nil)
	artifactRR := httptest.NewRecorder()
	srv.Routes().ServeHTTP(artifactRR, artifactReq)
	if artifactRR.Code != http.StatusOK {
		t.Fatalf("expected artifact 200, got %d: %s", artifactRR.Code, artifactRR.Body.String())
	}
	if got := artifactRR.Body.String(); got != "artifact-bytes" {
		t.Fatalf("unexpected artifact body: %q", got)
	}
}
