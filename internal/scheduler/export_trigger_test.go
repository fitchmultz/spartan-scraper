// Package scheduler provides focused regression coverage for export-trigger execution.
//
// Purpose:
// - Verify export-trigger delivery behavior for local and webhook destinations.
//
// Responsibilities:
// - Confirm transforms apply to local exports.
// - Confirm webhook exports require a real dispatcher and record failures correctly.
// - Confirm successful webhook exports deliver through the shared dispatcher path.
//
// Scope:
// - Direct ExportTrigger execution only; higher-level server wiring is covered elsewhere.
//
// Usage:
// - Run with `go test ./internal/scheduler`.
//
// Invariants/Assumptions:
// - Export history should record one logical export record per trigger execution, even when retries occur.
// - Webhook exports must not report success before dispatcher delivery completes.
package scheduler

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/exporter"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

func TestExportTriggerExportAppliesConfiguredTransform(t *testing.T) {
	dataDir := t.TempDir()
	store := NewExportStorage(dataDir)
	history := NewExportHistoryStore(dataDir)
	trigger := NewExportTrigger(dataDir, store, history, nil, nil)

	job := writeResultJob(t, dataDir, "job-transform", model.KindCrawl, strings.Join([]string{
		`{"url":"https://example.com/a","title":"A","status":200}`,
		`{"url":"https://example.com/b","title":"B","status":200}`,
	}, "\n"))
	schedule := &ExportSchedule{
		ID:      "schedule-transform",
		Name:    "Projected Export",
		Enabled: true,
		Filters: ExportFilters{JobKinds: []string{"crawl"}},
		Export: ExportConfig{
			Format:          "csv",
			DestinationType: "local",
			LocalPath:       "exports/{job_id}.csv",
			PathTemplate:    "exports/{job_id}.csv",
			Transform: exporter.TransformConfig{
				Expression: "{title: title, url: url}",
				Language:   "jmespath",
			},
		},
		Retry: DefaultRetryConfig(),
	}

	if err := trigger.Export(context.Background(), &job, schedule); err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	outputPath := filepath.Join(dataDir, "exports", "job-transform.csv")
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath): %v", err)
	}
	content := strings.TrimSpace(string(data))
	if !strings.Contains(content, "title,url") || strings.Contains(content, "status") {
		t.Fatalf("unexpected transformed export content: %s", content)
	}
	assertPrivateExportMode(t, outputPath)

	records, total, err := history.GetBySchedule(schedule.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetBySchedule() failed: %v", err)
	}
	if total != 1 || len(records) != 1 || records[0].Status != "succeeded" {
		t.Fatalf("unexpected export history: %#v total=%d", records, total)
	}
}

func TestExportTriggerRejectsLocalDestinationOutsideExportsRoot(t *testing.T) {
	dataDir := t.TempDir()
	store := NewExportStorage(dataDir)
	history := NewExportHistoryStore(dataDir)
	trigger := NewExportTrigger(dataDir, store, history, nil, nil)

	job := writeResultJob(t, dataDir, "job-outside-root", model.KindScrape, `{"title":"Example Domain"}`)
	schedule := &ExportSchedule{
		ID:      "schedule-outside-root",
		Name:    "Outside Root",
		Enabled: true,
		Filters: ExportFilters{JobKinds: []string{"scrape"}},
		Export: ExportConfig{
			Format:          "json",
			DestinationType: "local",
			LocalPath:       "jobs/{job_id}/override.json",
			PathTemplate:    "jobs/{job_id}/override.json",
		},
		Retry: DefaultRetryConfig(),
	}

	err := trigger.Export(context.Background(), &job, schedule)
	if err == nil || !strings.Contains(err.Error(), "DATA_DIR/exports") {
		t.Fatalf("expected exports-root validation error, got %v", err)
	}
	records, total, err := history.GetBySchedule(schedule.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetBySchedule() failed: %v", err)
	}
	if total != 0 || len(records) != 0 {
		t.Fatalf("expected no history records for rejected destination, got total=%d records=%#v", total, records)
	}
}

