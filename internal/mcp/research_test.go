// Tests for the research MCP tool.
// Verifies job creation with empty/default pipeline options and validates the
// tool's JSON schema definition.
//
// Does NOT handle:
// - Actual research workflow execution
// - Multi-source aggregation or result ranking
// - Pipeline processor/transformer execution
//
// Invariants:
// - research tool accepts optional pipeline options (defaults to empty)
// - Schema must include preProcessors, postProcessors, transformers fields
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
	for _, field := range []string{"preProcessors", "postProcessors", "transformers"} {
		if _, ok := props[field]; !ok {
			t.Errorf("expected %s in properties", field)
		}
	}
}
