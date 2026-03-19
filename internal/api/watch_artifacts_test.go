// Package api verifies watch artifact response shaping and downloads.
//
// Purpose:
// - Cover the public watch artifact contract and explicit download route.
//
// Responsibilities:
// - Assert watch-check responses expose artifact download URLs instead of paths.
// - Verify persisted watch artifacts stream through the API with stable metadata.
// - Confirm missing artifacts return stable not-found responses.
//
// Scope:
// - Watch artifact response builders and `/v1/watch/{id}/artifacts/{artifactKind}`.
//
// Usage:
// - Run with `go test ./internal/api`.
//
// Invariants/Assumptions:
// - Test fixtures use small fake PNG payloads that still sniff as `image/png`.
package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/watch"
)

func TestBuildWatchCheckInspectionUsesHistoryArtifactDownloads(t *testing.T) {
	record := watch.WatchCheckRecord{
		ID:        "check-123",
		WatchID:   "watch-123",
		URL:       "https://example.com",
		CheckedAt: time.Unix(1700000000, 0).UTC(),
		Status:    watch.CheckStatusChanged,
		Changed:   true,
		Artifacts: []watch.Artifact{{
			Kind:        watch.ArtifactKindCurrentScreenshot,
			Filename:    "watch-watch-123-current-screenshot.png",
			ContentType: "image/png",
			ByteSize:    42,
			Path:        "/tmp/private/current.png",
		}},
	}

	inspection := BuildWatchCheckInspection(record)
	if len(inspection.Artifacts) != 1 {
		t.Fatalf("expected one artifact response, got %d", len(inspection.Artifacts))
	}
	artifact := inspection.Artifacts[0]
	if artifact.DownloadURL != "/v1/watch/watch-123/history/check-123/artifacts/current-screenshot" {
		t.Fatalf("unexpected download URL: %s", artifact.DownloadURL)
	}

	raw, err := json.Marshal(inspection)
	if err != nil {
		t.Fatalf("marshal inspection: %v", err)
	}
	body := string(raw)
	if strings.Contains(body, "/tmp/private/current.png") {
		t.Fatalf("response leaked artifact path: %s", body)
	}
	if strings.Contains(body, "screenshotPath") || strings.Contains(body, "visualDiffPath") {
		t.Fatalf("response leaked legacy path fields: %s", body)
	}
}

func TestHandleWatchArtifact(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	storage := watch.NewFileStorage(srv.cfg.DataDir)
	created := createTestWatch(t, storage, "https://example.com/artifacts")

	sourcePath := filepath.Join(t.TempDir(), "current.png")
	payload := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, []byte("artifact-body")...)
	if err := os.WriteFile(sourcePath, payload, 0o600); err != nil {
		t.Fatalf("write source artifact: %v", err)
	}
	if _, _, err := watch.NewArtifactStore(srv.cfg.DataDir).ReplaceCurrent(created.ID, sourcePath); err != nil {
		t.Fatalf("persist watch artifact: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/watch/"+created.ID+"/artifacts/current-screenshot", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("expected image/png content type, got %s", got)
	}
	if !strings.Contains(rr.Header().Get("Content-Disposition"), "current-screenshot") {
		t.Fatalf("expected content disposition filename, got %s", rr.Header().Get("Content-Disposition"))
	}
	if rr.Body.Len() == 0 {
		t.Fatal("expected artifact bytes in response body")
	}
}

func TestHandleWatchArtifactNotFound(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	storage := watch.NewFileStorage(srv.cfg.DataDir)
	created := createTestWatch(t, storage, "https://example.com/missing-artifact")

	req := httptest.NewRequest(http.MethodGet, "/v1/watch/"+created.ID+"/artifacts/current-screenshot", nil)
	rr := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", rr.Code, rr.Body.String())
	}
}
