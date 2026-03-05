// Package e2e provides deterministic end-to-end integration tests for core product workflows.
//
// Purpose:
//   - Validate CLI, API, MCP, scheduler, export, and TUI flows against stable local infrastructure.
//
// Responsibilities:
//   - Build the test binary once per package run.
//   - Execute isolated command and server flows from temp working directories.
//   - Assert end-to-end behavior through the local test fixture and API surface.
//
// Scope:
//   - Core workflow coverage only; auth and web preview live in separate e2e test files.
//
// Usage:
//   - Runs as part of go test ./internal/e2e/...
//
// Invariants/Assumptions:
//   - Uses temp working directories so local .env files do not leak into tests.
//   - Uses the shared deterministic local fixture instead of third-party websites.
package e2e

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

	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/testsite"
)

var spartanPath string
var projectRoot string

func TestMain(m *testing.M) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	projectRoot = filepath.Clean(filepath.Join(cwd, "..", ".."))
	tmpDir, err := os.MkdirTemp("", "spartan-bin-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	spartanPath = filepath.Join(tmpDir, "spartan")
	build := exec.Command("go", "build", "-o", spartanPath, "./cmd/spartan")
	build.Dir = projectRoot
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

func Test00Preflight(t *testing.T) {
	t.Log("Checking browser availability...")
	usePlaywright := os.Getenv("USE_PLAYWRIGHT") == "1"
	if err := fetch.CheckBrowserAvailability(usePlaywright); err != nil {
		t.Fatalf("Pre-flight browser check failed: %v. Ensure Chromedp/Playwright is installed.", err)
	}
}

func TestCLIHelp(t *testing.T) {
	runOK(t, nil, "--help")
	runOK(t, nil, "auth", "--help")
	runOK(t, nil, "schedule", "--help")
	runOK(t, nil, "server", "--help")
	runOK(t, nil, "mcp", "--help")
	runOK(t, nil, "tui", "--help")
}

func TestTUISmoke(t *testing.T) {
	dataDir := t.TempDir()
	runOK(t, baseEnv(dataDir), "tui", "--smoke")
}

func TestCLIHTTPFlow(t *testing.T) {
	dataDir := t.TempDir()
	outDir := t.TempDir()
	env := baseEnv(dataDir)
	site := testsite.Start(t)

	scrapeOut := filepath.Join(outDir, "scrape.json")
	runOK(t, env, "scrape", "--url", site.ScrapeURL(), "--wait", "--wait-timeout", "60", "--out", scrapeOut)
	assertJSONContains(t, scrapeOut, "Example Domain")

	crawlOut := filepath.Join(outDir, "crawl.jsonl")
	runOK(t, env, "crawl", "--url", site.CrawlRootURL(), "--max-depth", "1", "--max-pages", "5", "--wait", "--wait-timeout", "90", "--out", crawlOut)
	requireLineCount(t, crawlOut, 2)

	researchOut := filepath.Join(outDir, "research.jsonl")
	runOK(t, env, "research", "--query", "example", "--urls", strings.Join(site.ResearchURLs(), ","), "--wait", "--wait-timeout", "60", "--out", researchOut)
	assertJSONContains(t, researchOut, `"summary"`)
}

func TestAPIMCPSchedulerExport(t *testing.T) {
	dataDir := t.TempDir()
	port := freePort(t)
	env := baseEnv(dataDir)
	env = append(env, "PORT="+strconv.Itoa(port))
	site := testsite.Start(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srvCmd, cleanup := startProcess(ctx, t, env, t.TempDir(), spartanPath, "server")
	defer cleanup()

	client := &http.Client{Timeout: 5 * time.Second}
	waitForHealth(t, client, port)

	jobID := postJob(t, client, port, "/v1/scrape", map[string]interface{}{
		"url":            site.ScrapeURL(),
		"headless":       false,
		"playwright":     false,
		"timeoutSeconds": 30,
	})
	waitForJob(t, client, port, jobID)
	assertJobResultContains(t, client, port, jobID, "Example Domain")

	exportOut := filepath.Join(t.TempDir(), "export.md")
	runOK(t, env, "export", "--job-id", jobID, "--format", "md", "--out", exportOut)
	assertFileNotEmpty(t, exportOut)

	runOK(t, env, "schedule", "add", "--kind", "scrape", "--interval", "1", "--url", site.ScrapeURL())
	waitForJobs(t, client, port, 1)

	runOK(t, env, "mcp", "--help")

	mcpOut := runMCP(t, env, []string{
		`{"id":1,"method":"initialize"}`,
		`{"id":2,"method":"tools/list"}`,
	})
	if !strings.Contains(mcpOut, `"tools"`) {
		t.Fatalf("expected MCP tools list in output")
	}
	if !strings.Contains(mcpOut, `"preProcessors"`) {
		t.Fatalf("expected preProcessors in tools schema")
	}
	if !strings.Contains(mcpOut, `"postProcessors"`) {
		t.Fatalf("expected postProcessors in tools schema")
	}
	if !strings.Contains(mcpOut, `"transformers"`) {
		t.Fatalf("expected transformers in tools schema")
	}

	mcpCallOut := runMCP(t, env, []string{
		fmt.Sprintf(`{"id":3,"method":"tools/call","params":{"name":"scrape_page","arguments":{"url":%q,"preProcessors":["prep1","prep2"],"postProcessors":["post1"],"transformers":["trans1"],"incremental":true}}}`, site.ScrapeURL()),
	})
	if strings.Contains(mcpCallOut, `"error"`) && strings.Contains(mcpCallOut, `"message"`) {
		var resp map[string]interface{}
		if err := json.Unmarshal([]byte(mcpCallOut), &resp); err == nil {
			if errMsg, ok := resp["error"].(map[string]interface{}); ok {
				msg, _ := errMsg["message"].(string)
				if !strings.Contains(msg, "job failed") {
					t.Fatalf("unexpected MCP error: %s", msg)
				}
			}
		}
	}

	cancel()
	_ = srvCmd.Wait()
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

	cmd := exec.CommandContext(ctx, spartanPath, args...)
	cmd.Dir = t.TempDir()
	cmd.Env = append(os.Environ(), env...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	t.Logf("Running: %s %s", spartanPath, strings.Join(args, " "))
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

	t.Logf("Starting process: %s %s in %s", name, strings.Join(args, " "), dir)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start process %s %v: %v", name, args, err)
	}

	cleanup := func() {
		if cmd.Process == nil {
			return
		}
		if runtime.GOOS != "windows" {
			// Kill the entire process group
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		} else {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}

	return cmd, cleanup
}

func runUntilContains(t *testing.T, env []string, outPath string, needle string, attempts int, args ...string) {
	t.Helper()
	var lastErr error
	for i := 0; i < attempts; i++ {
		t.Logf("Attempt %d/%d to find %q in %s", i+1, attempts, needle, outPath)
		if err := run(t, env, args...); err != nil {
			lastErr = err
			time.Sleep(2 * time.Second)
			continue
		}
		data, err := os.ReadFile(outPath)
		if err != nil {
			lastErr = err
			time.Sleep(500 * time.Millisecond)
			continue
		}
		text := string(data)
		if strings.Contains(text, needle) {
			return
		}
		lastErr = fmt.Errorf("missing %q in %s\n--- OUTPUT ---\n%s\n--------------", needle, outPath, text)
		time.Sleep(2 * time.Second)
	}
	if lastErr != nil {
		t.Fatalf("verification failed after %d attempts: %v", attempts, lastErr)
	}
	t.Fatalf("verification failed")
}

func lastLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return "... (truncated) ...\n" + strings.Join(lines[len(lines)-n:], "\n")
}

func runMCP(t *testing.T, env []string, lines []string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, spartanPath, "mcp")
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
			if _, err := io.WriteString(stdin, line+"\n"); err != nil {
				return
			}
		}
	}()

	out, err := io.ReadAll(stdout)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	select {
	case err := <-waitDone:
		if err != nil {
			t.Fatalf("mcp failed: %v", err)
		}
	case <-ctx.Done():
		if runtime.GOOS != "windows" {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		} else {
			_ = cmd.Process.Kill()
		}
		t.Fatalf("mcp timed out")
	}

	return string(out)
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

