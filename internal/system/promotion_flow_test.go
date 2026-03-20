// Package system provides deterministic promotion-flow regression coverage for the retained 1.0 core.
//
// Purpose:
// - Lock the verified job-promotion path down through the real HTTP API without browser automation.
//
// Responsibilities:
// - Fetch authoritative sanitized job detail by ID.
// - Derive template, watch, and export-schedule create requests from sanitized source-job data.
// - Prove promoted artifacts persist through their management endpoints without mutating the source job.
// - Assert promotion guardrails, detail-fetch fallback, and the redaction boundary.
//
// Scope:
// - PR-safe local system tests only. Browser proof stays in web tests and heavy browser checks stay outside this package.
//
// Usage:
// - Runs with `go test ./internal/system` and through `make test-ci`.
//
// Invariants/Assumptions:
// - Promotion uses existing CRUD endpoints instead of a dedicated server-side promotion API.
// - Job detail responses must stay sanitized and must never expose host-local paths or raw secrets.
// - Succeeded jobs are the intended promotion source contract for reusable automation.
package system

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/testsite"
)

type promotionJSON map[string]any

type jobDetailSnapshot struct {
	Raw      []byte
	Envelope promotionJSON
	Job      promotionJSON
}

func TestPromotionFlowEndToEndHTTPAPI(t *testing.T) {
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

	const (
		authBasicSentinel  = "user:passwd"
		authHeaderSentinel = "Bearer promotion-secret-token"
		authCookieSentinel = "session=promotion-cookie-value"
		authLoginSentinel  = "promotion-login-pass"
	)

	jobID := postJob(t, client, port, "/v1/scrape", map[string]any{
		"url":            site.URL("/auth/basic"),
		"headless":       false,
		"playwright":     false,
		"timeoutSeconds": 30,
		"auth": map[string]any{
			"basic":               authBasicSentinel,
			"headers":             map[string]string{"Authorization": authHeaderSentinel, "X-Test-Header": "kept"},
			"cookies":             []string{authCookieSentinel},
			"loginUrl":            site.URL("/login/chromedp"),
			"loginUserSelector":   "#username",
			"loginPassSelector":   "#password",
			"loginSubmitSelector": "button[type='submit']",
			"loginUser":           "promotion-user",
			"loginPass":           authLoginSentinel,
		},
		"extract": map[string]any{
			"inline": map[string]any{
				"name": "promotion-source-template",
				"selectors": []map[string]any{
					{"name": "body", "selector": "body", "trim": true},
					{"name": "title", "selector": "title", "trim": true},
				},
				"normalize": map[string]any{
					"titleField": "title",
				},
			},
		},
		"screenshot": map[string]any{
			"enabled":  true,
			"fullPage": true,
			"format":   "png",
		},
	})
	waitForJob(t, client, port, jobID)

	baseline := fetchJobDetail(t, client, port, jobID)
	if stringAt(t, baseline.Job, "id") != jobID {
		t.Fatalf("job id = %q, want %q", stringAt(t, baseline.Job, "id"), jobID)
	}
	if stringAt(t, baseline.Job, "kind") != "scrape" {
		t.Fatalf("job kind = %q, want scrape", stringAt(t, baseline.Job, "kind"))
	}
	if stringAt(t, baseline.Job, "status") != "succeeded" {
		t.Fatalf("job status = %q, want succeeded", stringAt(t, baseline.Job, "status"))
	}

	spec := objectAt(t, baseline.Job, "spec")
	exec := objectAt(t, spec, "execution")
	extract := objectAt(t, exec, "extract")
	inlineTemplate := objectAt(t, extract, "inline")
	if auth, ok := optionalStringAt(exec, "auth"); !ok || auth != "[REDACTED]" {
		t.Fatalf("expected execution.auth to be redacted, got %#v", exec["auth"])
	}
	if _, ok := optionalStringAt(baseline.Job, "resultPath"); ok {
		t.Fatalf("sanitized job unexpectedly exposed resultPath: %#v", baseline.Job["resultPath"])
	}
	if !bytes.Contains(baseline.Raw, []byte("[REDACTED]")) {
		t.Fatalf("expected sanitized job detail to contain a redaction marker: %s", string(baseline.Raw))
	}
	assertDoesNotContainLeaks(t, baseline.Raw,
		authBasicSentinel,
		authHeaderSentinel,
		authCookieSentinel,
		authLoginSentinel,
		dataDir,
	)

	templateName := "promoted-template-" + jobID
	templateCreate := map[string]any{
		"name":      templateName,
		"selectors": arrayAt(t, inlineTemplate, "selectors"),
	}
	if jsonld, ok := inlineTemplate["jsonld"]; ok {
		templateCreate["jsonld"] = jsonld
	}
	if regex, ok := inlineTemplate["regex"]; ok {
		templateCreate["regex"] = regex
	}
	if normalize, ok := inlineTemplate["normalize"]; ok {
		templateCreate["normalize"] = normalize
	}

	templateRaw, templateBody := postJSON(t, client,
		fmt.Sprintf("http://127.0.0.1:%d/v1/templates", port),
		templateCreate,
		http.StatusCreated,
	)
	if stringAt(t, templateBody, "name") != templateName {
		t.Fatalf("template name = %q, want %q", stringAt(t, templateBody, "name"), templateName)
	}
	assertDoesNotContainLeaks(t, templateRaw,
		authBasicSentinel,
		authHeaderSentinel,
		authCookieSentinel,
		authLoginSentinel,
	)

	_, fetchedTemplate := fetchJSON(t, client,
		fmt.Sprintf("http://127.0.0.1:%d/v1/templates/%s", port, templateName),
		http.StatusOK,
	)
	storedTemplate := objectAt(t, fetchedTemplate, "template")
	if !reflect.DeepEqual(storedTemplate["selectors"], inlineTemplate["selectors"]) {
		t.Fatalf("stored template selectors = %#v, want %#v", storedTemplate["selectors"], inlineTemplate["selectors"])
	}
	if !reflect.DeepEqual(storedTemplate["normalize"], inlineTemplate["normalize"]) {
		t.Fatalf("stored template normalize = %#v, want %#v", storedTemplate["normalize"], inlineTemplate["normalize"])
	}
	assertJobDetailUnchanged(t, baseline, fetchJobDetail(t, client, port, jobID))

	screenshot := objectAt(t, exec, "screenshot")
	watchCreate := map[string]any{
		"url":               stringAt(t, spec, "url"),
		"intervalSeconds":   300,
		"headless":          boolAt(t, exec, "headless"),
		"usePlaywright":     boolAt(t, exec, "playwright"),
		"screenshotEnabled": boolAt(t, screenshot, "enabled"),
	}
	watchRaw, watchBody := postJSON(t, client,
		fmt.Sprintf("http://127.0.0.1:%d/v1/watch", port),
		watchCreate,
		http.StatusCreated,
	)
	watchID := stringAt(t, watchBody, "id")
	if watchID == "" {
		t.Fatal("watch id was empty")
	}
	if stringAt(t, watchBody, "url") != stringAt(t, spec, "url") {
		t.Fatalf("watch url = %q, want %q", stringAt(t, watchBody, "url"), stringAt(t, spec, "url"))
	}
	if boolAt(t, watchBody, "headless") != boolAt(t, exec, "headless") {
		t.Fatalf("watch headless = %#v, want %#v", watchBody["headless"], exec["headless"])
	}
	if boolAt(t, watchBody, "usePlaywright") != boolAt(t, exec, "playwright") {
		t.Fatalf("watch usePlaywright = %#v, want %#v", watchBody["usePlaywright"], exec["playwright"])
	}
	if numberAt(t, watchBody, "intervalSeconds") != 300 {
		t.Fatalf("watch intervalSeconds = %#v, want 300", watchBody["intervalSeconds"])
	}
	assertDoesNotContainLeaks(t, watchRaw,
		authBasicSentinel,
		authHeaderSentinel,
		authCookieSentinel,
		authLoginSentinel,
	)

	_, fetchedWatch := fetchJSON(t, client,
		fmt.Sprintf("http://127.0.0.1:%d/v1/watch/%s", port, watchID),
		http.StatusOK,
	)
	if stringAt(t, fetchedWatch, "id") != watchID {
		t.Fatalf("fetched watch id = %q, want %q", stringAt(t, fetchedWatch, "id"), watchID)
	}
	if boolAt(t, fetchedWatch, "screenshotEnabled") != boolAt(t, screenshot, "enabled") {
		t.Fatalf("watch screenshotEnabled = %#v, want %#v", fetchedWatch["screenshotEnabled"], screenshot["enabled"])
	}
	assertJobDetailUnchanged(t, baseline, fetchJobDetail(t, client, port, jobID))

	exportPath := filepath.Join(t.TempDir(), "promotion-export-{job_id}.json")
	scheduleID := postExportSchedule(t, client, port, map[string]any{
		"name": "promoted-export-" + jobID,
		"filters": map[string]any{
			"job_kinds":   []string{stringAt(t, baseline.Job, "kind")},
			"job_status":  []string{stringAt(t, baseline.Job, "status")},
			"has_results": true,
		},
		"export": map[string]any{
			"format":           "json",
			"destination_type": "local",
			"local_path":       exportPath,
		},
	})
	_, scheduleBody := fetchJSON(t, client,
		fmt.Sprintf("http://127.0.0.1:%d/v1/export-schedules/%s", port, scheduleID),
		http.StatusOK,
	)
	if stringAt(t, scheduleBody, "id") != scheduleID {
		t.Fatalf("schedule id = %q, want %q", stringAt(t, scheduleBody, "id"), scheduleID)
	}
	filters := objectAt(t, scheduleBody, "filters")
	if !reflect.DeepEqual(stringArrayAt(t, filters, "job_kinds"), []string{stringAt(t, baseline.Job, "kind")}) {
		t.Fatalf("schedule job_kinds = %#v, want %#v", filters["job_kinds"], []string{stringAt(t, baseline.Job, "kind")})
	}
	if !reflect.DeepEqual(stringArrayAt(t, filters, "job_status"), []string{stringAt(t, baseline.Job, "status")}) {
		t.Fatalf("schedule job_status = %#v, want %#v", filters["job_status"], []string{stringAt(t, baseline.Job, "status")})
	}
	if !boolAt(t, filters, "has_results") {
		t.Fatalf("schedule has_results = %#v, want true", filters["has_results"])
	}
	exportConfig := objectAt(t, scheduleBody, "export")
	if stringAt(t, exportConfig, "format") != "json" {
		t.Fatalf("schedule format = %q, want json", stringAt(t, exportConfig, "format"))
	}
	if stringAt(t, exportConfig, "destination_type") != "local" {
		t.Fatalf("schedule destination_type = %q, want local", stringAt(t, exportConfig, "destination_type"))
	}
	if stringAt(t, exportConfig, "local_path") != exportPath {
		t.Fatalf("schedule local_path = %q, want %q", stringAt(t, exportConfig, "local_path"), exportPath)
	}
	assertJobDetailUnchanged(t, baseline, fetchJobDetail(t, client, port, jobID))

	cancel()
	_ = serverCmd.Wait()
}

