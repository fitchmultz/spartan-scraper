// Package mcp provides tests for the research MCP tool.
// Tests cover job creation with empty/default pipeline options and JSON schema validation.
// Does NOT test actual research workflow execution or multi-source aggregation.
package mcp

import (
	"context"
	"os"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func TestResearchWithEmptyPipelineOptions(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	ctx := context.Background()

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
		"",
		"",
		nil,
		"",
	)
	if err != nil {
		t.Fatalf("CreateResearchJob failed: %v", err)
	}

	pipelineMap := pipelineMapFromSpec(t, job.SpecMap())
	preProcessors, _ := pipelineMap["preProcessors"].([]interface{})
	postProcessors, _ := pipelineMap["postProcessors"].([]interface{})
	transformers, _ := pipelineMap["transformers"].([]interface{})
	if len(preProcessors) != 0 ||
		len(postProcessors) != 0 ||
		len(transformers) != 0 {
		t.Error("expected empty pipeline options")
	}
}

func TestResearchSchema(t *testing.T) {
	srv, tmpDir := testServer()
	defer os.RemoveAll(tmpDir)
	defer srv.Close()

	tools := srv.toolsList()
	toolMap := make(map[string]tool)
	for _, t := range tools {
		toolMap[t.Name] = t
	}

	researchTool, ok := toolMap["research"]
	if !ok {
		t.Fatal("research tool not found")
	}
	schema := researchTool.InputSchema
	props, _ := schema["properties"].(map[string]interface{})
	for _, field := range []string{"aiExtract", "aiMode", "aiPrompt", "aiSchema", "aiFields", "preProcessors", "postProcessors", "transformers"} {
		if _, ok := props[field]; !ok {
			t.Errorf("expected %s in properties", field)
		}
	}
}
