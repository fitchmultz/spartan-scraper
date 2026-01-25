package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
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

func TestMCPToolCallsWithPipelineAndIncremental(t *testing.T) {
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
	defer srv.Close()

	ctx := context.Background()

	t.Run("scrape_page with pipeline and incremental", func(t *testing.T) {
		job, err := srv.manager.CreateScrapeJob(
			ctx,
			"http://example.com",
			false,
			false,
			fetch.AuthOptions{},
			30,
			extract.ExtractOptions{},
			pipeline.Options{
				PreProcessors:  []string{"prep1", "prep2"},
				PostProcessors: []string{"post1"},
				Transformers:   []string{"trans1", "trans2", "trans3"},
			},
			true,
		)
		if err != nil {
			t.Fatalf("CreateScrapeJob failed: %v", err)
		}

		if job.Params["pipeline"] == nil {
			t.Error("pipeline options not stored in job params")
		}
		pipelineOpts, ok := job.Params["pipeline"].(pipeline.Options)
		if !ok {
			t.Fatalf("pipeline params is not pipeline.Options type")
		}
		if len(pipelineOpts.PreProcessors) != 2 {
			t.Errorf("expected 2 preProcessors, got %d", len(pipelineOpts.PreProcessors))
		}
		if len(pipelineOpts.PostProcessors) != 1 {
			t.Errorf("expected 1 postProcessor, got %d", len(pipelineOpts.PostProcessors))
		}
		if len(pipelineOpts.Transformers) != 3 {
			t.Errorf("expected 3 transformers, got %d", len(pipelineOpts.Transformers))
		}

		inc, ok := job.Params["incremental"].(bool)
		if !ok || !inc {
			t.Error("incremental flag not stored correctly in job params")
		}
	})

	t.Run("crawl_site with partial pipeline options", func(t *testing.T) {
		job, err := srv.manager.CreateCrawlJob(
			ctx,
			"http://example.com",
			2,
			100,
			false,
			false,
			fetch.AuthOptions{},
			30,
			extract.ExtractOptions{},
			pipeline.Options{
				PreProcessors: []string{"only-prep"},
			},
			false,
		)
		if err != nil {
			t.Fatalf("CreateCrawlJob failed: %v", err)
		}

		pipelineOpts, _ := job.Params["pipeline"].(pipeline.Options)
		if len(pipelineOpts.PreProcessors) != 1 {
			t.Errorf("expected 1 preProcessor, got %d", len(pipelineOpts.PreProcessors))
		}
		if len(pipelineOpts.PostProcessors) != 0 {
			t.Errorf("expected 0 postProcessors, got %d", len(pipelineOpts.PostProcessors))
		}
	})

	t.Run("research with empty pipeline options and no incremental", func(t *testing.T) {
		job, err := srv.manager.CreateResearchJob(
			ctx,
			"test query",
			[]string{"http://example.com"},
			2,
			100,
			false,
			false,
			fetch.AuthOptions{},
			30,
			extract.ExtractOptions{},
			pipeline.Options{},
			false,
		)
		if err != nil {
			t.Fatalf("CreateResearchJob failed: %v", err)
		}

		pipelineOpts, _ := job.Params["pipeline"].(pipeline.Options)
		if len(pipelineOpts.PreProcessors) != 0 ||
			len(pipelineOpts.PostProcessors) != 0 ||
			len(pipelineOpts.Transformers) != 0 {
			t.Error("expected empty pipeline options")
		}
		inc, _ := job.Params["incremental"].(bool)
		if inc {
			t.Error("expected incremental to be false")
		}
	})
}