func TestPromotionFlowGuardrailsAndDirectDetailFetchFallback(t *testing.T) {
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

	failedJobID := postJob(t, client, port, "/v1/scrape", map[string]any{
		"url":            site.URL("/auth/basic"),
		"headless":       false,
		"timeoutSeconds": 30,
	})
	failedDetail := waitForJobStatus(t, client, port, failedJobID, "failed")
	if stringAt(t, failedDetail.Job, "status") != "failed" {
		t.Fatalf("failed job status = %q, want failed", stringAt(t, failedDetail.Job, "status"))
	}
	if _, ok := optionalStringAt(failedDetail.Job, "resultPath"); ok {
		t.Fatalf("failed job unexpectedly exposed resultPath: %#v", failedDetail.Job["resultPath"])
	}
	assertDoesNotContainLeaks(t, failedDetail.Raw, dataDir)

	jobA := postJob(t, client, port, "/v1/scrape", map[string]any{
		"url":            site.ScrapeURL(),
		"headless":       false,
		"timeoutSeconds": 30,
	})
	jobB := postJob(t, client, port, "/v1/scrape", map[string]any{
		"url":            site.ArticleURL(),
		"headless":       false,
		"timeoutSeconds": 30,
	})
	waitForJob(t, client, port, jobA)
	waitForJob(t, client, port, jobB)

	_, page := fetchJSON(t, client,
		fmt.Sprintf("http://127.0.0.1:%d/v1/jobs?limit=1&offset=0", port),
		http.StatusOK,
	)
	jobs := arrayAt(t, page, "jobs")
	if len(jobs) != 1 {
		t.Fatalf("expected one job in limit=1 page, got %d (%#v)", len(jobs), page)
	}
	visibleJob := asPromotionJSON(t, jobs[0], "jobs[0]")
	visibleID := stringAt(t, visibleJob, "id")
	offPageID := jobA
	if visibleID == jobA {
		offPageID = jobB
	} else if visibleID != jobB {
		t.Fatalf("paged jobs response returned unexpected id %q (want %q or %q)", visibleID, jobA, jobB)
	}

	offPageDetail := fetchJobDetail(t, client, port, offPageID)
	if stringAt(t, offPageDetail.Job, "id") != offPageID {
		t.Fatalf("detail job id = %q, want %q", stringAt(t, offPageDetail.Job, "id"), offPageID)
	}
	if stringAt(t, offPageDetail.Job, "status") != "succeeded" {
		t.Fatalf("detail job status = %q, want succeeded", stringAt(t, offPageDetail.Job, "status"))
	}

	cancel()
	_ = serverCmd.Wait()
}

