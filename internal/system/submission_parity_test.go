package system

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/testsite"
)

type submissionParityFixture struct {
	name     string
	expected jobs.JobSpec
	restPath string
	restBody any
	cliArgs  func(t *testing.T, extractConfigPath string) []string
	mcpTool  string
	mcpArgs  map[string]any
}

func TestJobSubmissionTypedSpecParityAcrossSurfaces(t *testing.T) {
	site := testsite.Start(t)

	extractOpts := extract.ExtractOptions{
		Inline: &extract.Template{
			Name: "parity",
			Selectors: []extract.SelectorRule{{
				Name:     "title",
				Selector: "h1",
				Attr:     "text",
				Trim:     true,
			}},
		},
	}
	authOpts := fetch.AuthOptions{
		Basic:   "user:passwd",
		Headers: map[string]string{"X-Test-Header": "parity"},
		Cookies: []string{"session=parity"},
		ProxyHints: &fetch.ProxySelectionHints{
			PreferredRegion: "us-east",
			RequiredTags:    []string{"residential"},
			ExcludeProxyIDs: []string{"proxy-9"},
		},
	}
	device := fetch.GetDevicePreset("iphone15")
	if device == nil {
		t.Fatal("missing iphone15 device preset")
	}
	screenshot := &fetch.ScreenshotConfig{
		Enabled:  true,
		FullPage: false,
		Format:   fetch.ScreenshotFormatJPEG,
		Quality:  85,
		Width:    900,
		Height:   700,
	}
	intercept := &fetch.NetworkInterceptConfig{
		Enabled:             true,
		URLPatterns:         []string{"**/api/**"},
		ResourceTypes:       []fetch.InterceptedResourceType{fetch.ResourceTypeXHR, fetch.ResourceTypeFetch},
		CaptureRequestBody:  false,
		CaptureResponseBody: true,
		MaxBodySize:         2048,
		MaxEntries:          25,
	}

	fixtures := []submissionParityFixture{
		{
			name: "scrape",
			expected: jobs.JobSpec{
				Kind:             model.KindScrape,
				URL:              site.ScrapeURL(),
				Method:           httpMethodPost,
				Body:             []byte(`{"hello":"world"}`),
				ContentType:      "application/json",
				Headless:         false,
				UsePlaywright:    false,
				Auth:             authOpts,
				TimeoutSeconds:   30,
				Extract:          extractOpts,
				Incremental:      true,
				Screenshot:       screenshot,
				Device:           device,
				NetworkIntercept: intercept,
			},
			restPath: "/v1/scrape",
			restBody: api.ScrapeRequest{
				URL:              site.ScrapeURL(),
				Method:           httpMethodPost,
				Body:             `{"hello":"world"}`,
				ContentType:      "application/json",
				Headless:         false,
				TimeoutSeconds:   30,
				Auth:             cloneAuthOptions(authOpts),
				Extract:          cloneExtractOptions(extractOpts),
				Incremental:      boolPtr(true),
				Screenshot:       cloneScreenshotConfig(screenshot),
				Device:           cloneDevice(device),
				NetworkIntercept: cloneNetworkIntercept(intercept),
			},
			cliArgs: func(t *testing.T, extractConfigPath string) []string {
				t.Helper()
				return []string{
					"scrape",
					"--url", site.ScrapeURL(),
					"--method", httpMethodPost,
					"--body", `{"hello":"world"}`,
					"--content-type", "application/json",
					"--timeout", "30",
					"--auth-basic", "user:passwd",
					"--header", "X-Test-Header: parity",
					"--cookie", "session=parity",
					"--proxy-region", "us-east",
					"--proxy-tag", "residential",
					"--exclude-proxy-id", "proxy-9",
					"--extract-config", extractConfigPath,
					"--incremental",
					"--screenshot",
					"--screenshot-full-page=false",
					"--screenshot-format", "jpeg",
					"--screenshot-quality", "85",
					"--screenshot-width", "900",
					"--screenshot-height", "700",
					"--device", "iphone15",
					"--intercept-enabled",
					"--intercept-pattern", "**/api/**",
					"--intercept-resource-type", "xhr",
					"--intercept-resource-type", "fetch",
					"--intercept-request-body=false",
					"--intercept-response-body=true",
					"--intercept-max-body-size", "2048",
					"--intercept-max-entries", "25",
				}
			},
			mcpTool: "scrape_page",
			mcpArgs: mustJSONMap(t, api.ScrapeRequest{
				URL:              site.ScrapeURL(),
				Method:           httpMethodPost,
				Body:             `{"hello":"world"}`,
				ContentType:      "application/json",
				Headless:         false,
				TimeoutSeconds:   30,
				Auth:             cloneAuthOptions(authOpts),
				Extract:          cloneExtractOptions(extractOpts),
				Incremental:      boolPtr(true),
				Screenshot:       cloneScreenshotConfig(screenshot),
				Device:           cloneDevice(device),
				NetworkIntercept: cloneNetworkIntercept(intercept),
			}),
		},
		{
			name: "crawl",
			expected: jobs.JobSpec{
				Kind:             model.KindCrawl,
				URL:              site.CrawlRootURL(),
				MaxDepth:         1,
				MaxPages:         4,
				Headless:         false,
				UsePlaywright:    false,
				Auth:             authOpts,
				TimeoutSeconds:   30,
				Extract:          extractOpts,
				Incremental:      true,
				IncludePatterns:  []string{"/crawl/**"},
				ExcludePatterns:  []string{"/crawl/b"},
				RespectRobotsTxt: true,
				SkipDuplicates:   true,
				SimHashThreshold: 1,
				Screenshot:       screenshot,
				Device:           device,
				NetworkIntercept: intercept,
			},
			restPath: "/v1/crawl",
			restBody: api.CrawlRequest{
				URL:              site.CrawlRootURL(),
				MaxDepth:         1,
				MaxPages:         4,
				Headless:         false,
				TimeoutSeconds:   30,
				Auth:             cloneAuthOptions(authOpts),
				Extract:          cloneExtractOptions(extractOpts),
				Incremental:      boolPtr(true),
				IncludePatterns:  []string{"/crawl/**"},
				ExcludePatterns:  []string{"/crawl/b"},
				RespectRobotsTxt: boolPtr(true),
				SkipDuplicates:   boolPtr(true),
				SimHashThreshold: intPtr(1),
				Screenshot:       cloneScreenshotConfig(screenshot),
				Device:           cloneDevice(device),
				NetworkIntercept: cloneNetworkIntercept(intercept),
			},
			cliArgs: func(t *testing.T, extractConfigPath string) []string {
				t.Helper()
				return []string{
					"crawl",
					"--url", site.CrawlRootURL(),
					"--max-depth", "1",
					"--max-pages", "4",
					"--timeout", "30",
					"--auth-basic", "user:passwd",
					"--header", "X-Test-Header: parity",
					"--cookie", "session=parity",
					"--proxy-region", "us-east",
					"--proxy-tag", "residential",
					"--exclude-proxy-id", "proxy-9",
					"--extract-config", extractConfigPath,
					"--incremental",
					"--include", "/crawl/**",
					"--exclude", "/crawl/b",
					"--respect-robots",
					"--skip-duplicates",
					"--simhash-threshold", "1",
					"--screenshot",
					"--screenshot-full-page=false",
					"--screenshot-format", "jpeg",
					"--screenshot-quality", "85",
					"--screenshot-width", "900",
					"--screenshot-height", "700",
					"--device", "iphone15",
					"--intercept-enabled",
					"--intercept-pattern", "**/api/**",
					"--intercept-resource-type", "xhr",
					"--intercept-resource-type", "fetch",
					"--intercept-request-body=false",
					"--intercept-response-body=true",
					"--intercept-max-body-size", "2048",
					"--intercept-max-entries", "25",
				}
			},
			mcpTool: "crawl_site",
			mcpArgs: mustJSONMap(t, api.CrawlRequest{
				URL:              site.CrawlRootURL(),
				MaxDepth:         1,
				MaxPages:         4,
				Headless:         false,
				TimeoutSeconds:   30,
				Auth:             cloneAuthOptions(authOpts),
				Extract:          cloneExtractOptions(extractOpts),
				Incremental:      boolPtr(true),
				IncludePatterns:  []string{"/crawl/**"},
				ExcludePatterns:  []string{"/crawl/b"},
				RespectRobotsTxt: boolPtr(true),
				SkipDuplicates:   boolPtr(true),
				SimHashThreshold: intPtr(1),
				Screenshot:       cloneScreenshotConfig(screenshot),
				Device:           cloneDevice(device),
				NetworkIntercept: cloneNetworkIntercept(intercept),
			}),
		},
		{
			name: "research",
			expected: jobs.JobSpec{
				Kind:             model.KindResearch,
				Query:            "pricing model",
				URLs:             site.ResearchURLs(),
				MaxDepth:         0,
				MaxPages:         4,
				Headless:         false,
				UsePlaywright:    false,
				Auth:             authOpts,
				TimeoutSeconds:   30,
				Extract:          extractOpts,
				Screenshot:       screenshot,
				Device:           device,
				NetworkIntercept: intercept,
				Agentic: &model.ResearchAgenticConfig{
					Enabled:         true,
					Instructions:    "Prioritize pricing and support commitments",
					MaxRounds:       2,
					MaxFollowUpURLs: 4,
				},
			},
			restPath: "/v1/research",
			restBody: api.ResearchRequest{
				Query:            "pricing model",
				URLs:             site.ResearchURLs(),
				MaxDepth:         0,
				MaxPages:         4,
				Headless:         false,
				TimeoutSeconds:   30,
				Auth:             cloneAuthOptions(authOpts),
				Extract:          cloneExtractOptions(extractOpts),
				Screenshot:       cloneScreenshotConfig(screenshot),
				Device:           cloneDevice(device),
				NetworkIntercept: cloneNetworkIntercept(intercept),
				Agentic: &model.ResearchAgenticConfig{
					Enabled:         true,
					Instructions:    "Prioritize pricing and support commitments",
					MaxRounds:       2,
					MaxFollowUpURLs: 4,
				},
			},
			cliArgs: func(t *testing.T, extractConfigPath string) []string {
				t.Helper()
				return []string{
					"research",
					"--query", "pricing model",
					"--urls", strings.Join(site.ResearchURLs(), ","),
					"--max-depth", "0",
					"--max-pages", "4",
					"--timeout", "30",
					"--auth-basic", "user:passwd",
					"--header", "X-Test-Header: parity",
					"--cookie", "session=parity",
					"--proxy-region", "us-east",
					"--proxy-tag", "residential",
					"--exclude-proxy-id", "proxy-9",
					"--extract-config", extractConfigPath,
					"--agentic",
					"--agentic-instructions", "Prioritize pricing and support commitments",
					"--agentic-max-rounds", "2",
					"--agentic-max-follow-up-urls", "4",
					"--screenshot",
					"--screenshot-full-page=false",
					"--screenshot-format", "jpeg",
					"--screenshot-quality", "85",
					"--screenshot-width", "900",
					"--screenshot-height", "700",
					"--device", "iphone15",
					"--intercept-enabled",
					"--intercept-pattern", "**/api/**",
					"--intercept-resource-type", "xhr",
					"--intercept-resource-type", "fetch",
					"--intercept-request-body=false",
					"--intercept-response-body=true",
					"--intercept-max-body-size", "2048",
					"--intercept-max-entries", "25",
				}
			},
			mcpTool: "research",
			mcpArgs: mustJSONMap(t, api.ResearchRequest{
				Query:            "pricing model",
				URLs:             site.ResearchURLs(),
				MaxDepth:         0,
				MaxPages:         4,
				Headless:         false,
				TimeoutSeconds:   30,
				Auth:             cloneAuthOptions(authOpts),
				Extract:          cloneExtractOptions(extractOpts),
				Screenshot:       cloneScreenshotConfig(screenshot),
				Device:           cloneDevice(device),
				NetworkIntercept: cloneNetworkIntercept(intercept),
				Agentic: &model.ResearchAgenticConfig{
					Enabled:         true,
					Instructions:    "Prioritize pricing and support commitments",
					MaxRounds:       2,
					MaxFollowUpURLs: 4,
				},
			}),
		},
	}

	for _, fixture := range fixtures {
		fixture := fixture
		t.Run(fixture.name, func(t *testing.T) {
			wantVersion, wantTypedSpec, err := jobs.TypedSpecFromJobSpec(fixture.expected)
			if err != nil {
				t.Fatalf("TypedSpecFromJobSpec() failed: %v", err)
			}
			if wantVersion != model.JobSpecVersion1 {
				t.Fatalf("unexpected typed spec version %d", wantVersion)
			}
			wantJSON := normalizedTypedSpecJSON(t, wantTypedSpec)

			restJob := submitParityRESTJob(t, fixture)
			assertPersistedTypedSpecMatches(t, "rest", fixture.expected.Kind, wantJSON, restJob)

			cliJob := submitParityCLIJob(t, fixture)
			assertPersistedTypedSpecMatches(t, "cli", fixture.expected.Kind, wantJSON, cliJob)

			mcpJob := submitParityMCPJob(t, fixture)
			assertPersistedTypedSpecMatches(t, "mcp", fixture.expected.Kind, wantJSON, mcpJob)
		})
	}
}

