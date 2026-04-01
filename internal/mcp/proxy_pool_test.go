// Package mcp provides mcp functionality for Spartan Scraper.
//
// Purpose:
// - Verify proxy pool test behavior for package mcp.
//
// Responsibilities:
// - Define focused Go test coverage, fixtures, and assertions for the package behavior exercised here.
//
// Scope:
// - Automated test coverage only; production behavior stays in non-test package files.
//
// Usage:
// - Run with `go test` for package `mcp` or through `make test-ci`/`make ci`.
//
// Invariants/Assumptions:
// - Tests should remain deterministic and describe the package contract they protect.

package mcp

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

func TestProxyPoolStatusTool_ReturnsEmptyStatusWhenDisabled(t *testing.T) {
	srv, tmpDir := testServer()
	defer func() {
		srv.Close()
		_ = os.RemoveAll(tmpDir)
	}()

	result, err := srv.handleToolCall(context.Background(), map[string]json.RawMessage{
		"params": mustMarshalJSON(callParams{Name: "proxy_pool_status", Arguments: map[string]interface{}{}}),
	})
	if err != nil {
		t.Fatalf("proxy_pool_status failed: %v", err)
	}

	status, ok := result.(api.ProxyPoolStatusResponse)
	if !ok {
		t.Fatalf("unexpected response type %T", result)
	}
	if status.Strategy != "none" || status.TotalProxies != 0 || status.HealthyProxies != 0 || len(status.Proxies) != 0 {
		t.Fatalf("unexpected empty proxy-pool status: %#v", status)
	}
}

func TestProxyPoolStatusTool_ReturnsLoadedPoolStats(t *testing.T) {
	srv, tmpDir := testServer()
	defer func() {
		srv.Close()
		_ = os.RemoveAll(tmpDir)
	}()

	pool, err := fetch.NewProxyPool(fetch.ProxyPoolConfig{
		DefaultStrategy: "round_robin",
		HealthCheck:     fetch.HealthCheckConfig{Enabled: false},
		Proxies:         []fetch.ProxyEntry{{ID: "proxy-1", URL: "http://127.0.0.1:8080", Region: "us-east", Tags: []string{"residential"}}},
	})
	if err != nil {
		t.Fatalf("NewProxyPool() failed: %v", err)
	}
	defer pool.Stop()
	srv.manager.SetProxyPool(pool)
	pool.RecordSuccess("proxy-1", 90)

	result, err := srv.handleToolCall(context.Background(), map[string]json.RawMessage{
		"params": mustMarshalJSON(callParams{Name: "proxy_pool_status", Arguments: map[string]interface{}{}}),
	})
	if err != nil {
		t.Fatalf("proxy_pool_status failed: %v", err)
	}

	status, ok := result.(api.ProxyPoolStatusResponse)
	if !ok {
		t.Fatalf("unexpected response type %T", result)
	}
	if status.Strategy != "round_robin" || status.TotalProxies != 1 || status.HealthyProxies != 1 || len(status.Proxies) != 1 {
		t.Fatalf("unexpected proxy-pool summary: %#v", status)
	}
	if len(status.Regions) != 1 || status.Regions[0] != "us-east" {
		t.Fatalf("unexpected regions: %#v", status.Regions)
	}
	if len(status.Tags) != 1 || status.Tags[0] != "residential" {
		t.Fatalf("unexpected tags: %#v", status.Tags)
	}
	if status.Proxies[0].ID != "proxy-1" || status.Proxies[0].SuccessCount != 1 || status.Proxies[0].AvgLatencyMs != 90 {
		t.Fatalf("unexpected proxy status: %#v", status.Proxies[0])
	}
}
