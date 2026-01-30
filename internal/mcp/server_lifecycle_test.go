// Package mcp provides tests for MCP server lifecycle and shutdown behavior.
// Tests cover goroutine lifecycle management, shutdown requests, and idempotent close operations.
// Does NOT test tool execution behavior, schema validation, or job management operations.
package mcp

import (
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func TestServerCloseStopsManager(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)

	initialGoroutines := runtime.NumGoroutine()

	ctx := context.Background()
	job, err := srv.manager.CreateScrapeJob(
		ctx,
		"http://example.com",
		false,
		false,
		fetch.AuthOptions{},
		30,
		extract.ExtractOptions{},
		pipeline.Options{},
		false,
		"",
	)
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
	}

	if err := srv.manager.Enqueue(job); err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if err := srv.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	finalGoroutines := runtime.NumGoroutine()
	leaked := finalGoroutines - initialGoroutines

	if leaked > 5 {
		t.Errorf("Potential goroutine leak: started with %d, ended with %d (leaked %d)",
			initialGoroutines, finalGoroutines, leaked)
	}

	status := srv.manager.Status()
	if status.ActiveJobs > 0 {
		t.Errorf("Manager still has active jobs after Close: %d", status.ActiveJobs)
	}
}

func TestServerCloseIdempotent(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)

	if err := srv.Close(); err != nil {
		t.Fatalf("First Close failed: %v", err)
	}

	if err := srv.Close(); err != nil {
		t.Errorf("Second Close failed (should be idempotent): %v", err)
	}
}
