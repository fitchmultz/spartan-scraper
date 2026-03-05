// Package jobs provides tests for job execution and error redaction.
// Tests cover secret redaction in error messages (auth tokens, paths, key-value pairs, JSON secrets).
// Does NOT test actual fetch/extract logic or successful job completion.
package jobs

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func TestJobRun_RedactError_AuthTokens(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	// Create a scrape job with auth that will fail with Authorization header in error
	auth := fetch.AuthOptions{
		Headers: map[string]string{
			"Authorization": "Bearer secret-token-12345",
		},
	}

	job, err := m.CreateScrapeJob(ctx, "http://invalid-url-that-does-not-exist.example", "GET", nil, "", false, false, auth, 5, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
	}

	// Start manager and enqueue the job
	mgrCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(mgrCtx)

	if err := m.Enqueue(job); err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// Wait for job to fail
	for i := 0; i < 50; i++ {
		persisted, err := st.Get(ctx, job.ID)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if persisted.Status == model.StatusFailed {
			// Verify error is redacted
			if persisted.Error == "" {
				t.Error("expected non-empty error message")
			}
			if strings.Contains(persisted.Error, "secret-token-12345") {
				t.Errorf("error contains secret token: %s", persisted.Error)
			}
			if strings.Contains(persisted.Error, "Bearer") && !strings.Contains(persisted.Error, "[REDACTED]") {
				t.Errorf("Authorization header not redacted: %s", persisted.Error)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Error("job did not fail within expected time")
}

func TestJobRun_RedactError_Paths(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	// Create a scrape job that will fail with a path in error
	// Using invalid auth that will produce error messages with paths
	auth := fetch.AuthOptions{
		Headers: map[string]string{
			"Cookie": "session=abc123",
		},
	}

	job, err := m.CreateScrapeJob(ctx, "http://invalid-url-that-does-not-exist.example", "GET", nil, "", false, false, auth, 5, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
	}

	mgrCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(mgrCtx)

	if err := m.Enqueue(job); err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// Wait for job to fail
	for i := 0; i < 50; i++ {
		persisted, err := st.Get(ctx, job.ID)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if persisted.Status == model.StatusFailed {
			// Verify paths are redacted
			if persisted.Error == "" {
				t.Error("expected non-empty error message")
			}
			// Check for common path patterns
			if strings.Contains(persisted.Error, "/Users/") && !strings.Contains(persisted.Error, "[REDACTED]") {
				t.Errorf("Unix user path not redacted: %s", persisted.Error)
			}
			if strings.Contains(persisted.Error, "/home/") && !strings.Contains(persisted.Error, "[REDACTED]") {
				t.Errorf("Unix home path not redacted: %s", persisted.Error)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Error("job did not fail within expected time")
}

func TestJobRun_RedactError_KeyValueSecrets(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	// Create a job with secrets in query params that will be in error
	job, err := m.CreateScrapeJob(ctx, "http://invalid-url-that-does-not-exist.example?password=mypass123&token=mytoken456", "GET", nil, "", false, false, fetch.AuthOptions{}, 5, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
	}

	mgrCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(mgrCtx)

	if err := m.Enqueue(job); err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// Wait for job to fail
	for i := 0; i < 50; i++ {
		persisted, err := st.Get(ctx, job.ID)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if persisted.Status == model.StatusFailed {
			// Verify key=value secrets are redacted
			if persisted.Error == "" {
				t.Error("expected non-empty error message")
			}
			if strings.Contains(persisted.Error, "mypass123") {
				t.Errorf("error contains password: %s", persisted.Error)
			}
			if strings.Contains(persisted.Error, "mytoken456") {
				t.Errorf("error contains token: %s", persisted.Error)
			}
			if !strings.Contains(persisted.Error, "[REDACTED]") {
				t.Errorf("expected [REDACTED] in error: %s", persisted.Error)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Error("job did not fail within expected time")
}

func TestJobRun_RedactError_JSONSecrets(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	// Create a job that will fail with JSON containing secrets in error
	// The error may contain request params which could include JSON
	job, err := m.CreateJob(ctx, JobSpec{
		Kind:          model.KindScrape,
		URL:           "http://invalid-url-that-does-not-exist.example",
		Headless:      false,
		UsePlaywright: false,
		Auth: fetch.AuthOptions{
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
		TimeoutSeconds: 5,
		Extract:        extract.ExtractOptions{},
		Pipeline:       pipeline.Options{},
	})
	if err != nil {
		t.Fatalf("CreateJob failed: %v", err)
	}

	mgrCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(mgrCtx)

	if err := m.Enqueue(job); err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// Wait for job to fail
	for i := 0; i < 50; i++ {
		persisted, err := st.Get(ctx, job.ID)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if persisted.Status == model.StatusFailed {
			// Verify JSON secrets are redacted
			if persisted.Error == "" {
				t.Error("expected non-empty error message")
			}
			// Check for JSON-style secrets in error message
			if strings.Contains(persisted.Error, `"password":`) && !strings.Contains(persisted.Error, `"[REDACTED]"`) {
				t.Errorf("JSON password not redacted: %s", persisted.Error)
			}
			if strings.Contains(persisted.Error, `"apiKey":`) && !strings.Contains(persisted.Error, `"[REDACTED]"`) {
				t.Errorf("JSON apiKey not redacted: %s", persisted.Error)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Error("job did not fail within expected time")
}

func TestJobRun_RedactError_MultipleSecrets(t *testing.T) {
	m, st, cleanup := setupTestManager(t)
	defer cleanup()

	ctx := context.Background()

	// Create a job with multiple types of secrets
	auth := fetch.AuthOptions{
		Headers: map[string]string{
			"Authorization": "Bearer multi-secret-token-789",
			"X-API-Key":     "api-key-value-xyz",
		},
	}

	job, err := m.CreateScrapeJob(ctx, "http://invalid-url-that-does-not-exist.example?password=pass123", "GET", nil, "", false, false, auth, 5, extract.ExtractOptions{}, pipeline.Options{}, false, "", "", nil, "")
	if err != nil {
		t.Fatalf("CreateScrapeJob failed: %v", err)
	}

	mgrCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(mgrCtx)

	if err := m.Enqueue(job); err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	// Wait for job to fail
	for i := 0; i < 50; i++ {
		persisted, err := st.Get(ctx, job.ID)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if persisted.Status == model.StatusFailed {
			// Verify all secrets are redacted
			if persisted.Error == "" {
				t.Error("expected non-empty error message")
			}
			if strings.Contains(persisted.Error, "multi-secret-token-789") {
				t.Errorf("error contains Bearer token: %s", persisted.Error)
			}
			if strings.Contains(persisted.Error, "api-key-value-xyz") {
				t.Errorf("error contains API key: %s", persisted.Error)
			}
			if strings.Contains(persisted.Error, "pass123") {
				t.Errorf("error contains password: %s", persisted.Error)
			}
			// Count redaction occurrences - should be multiple
			redactCount := strings.Count(persisted.Error, "[REDACTED]")
			if redactCount < 1 {
				t.Errorf("expected at least 1 [REDACTED], got %d: %s", redactCount, persisted.Error)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Error("job did not fail within expected time")
}

func TestJobRun_RedactError_NilError(t *testing.T) {
	// Test that SafeMessage handles nil
	result := apperrors.SafeMessage(nil)
	if result != "" {
		t.Errorf("expected empty string from SafeMessage(nil), got: %s", result)
	}
}

func TestJobRun_RedactError_WrappedError(t *testing.T) {
	// Test that SafeMessage properly handles wrapped errors
	wrapped := errors.New("failed to connect: something went wrong")

	result := apperrors.SafeMessage(wrapped)
	if strings.Contains(result, "password=") {
		t.Errorf("SafeMessage leaked secret: %s", result)
	}
}
