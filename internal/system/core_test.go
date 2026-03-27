// Package system provides deterministic PR-gate integration coverage for the retained 1.0 core.
//
// Purpose:
// - Exercise the local-first REST, scheduler, MCP, export, manifest, and WebSocket flows in `make test-ci`.
//
// Responsibilities:
// - Build the CLI binary once for the package.
// - Start isolated local servers against the deterministic fixture site.
// - Assert core flows complete without relying on browser-heavy or live-network checks.
//
// Scope:
// - PR-safe local system tests only. Heavy browser and stress coverage stays in internal/e2e.
//
// Usage:
// - Runs automatically via `go test ./...` outside the excluded internal/e2e package.
//
// Invariants/Assumptions:
// - Uses temp data directories so local state and secrets are not touched.
// - Uses the shared local fixture instead of external websites.
package system

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/testharness"
	"github.com/fitchmultz/spartan-scraper/internal/testsite"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

var systemBinaryPath string

func TestMain(m *testing.M) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	projectRoot := filepath.Clean(filepath.Join(cwd, "..", ".."))
	binaryPath, cleanup, err := testharness.BuildBinary(
		projectRoot,
		"spartan-system-bin-*",
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	systemBinaryPath = binaryPath
	code := m.Run()
	cleanup()
	os.Exit(code)
}

func TestCoreRESTFlowsWriteResultsAndManifest(t *testing.T) {
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

	scrapeID := postJob(t, client, port, "/v1/scrape", map[string]any{
		"url":            site.ScrapeURL(),
		"headless":       false,
		"timeoutSeconds": 30,
	})
	waitForJob(t, client, port, scrapeID)
	assertJobResultContains(t, client, port, scrapeID, "Example Domain")
	assertManifestExists(t, dataDir, scrapeID)

	crawlID := postJob(t, client, port, "/v1/crawl", map[string]any{
		"url":            site.CrawlRootURL(),
		"maxDepth":       1,
		"maxPages":       5,
		"headless":       false,
		"timeoutSeconds": 30,
	})
	waitForJob(t, client, port, crawlID)
	assertManifestExists(t, dataDir, crawlID)

	researchID := postJob(t, client, port, "/v1/research", map[string]any{
		"query":          "example",
		"urls":           site.ResearchURLs(),
		"maxDepth":       1,
		"maxPages":       5,
		"headless":       false,
		"timeoutSeconds": 30,
	})
	waitForJob(t, client, port, researchID)
	assertJobResultContains(t, client, port, researchID, `"summary"`)
	assertManifestExists(t, dataDir, researchID)

	cancel()
	_ = serverCmd.Wait()
}

func TestScheduleExportAndWebSocketCoreFlow(t *testing.T) {
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

	wsConn, _, _, err := ws.Dial(context.Background(), fmt.Sprintf("ws://127.0.0.1:%d/v1/ws", port))
	if err != nil {
		t.Fatalf("ws dial: %v", err)
	}
	defer wsConn.Close()
	if err := wsutil.WriteClientText(wsConn, []byte(`{"type":"subscribe_jobs","timestamp":0,"payload":null}`)); err != nil {
		t.Fatalf("ws subscribe: %v", err)
	}

	jobID := postJob(t, client, port, "/v1/scrape", map[string]any{
		"url":            site.ScrapeURL(),
		"headless":       false,
		"timeoutSeconds": 30,
	})
	waitForJob(t, client, port, jobID)
	assertManifestExists(t, dataDir, jobID)

	events := waitForWebSocketJobLifecycle(t, wsConn, jobID)
	if !events["job_created"] {
		t.Fatalf("missing websocket job_created event for %s", jobID)
	}
	if !events["job_started"] && !events["job_status_changed"] {
		t.Fatalf("missing websocket running event for %s", jobID)
	}
	if !events["job_completed"] {
		t.Fatalf("missing websocket job_completed event for %s", jobID)
	}

	exportPath := filepath.Join(t.TempDir(), "result.md")
	runOK(t, env, "export", "--job-id", jobID, "--format", "md", "--out", exportPath)
	assertFileNotEmpty(t, exportPath)

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
	latestID := latestJobID(t, client, port)
	waitForJob(t, client, port, latestID)
	assertManifestExists(t, dataDir, latestID)

	cancel()
	_ = serverCmd.Wait()
}

