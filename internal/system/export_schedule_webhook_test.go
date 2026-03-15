// Package system provides deterministic server-path coverage for export-schedule webhooks.
//
// Purpose:
// - Prove scheduled webhook exports run through the real server dispatcher end to end.
//
// Responsibilities:
// - Create export schedules over the HTTP API after the server is already running.
// - Verify matching job completion triggers outbound webhook delivery.
// - Verify export history and webhook-delivery tracking both record the successful dispatch.
//
// Scope:
// - PR-safe local system coverage only.
//
// Usage:
// - Runs with `go test ./internal/system` and through `make test-ci`.
//
// Invariants/Assumptions:
// - The server path must update the live export trigger when schedules are created over HTTP.
// - Scheduled webhook exports must not report success unless dispatcher delivery succeeds.
package system

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/testsite"
)

func TestExportScheduleWebhookServerFlow(t *testing.T) {
	dataDir := t.TempDir()
	port := freePort(t)
	env := baseEnv(dataDir)
	env = append(env,
		"PORT="+strconv.Itoa(port),
		"WEBHOOK_ENABLED=true",
		"WEBHOOK_ALLOW_INTERNAL=true",
		"WEBHOOK_MAX_RETRIES=1",
		"WEBHOOK_TIMEOUT_MS=1000",
	)
	site := testsite.Start(t)

	received := make(chan receivedExportWebhook, 1)
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		request, err := decodeReceivedExportWebhook(r)
		if err != nil {
			t.Errorf("decodeReceivedExportWebhook(): %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		select {
		case received <- request:
		default:
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer receiver.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverCmd, cleanup := startProcess(ctx, t, env, t.TempDir(), systemBinaryPath, "server")
	defer cleanup()

	client := &http.Client{Timeout: 5 * time.Second}
	waitForHealth(t, client, port)

	scheduleID := postExportSchedule(t, client, port, map[string]any{
		"name": "system webhook export",
		"filters": map[string]any{
			"job_kinds":   []string{"scrape"},
			"job_status":  []string{"completed"},
			"has_results": true,
		},
		"export": map[string]any{
			"format":           "json",
			"destination_type": "webhook",
			"webhook_url":      receiver.URL,
		},
		"retry": map[string]any{
			"max_retries":   1,
			"base_delay_ms": 10,
		},
	})

	jobID := postJob(t, client, port, "/v1/scrape", map[string]any{
		"url":            site.ScrapeURL(),
		"headless":       false,
		"timeoutSeconds": 30,
	})
	waitForJob(t, client, port, jobID)
	assertManifestExists(t, dataDir, jobID)

	select {
	case request := <-received:
		if request.metadata["eventType"] != "export.completed" {
			t.Fatalf("unexpected webhook event payload: %#v", request.metadata)
		}
		if request.metadata["jobId"] != jobID {
			t.Fatalf("expected webhook jobId %s, got %#v", jobID, request.metadata)
		}
		if request.metadata["exportFormat"] != "json" {
			t.Fatalf("expected json export format, got %#v", request.metadata)
		}
		if request.metadata["filename"] != jobID+".json" {
			t.Fatalf("expected export filename %s.json, got %#v", jobID, request.metadata)
		}
		if request.metadata["contentType"] != "application/json" {
			t.Fatalf("expected application/json metadata content type, got %#v", request.metadata)
		}
		if request.metadata["resultUrl"] != "/v1/jobs/"+jobID+"/results" {
			t.Fatalf("expected resultUrl for %s, got %#v", jobID, request.metadata)
		}
		if request.exportFilename != jobID+".json" {
			t.Fatalf("expected multipart export filename %s.json, got %q", jobID, request.exportFilename)
		}
		if request.exportContentType != "application/json" {
			t.Fatalf("expected multipart export content type application/json, got %q", request.exportContentType)
		}
		if !bytes.Contains(request.exportBody, []byte("Example Domain")) {
			t.Fatalf("unexpected multipart export body: %s", string(request.exportBody))
		}
		if request.payloadType != "export-multipart" {
			t.Fatalf("expected export-multipart payload header, got %q", request.payloadType)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for scheduled webhook export delivery")
	}

	waitForExportHistorySuccess(t, client, port, scheduleID, jobID)
	waitForWebhookDeliveryRecord(t, client, port, jobID)

	cancel()
	_ = serverCmd.Wait()
}

func postExportSchedule(t *testing.T, client *http.Client, port int, body map[string]any) string {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("post export schedule marshal: %v", err)
	}
	resp, err := client.Post(fmt.Sprintf("http://127.0.0.1:%d/v1/export-schedules", port), "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("post export schedule: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		payload, _ := io.ReadAll(resp.Body)
		t.Fatalf("post export schedule status: %d body=%s", resp.StatusCode, string(payload))
	}
	var payload map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode export schedule: %v", err)
	}
	id, _ := payload["id"].(string)
	if id == "" {
		t.Fatalf("missing export schedule id: %#v", payload)
	}
	return id
}

func waitForExportHistorySuccess(t *testing.T, client *http.Client, port int, scheduleID, jobID string) {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/export-schedules/%s/history", port, scheduleID)
	for i := 0; i < 100; i++ {
		resp, err := client.Get(url)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		var payload map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		_ = resp.Body.Close()
		records, _ := payload["records"].([]any)
		if len(records) == 1 {
			record, _ := records[0].(map[string]any)
			if record["job_id"] == jobID && record["status"] == "success" {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("export history for schedule %s did not record a successful export for job %s", scheduleID, jobID)
}

type receivedExportWebhook struct {
	metadata          map[string]any
	exportBody        []byte
	exportFilename    string
	exportContentType string
	payloadType       string
}

func decodeReceivedExportWebhook(r *http.Request) (receivedExportWebhook, error) {
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return receivedExportWebhook{}, err
	}
	if mediaType != "multipart/form-data" {
		return receivedExportWebhook{}, fmt.Errorf("unexpected webhook content type %q", mediaType)
	}
	reader := multipart.NewReader(r.Body, params["boundary"])
	result := receivedExportWebhook{
		metadata:    map[string]any{},
		payloadType: r.Header.Get("X-Spartan-Webhook-Payload-Type"),
	}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return receivedExportWebhook{}, err
		}
		body, err := io.ReadAll(part)
		if err != nil {
			return receivedExportWebhook{}, err
		}
		switch part.FormName() {
		case "metadata":
			if err := json.Unmarshal(body, &result.metadata); err != nil {
				return receivedExportWebhook{}, err
			}
		case "export":
			result.exportBody = body
			result.exportFilename = part.FileName()
			result.exportContentType = part.Header.Get("Content-Type")
		}
	}
	return result, nil
}

func waitForWebhookDeliveryRecord(t *testing.T, client *http.Client, port int, jobID string) {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/webhooks/deliveries?job_id=%s", port, jobID)
	for i := 0; i < 100; i++ {
		resp, err := client.Get(url)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		var payload map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		_ = resp.Body.Close()
		deliveries, _ := payload["deliveries"].([]any)
		if len(deliveries) == 1 {
			record, _ := deliveries[0].(map[string]any)
			if record["jobId"] == jobID && record["eventType"] == "export.completed" && record["status"] == "delivered" {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("webhook deliveries did not record a delivered export webhook for job %s", jobID)
}
