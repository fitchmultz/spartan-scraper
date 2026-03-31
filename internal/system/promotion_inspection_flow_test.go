// Package system provides deterministic promotion-inspection regression coverage for saved automation artifacts.
//
// Purpose:
// - Lock the post-promotion save-and-inspect path down through the real HTTP API without browser automation.
//
// Responsibilities:
// - Create promoted templates, watches, and export schedules from authoritative sanitized job detail.
// - Verify saved templates remain previewable, watches persist inspectable history/detail, and export schedules record inspectable history.
// - Assert inspection responses stay sanitized and do not mutate the original source job.
//
// Scope:
// - PR-safe local system tests only. Browser workflow proof stays in web tests.
//
// Usage:
// - Runs with `go test ./internal/system` and through `make test-ci`.
//
// Invariants/Assumptions:
// - Promotion reuses existing CRUD and inspection endpoints instead of a dedicated server-side promotion API.
// - Watch inspection responses must not leak host-local artifact paths.
// - Post-promotion inspection should preserve source-job detail exactly.
package system

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/testsite"
)

func TestPromotionFlowPostSaveInspectionHTTPAPI(t *testing.T) {
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

	sourceJobID := postJob(t, client, port, "/v1/scrape", map[string]any{
		"url":            site.ScrapeURL(),
		"headless":       false,
		"timeoutSeconds": 30,
		"extract": map[string]any{
			"inline": map[string]any{
				"name": "promotion-inspection-source",
				"selectors": []map[string]any{
					{"name": "headline", "selector": "h1", "trim": true},
				},
				"normalize": map[string]any{
					"titleField": "headline",
				},
			},
		},
	})
	waitForJob(t, client, port, sourceJobID)

	baseline := fetchJobDetail(t, client, port, sourceJobID)
	if stringAt(t, baseline.Job, "status") != "succeeded" {
		t.Fatalf("source job status = %q, want succeeded", stringAt(t, baseline.Job, "status"))
	}

	spec := objectAt(t, baseline.Job, "spec")
	execution := objectAt(t, spec, "execution")
	extract := objectAt(t, execution, "extract")
	inlineTemplate := objectAt(t, extract, "inline")
	selectors := arrayAt(t, inlineTemplate, "selectors")
	if len(selectors) != 1 {
		t.Fatalf("inline selector count = %d, want 1", len(selectors))
	}
	headlineRule := asPromotionJSON(t, selectors[0], "selectors[0]")
	if stringAt(t, headlineRule, "selector") != "h1" {
		t.Fatalf("inline selector = %q, want h1", stringAt(t, headlineRule, "selector"))
	}

	templateName := "promotion-inspection-template-" + sourceJobID
	templateCreate := map[string]any{
		"name":      templateName,
		"selectors": selectors,
		"normalize": inlineTemplate["normalize"],
	}
	templateRaw, templateBody := postJSON(
		t,
		client,
		fmt.Sprintf("http://127.0.0.1:%d/v1/templates", port),
		templateCreate,
		http.StatusCreated,
	)
	if stringAt(t, templateBody, "name") != templateName {
		t.Fatalf("template name = %q, want %q", stringAt(t, templateBody, "name"), templateName)
	}
	assertDoesNotContainLeaks(t, templateRaw, dataDir)

	_, fetchedTemplate := fetchJSON(
		t,
		client,
		fmt.Sprintf("http://127.0.0.1:%d/v1/templates/%s", port, templateName),
		http.StatusOK,
	)
	storedTemplate := objectAt(t, fetchedTemplate, "template")
	if !reflect.DeepEqual(storedTemplate["selectors"], inlineTemplate["selectors"]) {
		t.Fatalf("stored template selectors = %#v, want %#v", storedTemplate["selectors"], inlineTemplate["selectors"])
	}

	previewRaw, previewBody := postJSON(
		t,
		client,
		fmt.Sprintf("http://127.0.0.1:%d/v1/template-preview/test-selector", port),
		map[string]any{
			"url":      site.ScrapeURL(),
			"selector": "h1",
		},
		http.StatusOK,
	)
	assertDoesNotContainLeaks(t, previewRaw, dataDir)
	if stringAt(t, previewBody, "selector") != "h1" {
		t.Fatalf("preview selector = %q, want h1", stringAt(t, previewBody, "selector"))
	}
	if numberAt(t, previewBody, "matches") < 1 {
		t.Fatalf("preview matches = %#v, want >= 1", previewBody["matches"])
	}
	previewElements := arrayAt(t, previewBody, "elements")
	if len(previewElements) == 0 {
		t.Fatal("preview elements were empty")
	}
	firstPreviewElement := asPromotionJSON(t, previewElements[0], "elements[0]")
	if text := requireNonEmptyString(t, firstPreviewElement, "text"); !strings.Contains(text, "Example Domain") {
		t.Fatalf("preview sample text = %q, want Example Domain", text)
	}

	watchRaw, watchBody := postJSON(
		t,
		client,
		fmt.Sprintf("http://127.0.0.1:%d/v1/watch", port),
		map[string]any{
			"url":               stringAt(t, spec, "url"),
			"intervalSeconds":   300,
			"headless":          boolAt(t, execution, "headless"),
			"usePlaywright":     boolAt(t, execution, "playwright"),
			"screenshotEnabled": false,
		},
		http.StatusCreated,
	)
	assertDoesNotContainLeaks(t, watchRaw, dataDir)
	watchID := stringAt(t, watchBody, "id")
	if watchID == "" {
		t.Fatal("watch id was empty")
	}

	watchCheckRaw, watchCheckBody := postJSON(
		t,
		client,
		fmt.Sprintf("http://127.0.0.1:%d/v1/watch/%s/check", port, watchID),
		map[string]any{},
		http.StatusOK,
	)
	assertDoesNotContainLeaks(t, watchCheckRaw, dataDir)
	watchCheck := objectAt(t, watchCheckBody, "check")
	checkID := requireNonEmptyString(t, watchCheck, "id")
	if stringAt(t, watchCheck, "watchId") != watchID {
		t.Fatalf("watch check watchId = %q, want %q", stringAt(t, watchCheck, "watchId"), watchID)
	}
	if stringAt(t, watchCheck, "url") != site.ScrapeURL() {
		t.Fatalf("watch check url = %q, want %q", stringAt(t, watchCheck, "url"), site.ScrapeURL())
	}
	requireNonEmptyString(t, watchCheck, "status")
	requireNonEmptyString(t, watchCheck, "title")
	requireNonEmptyString(t, watchCheck, "message")

	historyRecord := waitForWatchHistoryRecord(t, client, port, watchID, checkID)
	if stringAt(t, historyRecord, "id") != checkID {
		t.Fatalf("watch history id = %q, want %q", stringAt(t, historyRecord, "id"), checkID)
	}
	if stringAt(t, historyRecord, "watchId") != watchID {
		t.Fatalf("watch history watchId = %q, want %q", stringAt(t, historyRecord, "watchId"), watchID)
	}
	if stringAt(t, historyRecord, "url") != site.ScrapeURL() {
		t.Fatalf("watch history url = %q, want %q", stringAt(t, historyRecord, "url"), site.ScrapeURL())
	}
	requireNonEmptyString(t, historyRecord, "status")
	requireNonEmptyString(t, historyRecord, "title")
	requireNonEmptyString(t, historyRecord, "message")

	watchDetailRaw, watchDetailBody := fetchWatchCheckDetail(t, client, port, watchID, checkID)
	assertDoesNotContainLeaks(t, watchDetailRaw, dataDir)
	watchDetail := objectAt(t, watchDetailBody, "check")
	if stringAt(t, watchDetail, "id") != checkID {
		t.Fatalf("watch detail id = %q, want %q", stringAt(t, watchDetail, "id"), checkID)
	}
	if stringAt(t, watchDetail, "watchId") != watchID {
		t.Fatalf("watch detail watchId = %q, want %q", stringAt(t, watchDetail, "watchId"), watchID)
	}
	if stringAt(t, watchDetail, "url") != site.ScrapeURL() {
		t.Fatalf("watch detail url = %q, want %q", stringAt(t, watchDetail, "url"), site.ScrapeURL())
	}
	requireNonEmptyString(t, watchDetail, "title")
	requireNonEmptyString(t, watchDetail, "message")
	if actions, ok := watchDetail["actions"].([]any); ok {
		for index, actionValue := range actions {
			action := asPromotionJSON(t, actionValue, fmt.Sprintf("actions[%d]", index))
			requireNonEmptyString(t, action, "label")
			requireNonEmptyString(t, action, "kind")
		}
	}
	if artifacts, ok := watchDetail["artifacts"].([]any); ok {
		prefix := fmt.Sprintf("/v1/watch/%s/history/%s/artifacts/", watchID, checkID)
		for index, artifactValue := range artifacts {
			artifact := asPromotionJSON(t, artifactValue, fmt.Sprintf("artifacts[%d]", index))
			if url := requireNonEmptyString(t, artifact, "url"); !strings.HasPrefix(url, prefix) {
				t.Fatalf("artifact url = %q, want prefix %q", url, prefix)
			}
		}
	}

	exportPath := filepath.Join(dataDir, "exports", "promotion-inspection-export.json")
	exportTemplate := filepath.Join("exports", "promotion-inspection-export.json")
	scheduleID := postExportSchedule(t, client, port, map[string]any{
		"name": "promotion-inspection-export-" + sourceJobID,
		"filters": map[string]any{
			"job_kinds":   []string{stringAt(t, baseline.Job, "kind")},
			"job_status":  []string{"succeeded"},
			"has_results": true,
		},
		"export": map[string]any{
			"format":           "json",
			"destination_type": "local",
			"local_path":       exportTemplate,
		},
	})

	triggerJobID := postJob(t, client, port, "/v1/scrape", map[string]any{
		"url":            site.ScrapeURL(),
		"headless":       false,
		"timeoutSeconds": 30,
	})
	waitForJob(t, client, port, triggerJobID)

	exportRecord := waitForExportHistoryRecord(t, client, port, scheduleID, triggerJobID)
	if stringAt(t, exportRecord, "jobId") != triggerJobID {
		t.Fatalf("export history jobId = %q, want %q", stringAt(t, exportRecord, "jobId"), triggerJobID)
	}
	if stringAt(t, exportRecord, "status") != "succeeded" {
		t.Fatalf("export history status = %q, want succeeded", stringAt(t, exportRecord, "status"))
	}
	requireNonEmptyString(t, exportRecord, "title")
	requireNonEmptyString(t, exportRecord, "message")
	requireNonEmptyString(t, exportRecord, "destination")
	request := objectAt(t, exportRecord, "request")
	if stringAt(t, request, "format") != "json" {
		t.Fatalf("export request format = %q, want json", stringAt(t, request, "format"))
	}
	artifact := objectAt(t, exportRecord, "artifact")
	requireNonEmptyString(t, artifact, "filename")
	if stringAt(t, artifact, "contentType") != "application/json" {
		t.Fatalf("export artifact contentType = %q, want application/json", stringAt(t, artifact, "contentType"))
	}
	assertFileNotEmpty(t, exportPath)

	assertJobDetailUnchanged(t, baseline, fetchJobDetail(t, client, port, sourceJobID))

	cancel()
	_ = serverCmd.Wait()
}

