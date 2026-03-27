// Package testharness provides shared heavy-validation helpers for deterministic system and e2e suites.
//
// Purpose:
// - Centralize binary build, process lifecycle, browser startup, polling, and JSON-envelope helpers used by the heavy validation suites.
//
// Responsibilities:
// - Build the CLI binary for test packages.
// - Start and stop child processes safely.
// - Provide stable HTTP polling and JSON-envelope helpers.
// - Start local Vite preview instances and headless browsers for browser-visible checks.
//
// Scope:
// - Test-only helper code for internal/system and internal/e2e.
//
// Usage:
// - Import from test packages that need shared heavy-validation setup and assertions.
//
// Invariants/Assumptions:
// - Callers pass fully resolved project paths and explicit env slices.
// - All spawned processes must be cleaned up even on failure.
// - JSON helper functions fail fast when response envelopes drift.
package testharness

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
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
)

func BuildBinary(projectRoot string, tempPattern string) (string, func(), error) {
	tmpDir, err := os.MkdirTemp("", tempPattern)
	if err != nil {
		return "", nil, err
	}
	cleanup := func() {
		_ = os.RemoveAll(tmpDir)
	}

	binaryPath := filepath.Join(tmpDir, "spartan")
	build := exec.Command("go", "build", "-o", binaryPath, "./cmd/spartan")
	build.Dir = projectRoot
	build.Env = os.Environ()
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		cleanup()
		return "", nil, err
	}

	return binaryPath, cleanup, nil
}

func BaseEnv(dataDir string) []string {
	return []string{
		"DATA_DIR=" + dataDir,
		"RATE_LIMIT_QPS=50",
		"RATE_LIMIT_BURST=100",
		"MAX_CONCURRENCY=4",
		"REQUEST_TIMEOUT_SECONDS=15",
	}
}

func MergeEnv(overrides []string, extra ...string) []string {
	values := map[string]string{}
	for _, item := range os.Environ() {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		values[key] = value
	}
	for _, item := range append(overrides, extra...) {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			continue
		}
		values[key] = value
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		out = append(out, key+"="+values[key])
	}
	return out
}

func RunCommand(t *testing.T, binaryPath string, env []string, args ...string) error {
	t.Helper()
	return RunCommandInDir(t, binaryPath, t.TempDir(), env, args...)
}

func RunCommandInDir(t *testing.T, binaryPath string, dir string, env []string, args ...string) error {
	t.Helper()
	timeout := 120 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Dir = dir
	cmd.Env = env
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	t.Logf("Running: %s %s", binaryPath, strings.Join(args, " "))
	if err := cmd.Run(); err != nil {
		output := out.String()
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("command timed out after %v: %v\n--- OUTPUT ---\n%s\n--------------", timeout, err, lastLines(output, 20))
		}
		return fmt.Errorf("command failed: %v\n--- OUTPUT ---\n%s\n--------------", err, output)
	}
	return nil
}

func StartProcess(ctx context.Context, t *testing.T, env []string, dir string, name string, args ...string) (*exec.Cmd, func()) {
	t.Helper()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr

	if runtime.GOOS != "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	t.Logf("Starting process: %s %s in %s", name, strings.Join(args, " "), dir)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start process %s %v: %v", name, args, err)
	}

	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			if cmd.Process == nil {
				return
			}
			if runtime.GOOS != "windows" {
				_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
			} else {
				_ = cmd.Process.Kill()
			}
			_ = cmd.Wait()
		})
	}

	t.Cleanup(cleanup)
	return cmd, cleanup
}

func StartVitePreview(ctx context.Context, t *testing.T, webDir string, env []string, port int) (*exec.Cmd, func()) {
	t.Helper()
	cmd, cleanup := StartProcess(
		ctx,
		t,
		env,
		webDir,
		"pnpm",
		"exec",
		"vite",
		"preview",
		"--host",
		"127.0.0.1",
		"--port",
		strconv.Itoa(port),
	)
	client := &http.Client{Timeout: 2 * time.Second}
	WaitForPreview(t, client, port)
	return cmd, cleanup
}

func RunMCP(t *testing.T, binaryPath string, env []string, lines []string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, binaryPath, "mcp")
	cmd.Dir = t.TempDir()
	cmd.Env = env
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

func WaitForHealth(t *testing.T, client *http.Client, port int) {
	t.Helper()
	waitForStatusOK(t, client, fmt.Sprintf("http://127.0.0.1:%d/healthz", port), 50, 100*time.Millisecond, fmt.Sprintf("server not healthy on port %d", port))
}

func WaitForPreview(t *testing.T, client *http.Client, port int) {
	t.Helper()
	waitForStatusOK(t, client, fmt.Sprintf("http://127.0.0.1:%d/", port), 50, 100*time.Millisecond, fmt.Sprintf("web preview not reachable on port %d", port))
}