func TestGetPipelineOptions(t *testing.T) {
	tests := []struct {
		name     string
		args     map[string]interface{}
		expected pipeline.Options
	}{
		{
			name:     "nil args returns empty",
			args:     nil,
			expected: pipeline.Options{},
		},
		{
			name: "all fields populated",
			args: map[string]interface{}{
				"preProcessors":  []interface{}{"prep1", "prep2"},
				"postProcessors": []interface{}{"post1"},
				"transformers":   []interface{}{"trans1"},
			},
			expected: pipeline.Options{
				PreProcessors:  []string{"prep1", "prep2"},
				PostProcessors: []string{"post1"},
				Transformers:   []string{"trans1"},
			},
		},
		{
			name: "partial fields",
			args: map[string]interface{}{
				"preProcessors": []interface{}{"only-prep"},
			},
			expected: pipeline.Options{
				PreProcessors: []string{"only-prep"},
			},
		},
		{
			name:     "missing keys return empty slices",
			args:     map[string]interface{}{},
			expected: pipeline.Options{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getPipelineOptions(tt.args)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestHandleToolCallWithPipelineAndIncremental(t *testing.T) {
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
	defer srv.Close()

	ctx := context.Background()

	t.Run("scrape_page with all pipeline options and incremental true", func(t *testing.T) {
		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name": "scrape_page",
				"arguments": map[string]interface{}{
					"url":            "https://example.com",
					"headless":       false,
					"playwright":     false,
					"timeoutSeconds": 30,
					"preProcessors":  []string{"prep1", "prep2"},
					"postProcessors": []string{"post1"},
					"transformers":   []string{"trans1", "trans2"},
					"incremental":    true,
				},
			}),
		}

		_, err := srv.handleToolCall(ctx, base)
		if err != nil {
			t.Fatalf("handleToolCall failed: %v", err)
		}

		jobs, err := srv.store.List(ctx)
		if err != nil {
			t.Fatalf("failed to list jobs: %v", err)
		}
		if len(jobs) == 0 {
			t.Fatal("expected a job to be created")
		}

		job := jobs[0]
		pipelineMap, ok := job.Params["pipeline"].(map[string]interface{})
		if !ok {
			t.Fatal("pipeline params not found or wrong type")
		}
		preProcessors, _ := pipelineMap["preProcessors"].([]interface{})
		postProcessors, _ := pipelineMap["postProcessors"].([]interface{})
		transformers, _ := pipelineMap["transformers"].([]interface{})
		inc, ok := job.Params["incremental"].(bool)
		if !ok || !inc {
			t.Errorf("incremental: got %v, want true", inc)
		}

		preProcessorsStr := make([]string, len(preProcessors))
		for i, v := range preProcessors {
			preProcessorsStr[i] = v.(string)
		}
		postProcessorsStr := make([]string, len(postProcessors))
		for i, v := range postProcessors {
			postProcessorsStr[i] = v.(string)
		}
		transformersStr := make([]string, len(transformers))
		for i, v := range transformers {
			transformersStr[i] = v.(string)
		}

		if !reflect.DeepEqual(preProcessorsStr, []string{"prep1", "prep2"}) {
			t.Errorf("preProcessors: got %+v, want [prep1 prep2]", preProcessorsStr)
		}
		if !reflect.DeepEqual(postProcessorsStr, []string{"post1"}) {
			t.Errorf("postProcessors: got %+v, want [post1]", postProcessorsStr)
		}
		if !reflect.DeepEqual(transformersStr, []string{"trans1", "trans2"}) {
			t.Errorf("transformers: got %+v, want [trans1 trans2]", transformersStr)
		}
	})

	t.Run("crawl_site with partial pipeline options", func(t *testing.T) {
		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name": "crawl_site",
				"arguments": map[string]interface{}{
					"url":           "https://example.com",
					"maxDepth":      2,
					"maxPages":      10,
					"preProcessors": []string{"only-prep"},
					"incremental":   false,
				},
			}),
		}

		_, err := srv.handleToolCall(ctx, base)
		if err != nil {
			t.Fatalf("handleToolCall failed: %v", err)
		}

		jobs, err := srv.store.List(ctx)
		if err != nil {
			t.Fatalf("failed to list jobs: %v", err)
		}
		job := jobs[0]
		pipelineMap, _ := job.Params["pipeline"].(map[string]interface{})
		preProcessors, _ := pipelineMap["preProcessors"].([]interface{})
		postProcessors, _ := pipelineMap["postProcessors"].([]interface{})
		transformers, _ := pipelineMap["transformers"].([]interface{})
		inc, _ := job.Params["incremental"].(bool)
		if inc {
			t.Error("incremental: got true, want false")
		}

		preProcessorsStr := make([]string, len(preProcessors))
		for i, v := range preProcessors {
			preProcessorsStr[i] = v.(string)
		}

		if !reflect.DeepEqual(preProcessorsStr, []string{"only-prep"}) {
			t.Errorf("preProcessors: got %+v, want [only-prep]", preProcessorsStr)
		}
		if len(postProcessors) != 0 {
			t.Errorf("postProcessors: got %+v, want empty", postProcessors)
		}
		if len(transformers) != 0 {
			t.Errorf("transformers: got %+v, want empty", transformers)
		}
	})

	t.Run("research with empty pipeline options (default behavior)", func(t *testing.T) {
		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name": "research",
				"arguments": map[string]interface{}{
					"query": "test",
					"urls":  []string{"https://example.com"},
				},
			}),
		}

		_, err := srv.handleToolCall(ctx, base)
		if err != nil {
			t.Fatalf("handleToolCall failed: %v", err)
		}

		jobs, err := srv.store.List(ctx)
		if err != nil {
			t.Fatalf("failed to list jobs: %v", err)
		}
		job := jobs[0]
		pipelineMap, _ := job.Params["pipeline"].(map[string]interface{})
		preProcessors, _ := pipelineMap["preProcessors"].([]interface{})
		postProcessors, _ := pipelineMap["postProcessors"].([]interface{})
		transformers, _ := pipelineMap["transformers"].([]interface{})
		inc, _ := job.Params["incremental"].(bool)
		if inc {
			t.Error("incremental: got true, want false")
		}

		if len(preProcessors) != 0 || len(postProcessors) != 0 || len(transformers) != 0 {
			t.Error("expected all pipeline slices to be empty")
		}
	})
}