func fetchJobDetail(t *testing.T, client *http.Client, port int, jobID string) jobDetailSnapshot {
	t.Helper()
	raw, body := fetchJSON(t, client, fmt.Sprintf("http://127.0.0.1:%d/v1/jobs/%s", port, jobID), http.StatusOK)
	return jobDetailSnapshot{
		Raw:      raw,
		Envelope: body,
		Job:      objectAt(t, body, "job"),
	}
}

func waitForJobStatus(t *testing.T, client *http.Client, port int, jobID string, want string) jobDetailSnapshot {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		snapshot := fetchJobDetail(t, client, port, jobID)
		status := stringAt(t, snapshot.Job, "status")
		if status == want {
			return snapshot
		}
		if status == "succeeded" || status == "failed" || status == "canceled" {
			t.Fatalf("job %s reached terminal status %q, want %q", jobID, status, want)
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for job %s to reach %q", jobID, want)
	return jobDetailSnapshot{}
}

func fetchJSON(t *testing.T, client *http.Client, url string, wantStatus int) ([]byte, promotionJSON) {
	t.Helper()
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read %s: %v", url, err)
	}
	if resp.StatusCode != wantStatus {
		t.Fatalf("GET %s status = %d, want %d body=%s", url, resp.StatusCode, wantStatus, string(raw))
	}
	var payload promotionJSON
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode %s: %v body=%s", url, err, string(raw))
	}
	return raw, payload
}

