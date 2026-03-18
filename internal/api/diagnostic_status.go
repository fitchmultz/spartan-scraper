// Package api centralizes shared health and recovery diagnostics.
//
// Purpose:
// - Build canonical component health, diagnostic responses, and operator actions shared by REST, CLI, and MCP.
//
// Responsibilities:
// - Classify browser, AI, and proxy-pool readiness with consistent operator-facing language.
// - Build read-only diagnostic responses and recovery actions for degraded optional subsystems.
// - Translate one-click recovery actions into surface-appropriate commands for CLI and MCP consumers.
//
// Scope:
// - Shared diagnostics metadata and message generation only; HTTP handlers and CLI rendering live elsewhere.
//
// Usage:
// - Used by `/healthz`, `/v1/diagnostics/*`, `spartan health`, and MCP diagnostics tools.
//
// Invariants/Assumptions:
// - Optional subsystems should degrade without blocking core scraping workflows.
// - One-click diagnostic actions remain read-only and deterministic across surfaces.
package api

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

const (
	DiagnosticTargetBrowser   = "browser"
	DiagnosticTargetAI        = "ai"
	DiagnosticTargetProxyPool = "proxy_pool"
)

type ProxyPoolRuntimeState string

const (
	ProxyPoolRuntimeLoaded      ProxyPoolRuntimeState = "loaded"
	ProxyPoolRuntimeUnloaded    ProxyPoolRuntimeState = "unloaded"
	ProxyPoolRuntimeUnavailable ProxyPoolRuntimeState = "unavailable"
	ProxyPoolRuntimeSetupMode   ProxyPoolRuntimeState = "setup_mode"
)

// AIHealthChecker represents the minimal AI extractor health contract shared across surfaces.
type AIHealthChecker interface {
	HealthCheck(ctx context.Context) error
}

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

func NormalizeDiagnosticTarget(target string) string {
	return normalizeDiagnosticTarget(target)
}

func normalizeDiagnosticTarget(target string) string {
	normalized := strings.TrimSpace(strings.ToLower(target))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	switch normalized {
	case DiagnosticTargetBrowser:
		return DiagnosticTargetBrowser
	case DiagnosticTargetAI:
		return DiagnosticTargetAI
	case DiagnosticTargetProxyPool, "proxypool":
		return DiagnosticTargetProxyPool
	default:
		return ""
	}
}

func DiagnosticTargetFromActionValue(value string) string {
	switch strings.TrimSpace(value) {
	case "/v1/diagnostics/browser-check":
		return DiagnosticTargetBrowser
	case "/v1/diagnostics/ai-check":
		return DiagnosticTargetAI
	case "/v1/diagnostics/proxy-pool-check":
		return DiagnosticTargetProxyPool
	default:
		return ""
	}
}

func CLIRecommendedActions(actions []RecommendedAction, commandName string) []RecommendedAction {
	translated := make([]RecommendedAction, 0, len(actions))
	for _, action := range actions {
		translated = append(translated, translateCLIAction(action, commandName))
	}
	return translated
}

func translateCLIAction(action RecommendedAction, commandName string) RecommendedAction {
	translated := action
	if translated.Kind != ActionKindOneClick {
		return translated
	}
	target := DiagnosticTargetFromActionValue(translated.Value)
	if target == "" {
		return translated
	}
	translated.Kind = ActionKindCommand
	translated.Value = fmt.Sprintf("%s health --check %s", strings.TrimSpace(commandName), target)
	return translated
}

func MCPRecommendedActions(actions []RecommendedAction) []RecommendedAction {
	translated := make([]RecommendedAction, 0, len(actions))
	for _, action := range actions {
		translated = append(translated, translateMCPAction(action))
	}
	return translated
}