func TestExportTriggerWebhookExportFailsWithoutDispatcher(t *testing.T) {
	dataDir := t.TempDir()
	store := NewExportStorage(dataDir)
	history := NewExportHistoryStore(dataDir)
	trigger := NewExportTrigger(dataDir, store, history, nil, nil)

	job := writeResultJob(t, dataDir, "job-webhook-missing-dispatcher", model.KindScrape, `{"title":"Example Domain"}`)
	schedule := &ExportSchedule{
		ID:      "schedule-webhook-missing-dispatcher",
		Name:    "Webhook Export",
		Enabled: true,
		Filters: ExportFilters{JobKinds: []string{"scrape"}},
		Export: ExportConfig{
			Format:          "json",
			DestinationType: "webhook",
			WebhookURL:      "https://example.com/webhook",
		},
		Retry: ExportRetryConfig{MaxRetries: 1, BaseDelayMs: 1},
	}

	err := trigger.Export(context.Background(), &job, schedule)
	if err == nil || !strings.Contains(err.Error(), "dispatcher is not configured") {
		t.Fatalf("expected dispatcher unavailable error, got %v", err)
	}

	records, total, err := history.GetBySchedule(schedule.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetBySchedule() failed: %v", err)
	}
	if total != 1 || len(records) != 1 {
		t.Fatalf("expected exactly one export history record, got total=%d records=%#v", total, records)
	}
	if records[0].Status != "failed" {
		t.Fatalf("expected failed export record, got %#v", records[0])
	}
	if records[0].RetryCount != 1 {
		t.Fatalf("expected retry count 1, got %#v", records[0])
	}
	if !strings.Contains(records[0].ErrorMessage, "dispatcher is not configured") {
		t.Fatalf("expected dispatcher failure in history, got %#v", records[0])
	}
}