func submitParityRESTJob(t *testing.T, fixture submissionParityFixture) model.Job {
	t.Helper()
	dataDir := t.TempDir()
	port := freePort(t)
	env := parityEnv(baseEnv(dataDir), "PORT="+strconv.Itoa(port))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverCmd, cleanup := startProcessWithEnv(ctx, t, env, t.TempDir(), systemBinaryPath, "server")
	defer cleanup()

	client := &http.Client{Timeout: 5 * time.Second}
	waitForHealth(t, client, port)
	postJobPayload(t, client, port, fixture.restPath, fixture.restBody)

	cancel()
	_ = serverCmd.Wait()

	return loadOnlyJob(t, dataDir)
}

func submitParityCLIJob(t *testing.T, fixture submissionParityFixture) model.Job {
	t.Helper()
	dataDir := t.TempDir()
	extractConfigPath := writeExtractConfig(t, fixture.expected.Extract)
	env := parityEnv(baseEnv(dataDir))
	args := fixture.cliArgs(t, extractConfigPath)
	if err := runWithEnv(t, env, args...); err != nil {
		t.Fatalf("cli submission failed: %v", err)
	}
	return loadOnlyJob(t, dataDir)
}

func submitParityMCPJob(t *testing.T, fixture submissionParityFixture) model.Job {
	t.Helper()
	dataDir := t.TempDir()
	env := parityEnv(baseEnv(dataDir))
	_ = runMCPWithEnv(t, env, []string{
		`{"id":1,"method":"initialize"}`,
		fmt.Sprintf(`{"id":2,"method":"tools/call","params":{"name":%q,"arguments":%s}}`, fixture.mcpTool, mustMarshalJSONString(t, fixture.mcpArgs)),
	})
	return loadOnlyJob(t, dataDir)
}

