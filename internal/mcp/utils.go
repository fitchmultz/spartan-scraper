// Package mcp provides mcp functionality for Spartan Scraper.
//
// Purpose:
// - Implement utils support for package mcp.
//
// Responsibilities:
// - Define the file-local types, functions, and helpers that belong to this package concern.
//
// Scope:
// - Package-internal behavior owned by this file; broader orchestration stays in adjacent package files.
//
// Usage:
// - Used by other files in package `mcp` and any exported callers that depend on this package.
//
// Invariants/Assumptions:
// - This file should preserve the package contract and rely on surrounding package configuration as the source of truth.

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
