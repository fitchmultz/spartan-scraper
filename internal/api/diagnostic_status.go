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
	HealthStatus(ctx context.Context) (extract.AIHealthSnapshot, error)
}

// NormalizeDiagnosticTarget normalizes a diagnostic target name for consistent matching.
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

func aiDetails(cfg config.Config, snapshot extract.AIHealthSnapshot) map[string]any {
	details := map[string]any{
		"enabled":      true,
		"mode":         snapshot.Mode,
		"configPath":   strings.TrimSpace(cfg.AI.ConfigPath),
		"capabilities": snapshot.Capabilities,
	}
	if details["mode"] == "" {
		details["mode"] = cfg.AI.Mode
	}
	if snapshot.LoadError != "" {
		details["loadError"] = snapshot.LoadError
	}
	if len(snapshot.AuthErrors) > 0 {
		details["authErrors"] = snapshot.AuthErrors
	}
	return details
}

func addAIDiagnosticRuntimeDetails(details map[string]any, cfg config.Config, nodeBin string, nodePath string) map[string]any {
	if details == nil {
		details = map[string]any{}
	}
	details["nodeBinary"] = nodeBin
	details["bridgeScript"] = strings.TrimSpace(cfg.AI.BridgeScript)
	if strings.TrimSpace(nodePath) != "" {
		details["nodePath"] = nodePath
	}
	return details
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

	configured := extract.BuildConfiguredAIHealth(cfg.AI)
	if configured.Status == "disabled" {
		return ComponentStatus{
			Status:  "disabled",
			Message: configured.Message,
			Details: aiDetails(cfg, configured),
		}
	}

	if aiExtractor == nil {
		return ComponentStatus{
			Status:  "degraded",
			Message: "AI failed to initialize; core scraping still works without it.",
			Details: aiDetails(cfg, configured),
			Actions: aiRecoveryActions(cfg),
		}
	}

	snapshot, err := aiExtractor.HealthStatus(ctx)
	if err != nil {
		if len(snapshot.Capabilities) == 0 {
			snapshot = configured
		}
		return ComponentStatus{
			Status:  "degraded",
			Message: err.Error(),
			Details: aiDetails(cfg, snapshot),
			Actions: aiRecoveryActions(cfg),
		}
	}

	status := ComponentStatus{
		Status:  snapshot.Status,
		Message: snapshot.Message,
		Details: aiDetails(cfg, snapshot),
	}
	if snapshot.Status == "degraded" {
		status.Actions = aiRecoveryActions(cfg)
	}
	return status
}

func BuildProxyPoolComponentStatus(cfg config.Config, runtimeState ProxyPoolRuntimeState) ComponentStatus {
	path := strings.TrimSpace(cfg.ProxyPoolFile)
	if path == "" {
		return ComponentStatus{
			Status:  "disabled",
			Message: "Proxy pooling is currently off. Spartan does not need a proxy pool for normal operation.",
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

	configured := extract.BuildConfiguredAIHealth(cfg.AI)
	if configured.Status == "disabled" {
		return DiagnosticActionResponse{
			Status:  "disabled",
			Title:   "All AI capabilities are disabled",
			Message: configured.Message,
			Details: aiDetails(cfg, configured),
		}
	}

	nodeBin := strings.TrimSpace(cfg.AI.NodeBin)
	if nodeBin == "" {
		nodeBin = "node"
	}

	nodePath := ""
	details := addAIDiagnosticRuntimeDetails(aiDetails(cfg, configured), cfg, nodeBin, nodePath)
	issues := make([]string, 0, 2)

	resolvedNodePath, err := exec.LookPath(nodeBin)
	if err != nil {
		issues = append(issues, fmt.Sprintf("Node.js binary %q is not available on PATH", nodeBin))
	} else {
		nodePath = resolvedNodePath
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
			Message: "Node.js and bridge prerequisites look reachable, but the AI extractor is not initialized.",
			Details: details,
			Actions: aiRecoveryActions(cfg),
		}
	}

	snapshot, err := aiExtractor.HealthStatus(ctx)
	if err != nil {
		if len(snapshot.Capabilities) == 0 {
			snapshot = configured
		}
		return DiagnosticActionResponse{
			Status:  "degraded",
			Title:   "AI health check failed",
			Message: err.Error(),
			Details: addAIDiagnosticRuntimeDetails(aiDetails(cfg, snapshot), cfg, nodeBin, nodePath),
			Actions: aiRecoveryActions(cfg),
		}
	}

	title := "AI helpers are ready"
	switch snapshot.Status {
	case "degraded":
		title = "AI helpers are partially available"
	case "disabled":
		title = "All AI capabilities are disabled"
	}

	response := DiagnosticActionResponse{
		Status:  snapshot.Status,
		Title:   title,
		Message: snapshot.Message,
		Details: addAIDiagnosticRuntimeDetails(aiDetails(cfg, snapshot), cfg, nodeBin, nodePath),
	}
	if snapshot.Status == "degraded" {
		response.Actions = aiRecoveryActions(cfg)
	}
	return response
}

func BuildProxyPoolDiagnosticResponse(cfg config.Config, runtimeState ProxyPoolRuntimeState) DiagnosticActionResponse {
	path := strings.TrimSpace(cfg.ProxyPoolFile)
	if path == "" {
		return DiagnosticActionResponse{
			Status:  "disabled",
			Title:   "Proxy pool is off",
			Message: "No proxy-pool file is configured. Spartan does not need pooled proxy routing unless you explicitly opt into it.",
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