func TestMCPBasicFlow(t *testing.T) {
	dataDir := t.TempDir()
	env := baseEnv(dataDir)
	site := testsite.Start(t)

	out := runMCP(t, env, []string{
		`{"id":1,"method":"initialize"}`,
		`{"id":2,"method":"tools/list"}`,
		fmt.Sprintf(`{"id":3,"method":"tools/call","params":{"name":"scrape_page","arguments":{"url":%q,"timeoutSeconds":30}}}`, site.ScrapeURL()),
	})
	if !strings.Contains(out, `"tools"`) {
		t.Fatalf("expected tools/list output in MCP response")
	}
	if !strings.Contains(out, "Example Domain") {
		t.Fatalf("expected scrape_page MCP result to contain fixture content")
	}
}

func baseEnv(dataDir string) []string {
	return testharness.BaseEnv(dataDir)
}

func runOK(t *testing.T, env []string, args ...string) {
	t.Helper()
	if err := run(t, env, args...); err != nil {
		t.Fatalf("%v", err)
	}
}

func run(t *testing.T, env []string, args ...string) error {
	t.Helper()
	return testharness.RunCommand(
		t,
		systemBinaryPath,
		testharness.MergeEnv(env),
		args...,
	)
}

func startProcess(ctx context.Context, t *testing.T, env []string, dir string, name string, args ...string) (*exec.Cmd, func()) {
	t.Helper()
	return testharness.StartProcess(
		ctx,
		t,
		testharness.MergeEnv(env),
		dir,
		name,
		args...,
	)
}

func runMCP(t *testing.T, env []string, lines []string) string {
	t.Helper()
	return testharness.RunMCP(t, systemBinaryPath, testharness.MergeEnv(env), lines)
}

func waitForHealth(t *testing.T, client *http.Client, port int) {
	t.Helper()
	testharness.WaitForHealth(t, client, port)
}

func postJob(t *testing.T, client *http.Client, port int, path string, body map[string]any) string {
	t.Helper()
	return testharness.PostJob(t, client, port, path, body)
}

func postSchedule(t *testing.T, client *http.Client, port int, body map[string]any) string {
	t.Helper()
	return testharness.PostSchedule(t, client, port, body)
}

func waitForJob(t *testing.T, client *http.Client, port int, id string) {
	t.Helper()
	testharness.WaitForJob(t, client, port, id)
}

func waitForJobs(t *testing.T, client *http.Client, port int, minCount int) {
	t.Helper()
	testharness.WaitForJobs(t, client, port, minCount)
}

func latestJobID(t *testing.T, client *http.Client, port int) string {
	t.Helper()
	return testharness.LatestJobID(t, client, port)
}

func assertJobResultContains(t *testing.T, client *http.Client, port int, id string, needle string) {
	t.Helper()
	testharness.AssertJobResultContains(t, client, port, id, needle)
}

func assertManifestExists(t *testing.T, dataDir string, jobID string) {
	t.Helper()
	manifestPath := filepath.Join(dataDir, "jobs", jobID, "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest %s: %v", manifestPath, err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("parse manifest %s: %v", manifestPath, err)
	}
	if payload["jobId"] != jobID {
		t.Fatalf("manifest jobId = %v, want %s", payload["jobId"], jobID)
	}
	if _, ok := payload["specVersion"]; !ok {
		t.Fatalf("manifest missing specVersion: %s", manifestPath)
	}
	files, _ := payload["files"].([]any)
	if len(files) == 0 {
		t.Fatalf("manifest missing files: %s", manifestPath)
	}
}

func waitForWebSocketJobLifecycle(t *testing.T, conn net.Conn, jobID string) map[string]bool {
	t.Helper()
	found := map[string]bool{}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		data, err := wsutil.ReadServerText(conn)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			t.Fatalf("read websocket message: %v", err)
		}
		var msg struct {
			Type    string `json:"type"`
			Payload struct {
				JobID  string `json:"jobId"`
				Status string `json:"status"`
			} `json:"payload"`
		}
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("decode websocket message: %v", err)
		}
		if msg.Payload.JobID != jobID {
			continue
		}
		found[msg.Type] = true
		if msg.Type == "job_status_changed" && msg.Payload.Status == "running" {
			found["job_status_changed"] = true
		}
		if msg.Type == "job_completed" {
			return found
		}
	}
	return found
}

func assertFileNotEmpty(t *testing.T, path string) {
	t.Helper()
	testharness.AssertFileNotEmpty(t, path)
}

func freePort(t *testing.T) int {
	t.Helper()
	return testharness.FreePort(t)
}

func requireLineCount(t *testing.T, path string, min int) {
	t.Helper()
	testharness.RequireLineCount(t, path, min)
}
