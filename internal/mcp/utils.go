// MCP server utility functions.
//
// Responsibilities:
// - Poll job store for completion (waitForJob)
// - Decode tool arguments with the shared persisted-parameter semantics.
// - Resolve authentication profiles for tools
//
// Does NOT handle:
// - Tool execution logic or business operations
// - Server lifecycle management
//
// Invariants:
// - waitForJob has independent timeout timer (not cancellable by caller context)
// - Missing or invalid tool arguments use explicit defaults.
package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/paramdecode"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

func waitForJob(ctx context.Context, store jobStore, id string, timeoutSeconds int) error {
	pollInterval := 200 * time.Millisecond
	timer := time.NewTimer(pollInterval)
	defer timer.Stop()

	var timeoutTimer <-chan time.Time
	if timeoutSeconds > 0 {
		timeoutDuration := time.Duration(timeoutSeconds) * time.Second
		timeoutTimer = time.After(timeoutDuration)
	}

	for {
		job, err := store.Get(ctx, id)
		if err != nil {
			return err
		}
		switch job.Status {
		case model.StatusSucceeded:
			return nil
		case model.StatusFailed:
			if job.Error != "" {
				return fmt.Errorf("job failed: %s", job.Error)
			}
			return apperrors.Internal("job failed")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			timer.Reset(pollInterval)
		case <-timeoutTimer:
			return apperrors.Internal(fmt.Sprintf("job timed out after %d seconds", timeoutSeconds))
		}
	}
}

func getPipelineOptions(args map[string]interface{}) pipeline.Options {
	return pipeline.Options{
		PreProcessors:  paramdecode.StringSlice(args, "preProcessors"),
		PostProcessors: paramdecode.StringSlice(args, "postProcessors"),
		Transformers:   paramdecode.StringSlice(args, "transformers"),
	}
}