func TestExportTriggerWebhookExportDeliversViaDispatcher(t *testing.T) {
	dataDir := t.TempDir()
	store := NewExportStorage(dataDir)
	history := NewExportHistoryStore(dataDir)
	deliveryStore := webhook.NewStore(dataDir)
	dispatcher := webhook.NewDispatcherWithStore(webhook.Config{
		AllowInternal: true,
		MaxRetries:    1,
		Timeout:       time.Second,
	}, deliveryStore)
	defer dispatcher.Close()
	trigger := NewExportTrigger(dataDir, store, history, nil, dispatcher)

	received := make(chan receivedExportWebhook, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer server.Close()

	job := writeResultJob(t, dataDir, "job-webhook-success", model.KindScrape, `{"title":"Example Domain"}`)
	schedule := &ExportSchedule{
		ID:      "schedule-webhook-success",
		Name:    "Webhook Export",
		Enabled: true,
		Filters: ExportFilters{JobKinds: []string{"scrape"}},
		Export: ExportConfig{
			Format:          "json",
			DestinationType: "webhook",
			WebhookURL:      server.URL,
		},
		Retry: ExportRetryConfig{MaxRetries: 1, BaseDelayMs: 1},
	}

	if err := trigger.Export(context.Background(), &job, schedule); err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	select {
	case request := <-received:
		if request.payload.EventType != webhook.EventExportCompleted {
			t.Fatalf("expected export completed event, got %#v", request.payload)
		}
		if request.payload.JobID != job.ID {
			t.Fatalf("expected payload for job %s, got %#v", job.ID, request.payload)
		}
		if request.payload.ExportFormat != "json" {
			t.Fatalf("expected json export format, got %#v", request.payload)
		}
		if request.payload.Filename != job.ID+".json" {
			t.Fatalf("expected export filename %s.json, got %#v", job.ID, request.payload)
		}
		if request.payload.ContentType != "application/json" {
			t.Fatalf("expected application/json content type, got %#v", request.payload)
		}
		if request.payload.ResultURL != "/v1/jobs/"+job.ID+"/results" {
			t.Fatalf("expected result URL for job %s, got %#v", job.ID, request.payload)
		}
		if request.payload.RecordCount != 1 || request.payload.ExportSize != int64(len(request.exportBody)) {
			t.Fatalf("unexpected export metadata: %#v", request.payload)
		}
		if request.exportContentType != "application/json" {
			t.Fatalf("expected export part content type application/json, got %q", request.exportContentType)
		}
		if request.exportFilename != job.ID+".json" {
			t.Fatalf("expected export part filename %s.json, got %q", job.ID, request.exportFilename)
		}
		if got := strings.TrimSpace(string(request.exportBody)); !strings.Contains(got, "Example Domain") {
			t.Fatalf("unexpected export body: %s", got)
		}
		if request.payloadType != "export-multipart" {
			t.Fatalf("expected export-multipart payload type header, got %q", request.payloadType)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for webhook delivery")
	}

	records, total, err := history.GetBySchedule(schedule.ID, 10, 0)
	if err != nil {
		t.Fatalf("GetBySchedule() failed: %v", err)
	}
	if total != 1 || len(records) != 1 || records[0].Status != "succeeded" {
		t.Fatalf("unexpected export history: %#v total=%d", records, total)
	}

	deliveries, err := deliveryStore.ListRecords(context.Background(), job.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListRecords() failed: %v", err)
	}
	if len(deliveries) != 1 || deliveries[0].Status != webhook.DeliveryStatusDelivered {
		t.Fatalf("unexpected delivery records: %#v", deliveries)
	}
}

func TestExportTriggerStopCancelsPendingRetry(t *testing.T) {
	dataDir := t.TempDir()
	store := NewExportStorage(dataDir)
	history := NewExportHistoryStore(dataDir)
	trigger := NewExportTrigger(dataDir, store, history, nil, nil)
	if err := trigger.Start(); err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	schedule := &ExportSchedule{
		ID:      "schedule-stop-cancel",
		Name:    "Webhook Export",
		Enabled: true,
		Filters: ExportFilters{JobKinds: []string{"scrape"}, JobStatus: []string{"succeeded"}},
		Export: ExportConfig{
			Format:          "json",
			DestinationType: "webhook",
			WebhookURL:      "https://example.com/webhook",
		},
		Retry: ExportRetryConfig{MaxRetries: 1, BaseDelayMs: 5000},
	}
	trigger.AddSchedule(schedule)

	job := writeResultJob(t, dataDir, "job-stop-cancel", model.KindScrape, `{"title":"Example Domain"}`)
	trigger.HandleJobEvent(jobs.JobEvent{Type: jobs.JobEventCompleted, Job: job})

	deadline := time.Now().Add(time.Second)
	for {
		records, total, err := history.GetBySchedule(schedule.ID, 10, 0)
		if err != nil {
			t.Fatalf("GetBySchedule() failed: %v", err)
		}
		if total == 1 && len(records) == 1 && records[0].RetryCount == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for retry scheduling, got total=%d records=%#v", total, records)
		}
		time.Sleep(10 * time.Millisecond)
	}

	started := time.Now()
	if err := trigger.Stop(); err != nil {
		t.Fatalf("Stop() failed: %v", err)
	}
	if elapsed := time.Since(started); elapsed >= time.Second {
		t.Fatalf("expected Stop() to cancel pending retry promptly, took %v", elapsed)
	}
}

type receivedExportWebhook struct {
	payload           webhook.Payload
	exportBody        []byte
	exportFilename    string
	exportContentType string
	payloadType       string
}

func decodeReceivedExportWebhook(r *http.Request) (receivedExportWebhook, error) {
	if got := r.Header.Get("X-Spartan-Webhook-Event-Type"); got != string(webhook.EventExportCompleted) {
		return receivedExportWebhook{}, io.ErrUnexpectedEOF
	}
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return receivedExportWebhook{}, err
	}
	if mediaType != "multipart/form-data" {
		return receivedExportWebhook{}, io.ErrUnexpectedEOF
	}
	reader := multipart.NewReader(r.Body, params["boundary"])
	result := receivedExportWebhook{payloadType: r.Header.Get("X-Spartan-Webhook-Payload-Type")}
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
			if err := json.Unmarshal(body, &result.payload); err != nil {
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

func assertPrivateExportMode(t *testing.T, path string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat export file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("export file permissions = %o, want %o", got, 0o600)
	}
}

func writeResultJob(t *testing.T, dataDir, jobID string, kind model.Kind, result string) model.Job {
	t.Helper()

	jobDir := filepath.Join(dataDir, "jobs", jobID)
	if err := os.MkdirAll(jobDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(jobDir): %v", err)
	}
	resultPath := filepath.Join(jobDir, "results.jsonl")
	if err := os.WriteFile(resultPath, []byte(result), 0o644); err != nil {
		t.Fatalf("WriteFile(resultPath): %v", err)
	}

	return model.Job{
		ID:         jobID,
		Kind:       kind,
		Status:     model.StatusSucceeded,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ResultPath: resultPath,
	}
}
