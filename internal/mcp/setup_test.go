// Package mcp provides common test helpers for MCP server tests.
// Tests cover test server creation, configuration setup, and JSON marshaling utilities.
// Does NOT handle server lifecycle management or test-specific assertions.
package mcp

import (
	"encoding/json"
	"os"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func testConfig(cfg config.Config) config.Config {
	return config.Config{
		DataDir:            cfg.DataDir,
		ProxyPoolFile:      cfg.ProxyPoolFile,
		AI:                 cfg.AI,
		UserAgent:          "test-agent",
		RequestTimeoutSecs: 30,
		MaxConcurrency:     1,
		RateLimitQPS:       10,
		RateLimitBurst:     5,
		MaxRetries:         3,
		RetryBaseMs:        100,
		MaxResponseBytes:   10 * 1024 * 1024,
		UsePlaywright:      cfg.UsePlaywright,
	}
}

func testServer() (*Server, string) {
	tmpDir, err := os.MkdirTemp("", "mcp-server-test-*")
	if err != nil {
		panic(err)
	}

	cfg := testConfig(config.Config{DataDir: tmpDir})
	srv, err := NewServer(cfg)
	if err != nil {
		os.RemoveAll(tmpDir)
		panic(err)
	}

	return srv, tmpDir
}

func mustMarshalJSON(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
