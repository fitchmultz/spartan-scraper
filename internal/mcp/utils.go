// MCP server utility functions.
//
// Responsibilities:
// - Poll job store for completion (waitForJob)
// - Extract and type-safe parse tool arguments (getString, getBool, etc.)
// - Resolve authentication profiles for tools
//
// Does NOT handle:
// - Tool execution logic or business operations
// - Server lifecycle management
//
// Invariants:
// - waitForJob has independent timeout timer (not cancellable by caller context)
// - Argument getters return safe defaults for missing/invalid values
// - Type assertions are nil-safe and return defaults on failure
package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
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

func getString(args map[string]interface{}, key string) string {
	if args == nil {
		return ""
	}
	if value, ok := args[key].(string); ok {
		return value
	}
	return ""
}

func getBool(args map[string]interface{}, key string) bool {
	if args == nil {
		return false
	}
	if value, ok := args[key].(bool); ok {
		return value
	}
	return false
}

func getBoolDefault(args map[string]interface{}, key string, fallback bool) bool {
	if args == nil {
		return fallback
	}
	if _, ok := args[key]; !ok {
		return fallback
	}
	if value, ok := args[key].(bool); ok {
		return value
	}
	return fallback
}

func getInt(args map[string]interface{}, key string, fallback int) int {
	if args == nil {
		return fallback
	}
	switch value := args[key].(type) {
	case float64:
		if int(value) <= 0 {
			return fallback
		}
		return int(value)
	case int:
		if value <= 0 {
			return fallback
		}
		return value
	default:
		return fallback
	}
}

func getStringSlice(args map[string]interface{}, key string) []string {
	if args == nil {
		return nil
	}
	values, ok := args[key]
	if !ok {
		return nil
	}
	switch v := values.(type) {
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return v
	default:
		return nil
	}
}

func getPipelineOptions(args map[string]interface{}) pipeline.Options {
	if args == nil {
		return pipeline.Options{}
	}
	return pipeline.Options{
		PreProcessors:  getStringSlice(args, "preProcessors"),
		PostProcessors: getStringSlice(args, "postProcessors"),
		Transformers:   getStringSlice(args, "transformers"),
	}
}