func TestToolsListSchemaIncludesPipelineAndIncremental(t *testing.T) {
	tmpDir := t.TempDir()
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
	defer srv.Close()

	tools := srv.toolsList()
	toolMap := make(map[string]tool)
	for _, t := range tools {
		toolMap[t.Name] = t
	}

	t.Run("scrape_page schema", func(t *testing.T) {
		tool, ok := toolMap["scrape_page"]
		if !ok {
			t.Fatal("scrape_page tool not found")
		}
		schema := tool.InputSchema
		props, ok := schema["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("properties not found in schema")
		}
		requiredFields := schema["required"]
		requiredSlice, _ := requiredFields.([]interface{})
		requiredSet := make(map[string]bool)
		for _, f := range requiredSlice {
			requiredSet[f.(string)] = true
		}

		for _, field := range []string{"preProcessors", "postProcessors", "transformers", "incremental"} {
			if _, ok := props[field]; !ok {
				t.Errorf("expected %s in properties", field)
			}
			if requiredSet[field] {
				t.Errorf("expected %s to be optional, but it's in required", field)
			}
		}

		preProcessorsType, ok := props["preProcessors"].(map[string]string)
		if !ok || preProcessorsType["type"] != "array" {
			t.Error("preProcessors should be array type")
		}
		postProcessorsType, ok := props["postProcessors"].(map[string]string)
		if !ok || postProcessorsType["type"] != "array" {
			t.Error("postProcessors should be array type")
		}
		transformersType, ok := props["transformers"].(map[string]string)
		if !ok || transformersType["type"] != "array" {
			t.Error("transformers should be array type")
		}
		incrementalType, ok := props["incremental"].(map[string]string)
		if !ok || incrementalType["type"] != "boolean" {
			t.Error("incremental should be boolean type")
		}
	})

	t.Run("crawl_site schema", func(t *testing.T) {
		tool, ok := toolMap["crawl_site"]
		if !ok {
			t.Fatal("crawl_site tool not found")
		}
		schema := tool.InputSchema
		props, _ := schema["properties"].(map[string]interface{})
		for _, field := range []string{"preProcessors", "postProcessors", "transformers", "incremental"} {
			if _, ok := props[field]; !ok {
				t.Errorf("expected %s in properties", field)
			}
		}
	})

	t.Run("research schema", func(t *testing.T) {
		tool, ok := toolMap["research"]
		if !ok {
			t.Fatal("research tool not found")
		}
		schema := tool.InputSchema
		props, _ := schema["properties"].(map[string]interface{})
		for _, field := range []string{"preProcessors", "postProcessors", "transformers", "incremental"} {
			if _, ok := props[field]; !ok {
				t.Errorf("expected %s in properties", field)
			}
		}
	})
}

