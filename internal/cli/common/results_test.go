package common

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func TestWaitForJob_RedactedErrorSurfaced(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer st.Close()

	jobID := "test-job-redacted"

	job := model.Job{
		ID:        jobID,
		Kind:      model.KindScrape,
		Status:    model.StatusFailed,
		Error:     "connection timeout after 30s - Authorization: [REDACTED]",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := st.Create(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	ctx := context.Background()
	err = waitForJob(ctx, st, jobID, 10*time.Second)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expected := "job failed: connection timeout after 30s - Authorization: [REDACTED]"
	if err.Error() != expected {
		t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestWaitForJob_RedactedErrorWithSecrets(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer st.Close()

	jobID := "test-job-secrets"

	job := model.Job{
		ID:        jobID,
		Kind:      model.KindScrape,
		Status:    model.StatusFailed,
		Error:     "auth failed: password=[REDACTED], token=[REDACTED], path=[REDACTED]",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := st.Create(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	ctx := context.Background()
	err = waitForJob(ctx, st, jobID, 10*time.Second)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "[REDACTED]") {
		t.Errorf("expected [REDACTED] in error message, got: %s", errMsg)
	}

	if strings.Contains(errMsg, "password=abc") || strings.Contains(errMsg, "token=xyz") {
		t.Errorf("error message contains unredacted secrets: %s", errMsg)
	}
}

func TestWaitForJob_EmptyRedactedError(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer st.Close()

	jobID := "test-job-empty-error"

	job := model.Job{
		ID:        jobID,
		Kind:      model.KindScrape,
		Status:    model.StatusFailed,
		Error:     "",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := st.Create(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	ctx := context.Background()
	err = waitForJob(ctx, st, jobID, 10*time.Second)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	expected := "job failed"
	if err.Error() != expected {
		t.Errorf("expected error message '%s', got '%s'", expected, err.Error())
	}
}

func TestWaitForJob_SuccessfulJob(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer st.Close()

	jobID := "test-job-success"

	job := model.Job{
		ID:        jobID,
		Kind:      model.KindScrape,
		Status:    model.StatusSucceeded,
		Error:     "",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := st.Create(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	ctx := context.Background()
	err = waitForJob(ctx, st, jobID, 10*time.Second)

	if err != nil {
		t.Fatalf("expected no error for successful job, got: %v", err)
	}
}

func TestWaitForJob_ErrorWithFilesystemPath(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer st.Close()

	jobID := "test-job-path"

	job := model.Job{
		ID:        jobID,
		Kind:      model.KindScrape,
		Status:    model.StatusFailed,
		Error:     "failed to write to [REDACTED]: permission denied",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := st.Create(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	ctx := context.Background()
	err = waitForJob(ctx, st, jobID, 10*time.Second)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "[REDACTED]") {
		t.Errorf("expected [REDACTED] in error message, got: %s", errMsg)
	}

	if strings.Contains(errMsg, "/Users/") || strings.Contains(errMsg, "/home/") || strings.Contains(errMsg, "C:\\") {
		t.Errorf("error message contains unredacted path: %s", errMsg)
	}
}

func TestWaitForJob_ErrorWithBearerToken(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer st.Close()

	jobID := "test-job-bearer"

	job := model.Job{
		ID:        jobID,
		Kind:      model.KindScrape,
		Status:    model.StatusFailed,
		Error:     "authentication failed: Bearer [REDACTED]",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := st.Create(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	ctx := context.Background()
	err = waitForJob(ctx, st, jobID, 10*time.Second)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "Bearer [REDACTED]") {
		t.Errorf("expected 'Bearer [REDACTED]' in error message, got: %s", errMsg)
	}

	if strings.Contains(errMsg, "Bearer abc") || strings.Contains(errMsg, "Bearer xyz") {
		t.Errorf("error message contains unredacted token: %s", errMsg)
	}
}