func translateMCPAction(action RecommendedAction) RecommendedAction {
	translated := action
	if translated.Kind != ActionKindOneClick {
		return translated
	}
	target := DiagnosticTargetFromActionValue(translated.Value)
	if target == "" {
		return translated
	}
	translated.Kind = ActionKindCommand
	translated.Value = fmt.Sprintf("diagnostic_check component=%s", target)
	return translated
}

func BuildConfigRuntimeNotices(in []config.StartupNotice) []RuntimeNotice {
	out := make([]RuntimeNotice, 0, len(in))
	for _, notice := range in {
		item := RuntimeNotice{
			ID:       notice.ID,
			Scope:    "config",
			Severity: notice.Severity,
			Title:    notice.Title,
			Message:  notice.Message,
		}
		if notice.Action != "" {
			item.Actions = []RecommendedAction{{
				Label: "Resolve configuration issue",
				Kind:  ActionKindDoc,
				Value: notice.Action,
			}}
		}
		out = append(out, item)
	}
	return out
}

func BuildBrowserComponentStatus(cfg config.Config) ComponentStatus {
	details := map[string]any{
		"platform":          runtime.GOOS,
		"playwrightEnabled": cfg.UsePlaywright,
	}

	if err := fetch.CheckBrowserAvailability(false); err != nil {
		status := ComponentStatus{
			Status:  "degraded",
			Message: fmt.Sprintf("Chrome/Chromium browser automation is unavailable: %v. HTTP scraping still works.", err),
			Details: details,
			Actions: browserRecoveryActions(runtime.GOOS, cfg.UsePlaywright),
		}
		if cfg.UsePlaywright {
			status.Message += " Playwright checks may also fail until browser tooling is installed."
		}
		return status
	}

	if chromePath, err := fetch.FindChrome(); err == nil {
		details["chromePath"] = chromePath
	}

	if cfg.UsePlaywright {
		if err := fetch.CheckBrowserAvailability(true); err != nil {
			return ComponentStatus{
				Status:  "degraded",
				Message: fmt.Sprintf("Playwright is enabled but unavailable: %v. Chrome-backed browser automation still works.", err),
				Details: details,
				Actions: browserRecoveryActions(runtime.GOOS, true),
			}
		}
	}

	return ComponentStatus{
		Status:  "ok",
		Message: "Browser automation is ready.",
		Details: details,
	}
}

func BuildAIComponentStatus(ctx context.Context, cfg config.Config, aiExtractor AIHealthChecker) ComponentStatus {
	if !extract.IsAIEnabled(cfg.AI) {
		return ComponentStatus{
			Status:  "disabled",
			Message: "AI helpers are optional and currently disabled.",
			Details: map[string]any{
				"enabled": false,
			},
		}
	}

	if aiExtractor == nil {
		return ComponentStatus{
			Status:  "degraded",
			Message: "AI failed to initialize; core scraping still works without it.",
			Details: map[string]any{
				"enabled": true,
				"mode":    cfg.AI.Mode,
			},
			Actions: aiRecoveryActions(cfg),
		}
	}

	status := ComponentStatus{
		Status:  "ok",
		Message: "AI helpers are ready.",
		Details: map[string]any{
			"enabled": true,
			"mode":    cfg.AI.Mode,
		},
	}
	if err := aiExtractor.HealthCheck(ctx); err != nil {
		status.Status = "degraded"
		status.Message = err.Error()
		status.Actions = aiRecoveryActions(cfg)
	}
	return status
}