func TestJobListTool(t *testing.T) {
	tmpDir := t.TempDir()
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
	defer srv.Close()

	t.Run("job_list in toolsList", func(t *testing.T) {
		tools := srv.toolsList()
		toolMap := make(map[string]tool)
		for _, t := range tools {
			toolMap[t.Name] = t
		}
		jobListTool, ok := toolMap["job_list"]
		if !ok {
			t.Fatal("job_list tool not found in toolsList")
		}
		if jobListTool.Description != "List all jobs with pagination" {
			t.Errorf("expected description 'List all jobs with pagination', got '%s'", jobListTool.Description)
		}
		schema := jobListTool.InputSchema
		props, ok := schema["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("properties not found in schema")
		}
		if _, ok := props["limit"]; !ok {
			t.Error("expected 'limit' in properties")
		}
		if _, ok := props["offset"]; !ok {
			t.Error("expected 'offset' in properties")
		}
		required := schema["required"]
		if len(required.([]string)) != 0 {
			t.Error("expected no required fields")
		}
	})
}

func TestHandleJobList(t *testing.T) {
	tmpDir := t.TempDir()
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
	defer srv.Close()

	ctx := context.Background()

	t.Run("list all jobs", func(t *testing.T) {
		_, err := srv.manager.CreateScrapeJob(ctx, "http://example.com/1", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
		if err != nil {
			t.Fatalf("CreateScrapeJob failed: %v", err)
		}
		_, err = srv.manager.CreateScrapeJob(ctx, "http://example.com/2", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
		if err != nil {
			t.Fatalf("CreateScrapeJob failed: %v", err)
		}

		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_list",
				"arguments": map[string]interface{}{},
			}),
		}

		result, err := srv.handleToolCall(ctx, base)
		if err != nil {
			t.Fatalf("handleToolCall failed: %v", err)
		}

		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Fatal("result is not a map")
		}
		jobs := resultMap["jobs"]
		if jobs == nil {
			t.Fatal("jobs not found in result")
		}

		jobCount := reflect.ValueOf(jobs).Len()
		if jobCount != 2 {
			t.Errorf("expected 2 jobs, got %d", jobCount)
		}
	})

	t.Run("list with limit and offset", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			_, err := srv.manager.CreateScrapeJob(ctx, fmt.Sprintf("http://example.com/%d", i), false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
			if err != nil {
				t.Fatalf("CreateScrapeJob failed: %v", err)
			}
		}

		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_list",
				"arguments": map[string]interface{}{"limit": 2, "offset": 2},
			}),
		}

		result, err := srv.handleToolCall(ctx, base)
		if err != nil {
			t.Fatalf("handleToolCall failed: %v", err)
		}

		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Fatal("result is not a map")
		}
		jobs := resultMap["jobs"]
		if jobs == nil {
			t.Fatal("jobs not found in result")
		}
		jobCount := reflect.ValueOf(jobs).Len()
		if jobCount != 2 {
			t.Errorf("expected 2 jobs (offset 2, limit 2), got %d", jobCount)
		}
	})
}