func assertPersistedTypedSpecMatches(t *testing.T, surface string, kind model.Kind, wantJSON []byte, job model.Job) {
	t.Helper()
	if job.Kind != kind {
		t.Fatalf("%s job kind = %s, want %s", surface, job.Kind, kind)
	}
	if job.SpecVersion != model.JobSpecVersion1 {
		t.Fatalf("%s spec version = %d, want %d", surface, job.SpecVersion, model.JobSpecVersion1)
	}
	gotJSON := normalizedTypedSpecJSON(t, job.Spec)
	if !bytes.Equal(gotJSON, wantJSON) {
		t.Fatalf("%s typed spec mismatch\nwant: %s\n got: %s", surface, string(wantJSON), string(gotJSON))
	}
}

func loadOnlyJob(t *testing.T, dataDir string) model.Job {
	t.Helper()
	var lastErr error
	for i := 0; i < 20; i++ {
		st, err := store.Open(dataDir)
		if err != nil {
			lastErr = err
			time.Sleep(50 * time.Millisecond)
			continue
		}
		jobs, err := st.List(context.Background())
		_ = st.Close()
		if err != nil {
			lastErr = err
			time.Sleep(50 * time.Millisecond)
			continue
		}
		if len(jobs) == 1 {
			return jobs[0]
		}
		lastErr = fmt.Errorf("expected exactly 1 job, got %d", len(jobs))
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("failed to load single persisted job from %s: %v", dataDir, lastErr)
	return model.Job{}
}

func normalizedTypedSpecJSON(t *testing.T, spec any) []byte {
	t.Helper()
	raw, err := json.Marshal(spec)
	if err != nil {
		t.Fatalf("json.Marshal(%T): %v", spec, err)
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("json.Unmarshal(%T): %v", spec, err)
	}
	if execution, ok := payload["execution"].(map[string]any); ok {
		delete(execution, "requestId")
	}
	normalized, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent(%T): %v", spec, err)
	}
	return normalized
}