func BuildProxyPoolComponentStatus(cfg config.Config, runtimeState ProxyPoolRuntimeState) ComponentStatus {
	path := strings.TrimSpace(cfg.ProxyPoolFile)
	if path == "" {
		return ComponentStatus{
			Status:  "disabled",
			Message: "Proxy pool is intentionally disabled. This is optional unless you need pooled proxy routing.",
			Actions: proxyPoolEnableActions(),
		}
	}

	info, statErr := os.Stat(path)
	if statErr != nil {
		return ComponentStatus{
			Status:  "degraded",
			Message: fmt.Sprintf("Configured proxy pool file %s is missing or unreadable: %v", path, statErr),
			Details: map[string]any{
				"path": path,
			},
			Actions: proxyPoolRecoveryActions(path),
		}
	}

	details := map[string]any{
		"path":  path,
		"bytes": info.Size(),
	}

	switch runtimeState {
	case ProxyPoolRuntimeLoaded:
		return ComponentStatus{
			Status:  "ok",
			Message: "Proxy pool loaded.",
			Details: details,
		}
	case ProxyPoolRuntimeSetupMode:
		return ComponentStatus{
			Status:  "degraded",
			Message: "Proxy pool configuration is present, but Spartan is still in setup mode. Finish setup, restart, then re-check that the pool loads.",
			Details: details,
			Actions: proxyPoolRecoveryActions(path),
		}
	case ProxyPoolRuntimeUnavailable:
		return ComponentStatus{
			Status:  "degraded",
			Message: "Proxy pool configuration is present, but the Spartan runtime is not running. Start the server, then re-check that the pool loads.",
			Details: details,
			Actions: proxyPoolRecoveryActions(path),
		}
	default:
		return ComponentStatus{
			Status:  "degraded",
			Message: "The proxy pool file exists, but the runtime does not currently have a loaded pool.",
			Details: details,
			Actions: proxyPoolRecoveryActions(path),
		}
	}
}

func BuildBrowserDiagnosticResponse(cfg config.Config) DiagnosticActionResponse {
	details := map[string]any{
		"platform":          runtime.GOOS,
		"playwrightEnabled": cfg.UsePlaywright,
	}

	chromePath, err := fetch.FindChrome()
	if err != nil {
		return DiagnosticActionResponse{
			Status:  "degraded",
			Title:   "Browser tooling is still unavailable",
			Message: fmt.Sprintf("Chrome/Chromium is not available on the server host: %v", err),
			Details: details,
			Actions: browserRecoveryActions(runtime.GOOS, cfg.UsePlaywright),
		}
	}

	details["chromePath"] = chromePath
	if cfg.UsePlaywright {
		if err := fetch.CheckBrowserAvailabilityFresh(true); err != nil {
			return DiagnosticActionResponse{
				Status:  "degraded",
				Title:   "Playwright is still unavailable",
				Message: fmt.Sprintf("Chrome is present at %s, but Playwright is not ready: %v", chromePath, err),
				Details: details,
				Actions: browserRecoveryActions(runtime.GOOS, true),
			}
		}
	}

	return DiagnosticActionResponse{
		Status:  "ok",
		Title:   "Browser automation is ready",
		Message: fmt.Sprintf("Detected browser tooling at %s.", chromePath),
		Details: details,
	}
}

