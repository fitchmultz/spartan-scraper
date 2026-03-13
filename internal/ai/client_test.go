package ai

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func TestResolveBridgeScriptPathFindsExecutableParent(t *testing.T) {
	rootDir := t.TempDir()
	scriptPath := filepath.Join(rootDir, "tools", "pi-bridge", "dist", "main.js")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("mkdir script dir: %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("// test bridge\n"), 0o644); err != nil {
		t.Fatalf("write bridge script: %v", err)
	}

	searchRoots := bridgeScriptSearchRoots(
		"",
		filepath.Join(rootDir, "bin", "spartan"),
		"",
	)

	resolved, err := resolveBridgeScriptPath(config.DefaultPIBridgeScript, searchRoots)
	if err != nil {
		t.Fatalf("resolveBridgeScriptPath() failed: %v", err)
	}
	if resolved != scriptPath {
		t.Fatalf("expected %q, got %q", scriptPath, resolved)
	}
}

func TestResolveBridgeScriptPathPrefersConfigDirectory(t *testing.T) {
	configDir := t.TempDir()
	scriptPath := filepath.Join(configDir, "tools", "pi-bridge", "dist", "main.js")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
		t.Fatalf("mkdir script dir: %v", err)
	}
	if err := os.WriteFile(scriptPath, []byte("// test bridge\n"), 0o644); err != nil {
		t.Fatalf("write bridge script: %v", err)
	}

	searchRoots := bridgeScriptSearchRoots(
		"",
		"",
		filepath.Join(configDir, "routes.json"),
	)

	resolved, err := resolveBridgeScriptPath(config.DefaultPIBridgeScript, searchRoots)
	if err != nil {
		t.Fatalf("resolveBridgeScriptPath() failed: %v", err)
	}
	if resolved != scriptPath {
		t.Fatalf("expected %q, got %q", scriptPath, resolved)
	}
}

func TestClientHealthCheckHonorsStartupTimeout(t *testing.T) {
	nodeBin := requireNode(t)
	scriptPath := writeBridgeScript(t, `
const readline = require("node:readline");
const rl = readline.createInterface({ input: process.stdin, crlfDelay: Infinity });
rl.on("line", () => {
  // Intentionally never respond so startup readiness times out.
});
`)

	client := NewClient(config.AIConfig{
		Enabled:            true,
		NodeBin:            nodeBin,
		BridgeScript:       scriptPath,
		StartupTimeoutSecs: 1,
		RequestTimeoutSecs: 5,
	})
	defer func() { _ = client.Close() }()

	start := time.Now()
	err := client.HealthCheck(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("expected startup timeout to stop quickly, took %v", elapsed)
	}
}

func TestClientExtractHonorsConfiguredRequestTimeout(t *testing.T) {
	nodeBin := requireNode(t)
	scriptPath := writeBridgeScript(t, `
const readline = require("node:readline");
const rl = readline.createInterface({ input: process.stdin, crlfDelay: Infinity });
rl.on("line", (line) => {
  const request = JSON.parse(line);
  if (request.op === "health") {
    process.stdout.write(JSON.stringify({
      id: request.id,
      ok: true,
      result: { mode: "fixture" },
    }) + "\n");
    return;
  }
  if (request.op === "extract_preview") {
    return;
  }
});
`)

	client := NewClient(config.AIConfig{
		Enabled:            true,
		NodeBin:            nodeBin,
		BridgeScript:       scriptPath,
		StartupTimeoutSecs: 1,
		RequestTimeoutSecs: 1,
	})
	defer func() { _ = client.Close() }()

	start := time.Now()
	_, err := client.Extract(context.Background(), ExtractRequest{
		HTML: "<html></html>",
		URL:  "https://example.com",
		Mode: "natural_language",
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("expected request timeout to stop quickly, took %v", elapsed)
	}
}

func TestClientHealthFailsWhenNoAuthReadyRoutesExist(t *testing.T) {
	nodeBin := requireNode(t)
	scriptPath := writeBridgeScript(t, `
const readline = require("node:readline");
const rl = readline.createInterface({ input: process.stdin, crlfDelay: Infinity });
const result = {
  mode: "sdk",
  resolved: {
    "extract.natural_language": ["openai/gpt-5.4", "kimi-coding/k2p5"]
  },
  available: {
    "extract.natural_language": []
  },
  route_status: {
    "extract.natural_language": [
      {
        route_id: "openai/gpt-5.4",
        provider: "openai",
        model: "gpt-5.4",
        status: "missing_auth",
        message: "no auth configured for provider openai",
        model_found: true,
        auth_configured: false
      },
      {
        route_id: "kimi-coding/k2p5",
        provider: "kimi-coding",
        model: "k2p5",
        status: "missing_auth",
        message: "no auth configured for provider kimi-coding",
        model_found: true,
        auth_configured: false
      }
    ]
  }
};
rl.on("line", (line) => {
  const request = JSON.parse(line);
  if (request.op === "health") {
    process.stdout.write(JSON.stringify({ id: request.id, ok: true, result }) + "\n");
  }
});
`)

	client := NewClient(config.AIConfig{
		Enabled:            true,
		NodeBin:            nodeBin,
		BridgeScript:       scriptPath,
		StartupTimeoutSecs: 1,
		RequestTimeoutSecs: 5,
	})
	defer func() { _ = client.Close() }()

	err := client.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("expected startup diagnostics error")
	}
	if !strings.Contains(err.Error(), "no auth-ready pi routes available") {
		t.Fatalf("expected missing auth diagnostic, got %v", err)
	}
	if !strings.Contains(err.Error(), "openai/gpt-5.4") {
		t.Fatalf("expected route diagnostics in error, got %v", err)
	}
}

func requireNode(t *testing.T) string {
	t.Helper()

	nodeBin, err := exec.LookPath(config.DefaultPINodeBin)
	if err != nil {
		t.Skipf("node is required for bridge client tests: %v", err)
	}
	return nodeBin
}

func writeBridgeScript(t *testing.T, contents string) string {
	t.Helper()

	scriptPath := filepath.Join(t.TempDir(), "bridge.js")
	if err := os.WriteFile(scriptPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("write bridge script: %v", err)
	}
	return scriptPath
}
