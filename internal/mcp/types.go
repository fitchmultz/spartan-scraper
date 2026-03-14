// Package mcp type definitions for the MCP server.
//
// Responsibilities:
// - Define core server types (Server, request, response)
// - Define MCP protocol types (tool, callParams, rpcError)
// - Define minimal interfaces for testing (jobStore)
//
// Does NOT handle:
// - Business logic or server operations
// - Request processing or response generation
//
// Invariants:
// - All types are JSON-serializable for MCP protocol compliance
// - jobStore interface is minimal and test-focused, not for production use
package mcp

import (
	"context"

	"github.com/fitchmultz/spartan-scraper/internal/aiauthoring"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

type Server struct {
	store       *store.Store
	manager     *jobs.Manager
	cfg         config.Config
	aiAuthoring *aiauthoring.Service
	ctx         context.Context
	cancel      context.CancelFunc
}

type request struct {
	ID     interface{}       `json:"id"`
	Method string            `json:"method"`
	Params map[string]string `json:"params"`
}

type response struct {
	ID     interface{} `json:"id"`
	Result interface{} `json:"result,omitempty"`
	Error  *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type callParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type jobStore interface {
	Get(ctx context.Context, id string) (model.Job, error)
}