func writeExtractConfig(t *testing.T, opts extract.ExtractOptions) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "extract.json")
	content, err := json.Marshal(opts.Inline)
	if err != nil {
		t.Fatalf("json.Marshal extract config: %v", err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write extract config: %v", err)
	}
	return path
}

func mustJSONMap(t *testing.T, value any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%T): %v", value, err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("json.Unmarshal(%T): %v", value, err)
	}
	return out
}

func mustMarshalJSONString(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%T): %v", value, err)
	}
	return string(raw)
}

func cloneAuthOptions(value fetch.AuthOptions) *fetch.AuthOptions {
	clone := value
	return &clone
}

func cloneExtractOptions(value extract.ExtractOptions) *extract.ExtractOptions {
	clone := value
	return &clone
}

func cloneScreenshotConfig(value *fetch.ScreenshotConfig) *fetch.ScreenshotConfig {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func cloneDevice(value *fetch.DeviceEmulation) *fetch.DeviceEmulation {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func cloneNetworkIntercept(value *fetch.NetworkInterceptConfig) *fetch.NetworkInterceptConfig {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func boolPtr(value bool) *bool {
	return &value
}

func intPtr(value int) *int {
	return &value
}

const httpMethodPost = "POST"

func postJobPayload(t *testing.T, client *http.Client, port int, path string, body any) string {
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
	if resp.StatusCode != http.StatusCreated {
		payload, _ := io.ReadAll(resp.Body)
		t.Fatalf("post job status: %d body=%s", resp.StatusCode, string(payload))
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode job: %v", err)
	}
	jobPayload, ok := payload["job"].(map[string]any)
	if !ok {
		t.Fatalf("missing job envelope: %#v", payload)
	}
	id, _ := jobPayload["id"].(string)
	if id == "" {
		t.Fatalf("missing job id")
	}
	return id
}

func parityEnv(overrides []string, extra ...string) []string {
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
	values["PI_ENABLED"] = ""
	values["PI_CONFIG_PATH"] = ""
	values["PROXY_URL"] = ""
	values["PROXY_USERNAME"] = ""
	values["PROXY_PASSWORD"] = ""
	values["AUTH_BASIC"] = ""
	values["AUTH_BEARER"] = ""
	values["AUTH_API_KEY"] = ""
	out := make([]string, 0, len(values))
	for key, value := range values {
		out = append(out, key+"="+value)
	}
	return out
}

func runWithEnv(t *testing.T, env []string, args ...string) error {
	t.Helper()
	timeout := 120 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, systemBinaryPath, args...)
	cmd.Dir = t.TempDir()
	cmd.Env = env
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

func startProcessWithEnv(ctx context.Context, t *testing.T, env []string, dir string, name string, args ...string) (*exec.Cmd, func()) {
	t.Helper()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Env = env
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

func runMCPWithEnv(t *testing.T, env []string, lines []string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, systemBinaryPath, "mcp")
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
