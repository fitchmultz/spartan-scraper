// Package mcp routes MCP tool calls onto Spartan runtime operations.
//
// Purpose:
// - Decode MCP tool call envelopes and dispatch them through typed tool registries.
//
// Responsibilities:
// - Parse tool-call params from JSON-RPC requests.
// - Enforce setup-mode tool restrictions before runtime execution.
// - Reuse shared decoding helpers so domain handlers stay focused on business logic.
//
// Scope:
// - MCP tool dispatch only; individual tool implementations live in focused handler files.
//
// Usage:
// - Called by the MCP server when handling `tools/call` JSON-RPC messages.
//
// Invariants/Assumptions:
// - All tool names are registered exactly once in the active registry.
// - Setup mode only exposes diagnostic tools until recovery is complete.
// - Unknown tool names return validation errors instead of transport failures.
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

func decodeToolArguments(args map[string]interface{}, dst any) error {
	payload, err := json.Marshal(args)
	if err != nil {
		return apperrors.Wrap(apperrors.KindValidation, "invalid tool arguments", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return apperrors.Validation("invalid tool arguments: " + err.Error())
	}
	var extra any
	if err := decoder.Decode(&extra); err != nil {
		if err == io.EOF {
			return nil
		}
		return apperrors.Validation("invalid tool arguments: " + err.Error())
	}
	return apperrors.Validation("invalid tool arguments: expected a single JSON object")
}

func (s *Server) handleToolCall(ctx context.Context, base map[string]json.RawMessage) (interface{}, error) {
	var params callParams
	if raw, ok := base["params"]; ok {
		if err := json.Unmarshal(raw, &params); err != nil {
			return nil, err
		}
	}

	handler, ok := s.activeToolRegistry()[params.Name]
	if !ok {
		return nil, apperrors.Validation(fmt.Sprintf("unknown tool: %s", params.Name))
	}
	return handler(s, ctx, params)
}
