// Common test helpers for MCP server tests.
// Provides test server, configuration, and JSON marshaling utilities used across
// all test files in the mcp package. These helpers create isolated test environments
// with temporary directories for storage.
//
// Does NOT handle:
// - Server lifecycle management (defer Close() is caller's responsibility)
// - Cleanup of temporary directories (caller must use defer os.RemoveAll(tmpDir))
// - Test-specific assertions or validation logic
//
// Invariants:
// - testServer() returns a server with a temporary data directory
// - testConfig() returns a config with safe defaults for testing (rate limiting, timeouts, etc.)
// - Temporary directories are created in OS temp space with prefix "mcp-server-test-*"
// - All test configs use concurrency=1 to avoid race conditions in tests
package mcp

import (
	"encoding/json"
	"os"

	"spartan-scraper/internal/config"
)

func testConfig(cfg config.Config) config.Config {
	return config.Config{
		DataDir:            cfg.DataDir,
		UserAgent:          "test-agent",
		RequestTimeoutSecs: 30,
		MaxConcurrency:     1,
		RateLimitQPS:       10,
		RateLimitBurst:     5,
		MaxRetries:         3,
		RetryBaseMs:        100,
		MaxResponseBytes:   10 * 1024 * 1024,
		UsePlaywright:      false,
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