func TestJobCancelTool(t *testing.T) {
	tmpDir := t.TempDir()
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
	defer srv.Close()

	t.Run("job_cancel in toolsList", func(t *testing.T) {
		tools := srv.toolsList()
		toolMap := make(map[string]tool)
		for _, t := range tools {
			toolMap[t.Name] = t
		}
		jobCancelTool, ok := toolMap["job_cancel"]
		if !ok {
			t.Fatal("job_cancel tool not found in toolsList")
		}
		if jobCancelTool.Description != "Cancel a running or queued job by id" {
			t.Errorf("expected description 'Cancel a running or queued job by id', got '%s'", jobCancelTool.Description)
		}
		schema := jobCancelTool.InputSchema
		props, ok := schema["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("properties not found in schema")
		}
		if _, ok := props["id"]; !ok {
			t.Error("expected 'id' in properties")
		}
		required := schema["required"].([]string)
		if len(required) != 1 || required[0] != "id" {
			t.Error("expected 'id' to be required")
		}
	})
}

func TestHandleJobCancel(t *testing.T) {
	tmpDir := t.TempDir()
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
	defer srv.Close()

	ctx := context.Background()

	t.Run("cancel queued job", func(t *testing.T) {
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
		if err != nil {
			t.Fatalf("CreateScrapeJob failed: %v", err)
		}

		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_cancel",
				"arguments": map[string]interface{}{"id": job.ID},
			}),
		}

		result, err := srv.handleToolCall(ctx, base)
		if err != nil {
			t.Fatalf("handleToolCall failed: %v", err)
		}

		resultMap, ok := result.(map[string]interface{})
		if !ok {
			t.Fatal("result is not a map")
		}
		if resultMap["status"] != "canceled" {
			t.Errorf("expected status 'canceled', got '%v'", resultMap["status"])
		}
		if resultMap["id"] != job.ID {
			t.Errorf("expected id '%s', got '%v'", job.ID, resultMap["id"])
		}

		updatedJob, err := srv.store.Get(ctx, job.ID)
		if err != nil {
			t.Fatalf("Get job failed: %v", err)
		}
		if updatedJob.Status != "canceled" {
			t.Errorf("expected job status 'canceled', got '%s'", updatedJob.Status)
		}
	})

	t.Run("cancel non-existent job", func(t *testing.T) {
		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_cancel",
				"arguments": map[string]interface{}{"id": "non-existent-id"},
			}),
		}

		_, err := srv.handleToolCall(ctx, base)
		if err == nil {
			t.Error("expected error for non-existent job")
		}
	})

	t.Run("cancel without id", func(t *testing.T) {
		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_cancel",
				"arguments": map[string]interface{}{},
			}),
		}

		_, err := srv.handleToolCall(ctx, base)
		if err == nil {
			t.Error("expected error for missing id")
		}
	})
}

func TestJobExportTool(t *testing.T) {
	tmpDir := t.TempDir()
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
	defer srv.Close()

	t.Run("job_export in toolsList", func(t *testing.T) {
		tools := srv.toolsList()
		toolMap := make(map[string]tool)
		for _, t := range tools {
			toolMap[t.Name] = t
		}
		jobExportTool, ok := toolMap["job_export"]
		if !ok {
			t.Fatal("job_export tool not found in toolsList")
		}
		if jobExportTool.Description != "Export job results in specified format (jsonl, json, md, csv)" {
			t.Errorf("expected description 'Export job results in specified format (jsonl, json, md, csv)', got '%s'", jobExportTool.Description)
		}
		schema := jobExportTool.InputSchema
		props, ok := schema["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("properties not found in schema")
		}
		if _, ok := props["id"]; !ok {
			t.Error("expected 'id' in properties")
		}
		if _, ok := props["format"]; !ok {
			t.Error("expected 'format' in properties")
		}
		required := schema["required"].([]string)
		if len(required) != 1 || required[0] != "id" {
			t.Error("expected 'id' to be required and 'format' to be optional")
		}
	})
}

