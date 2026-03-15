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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/testsite"
	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

var systemBinaryPath string
var systemProjectRoot string

func TestMain(m *testing.M) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	systemProjectRoot = filepath.Clean(filepath.Join(cwd, "..", ".."))
	tmpDir, err := os.MkdirTemp("", "spartan-system-bin-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	systemBinaryPath = filepath.Join(tmpDir, "spartan")
	build := exec.Command("go", "build", "-o", systemBinaryPath, "./cmd/spartan")
	build.Dir = systemProjectRoot
	build.Env = os.Environ()
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(tmpDir)
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
	return []string{
		"DATA_DIR=" + dataDir,
		"RATE_LIMIT_QPS=50",
		"RATE_LIMIT_BURST=100",
		"MAX_CONCURRENCY=4",
		"REQUEST_TIMEOUT_SECONDS=15",
	}
}

func runOK(t *testing.T, env []string, args ...string) {
	t.Helper()
	if err := run(t, env, args...); err != nil {
		t.Fatalf("%v", err)
	}
}

func run(t *testing.T, env []string, args ...string) error {
	t.Helper()
	timeout := 120 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, systemBinaryPath, args...)
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), env...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		output := out.String()
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("command timed out after %v: %v\n--- OUTPUT ---\n%s\n--------------", timeout, err, lastLines(output, 20))
		}
		return fmt.Errorf("command failed: %v\n--- OUTPUT ---\n%s\n--------------", err, output)
	}
	return nil
}

func startProcess(ctx context.Context, t *testing.T, env []string, dir string, name string, args ...string) (*exec.Cmd, func()) {
	t.Helper()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr

	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start process %s %v: %v", name, args, err)
	}

	cleanup := func() {
		if cmd.Process == nil {
			return
		}
		if runtime.GOOS != "windows" {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		} else {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}

	return cmd, cleanup
}

func runMCP(t *testing.T, env []string, lines []string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, systemBinaryPath, "mcp")
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), env...)
	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("stdout: %v", err)
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("start mcp: %v", err)
	}

	go func() {
		defer stdin.Close()
		for _, line := range lines {
			_, _ = io.WriteString(stdin, line+"\n")
		}
	}()

	out, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("mcp failed: %v", err)
	}
	return string(out)
}

func waitForHealth(t *testing.T, client *http.Client, port int) {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/healthz", port)
	for i := 0; i < 50; i++ {
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("server not healthy on port %d", port)
}

func postJob(t *testing.T, client *http.Client, port int, path string, body map[string]any) string {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("post job marshal: %v", err)
	}
	resp, err := client.Post(fmt.Sprintf("http://127.0.0.1:%d%s", port, path), "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("post job: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(resp.Body)
		t.Fatalf("post job status: %d body=%s", resp.StatusCode, string(payload))
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode job: %v", err)
	}
	id, _ := payload["id"].(string)
	if id == "" {
		t.Fatalf("missing job id")
	}
	return id
}

func postSchedule(t *testing.T, client *http.Client, port int, body map[string]any) string {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("post schedule marshal: %v", err)
	}
	resp, err := client.Post(fmt.Sprintf("http://127.0.0.1:%d/v1/schedules", port), "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("post schedule: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		payload, _ := io.ReadAll(resp.Body)
		t.Fatalf("post schedule status: %d body=%s", resp.StatusCode, string(payload))
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode schedule: %v", err)
	}
	id, _ := payload["id"].(string)
	if id == "" {
		t.Fatalf("missing schedule id")
	}
	if _, hasParams := payload["params"]; hasParams {
		t.Fatalf("schedule response unexpectedly exposed legacy params")
	}
	if _, hasSpecVersion := payload["specVersion"]; hasSpecVersion {
		t.Fatalf("schedule response unexpectedly exposed legacy specVersion")
	}
	request, ok := payload["request"].(map[string]any)
	if !ok {
		t.Fatalf("schedule response missing request object: %#v", payload)
	}
	if request["url"] == nil {
		t.Fatalf("schedule response request missing url: %#v", request)
	}
	return id
}

func waitForJob(t *testing.T, client *http.Client, port int, id string) {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/jobs/%s", port, id)
	for i := 0; i < 100; i++ {
		resp, err := client.Get(url)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		var payload map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		_ = resp.Body.Close()
		switch payload["status"] {
		case "succeeded":
			return
		case "failed":
			t.Fatalf("job %s failed", id)
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("job %s timeout", id)
}

func waitForJobs(t *testing.T, client *http.Client, port int, minCount int) {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/jobs", port)
	for i := 0; i < 100; i++ {
		resp, err := client.Get(url)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		var payload map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		_ = resp.Body.Close()
		jobs, _ := payload["jobs"].([]any)
		if len(jobs) >= minCount {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("jobs not scheduled")
}

func latestJobID(t *testing.T, client *http.Client, port int) string {
	t.Helper()
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/v1/jobs?limit=1", port))
	if err != nil {
		t.Fatalf("latest job request: %v", err)
	}
	defer resp.Body.Close()
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode latest job payload: %v", err)
	}
	jobs, _ := payload["jobs"].([]any)
	if len(jobs) == 0 {
		t.Fatal("latestJobID(): no jobs returned")
	}
	job, _ := jobs[0].(map[string]any)
	id, _ := job["id"].(string)
	if id == "" {
		t.Fatal("latestJobID(): missing id")
	}
	return id
}

func assertJobResultContains(t *testing.T, client *http.Client, port int, id string, needle string) {
	t.Helper()
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/v1/jobs/%s/results", port, id))
	if err != nil {
		t.Fatalf("job results: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("job results status: %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), needle) {
		t.Fatalf("expected %q in job results", needle)
	}
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
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.Size() == 0 {
		t.Fatalf("empty file %s", path)
	}
}

func lastLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return "... (truncated) ...\n" + strings.Join(lines[len(lines)-n:], "\n")
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func requireLineCount(t *testing.T, path string, min int) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	count := 0
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) != "" {
			count++
		}
	}
	if count < min {
		t.Fatalf("expected at least %d lines in %s, got %d", min, path, count)
	}
}
