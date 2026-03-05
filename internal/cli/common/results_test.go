// Package common contains tests for job result handling utilities.
//
// Responsibilities:
// - Testing waitForJob with secret redaction in error messages
// - Testing copyResults for job output retrieval
// - Validating that secrets, tokens, and paths are properly redacted
//
// Non-goals:
// - Testing actual job execution or worker pools
// - Testing external storage backends
//
// Assumptions:
// - Tests use temporary directories for data storage
// - Tests are isolated and do not depend on external services
package common

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
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

func TestCopyResults_Success(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer st.Close()

	resultPath := filepath.Join(dataDir, "result.json")
	resultContent := `{"results": ["test"]}`
	if err := os.WriteFile(resultPath, []byte(resultContent), 0o644); err != nil {
		t.Fatalf("failed to write result file: %v", err)
	}

	jobID := "test-job-copy"
	job := model.Job{
		ID:         jobID,
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		ResultPath: resultPath,
	}

	if err := st.Create(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	outPath := filepath.Join(dataDir, "output.json")
	err = copyResults(context.Background(), st, jobID, outPath)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if string(content) != resultContent {
		t.Errorf("expected content %q, got %q", resultContent, string(content))
	}
}

func TestCopyResults_NoResultPath(t *testing.T) {
	dataDir := t.TempDir()
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	defer st.Close()

	jobID := "test-job-no-result"
	job := model.Job{
		ID:         jobID,
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		ResultPath: "",
	}

	if err := st.Create(context.Background(), job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	outPath := filepath.Join(dataDir, "output.json")
	err = copyResults(context.Background(), st, jobID, outPath)

	if err == nil {
		t.Fatal("expected error for missing result path, got nil")
	}

	if !apperrors.IsKind(err, apperrors.KindNotFound) {
		t.Errorf("expected KindNotFound, got %v", apperrors.KindOf(err))
	}
}
