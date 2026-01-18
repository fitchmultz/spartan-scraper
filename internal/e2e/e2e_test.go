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
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
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
	server := newFixtureServer()
	defer server.Close()

	dataDir := t.TempDir()
	outDir := t.TempDir()
	env := baseEnv(dataDir)

	scrapeOut := filepath.Join(outDir, "scrape.json")
	runOK(t, env, "scrape", "--url", server.URL+"/", "--wait", "--wait-timeout", "30", "--out", scrapeOut)
	assertJSONContains(t, scrapeOut, "Example Page")

	crawlOut := filepath.Join(outDir, "crawl.jsonl")
	runOK(t, env, "crawl", "--url", server.URL+"/", "--max-depth", "1", "--max-pages", "5", "--wait", "--wait-timeout", "30", "--out", crawlOut)
	requireLineCount(t, crawlOut, 2)

	researchOut := filepath.Join(outDir, "research.jsonl")
	runOK(t, env, "research", "--query", "example", "--urls", server.URL+"/,"+server.URL+"/page1", "--wait", "--wait-timeout", "30", "--out", researchOut)
	assertJSONContains(t, researchOut, "Summary")
}

func TestAuthProfilesAndHeadlessLogin(t *testing.T) {
	server := newFixtureServer()
	defer server.Close()

	dataDir := t.TempDir()
	outDir := t.TempDir()
	env := baseEnv(dataDir)

	runOK(t, env, "auth", "set",
		"--name", "fixture",
		"--login-url", server.URL+"/login",
		"--login-user-selector", "#username",
		"--login-pass-selector", "#password",
		"--login-submit-selector", "#submit",
		"--login-user", "user",
		"--login-pass", "pass",
	)

	runOK(t, env, "auth", "list")

	headlessOut := filepath.Join(outDir, "secure-headless.json")
	runOK(t, env, "scrape", "--url", server.URL+"/secure", "--headless", "--auth-profile", "fixture", "--wait", "--wait-timeout", "60", "--out", headlessOut)
	assertJSONContains(t, headlessOut, "Secure Area")

	playwrightOut := filepath.Join(outDir, "secure-playwright.json")
	runOK(t, env, "scrape", "--url", server.URL+"/secure", "--headless", "--playwright", "--auth-profile", "fixture", "--wait", "--wait-timeout", "60", "--out", playwrightOut)
	assertJSONContains(t, playwrightOut, "Secure Area")

	runOK(t, env, "auth", "delete", "--name", "fixture")
}

func TestAPIMCPSchedulerExport(t *testing.T) {
	server := newFixtureServer()
	defer server.Close()

	dataDir := t.TempDir()
	port := freePort(t)
	env := baseEnv(dataDir)
	env = append(env, "PORT="+strconv.Itoa(port))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srvCmd := exec.CommandContext(ctx, spartanPath, "server")
	srvCmd.Dir = projectRoot
	srvCmd.Env = env
	srvCmd.Stdout = io.Discard
	srvCmd.Stderr = os.Stderr
	if err := srvCmd.Start(); err != nil {
		t.Fatalf("start server: %v", err)
	}
	waitForHealth(t, port)

	jobID := postJob(t, port, "/v1/scrape", map[string]interface{}{
		"url":            server.URL + "/",
		"headless":       false,
		"playwright":     false,
		"timeoutSeconds": 10,
	})
	waitForJob(t, port, jobID)

	exportOut := filepath.Join(t.TempDir(), "export.md")
	runOK(t, env, "export", "--job-id", jobID, "--format", "md", "--out", exportOut)
	assertFileNotEmpty(t, exportOut)

	runOK(t, env, "schedule", "add", "--kind", "scrape", "--interval", "1", "--url", server.URL+"/")
	waitForJobs(t, port, 1)

	runOK(t, env, "mcp", "--help")
	mcpOut := runMCP(t, env, []string{
		`{"id":1,"method":"initialize"}`,
		`{"id":2,"method":"tools/list"}`,
		fmt.Sprintf(`{"id":3,"method":"tools/call","params":{"name":"scrape_page","arguments":{"url":"%s"}}}`, server.URL+"/"),
	})
	if !strings.Contains(mcpOut, `"tools"`) {
		t.Fatalf("expected MCP tools list in output")
	}

	cancel()
	_ = srvCmd.Wait()
}

func runOK(t *testing.T, env []string, args ...string) {
	t.Helper()
	cmd := exec.Command(spartanPath, args...)
	cmd.Dir = projectRoot
	cmd.Env = append(os.Environ(), env...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("command failed: %v\n%s", err, out.String())
	}
}

func runMCP(t *testing.T, env []string, lines []string) string {
	t.Helper()
	cmd := exec.Command(spartanPath, "mcp")
	cmd.Dir = projectRoot
	cmd.Env = append(os.Environ(), env...)
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
	for _, line := range lines {
		_, _ = io.WriteString(stdin, line+"\n")
	}
	_ = stdin.Close()
	out, _ := io.ReadAll(stdout)
	_ = cmd.Wait()
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

func newFixtureServer() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `<html><head><title>Example Page</title></head><body>`+
			`<a href="/page1">Page 1</a>`+
			`<a href="/page2">Page 2</a>`+
			`<a href="https://example.com">External</a>`+
			`</body></html>`)
	})
	mux.HandleFunc("/page1", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `<html><head><title>Page One</title></head><body>Summary content</body></html>`)
	})
	mux.HandleFunc("/page2", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `<html><head><title>Page Two</title></head><body>More content</body></html>`)
	})
	mux.HandleFunc("/login", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `<html><body><form action="/do_login" method="post">`+
			`<input id="username" name="username">`+
			`<input id="password" name="password" type="password">`+
			`<button id="submit" type="submit">Login</button>`+
			`</form></body></html>`)
	})
	mux.HandleFunc("/do_login", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.Form.Get("username") == "user" && r.Form.Get("password") == "pass" {
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "ok", Path: "/"})
			http.Redirect(w, r, "/secure", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
	})
	mux.HandleFunc("/secure", func(w http.ResponseWriter, r *http.Request) {
		_, err := r.Cookie("session")
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, "Unauthorized")
			return
		}
		fmt.Fprint(w, `<html><head><title>Secure</title></head><body>Secure Area</body></html>`)
	})
	return httptest.NewServer(mux)
}

func waitForHealth(t *testing.T, port int) {
	t.Helper()
	base := fmt.Sprintf("http://127.0.0.1:%d/healthz", port)
	for i := 0; i < 50; i++ {
		resp, err := http.Get(base)
		if err == nil && resp.StatusCode == http.StatusOK {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("server not healthy on port %d", port)
}

func postJob(t *testing.T, port int, path string, body map[string]interface{}) string {
	t.Helper()
	data, _ := json.Marshal(body)
	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d%s", port, path), "application/json", bytes.NewReader(data))
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

func waitForJob(t *testing.T, port int, id string) {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/jobs/%s", port, id)
	for i := 0; i < 100; i++ {
		resp, err := http.Get(url)
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

func waitForJobs(t *testing.T, port int, minCount int) {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/jobs", port)
	for i := 0; i < 100; i++ {
		resp, err := http.Get(url)
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
