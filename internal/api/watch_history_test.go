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
	var firstCheck WatchCheckInspectionResponse
	if err := json.Unmarshal(firstCheckRR.Body.Bytes(), &firstCheck); err != nil {
		t.Fatalf("failed to decode first check: %v", err)
	}
	if firstCheck.Check.ID == "" {
		t.Fatal("expected first check to include check id")
	}
	if !firstCheck.Check.Baseline {
		t.Fatal("expected first check to establish a baseline")
	}

	body = "<html><body><main>beta</main></body></html>"
	secondCheckReq := httptest.NewRequest(http.MethodPost, "/v1/watch/"+created.ID+"/check", nil)
	secondCheckRR := httptest.NewRecorder()
	srv.Routes().ServeHTTP(secondCheckRR, secondCheckReq)
	if secondCheckRR.Code != http.StatusOK {
		t.Fatalf("expected second check 200, got %d: %s", secondCheckRR.Code, secondCheckRR.Body.String())
	}
	var secondCheck WatchCheckInspectionResponse
	if err := json.Unmarshal(secondCheckRR.Body.Bytes(), &secondCheck); err != nil {
		t.Fatalf("failed to decode second check: %v", err)
	}
	if secondCheck.Check.ID == "" {
		t.Fatal("expected second check to include check id")
	}
	if !secondCheck.Check.Changed {
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
	if historyResp.Checks[0].ID != secondCheck.Check.ID {
		t.Fatalf("expected latest history record first, got %q want %q", historyResp.Checks[0].ID, secondCheck.Check.ID)
	}
	if historyResp.Checks[0].Status != "changed" {
		t.Fatalf("expected changed status, got %q", historyResp.Checks[0].Status)
	}
	assertActionValue(t, historyResp.Checks[0].Actions, "Open watch automation workspace", "/automation/watches")

	detailReq := httptest.NewRequest(http.MethodGet, "/v1/watch/"+created.ID+"/history/"+firstCheck.Check.ID, nil)
	detailRR := httptest.NewRecorder()
	srv.Routes().ServeHTTP(detailRR, detailReq)
	if detailRR.Code != http.StatusOK {
		t.Fatalf("expected detail 200, got %d: %s", detailRR.Code, detailRR.Body.String())
	}
	var detailResp WatchCheckInspectionResponse
	if err := json.Unmarshal(detailRR.Body.Bytes(), &detailResp); err != nil {
		t.Fatalf("failed to decode detail response: %v", err)
	}
	if detailResp.Check.ID != firstCheck.Check.ID {
		t.Fatalf("unexpected detail id: %q", detailResp.Check.ID)
	}
	if detailResp.Check.Status != "baseline" {
		t.Fatalf("expected baseline status, got %q", detailResp.Check.Status)
	}
	assertActionValue(t, firstCheck.Check.Actions, "Open watch automation workspace", "/automation/watches")
	assertActionValue(t, detailResp.Check.Actions, "Open watch automation workspace", "/automation/watches")
}

func TestHandleWatchCheckFailurePersistsFailedHistory(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	createBody, _ := json.Marshal(WatchRequest{URL: "http://127.0.0.1:1", IntervalSeconds: 1800})
	createReq := httptest.NewRequest(http.MethodPost, "/v1/watch", bytes.NewReader(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRR := httptest.NewRecorder()
	srv.Routes().ServeHTTP(createRR, createReq)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("expected create 201, got %d: %s", createRR.Code, createRR.Body.String())
	}

	var created WatchResponse
	if err := json.Unmarshal(createRR.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	checkReq := httptest.NewRequest(http.MethodPost, "/v1/watch/"+created.ID+"/check", nil)
	checkRR := httptest.NewRecorder()
	srv.Routes().ServeHTTP(checkRR, checkReq)
	if checkRR.Code != http.StatusOK {
		t.Fatalf("expected check 200, got %d: %s", checkRR.Code, checkRR.Body.String())
	}

	var inspection WatchCheckInspectionResponse
	if err := json.Unmarshal(checkRR.Body.Bytes(), &inspection); err != nil {
		t.Fatalf("decode check response: %v", err)
	}
	if inspection.Check.Status != "failed" {
		t.Fatalf("status = %q, want failed", inspection.Check.Status)
	}
	if inspection.Check.Error == "" {
		t.Fatal("expected failure error message")
	}

	historyReq := httptest.NewRequest(http.MethodGet, "/v1/watch/"+created.ID+"/history?limit=10", nil)
	historyRR := httptest.NewRecorder()
	srv.Routes().ServeHTTP(historyRR, historyReq)
	if historyRR.Code != http.StatusOK {
		t.Fatalf("expected history 200, got %d: %s", historyRR.Code, historyRR.Body.String())
	}

	var history WatchCheckHistoryResponse
	if err := json.Unmarshal(historyRR.Body.Bytes(), &history); err != nil {
		t.Fatalf("decode history response: %v", err)
	}
	if len(history.Checks) != 1 {
		t.Fatalf("expected one history record, got %#v", history.Checks)
	}
	if history.Checks[0].Status != "failed" {
		t.Fatalf("persisted status = %q, want failed", history.Checks[0].Status)
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

func assertActionValue(t *testing.T, actions []RecommendedAction, label, want string) {
	t.Helper()
	for _, action := range actions {
		if action.Label == label {
			if action.Value != want {
				t.Fatalf("action %q value = %q, want %q", label, action.Value, want)
			}
			return
		}
	}
	t.Fatalf("missing action %q in %#v", label, actions)
}
