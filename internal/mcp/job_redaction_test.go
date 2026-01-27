package mcp

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"spartan-scraper/internal/model"
)

func TestJobStatus_RedactsSensitiveData(t *testing.T) {
	server, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer server.Close()

	ctx := context.Background()

	// Create job with sensitive data
	job := model.Job{
		ID:         "job-1",
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ResultPath: "/Users/admin/.data/results/job-1.jsonl",
		Params: map[string]interface{}{
			"url":      "https://example.com",
			"password": "secret123",
			"apiKey":   "abc-def",
			"headers": map[string]interface{}{
				"Authorization": "Bearer secret-token",
				"Content-Type":  "application/json",
			},
		},
	}

	if err := server.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	base := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name":      "job_status",
			"arguments": map[string]interface{}{"id": "job-1"},
		}),
	}

	result, err := server.handleToolCall(ctx, base)
	if err != nil {
		t.Fatalf("handleToolCall failed: %v", err)
	}

	resultJob, ok := result.(model.Job)
	if !ok {
		t.Fatalf("Expected model.Job, got %T", result)
	}

	// Verify ResultPath is empty
	if resultJob.ResultPath != "" {
		t.Errorf("ResultPath should be empty, got: %s", resultJob.ResultPath)
	}

	// Verify sensitive params are redacted
	if resultJob.Params["password"] != "[REDACTED]" {
		t.Errorf("password should be redacted, got: %v", resultJob.Params["password"])
	}
	if resultJob.Params["apiKey"] != "[REDACTED]" {
		t.Errorf("apiKey should be redacted, got: %v", resultJob.Params["apiKey"])
	}

	// Verify headers are redacted
	headers := resultJob.Params["headers"].(map[string]interface{})
	if headers["Authorization"] != "[REDACTED]" {
		t.Errorf("Authorization header should be redacted, got: %v", headers["Authorization"])
	}
	if headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type header should not be redacted, got: %v", headers["Content-Type"])
	}

	// Verify non-sensitive params are preserved
	if resultJob.Params["url"] != "https://example.com" {
		t.Errorf("url should not be redacted, got: %v", resultJob.Params["url"])
	}
}

func TestJobList_RedactsSensitiveData(t *testing.T) {
	server, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer server.Close()

	ctx := context.Background()

	// Create jobs with sensitive data
	job1 := model.Job{
		ID:         "job-1",
		Kind:       model.KindScrape,
		Status:     model.StatusSucceeded,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ResultPath: "/Users/admin/.data/results/job-1.jsonl",
		Params: map[string]interface{}{
			"url":    "https://example.com",
			"secret": "top-secret-1",
		},
	}
	job2 := model.Job{
		ID:         "job-2",
		Kind:       model.KindCrawl,
		Status:     model.StatusRunning,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		ResultPath: "/home/user/results/job-2.jsonl",
		Params: map[string]interface{}{
			"url":   "https://test.com",
			"token": "bearer-token-2",
		},
	}

	if err := server.store.Create(ctx, job1); err != nil {
		t.Fatalf("failed to create job1: %v", err)
	}
	if err := server.store.Create(ctx, job2); err != nil {
		t.Fatalf("failed to create job2: %v", err)
	}

	base := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name":      "job_list",
			"arguments": map[string]interface{}{"limit": 10, "offset": 0},
		}),
	}

	result, err := server.handleToolCall(ctx, base)
	if err != nil {
		t.Fatalf("handleToolCall failed: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map[string]interface{}, got %T", result)
	}

	jobs, ok := resultMap["jobs"].([]model.Job)
	if !ok {
		t.Fatalf("Expected []model.Job, got %T", resultMap["jobs"])
	}

	if len(jobs) != 2 {
		t.Fatalf("Expected 2 jobs, got %d", len(jobs))
	}

	// Verify all jobs have ResultPath redacted
	for _, job := range jobs {
		if job.ResultPath != "" {
			t.Errorf("Job %s: ResultPath should be empty, got: %s", job.ID, job.ResultPath)
		}
	}

	// Find job-1 and verify secrets are redacted
	var foundJob1 *model.Job
	for i := range jobs {
		if jobs[i].ID == "job-1" {
			foundJob1 = &jobs[i]
			break
		}
	}

	if foundJob1 == nil {
		t.Fatal("job-1 not found in response")
	}

	if foundJob1.Params["secret"] != "[REDACTED]" {
		t.Errorf("secret should be redacted, got: %v", foundJob1.Params["secret"])
	}

	// Find job-2 and verify token is redacted
	var foundJob2 *model.Job
	for i := range jobs {
		if jobs[i].ID == "job-2" {
			foundJob2 = &jobs[i]
			break
		}
	}

	if foundJob2 == nil {
		t.Fatal("job-2 not found in response")
	}

	if foundJob2.Params["token"] != "[REDACTED]" {
		t.Errorf("token should be redacted, got: %v", foundJob2.Params["token"])
	}
}

func TestJobStatus_ErrorWithPath_Redacted(t *testing.T) {
	server, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer server.Close()

	ctx := context.Background()

	// Create job with error containing path
	job := model.Job{
		ID:        "job-1",
		Kind:      model.KindScrape,
		Status:    model.StatusFailed,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Error:     "Failed to write to /Users/admin/.data/temp/file.txt: permission denied",
		Params:    map[string]interface{}{"url": "https://example.com"},
	}

	if err := server.store.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	base := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name":      "job_status",
			"arguments": map[string]interface{}{"id": "job-1"},
		}),
	}

	result, err := server.handleToolCall(ctx, base)
	if err != nil {
		t.Fatalf("handleToolCall failed: %v", err)
	}

	resultJob := result.(model.Job)

	// Verify error has paths redacted
	if strings.Contains(resultJob.Error, "/Users/admin/.data") {
		t.Errorf("Error should not contain filesystem path, got: %s", resultJob.Error)
	}
	if !strings.Contains(resultJob.Error, "[REDACTED]") {
		t.Errorf("Error should contain [REDACTED] placeholder, got: %s", resultJob.Error)
	}
}

func TestJobStatus_NotFound(t *testing.T) {
	server, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer server.Close()

	base := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name":      "job_status",
			"arguments": map[string]interface{}{"id": "nonexistent"},
		}),
	}

	_, err := server.handleToolCall(context.Background(), base)
	if err == nil {
		t.Error("Expected error for nonexistent job")
	}
}

func TestJobStatus_MissingID(t *testing.T) {
	server, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer server.Close()

	base := map[string]json.RawMessage{
		"params": mustMarshalJSON(map[string]interface{}{
			"name":      "job_status",
			"arguments": map[string]interface{}{},
		}),
	}

	_, err := server.handleToolCall(context.Background(), base)
	if err == nil {
		t.Error("Expected error for missing id")
	}
}
