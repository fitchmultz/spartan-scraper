// Package api provides operator-facing recovery actions for degraded optional subsystems.
//
// Purpose:
// - Build recommended actions for browser, AI, and proxy-pool subsystems when diagnostics detect degradation.
//
// Responsibilities:
// - Generate platform-specific browser install actions.
// - Generate AI prerequisite verification and installation actions.
// - Generate proxy-pool configuration and recovery actions.
//
// Scope:
// - Recovery action generation only; component status builders live in diagnostic_status.go.
//
// Usage:
// - Called by BuildBrowserComponentStatus, BuildAIComponentStatus, and BuildProxyPoolComponentStatus.
//
// Invariants/Assumptions:
// - Actions are read-only and deterministic across surfaces (API, CLI, MCP).
package api

import (
	"fmt"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func browserRecoveryActions(platform string, usePlaywright bool) []RecommendedAction {
	actions := []RecommendedAction{{
		Label: "Re-check browser tooling",
		Kind:  ActionKindOneClick,
		Value: DiagnosticActionPath(DiagnosticTargetBrowser),
	}}

	switch platform {
	case "darwin":
		actions = append(actions, RecommendedAction{
			Label: "Install Chrome on macOS",
			Kind:  ActionKindCopy,
			Value: "brew install --cask google-chrome",
		})
	case "windows":
		actions = append(actions, RecommendedAction{
			Label: "Install Chrome on Windows",
			Kind:  ActionKindCopy,
			Value: "winget install Google.Chrome",
		})
	default:
		actions = append(actions,
			RecommendedAction{
				Label: "Install Chromium on Ubuntu/Debian",
				Kind:  ActionKindCopy,
				Value: "sudo apt-get install chromium-browser",
			},
			RecommendedAction{
				Label: "Install Chromium on Fedora",
				Kind:  ActionKindCopy,
				Value: "sudo dnf install chromium",
			},
			RecommendedAction{
				Label: "Install Chromium on Arch",
				Kind:  ActionKindCopy,
				Value: "sudo pacman -S chromium",
			},
		)
	}

	if usePlaywright {
		actions = append(actions,
			RecommendedAction{
				Label: "Install Playwright drivers",
				Kind:  ActionKindCopy,
				Value: "go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5700.1 install --with-deps",
			},
			RecommendedAction{
				Label: "Playwright setup guide",
				Kind:  ActionKindExternalLink,
				Value: "https://playwright.dev/docs/intro",
			},
		)
	}

	return append(actions, RecommendedAction{
		Label: "Browser install guide",
		Kind:  ActionKindExternalLink,
		Value: "https://playwright.dev/docs/browsers",
	})
}

func aiRecoveryActions(cfg config.Config) []RecommendedAction {
	nodeBin := strings.TrimSpace(cfg.AI.NodeBin)
	if nodeBin == "" {
		nodeBin = "node"
	}

	actions := []RecommendedAction{
		{
			Label: "Re-check AI prerequisites",
			Kind:  ActionKindOneClick,
			Value: DiagnosticActionPath(DiagnosticTargetAI),
		},
		{
			Label: "Verify Node.js is available",
			Kind:  ActionKindCopy,
			Value: fmt.Sprintf("%s --version", nodeBin),
		},
		{
			Label: "Install bridge dependencies",
			Kind:  ActionKindCopy,
			Value: "pnpm --dir tools/pi-bridge install",
		},
		{
			Label: "Build the bridge bundle",
			Kind:  ActionKindCopy,
			Value: "pnpm --dir tools/pi-bridge build",
		},
		{
			Label: "Install Node.js",
			Kind:  ActionKindExternalLink,
			Value: "https://nodejs.org/en/download",
		},
	}

	if bridgeScript := strings.TrimSpace(cfg.AI.BridgeScript); bridgeScript != "" {
		actions = append(actions, RecommendedAction{
			Label: "Inspect configured bridge script",
			Kind:  ActionKindCopy,
			Value: fmt.Sprintf("ls -l %q", bridgeScript),
		})
	}
	if configPath := strings.TrimSpace(cfg.AI.ConfigPath); configPath != "" {
		actions = append(actions, RecommendedAction{
			Label: "Inspect AI route overrides",
			Kind:  ActionKindCopy,
			Value: fmt.Sprintf("ls -l %q", configPath),
		})
	}

	return actions
}

func proxyPoolEnableActions() []RecommendedAction {
	return []RecommendedAction{
		{
			Label: "Set PROXY_POOL_FILE when you need pooled routing",
			Kind:  ActionKindEnv,
			Value: "PROXY_POOL_FILE=/absolute/path/to/proxy-pool.json",
		},
		{
			Label: "Re-check proxy pool configuration",
			Kind:  ActionKindOneClick,
			Value: DiagnosticActionPath(DiagnosticTargetProxyPool),
		},
	}
}

func proxyPoolRecoveryActions(proxyPoolFile string) []RecommendedAction {
	actions := []RecommendedAction{
		{
			Label: "Re-check proxy pool configuration",
			Kind:  ActionKindOneClick,
			Value: DiagnosticActionPath(DiagnosticTargetProxyPool),
		},
		{
			Label: "Disable proxy pool intentionally",
			Kind:  ActionKindEnv,
			Value: "PROXY_POOL_FILE=",
		},
	}

	if path := strings.TrimSpace(proxyPoolFile); path != "" {
		actions = append(actions,
			RecommendedAction{
				Label: "Inspect configured proxy pool file",
				Kind:  ActionKindCopy,
				Value: fmt.Sprintf("ls -l %q", path),
			},
			RecommendedAction{
				Label: "Expected proxy pool file",
				Kind:  ActionKindCopy,
				Value: path,
			},
		)
	}

	return actions
}

// DiagnosticActionPath returns the API path for a one-click diagnostic action.
func DiagnosticActionPath(target string) string {
	switch normalizeDiagnosticTarget(target) {
	case DiagnosticTargetBrowser:
		return "/v1/diagnostics/browser-check"
	case DiagnosticTargetAI:
		return "/v1/diagnostics/ai-check"
	case DiagnosticTargetProxyPool:
		return "/v1/diagnostics/proxy-pool-check"
	default:
		return ""
	}
}