func waitForWatchHistoryRecord(t *testing.T, client *http.Client, port int, watchID, checkID string) promotionJSON {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/watch/%s/history?limit=10&offset=0", port, watchID)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		_, payload := fetchJSON(t, client, url, http.StatusOK)
		records := arrayAt(t, payload, "checks")
		for _, recordValue := range records {
			record := asPromotionJSON(t, recordValue, "checks[]")
			if id, ok := optionalStringAt(record, "id"); ok && id == checkID {
				return record
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("watch history for %s did not record check %s", watchID, checkID)
	return nil
}

func fetchWatchCheckDetail(t *testing.T, client *http.Client, port int, watchID, checkID string) ([]byte, promotionJSON) {
	t.Helper()
	return fetchJSON(t, client, fmt.Sprintf("http://127.0.0.1:%d/v1/watch/%s/history/%s", port, watchID, checkID), http.StatusOK)
}

func waitForExportHistoryRecord(t *testing.T, client *http.Client, port int, scheduleID, jobID string) promotionJSON {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/export-schedules/%s/history?limit=10&offset=0", port, scheduleID)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		_, payload := fetchJSON(t, client, url, http.StatusOK)
		records := arrayAt(t, payload, "exports")
		for _, recordValue := range records {
			record := asPromotionJSON(t, recordValue, "exports[]")
			if id, ok := optionalStringAt(record, "jobId"); ok && id == jobID {
				return record
			}
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Fatalf("export history for schedule %s did not record job %s", scheduleID, jobID)
	return nil
}

func requireNonEmptyString(t *testing.T, obj promotionJSON, key string) string {
	t.Helper()
	value, ok := optionalStringAt(obj, key)
	if !ok || strings.TrimSpace(value) == "" {
		t.Fatalf("expected %s to be a non-empty string in %#v", key, obj)
	}
	return value
}
