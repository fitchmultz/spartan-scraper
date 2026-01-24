package mcp

import (
	"context"
	"os"
	"runtime"
	"testing"
	"time"

	"spartan-scraper/internal/config"
	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/fetch"
	"spartan-scraper/internal/pipeline"
)

func TestServerCloseStopsManager(t *testing.T) {
	// Create a temporary data directory
	tmpDir, err := os.MkdirTemp("", "mcp-server-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test config
	cfg := config.Config{
		DataDir:            tmpDir,
		UserAgent:          "test-agent",
		RequestTimeoutSecs: 30,
		MaxConcurrency:     2,
		RateLimitQPS:       10,
		RateLimitBurst:     5,
		MaxRetries:         3,
		RetryBaseMs:        100,
		MaxResponseBytes:   10 * 1024 * 1024,
		UsePlaywright:      false,
	}

	// Create server
	srv, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	// Get initial goroutine count
	initialGoroutines := runtime.NumGoroutine()

	// Create a job that will take some time (use a long timeout URL)
	// This ensures the manager has active work
	ctx := context.Background()
	job, err := srv.manager.CreateScrapeJob(
		ctx,
		"http://example.com", // will fail but that's okay
		false,
		false,
		fetch.AuthOptions{},
		30,
		extract.ExtractOptions{},
		pipeline.Options{},
		false,
	)
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
	}

	// Enqueue the job
	if err := srv.manager.Enqueue(job); err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// Give the manager a moment to pick up the job
	time.Sleep(100 * time.Millisecond)

	// Now close the server
	// This should cancel the context and wait for the manager
	if err := srv.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Give goroutines time to exit
	time.Sleep(200 * time.Millisecond)

	// Check that goroutines have cleaned up
	// We allow some tolerance for other goroutines (testing, GC, etc.)
	finalGoroutines := runtime.NumGoroutine()
	leaked := finalGoroutines - initialGoroutines

	// If we leaked more than 5 goroutines, something is wrong
	if leaked > 5 {
		t.Errorf("Potential goroutine leak: started with %d, ended with %d (leaked %d)",
			initialGoroutines, finalGoroutines, leaked)
	}

	// Verify manager status shows no active jobs
	status := srv.manager.Status()
	if status.ActiveJobs > 0 {
		t.Errorf("Manager still has active jobs after Close: %d", status.ActiveJobs)
	}
}

func TestServerCloseIdempotent(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mcp-server-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cfg := config.Config{
		DataDir:            tmpDir,
		UserAgent:          "test-agent",
		RequestTimeoutSecs: 30,
		MaxConcurrency:     1,
		RateLimitQPS:       10,
		RateLimitBurst:     5,
		MaxRetries:         3,
		RetryBaseMs:        100,
		MaxResponseBytes:   10 * 1024 * 1024,
		UsePlaywright:      false,
	}

	srv, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("NewServer failed: %v", err)
	}

	// First close
	if err := srv.Close(); err != nil {
		t.Fatalf("First Close failed: %v", err)
	}

	// Second close should be safe (idempotent)
	if err := srv.Close(); err != nil {
		t.Errorf("Second Close failed (should be idempotent): %v", err)
	}
}