func waitForHealth(t *testing.T, client *http.Client, port int) {
	t.Helper()
	base := fmt.Sprintf("http://127.0.0.1:%d/healthz", port)
	for i := 0; i < 50; i++ {
		resp, err := client.Get(base)
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("server not healthy on port %d", port)
}

func postJob(t *testing.T, client *http.Client, port int, path string, body map[string]interface{}) string {
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
		t.Fatalf("post job status: %d", resp.StatusCode)
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode job: %v", err)
	}
	id, _ := payload["id"].(string)
	if id == "" {
		t.Fatalf("missing job id")
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
		var payload map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		_ = resp.Body.Close()
		if payload["status"] == "succeeded" {
			return
		}
		if payload["status"] == "failed" {
			t.Fatalf("job failed")
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("job timeout")
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
		var payload map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		_ = resp.Body.Close()
		jobs, _ := payload["jobs"].([]interface{})
		if len(jobs) >= minCount {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("jobs not scheduled")
}

func assertJobResultContains(t *testing.T, client *http.Client, port int, id string, needle string) {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/jobs/%s/results", port, id)
	resp, err := client.Get(url)
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

func assertJSONContains(t *testing.T, path, contains string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(raw), contains) {
		t.Fatalf("expected %q in %s", contains, path)
	}
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

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}
