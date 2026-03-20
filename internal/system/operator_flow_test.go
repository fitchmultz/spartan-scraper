// Package system provides deterministic PR-gate integration coverage for operator-facing flows.
//
// Purpose:
// - Lock the fresh-start-to-daily-use operator journey into PR-safe system coverage.
//
// Responsibilities:
//   - Start an isolated server against the deterministic fixture site.
//   - Verify health, static web shell delivery, first-job creation, canonical jobs-list envelopes,
//     export, schedule, and WebSocket lifecycle behavior.
//   - Ensure the operator-facing surface stays intact after work has begun.
//
// Scope:
// - PR-safe local integration coverage only.
//
// Usage:
// - Runs automatically in `go test ./...` outside the excluded internal/e2e package.
//
// Invariants/Assumptions:
// - Uses temp data directories and the deterministic local fixture.
// - Verifies the server serves the web shell and canonical job envelopes without external network access.
package system

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/testsite"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

type operatorFlowHealthResponse struct {
	Status string `json:"status"`
}

type operatorFlowJobsEnvelope struct {
	Jobs []struct {
		ID string `json:"id"`
	} `json:"jobs"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func TestOperatorFlowFreshStartToDailyUse(t *testing.T) {
	dataDir := t.TempDir()
	port := freePort(t)
	env := baseEnv(dataDir)
	env = append(env, "PORT="+strconv.Itoa(port))
	site := testsite.Start(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverCmd, cleanup := startProcess(ctx, t, env, t.TempDir(), systemBinaryPath, "server")
	defer cleanup()

	client := &http.Client{Timeout: 5 * time.Second}
	waitForHealth(t, client, port)

	assertOperatorFlowHealthOK(t, client, port)

	firstJobID := postJob(t, client, port, "/v1/scrape", map[string]any{
		"url":            site.ScrapeURL(),
		"headless":       false,
		"timeoutSeconds": 30,
	})
	waitForJob(t, client, port, firstJobID)
	assertManifestExists(t, dataDir, firstJobID)

	firstEnvelope := fetchOperatorFlowJobsEnvelope(t, client, port, 10, 0)
	assertOperatorFlowJobsEnvelope(t, firstEnvelope, 1, 10, 0)
	if !operatorFlowEnvelopeContainsJob(firstEnvelope, firstJobID) {
		t.Fatalf("GET /v1/jobs did not include first job %s", firstJobID)
	}

	exportPath := filepath.Join(t.TempDir(), "first-job.md")
	runOK(t, env, "export", "--job-id", firstJobID, "--format", "md", "--out", exportPath)
	assertFileNotEmpty(t, exportPath)

	wsConn, _, _, err := ws.Dial(context.Background(), fmt.Sprintf("ws://127.0.0.1:%d/v1/ws", port))
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer wsConn.Close()

	if err := wsutil.WriteClientText(wsConn, []byte(`{"type":"subscribe_jobs","timestamp":0,"payload":null}`)); err != nil {
		t.Fatalf("ws subscribe: %v", err)
	}

	postSchedule(t, client, port, map[string]any{
		"kind":            "scrape",
		"intervalSeconds": 1,
		"request": map[string]any{
			"url":            site.ScrapeURL(),
			"headless":       false,
			"timeoutSeconds": 30,
		},
	})
	waitForJobs(t, client, port, 2)

	secondEnvelope := fetchOperatorFlowJobsEnvelope(t, client, port, 10, 0)
	assertOperatorFlowJobsEnvelope(t, secondEnvelope, 2, 10, 0)

	scheduledJobID := findAdditionalOperatorFlowJobID(t, secondEnvelope, firstJobID)
	waitForJob(t, client, port, scheduledJobID)
	assertManifestExists(t, dataDir, scheduledJobID)

	events := waitForWebSocketJobLifecycle(t, wsConn, scheduledJobID)
	if !events["job_created"] {
		t.Fatalf("missing websocket job_created event for %s", scheduledJobID)
	}
	if !events["job_started"] && !events["job_status_changed"] {
		t.Fatalf("missing websocket running event for %s", scheduledJobID)
	}
	if !events["job_completed"] {
		t.Fatalf("missing websocket job_completed event for %s", scheduledJobID)
	}

	cancel()
	_ = serverCmd.Wait()
}

func assertOperatorFlowHealthOK(t *testing.T, client *http.Client, port int) {
	t.Helper()

	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/healthz", port))
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET /healthz status=%d body=%s", resp.StatusCode, string(body))
	}

	var payload operatorFlowHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode /healthz: %v", err)
	}
	if payload.Status != "ok" {
		t.Fatalf("health status=%q, want ok", payload.Status)
	}
}

func fetchOperatorFlowJobsEnvelope(t *testing.T, client *http.Client, port int, limit int, offset int) operatorFlowJobsEnvelope {
	t.Helper()

	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/v1/jobs?limit=%d&offset=%d", port, limit, offset))
	if err != nil {
		t.Fatalf("GET /v1/jobs: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET /v1/jobs status=%d body=%s", resp.StatusCode, string(body))
	}

	var payload operatorFlowJobsEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode /v1/jobs: %v", err)
	}
	return payload
}

func assertOperatorFlowJobsEnvelope(t *testing.T, payload operatorFlowJobsEnvelope, minTotal int, wantLimit int, wantOffset int) {
	t.Helper()

	if payload.Jobs == nil {
		t.Fatalf("jobs envelope missing jobs array")
	}
	if payload.Total < minTotal {
		t.Fatalf("jobs envelope total=%d, want at least %d", payload.Total, minTotal)
	}
	if payload.Limit != wantLimit {
		t.Fatalf("jobs envelope limit=%d, want %d", payload.Limit, wantLimit)
	}
	if payload.Offset != wantOffset {
		t.Fatalf("jobs envelope offset=%d, want %d", payload.Offset, wantOffset)
	}
}

func operatorFlowEnvelopeContainsJob(payload operatorFlowJobsEnvelope, wantID string) bool {
	for _, job := range payload.Jobs {
		if job.ID == wantID {
			return true
		}
	}
	return false
}

func findAdditionalOperatorFlowJobID(t *testing.T, payload operatorFlowJobsEnvelope, excludeID string) string {
	t.Helper()

	for _, job := range payload.Jobs {
		if job.ID != "" && job.ID != excludeID {
			return job.ID
		}
	}
	t.Fatalf("no additional job found besides %s", excludeID)
	return ""
}
