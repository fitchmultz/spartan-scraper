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
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/testharness"
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
	binaryPath, cleanup, err := testharness.BuildBinary(projectRoot, "spartan-bin-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	spartanPath = binaryPath
	code := m.Run()
	cleanup()
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
	return testharness.RunCommand(t, spartanPath, testharness.MergeEnv(env), args...)
}

func startProcess(ctx context.Context, t *testing.T, env []string, dir string, name string, args ...string) (*exec.Cmd, func()) {
	t.Helper()
	return testharness.StartProcess(ctx, t, testharness.MergeEnv(env), dir, name, args...)
}

func startWebPreview(ctx context.Context, t *testing.T, env []string, port int) (*exec.Cmd, func()) {
	t.Helper()
	return testharness.StartVitePreview(
		ctx,
		t,
		filepath.Join(projectRoot, "web"),
		testharness.MergeEnv(env),
		port,
	)
}

func newHeadlessBrowserContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return testharness.NewHeadlessBrowserContext(t)
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

func baseEnv(dataDir string) []string {
	return testharness.BaseEnv(dataDir)
}

func waitForHealth(t *testing.T, client *http.Client, port int) {
	t.Helper()
	testharness.WaitForHealth(t, client, port)
}

func postJob(t *testing.T, client *http.Client, port int, path string, body map[string]interface{}) string {
	t.Helper()
	return testharness.PostJob(t, client, port, path, body)
}

func waitForJob(t *testing.T, client *http.Client, port int, id string) {
	t.Helper()
	testharness.WaitForJob(t, client, port, id)
}

func waitForJobs(t *testing.T, client *http.Client, port int, minCount int) {
	t.Helper()
	testharness.WaitForJobs(t, client, port, minCount)
}

func assertJobResultContains(t *testing.T, client *http.Client, port int, id string, needle string) {
	t.Helper()
	testharness.AssertJobResultContains(t, client, port, id, needle)
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
	testharness.AssertFileNotEmpty(t, path)
}

func requireLineCount(t *testing.T, path string, min int) {
	t.Helper()
	testharness.RequireLineCount(t, path, min)
}

func freePort(t *testing.T) int {
	t.Helper()
	return testharness.FreePort(t)
}
