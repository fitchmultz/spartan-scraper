// Package mcp provides integration tests for MCP tool execution.
// Tests cover handleToolCall routing, argument parsing, and job creation for scrape_page,
// crawl_site, and research tools with pipeline/incremental options.
// Does NOT test schema validation, server lifecycle, or job management operations.
package mcp

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestHandleToolCallWithPipelineAndIncremental(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
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
		pipelineMap, ok := job.SpecMap()["pipeline"].(map[string]interface{})
		if !ok {
			t.Fatal("pipeline params not found or wrong type")
		}
		preProcessors, _ := pipelineMap["preProcessors"].([]interface{})
		postProcessors, _ := pipelineMap["postProcessors"].([]interface{})
		transformers, _ := pipelineMap["transformers"].([]interface{})
		inc, ok := job.SpecMap()["incremental"].(bool)
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
		pipelineMap, _ := job.SpecMap()["pipeline"].(map[string]interface{})
		preProcessors, _ := pipelineMap["preProcessors"].([]interface{})
		postProcessors, _ := pipelineMap["postProcessors"].([]interface{})
		transformers, _ := pipelineMap["transformers"].([]interface{})
		inc, _ := job.SpecMap()["incremental"].(bool)
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
		pipelineMap, _ := job.SpecMap()["pipeline"].(map[string]interface{})
		preProcessors, _ := pipelineMap["preProcessors"].([]interface{})
		postProcessors, _ := pipelineMap["postProcessors"].([]interface{})
		transformers, _ := pipelineMap["transformers"].([]interface{})
		inc, _ := job.SpecMap()["incremental"].(bool)
		if inc {
			t.Error("incremental: got true, want false")
		}

		if len(preProcessors) != 0 || len(postProcessors) != 0 || len(transformers) != 0 {
			t.Error("expected all pipeline slices to be empty")
		}
	})
}
