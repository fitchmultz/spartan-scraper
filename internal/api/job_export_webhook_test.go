// Package api provides webhook export contract coverage for direct job exports.
//
// Purpose:
//   - Prove direct exports use the same multipart webhook delivery contract as
//     scheduled webhook exports while returning a guided outcome envelope.
//
// Responsibilities:
//   - Verify rendered direct-export bytes are returned inline inside the outcome envelope.
//   - Verify the same rendered bytes are sent to export_completed webhook
//     receivers with stable metadata and without local filesystem paths.
//
// Scope:
// - API-handler integration coverage only.
//
// Usage:
// - Runs with `go test ./internal/api`.
//
// Invariants/Assumptions:
//   - Export-completed webhook deliveries are best-effort side effects for direct
//     exports, but they must use the canonical multipart contract when configured.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
)

func TestHandleJobExportDispatchesMultipartExportWebhook(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	deliveryStore := webhook.NewStore(srv.cfg.DataDir)
	srv.webhookDispatcher = webhook.NewDispatcherWithStore(webhook.Config{
		AllowInternal: true,
		MaxRetries:    1,
		Timeout:       time.Second,
	}, deliveryStore)

	received := make(chan apiExportWebhookRequest, 1)
	const receiverDelay = 100 * time.Millisecond
	receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request, err := decodeAPIExportWebhookRequest(r)
		if err != nil {
			t.Errorf("decodeAPIExportWebhookRequest(): %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		time.Sleep(receiverDelay)
		select {
		case received <- request:
		default:
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer receiver.Close()

	ctx := context.Background()
	jobID := "test-job-export-webhook"
	createSucceededJobWithWebhookResult(t, srv, ctx, jobID, model.KindScrape, receiver.URL, `{"url":"https://example.com","status":200,"title":"Example Domain"}`)

	body := `{"format":"csv","transform":{"expression":"{title: title}","language":"jmespath"}}`
	req := newJSONExportRequest(http.MethodPost, fmt.Sprintf("/v1/jobs/%s/export", jobID), body)
	rr := httptest.NewRecorder()
	started := time.Now()
	srv.Routes().ServeHTTP(rr, req)
	if elapsed := time.Since(started); elapsed < receiverDelay {
		t.Fatalf("expected export handler to wait at least %v for webhook delivery, took %v", receiverDelay, elapsed)
	}

	response := decodeExportOutcomeResponse(t, rr)
	if response.Export.Status != "succeeded" {
		t.Fatalf("expected succeeded export, got %#v", response.Export)
	}
	if response.Export.Artifact == nil {
		t.Fatalf("expected export artifact, got %#v", response.Export)
	}
	if response.Export.Artifact.ContentType != "text/csv; charset=utf-8" {
		t.Fatalf("expected csv content type, got %#v", response.Export.Artifact)
	}
	if response.Export.Artifact.Content != "title\nExample Domain\n" {
		t.Fatalf("unexpected direct export body: %q", response.Export.Artifact.Content)
	}

	select {
	case request := <-received:
		if request.metadata["eventType"] != "export.completed" {
			t.Fatalf("unexpected webhook metadata: %#v", request.metadata)
		}
		if request.metadata["jobId"] != jobID {
			t.Fatalf("expected webhook jobId %s, got %#v", jobID, request.metadata)
		}
		if request.metadata["exportFormat"] != "csv" {
			t.Fatalf("expected csv export format, got %#v", request.metadata)
		}
		if request.metadata["filename"] != jobID+".csv" {
			t.Fatalf("expected filename %s.csv, got %#v", jobID, request.metadata)
		}
		if request.metadata["contentType"] != "text/csv; charset=utf-8" {
			t.Fatalf("expected csv metadata content type, got %#v", request.metadata)
		}
		if request.metadata["resultUrl"] != "/v1/jobs/"+jobID+"/results" {
			t.Fatalf("expected resultUrl for %s, got %#v", jobID, request.metadata)
		}
		if _, exists := request.metadata["resultPath"]; exists {
			t.Fatalf("unexpected resultPath leak in metadata: %#v", request.metadata)
		}
		if _, exists := request.metadata["exportPath"]; exists {
			t.Fatalf("unexpected exportPath leak in metadata: %#v", request.metadata)
		}
		if request.exportFilename != jobID+".csv" {
			t.Fatalf("expected export part filename %s.csv, got %q", jobID, request.exportFilename)
		}
		if request.exportContentType != "text/csv; charset=utf-8" {
			t.Fatalf("expected export part content type text/csv; charset=utf-8, got %q", request.exportContentType)
		}
		if string(request.exportBody) != response.Export.Artifact.Content {
			t.Fatalf("expected webhook export body to match HTTP response: body=%q webhook=%q", response.Export.Artifact.Content, string(request.exportBody))
		}
		if request.payloadType != "export-multipart" {
			t.Fatalf("expected export-multipart payload type header, got %q", request.payloadType)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for export webhook delivery")
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		deliveries, err := deliveryStore.ListRecords(context.Background(), jobID, 10, 0)
		if err != nil {
			t.Fatalf("ListRecords() failed: %v", err)
		}
		if len(deliveries) == 1 && deliveries[0].Status == webhook.DeliveryStatusDelivered {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("unexpected delivery records: %#v", deliveries)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

type apiExportWebhookRequest struct {
	metadata          map[string]any
	exportBody        []byte
	exportFilename    string
	exportContentType string
	payloadType       string
}

func decodeAPIExportWebhookRequest(r *http.Request) (apiExportWebhookRequest, error) {
	mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return apiExportWebhookRequest{}, err
	}
	if mediaType != "multipart/form-data" {
		return apiExportWebhookRequest{}, fmt.Errorf("unexpected webhook content type %q", mediaType)
	}
	reader := multipart.NewReader(r.Body, params["boundary"])
	result := apiExportWebhookRequest{
		metadata:    map[string]any{},
		payloadType: r.Header.Get("X-Spartan-Webhook-Payload-Type"),
	}
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return apiExportWebhookRequest{}, err
		}
		body, err := io.ReadAll(part)
		if err != nil {
			return apiExportWebhookRequest{}, err
		}
		switch part.FormName() {
		case "metadata":
			if err := json.Unmarshal(body, &result.metadata); err != nil {
				return apiExportWebhookRequest{}, err
			}
		case "export":
			result.exportBody = body
			result.exportFilename = part.FileName()
			result.exportContentType = part.Header.Get("Content-Type")
		}
	}
	return result, nil
}

func createSucceededJobWithWebhookResult(t *testing.T, srv *Server, ctx context.Context, jobID string, kind model.Kind, webhookURL string, resultContent string) {
	t.Helper()
	job := model.Job{
		ID:        jobID,
		Kind:      kind,
		Status:    model.StatusQueued,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Spec: model.ScrapeSpecV1{
			Version: model.JobSpecVersion1,
			URL:     "https://example.com",
			Execution: model.ExecutionSpec{
				Webhook: &model.WebhookSpec{
					URL:    webhookURL,
					Events: []string{"export_completed"},
				},
			},
		},
	}
	if err := srv.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}
	resultDir := filepath.Join(srv.store.DataDir(), "jobs", jobID)
	if err := fsutil.MkdirAllSecure(resultDir); err != nil {
		t.Fatalf("failed to create result directory: %v", err)
	}
	resultPath := filepath.Join(resultDir, "results.jsonl")
	if err := os.WriteFile(resultPath, []byte(resultContent), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}
	if err := srv.store.UpdateResultPath(ctx, jobID, resultPath); err != nil {
		t.Fatalf("failed to update job result path: %v", err)
	}
	if err := srv.store.UpdateStatus(ctx, jobID, model.StatusSucceeded, ""); err != nil {
		t.Fatalf("failed to update job status: %v", err)
	}
}
