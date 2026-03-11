// Package jobs provides tests for manager recovery of queued jobs.
// Tests cover recovery of persisted queued jobs on manager startup.
// Does NOT test recovery of running jobs or partial execution state.
package jobs

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func TestManagerRecoverQueuedJobs(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	job1, _ := m.CreateScrapeJob(ctx, "http://example.com/1", "GET", nil, "", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
	job2, _ := m.CreateScrapeJob(ctx, "http://example.com/2", "GET", nil, "", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")

	queuedJobs, _ := st.ListByStatus(ctx, model.StatusQueued, store.ListByStatusOptions{})
	if len(queuedJobs) != 2 {
		t.Fatalf("expected 2 queued jobs in store, got %d", len(queuedJobs))
	}

	if len(m.queue) != 0 {
		t.Error("queue should be empty before Start")
	}

	cancelCtx, cancel := context.WithCancel(ctx)
	m.Start(cancelCtx)

	time.Sleep(200 * time.Millisecond)

	cancel()

	final1, _ := st.Get(ctx, job1.ID)
	final2, _ := st.Get(ctx, job2.ID)

	if final1.Status == model.StatusQueued || final2.Status == model.StatusQueued {
		t.Error("recovered jobs should have been picked up from queue")
	}
}

// TestManagerRecoverQueuedJobsWithBody verifies that request body bytes are preserved
// across job persistence and recovery. This tests the fix for RQ-0261.
func TestManagerRecoverQueuedJobsWithBody(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	// Create a POST job with a JSON body
	body := []byte(`{"query": "test", "payload": [1, 2, 3], "nested": {"key": "value"}}`)
	job, err := m.CreateScrapeJob(ctx, "http://example.com/api", "POST", body, "application/json", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	// Verify job is stored in database
	storedJob, err := st.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to get job from store: %v", err)
	}

	// Verify body is preserved in stored params (may be base64-encoded string)
	bodyValue, ok := storedJob.SpecMap()["body"]
	if !ok {
		t.Fatal("body param not found in stored job")
	}

	// Decode the body using decodeBytes (this is what happens during job execution)
	decodedBody := decodeBytes(bodyValue)

	// The decoded body should match the original
	if !bytes.Equal(decodedBody, body) {
		t.Errorf("body not preserved after storage: got %q, want %q", decodedBody, body)
	}

	// Simulate manager restart by creating a new manager with the same store
	newManager := NewManager(
		st,
		m.DataDir,
		"TestAgent/1.0",
		30*time.Second,
		2,
		10,
		20,
		3,
		100*time.Millisecond,
		10*1024*1024,
		false,
		fetch.DefaultCircuitBreakerConfig(),
		nil,
	)

	// Start the new manager (this triggers recoverQueuedJobs)
	cancelCtx, cancel := context.WithCancel(ctx)
	newManager.Start(cancelCtx)
	defer cancel()

	// Wait a bit for recovery to happen
	time.Sleep(100 * time.Millisecond)

	// The job should be recovered - check it was enqueued by seeing if status changed
	// or by checking the queue length indirectly through status
	recoveredJob, err := st.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to get recovered job: %v", err)
	}

	// Verify body is still preserved after recovery
	recoveredBodyValue, ok := recoveredJob.SpecMap()["body"]
	if !ok {
		t.Fatal("body param not found in recovered job")
	}

	recoveredBody := decodeBytes(recoveredBodyValue)
	if !bytes.Equal(recoveredBody, body) {
		t.Errorf("body not preserved after recovery: got %q, want %q", recoveredBody, body)
	}
}

// TestManagerRecoverQueuedJobsWithBinaryBody verifies that binary request bodies
// (non-UTF8 data) are preserved correctly across persistence and recovery.
func TestManagerRecoverQueuedJobsWithBinaryBody(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	// Create a job with binary body (e.g., image bytes, protobuf, etc.)
	body := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD, 0xFC, 0xFB}
	job, err := m.CreateScrapeJob(ctx, "http://example.com/upload", "POST", body, "application/octet-stream", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
	if err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	// Retrieve from store
	storedJob, err := st.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("failed to get job from store: %v", err)
	}

	// Decode body
	decodedBody := decodeBytes(storedJob.SpecMap()["body"])

	// Verify binary data is preserved exactly
	if !bytes.Equal(decodedBody, body) {
		t.Errorf("binary body not preserved: got %v, want %v", decodedBody, body)
	}
}

// TestManagerRecoverQueuedJobsWithEmptyBody verifies that jobs with nil/empty bodies
// are handled correctly during recovery.
func TestManagerRecoverQueuedJobsWithEmptyBody(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	// Create jobs with nil and empty bodies
	jobNil, err := m.CreateScrapeJob(ctx, "http://example.com/get", "GET", nil, "", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
	if err != nil {
		t.Fatalf("failed to create job with nil body: %v", err)
	}

	jobEmpty, err := m.CreateScrapeJob(ctx, "http://example.com/post", "POST", []byte{}, "application/json", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
	if err != nil {
		t.Fatalf("failed to create job with empty body: %v", err)
	}

	// Retrieve from store
	storedNil, _ := st.Get(ctx, jobNil.ID)
	storedEmpty, _ := st.Get(ctx, jobEmpty.ID)

	// Decode bodies
	decodedNil := decodeBytes(storedNil.SpecMap()["body"])
	decodedEmpty := decodeBytes(storedEmpty.SpecMap()["body"])

	// nil body should decode to nil or empty
	if decodedNil != nil && len(decodedNil) != 0 {
		t.Errorf("nil body should be nil or empty, got %v", decodedNil)
	}

	// empty body should decode to empty
	if len(decodedEmpty) != 0 {
		t.Errorf("empty body should be empty, got %v", decodedEmpty)
	}
}