func PostJob(t *testing.T, client *http.Client, port int, path string, body any) string {
	t.Helper()
	payload := postCreatedJSON(t, client, fmt.Sprintf("http://127.0.0.1:%d%s", port, path), body, "job")
	return stringField(t, objectField(t, payload, "job"), "id")
}

func PostSchedule(t *testing.T, client *http.Client, port int, body any) string {
	t.Helper()
	payload := postCreatedJSON(t, client, fmt.Sprintf("http://127.0.0.1:%d/v1/schedules", port), body, "schedule")
	id := stringField(t, payload, "id")
	if _, hasParams := payload["params"]; hasParams {
		t.Fatalf("schedule response unexpectedly exposed legacy params")
	}
	if _, hasSpecVersion := payload["specVersion"]; hasSpecVersion {
		t.Fatalf("schedule response unexpectedly exposed legacy specVersion")
	}
	request := objectField(t, payload, "request")
	if request["url"] == nil {
		t.Fatalf("schedule response request missing url: %#v", request)
	}
	return id
}

func WaitForJob(t *testing.T, client *http.Client, port int, id string) {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/jobs/%s", port, id)
	for i := 0; i < 100; i++ {
		resp, err := client.Get(url)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		payload := decodeJSONResponse(t, resp, "job detail")
		jobPayload := objectField(t, payload, "job")
		switch status := stringField(t, jobPayload, "status"); status {
		case "succeeded":
			return
		case "failed":
			t.Fatalf("job %s failed", id)
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("job %s timeout", id)
}

func WaitForJobs(t *testing.T, client *http.Client, port int, minCount int) {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/jobs", port)
	for i := 0; i < 100; i++ {
		resp, err := client.Get(url)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		payload := decodeJSONResponse(t, resp, "jobs list")
		jobs := arrayField(t, payload, "jobs")
		if len(jobs) >= minCount {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("jobs not scheduled")
}

func LatestJobID(t *testing.T, client *http.Client, port int) string {
	t.Helper()
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/v1/jobs?limit=1", port))
	if err != nil {
		t.Fatalf("latest job request: %v", err)
	}
	payload := decodeJSONResponse(t, resp, "latest job payload")
	jobs := arrayField(t, payload, "jobs")
	if len(jobs) == 0 {
		t.Fatal("latestJobID(): no jobs returned")
	}
	job, ok := jobs[0].(map[string]any)
	if !ok {
		t.Fatalf("latest job item was not an object: %#v", jobs[0])
	}
	return stringField(t, job, "id")
}

func AssertJobResultContains(t *testing.T, client *http.Client, port int, id string, needle string) {
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

func AssertFileNotEmpty(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.Size() == 0 {
		t.Fatalf("empty file %s", path)
	}
}

func RequireLineCount(t *testing.T, path string, min int) {
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

func FreePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("free port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func NewHeadlessBrowserContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	allocatorOpts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	allocatorOpts = append(allocatorOpts,
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.WindowSize(1365, 780),
	)
	allocCtx, cancelAlloc := chromedp.NewExecAllocator(context.Background(), allocatorOpts...)
	browserCtx, cancelBrowser := chromedp.NewContext(allocCtx)
	ctx, cancelTimeout := context.WithTimeout(browserCtx, 30*time.Second)
	return ctx, func() {
		cancelTimeout()
		cancelBrowser()
		cancelAlloc()
	}
}

func waitForStatusOK(t *testing.T, client *http.Client, url string, attempts int, delay time.Duration, timeoutMessage string) {
	t.Helper()
	for i := 0; i < attempts; i++ {
		resp, err := client.Get(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			return
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		time.Sleep(delay)
	}
	t.Fatal(timeoutMessage)
}

func postCreatedJSON(t *testing.T, client *http.Client, url string, body any, label string) map[string]any {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("post %s marshal: %v", label, err)
	}
	resp, err := client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("post %s: %v", label, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		payload, _ := io.ReadAll(resp.Body)
		t.Fatalf("post %s status: %d body=%s", label, resp.StatusCode, string(payload))
	}
	payload := map[string]any{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode %s: %v", label, err)
	}
	return payload
}

func decodeJSONResponse(t *testing.T, resp *http.Response, label string) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	payload := map[string]any{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode %s: %v", label, err)
	}
	return payload
}

func objectField(t *testing.T, payload map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := payload[key].(map[string]any)
	if !ok {
		t.Fatalf("missing %s object: %#v", key, payload)
	}
	return value
}

func arrayField(t *testing.T, payload map[string]any, key string) []any {
	t.Helper()
	value, ok := payload[key].([]any)
	if !ok {
		t.Fatalf("missing %s array: %#v", key, payload)
	}
	return value
}

func stringField(t *testing.T, payload map[string]any, key string) string {
	t.Helper()
	value, _ := payload[key].(string)
	if value == "" {
		t.Fatalf("missing %s string: %#v", key, payload)
	}
	return value
}

func lastLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return "... (truncated) ...\n" + strings.Join(lines[len(lines)-n:], "\n")
}
