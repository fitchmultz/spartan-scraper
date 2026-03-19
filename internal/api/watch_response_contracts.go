// Package api centralizes watch response envelope construction.
//
// Purpose:
// - Provide one canonical response-shaping path for watch collections and single-watch payloads.
//
// Responsibilities:
// - Normalize watch response arrays for JSON collection envelopes.
// - Wrap paginated watch collections with stable metadata used across REST, Web, and MCP.
// - Reuse the existing watch response mapping so transports stay aligned.
//
// Scope:
// - Watch configuration response construction only; persistence and execution remain elsewhere.
//
// Usage:
// - Called by REST handlers and MCP tool handlers before encoding watch payloads.
//
// Invariants/Assumptions:
// - Collection responses always emit arrays, never null.
// - Watch payloads should match the API-facing watch schema instead of exposing raw storage structs.
package api

import "github.com/fitchmultz/spartan-scraper/internal/watch"

func normalizeWatchResponses(items []WatchResponse) []WatchResponse {
	if items == nil {
		return []WatchResponse{}
	}
	return items
}

// BuildWatchResponse returns the canonical single-watch payload.
func BuildWatchResponse(item watch.Watch) WatchResponse {
	return toWatchResponse(item)
}

// BuildWatchListResponse returns the canonical paginated watch collection envelope.
func BuildWatchListResponse(items []watch.Watch, total, limit, offset int) WatchListResponse {
	responses := make([]WatchResponse, 0, len(items))
	for _, item := range items {
		responses = append(responses, BuildWatchResponse(item))
	}
	return WatchListResponse{
		Watches: normalizeWatchResponses(responses),
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}
}