func TestHandleJobExport(t *testing.T) {
	tmpDir := t.TempDir()
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
	defer srv.Close()

	ctx := context.Background()

	t.Run("export job as jsonl", func(t *testing.T) {
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
		if err != nil {
			t.Fatalf("CreateScrapeJob failed: %v", err)
		}

		resultFile := job.ResultPath
		resultDir := filepath.Join(tmpDir, "jobs", job.ID)
		if err := os.MkdirAll(resultDir, 0o755); err != nil {
			t.Fatalf("failed to create job directory: %v", err)
		}
		resultContent := `{"url":"http://example.com","status":200,"title":"Test","text":"Content"}`
		if err := os.WriteFile(resultFile, []byte(resultContent), 0o644); err != nil {
			t.Fatalf("failed to write result file: %v", err)
		}

		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_export",
				"arguments": map[string]interface{}{"id": job.ID, "format": "jsonl"},
			}),
		}

		result, err := srv.handleToolCall(ctx, base)
		if err != nil {
			t.Fatalf("handleToolCall failed: %v", err)
		}

		resultStr, ok := result.(string)
		if !ok {
			t.Fatal("result is not a string")
		}
		resultStr = strings.TrimSpace(resultStr)
		if resultStr != resultContent {
			t.Errorf("expected result '%s', got '%s'", resultContent, resultStr)
		}
	})

	t.Run("export job with default format", func(t *testing.T) {
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
		if err != nil {
			t.Fatalf("CreateScrapeJob failed: %v", err)
		}

		resultFile := job.ResultPath
		resultDir := filepath.Join(tmpDir, "jobs", job.ID)
		if err := os.MkdirAll(resultDir, 0o755); err != nil {
			t.Fatalf("failed to create job directory: %v", err)
		}
		resultContent := `{"url":"http://example.com","status":200}`
		if err := os.WriteFile(resultFile, []byte(resultContent), 0o644); err != nil {
			t.Fatalf("failed to write result file: %v", err)
		}

		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_export",
				"arguments": map[string]interface{}{"id": job.ID},
			}),
		}

		result, err := srv.handleToolCall(ctx, base)
		if err != nil {
			t.Fatalf("handleToolCall failed: %v", err)
		}

		resultStr, ok := result.(string)
		if !ok {
			t.Fatal("result is not a string")
		}
		resultStr = strings.TrimSpace(resultStr)
		if resultStr != resultContent {
			t.Errorf("expected result '%s', got '%s'", resultContent, resultStr)
		}
	})

	t.Run("export job with invalid format", func(t *testing.T) {
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
		if err != nil {
			t.Fatalf("CreateScrapeJob failed: %v", err)
		}

		resultFile := job.ResultPath
		resultDir := filepath.Join(tmpDir, "jobs", job.ID)
		if err := os.MkdirAll(resultDir, 0o755); err != nil {
			t.Fatalf("failed to create job directory: %v", err)
		}
		resultContent := `{"url":"http://example.com","status":200}`
		if err := os.WriteFile(resultFile, []byte(resultContent), 0o644); err != nil {
			t.Fatalf("failed to write result file: %v", err)
		}

		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_export",
				"arguments": map[string]interface{}{"id": job.ID, "format": "txt"},
			}),
		}

		_, err = srv.handleToolCall(ctx, base)
		if err == nil {
			t.Error("expected error for invalid format")
		}
	})

	t.Run("export job without results", func(t *testing.T) {
		job, err := srv.manager.CreateScrapeJob(ctx, "http://example.com", false, false, fetch.AuthOptions{}, 30, extract.ExtractOptions{}, pipeline.Options{}, false)
		if err != nil {
			t.Fatalf("CreateScrapeJob failed: %v", err)
		}

		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_export",
				"arguments": map[string]interface{}{"id": job.ID, "format": "jsonl"},
			}),
		}

		_, err = srv.handleToolCall(ctx, base)
		if err == nil {
			t.Error("expected error for job without results")
		}
	})

	t.Run("export non-existent job", func(t *testing.T) {
		base := map[string]json.RawMessage{
			"params": mustMarshalJSON(map[string]interface{}{
				"name":      "job_export",
				"arguments": map[string]interface{}{"id": "non-existent-id", "format": "jsonl"},
			}),
		}

		_, err := srv.handleToolCall(ctx, base)
		if err == nil {
			t.Error("expected error for non-existent job")
		}
	})
}

func mustMarshalJSON(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