func postJSON(t *testing.T, client *http.Client, url string, payload any, wantStatus int) ([]byte, promotionJSON) {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal POST %s: %v", url, err)
	}
	resp, err := client.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read POST %s: %v", url, err)
	}
	if resp.StatusCode != wantStatus {
		t.Fatalf("POST %s status = %d, want %d body=%s", url, resp.StatusCode, wantStatus, string(raw))
	}
	var body promotionJSON
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode POST %s: %v body=%s", url, err, string(raw))
	}
	return raw, body
}

func objectAt(t *testing.T, obj promotionJSON, key string) promotionJSON {
	t.Helper()
	return asPromotionJSON(t, obj[key], key)
}

func asPromotionJSON(t *testing.T, value any, label string) promotionJSON {
	t.Helper()
	mapped, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected %s to be an object, got %#v", label, value)
	}
	return promotionJSON(mapped)
}

func stringAt(t *testing.T, obj promotionJSON, key string) string {
	t.Helper()
	value, ok := obj[key].(string)
	if !ok {
		t.Fatalf("expected %s to be a string in %#v", key, obj)
	}
	return value
}

func optionalStringAt(obj promotionJSON, key string) (string, bool) {
	value, ok := obj[key]
	if !ok || value == nil {
		return "", false
	}
	str, ok := value.(string)
	return str, ok
}

func boolAt(t *testing.T, obj promotionJSON, key string) bool {
	t.Helper()
	value, ok := obj[key].(bool)
	if !ok {
		t.Fatalf("expected %s to be a bool in %#v", key, obj)
	}
	return value
}

func numberAt(t *testing.T, obj promotionJSON, key string) float64 {
	t.Helper()
	value, ok := obj[key].(float64)
	if !ok {
		t.Fatalf("expected %s to be a number in %#v", key, obj)
	}
	return value
}

func arrayAt(t *testing.T, obj promotionJSON, key string) []any {
	t.Helper()
	value, ok := obj[key].([]any)
	if !ok {
		t.Fatalf("expected %s to be an array in %#v", key, obj)
	}
	return value
}

func stringArrayAt(t *testing.T, obj promotionJSON, key string) []string {
	t.Helper()
	values := arrayAt(t, obj, key)
	out := make([]string, 0, len(values))
	for _, value := range values {
		str, ok := value.(string)
		if !ok {
			t.Fatalf("expected %s to contain only strings, got %#v", key, values)
		}
		out = append(out, str)
	}
	return out
}

func assertDoesNotContainLeaks(t *testing.T, raw []byte, banned ...string) {
	t.Helper()
	body := string(raw)
	for _, token := range banned {
		if token == "" {
			continue
		}
		if strings.Contains(body, token) {
			t.Fatalf("response leaked %q: %s", token, body)
		}
	}
}

func assertJobDetailUnchanged(t *testing.T, before, after jobDetailSnapshot) {
	t.Helper()
	if reflect.DeepEqual(before.Envelope, after.Envelope) {
		return
	}
	beforePretty, _ := json.MarshalIndent(before.Envelope, "", "  ")
	afterPretty, _ := json.MarshalIndent(after.Envelope, "", "  ")
	t.Fatalf("job detail changed after promotion side effect\nBEFORE:\n%s\nAFTER:\n%s", string(beforePretty), string(afterPretty))
}