func BuildAIDiagnosticResponse(ctx context.Context, cfg config.Config, aiExtractor AIHealthChecker) DiagnosticActionResponse {
	if !extract.IsAIEnabled(cfg.AI) {
		return DiagnosticActionResponse{
			Status:  "disabled",
			Title:   "AI helpers are disabled",
			Message: "AI is optional and currently disabled in configuration.",
			Details: map[string]any{
				"enabled": false,
				"mode":    cfg.AI.Mode,
			},
		}
	}

	nodeBin := strings.TrimSpace(cfg.AI.NodeBin)
	if nodeBin == "" {
		nodeBin = "node"
	}

	details := map[string]any{
		"enabled":      true,
		"mode":         cfg.AI.Mode,
		"nodeBinary":   nodeBin,
		"bridgeScript": strings.TrimSpace(cfg.AI.BridgeScript),
	}
	issues := make([]string, 0, 2)

	nodePath, err := exec.LookPath(nodeBin)
	if err != nil {
		issues = append(issues, fmt.Sprintf("Node.js binary %q is not available on PATH", nodeBin))
	} else {
		details["nodePath"] = nodePath
	}

	if bridgeScript := strings.TrimSpace(cfg.AI.BridgeScript); bridgeScript != "" {
		if _, err := os.Stat(bridgeScript); err != nil {
			issues = append(issues, fmt.Sprintf("bridge script %q was not found", bridgeScript))
		}
	}

	if len(issues) > 0 {
		return DiagnosticActionResponse{
			Status:  "degraded",
			Title:   "AI prerequisites are incomplete",
			Message: strings.Join(issues, ". "),
			Details: details,
			Actions: aiRecoveryActions(cfg),
		}
	}

	if aiExtractor == nil {
		return DiagnosticActionResponse{
			Status:  "degraded",
			Title:   "AI failed to initialize",
			Message: "Node.js and the configured bridge path look reachable, but the AI extractor is not initialized yet.",
			Details: details,
			Actions: aiRecoveryActions(cfg),
		}
	}

	if err := aiExtractor.HealthCheck(ctx); err != nil {
		return DiagnosticActionResponse{
			Status:  "degraded",
			Title:   "AI health check failed",
			Message: err.Error(),
			Details: details,
			Actions: aiRecoveryActions(cfg),
		}
	}

	return DiagnosticActionResponse{
		Status:  "ok",
		Title:   "AI helpers are ready",
		Message: "AI prerequisites look healthy.",
		Details: details,
	}
}

func BuildProxyPoolDiagnosticResponse(cfg config.Config, runtimeState ProxyPoolRuntimeState) DiagnosticActionResponse {
	path := strings.TrimSpace(cfg.ProxyPoolFile)
	if path == "" {
		return DiagnosticActionResponse{
			Status:  "disabled",
			Title:   "Proxy pool is disabled",
			Message: "PROXY_POOL_FILE is empty, so pooled proxy routing is intentionally disabled.",
			Actions: proxyPoolEnableActions(),
		}
	}

	info, err := os.Stat(path)
	if err != nil {
		return DiagnosticActionResponse{
			Status:  "degraded",
			Title:   "Proxy pool file is missing",
			Message: fmt.Sprintf("Configured proxy pool file %q is missing or unreadable: %v", path, err),
			Details: map[string]any{"path": path},
			Actions: proxyPoolRecoveryActions(path),
		}
	}

	details := map[string]any{
		"path":  path,
		"bytes": info.Size(),
	}

	switch runtimeState {
	case ProxyPoolRuntimeLoaded:
		return DiagnosticActionResponse{
			Status:  "ok",
			Title:   "Proxy pool is loaded",
			Message: "Proxy pool configuration is present and loaded.",
			Details: details,
		}
	case ProxyPoolRuntimeSetupMode:
		return DiagnosticActionResponse{
			Status:  "degraded",
			Title:   "Proxy pool is waiting on setup recovery",
			Message: "The proxy pool file exists, but Spartan is still in setup mode. Finish setup, restart, then re-check that the pool loads.",
			Details: details,
			Actions: proxyPoolRecoveryActions(path),
		}
	case ProxyPoolRuntimeUnavailable:
		return DiagnosticActionResponse{
			Status:  "degraded",
			Title:   "Proxy pool is waiting on the runtime",
			Message: "The proxy pool file exists, but the Spartan runtime is not running yet. Start the server, then re-check that the pool loads.",
			Details: details,
			Actions: proxyPoolRecoveryActions(path),
		}
	default:
		return DiagnosticActionResponse{
			Status:  "degraded",
			Title:   "Proxy pool is not loaded",
			Message: "The proxy pool file exists, but the runtime does not currently have a loaded pool.",
			Details: details,
			Actions: proxyPoolRecoveryActions(path),
		}
	}
}

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

	return actions
}

func proxyPoolEnableActions() []RecommendedAction {
	return []RecommendedAction{
		{
			Label: "Set PROXY_POOL_FILE when you need pooled routing",
			Kind:  ActionKindEnv,
			Value: "PROXY_POOL_FILE=/absolute/path/to/proxy-pool.txt",
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
