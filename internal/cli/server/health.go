// Package server contains health CLI command wiring.
//
// Purpose:
// - Provide the `spartan health` command with both API-backed and local fallback diagnostics.
//
// Responsibilities:
// - Prefer the running server's structured `/healthz` payload when available.
// - Fall back to local store/browser checks when the server is offline.
// - Surface setup-mode guidance consistently with server startup preflight.
//
// Scope:
// - CLI health command behavior only; the HTTP endpoint lives in `internal/api`.
//
// Usage:
// - Invoked from `spartan health`.
//
// Invariants/Assumptions:
// - Non-ok health payloads should return a non-zero exit code.
// - Local fallback checks must never hide setup-mode requirements.
package server

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/buildinfo"
	"github.com/fitchmultz/spartan-scraper/internal/cli/common"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func RunHealth(ctx context.Context, cfg config.Config, args []string) int {
	fs := flag.NewFlagSet("health", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOutput := fs.Bool("json", false, "Print raw JSON instead of human-readable diagnostics")
	checkComponent := fs.String("check", "", "Run a read-only diagnostic check for browser, ai, or proxy_pool")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, `Usage:
  spartan health [--json]
  spartan health --check <browser|ai|proxy_pool> [--json]

Examples:
  spartan health
  spartan health --json
  spartan health --check browser
  spartan health --check ai --json

Exit codes:
  0  health is ok, or the requested subsystem is intentionally disabled
  1  setup is required, or a degraded/error state needs operator action
  2  invalid arguments
`)
	}

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() > 0 {
		fs.Usage()
		return 2
	}

	commandName := currentCommandName()
	if target := api.NormalizeDiagnosticTarget(*checkComponent); target != "" || strings.TrimSpace(*checkComponent) != "" {
		if target == "" {
			fmt.Fprintln(os.Stderr, "invalid --check value: use browser, ai, or proxy_pool")
			return 2
		}
		response, source, err := runDiagnosticCheck(ctx, cfg, target)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if *jsonOutput {
			printJSON(response)
		} else {
			fmt.Print(renderDiagnosticResponse(target, response, commandName, source))
		}
		if response.Status == "ok" || response.Status == "disabled" {
			return 0
		}
		return 1
	}

	health, source, err := loadHealthSnapshot(ctx, cfg, commandName)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if *jsonOutput {
		printJSON(health)
	} else {
		fmt.Print(renderHealthResponse(health, commandName, source))
	}
	if health.Status == "ok" {
		return 0
	}
	return 1
}

func runDiagnosticCheck(ctx context.Context, cfg config.Config, target string) (api.DiagnosticActionResponse, string, error) {
	url := fmt.Sprintf("http://localhost:%s%s", cfg.Port, api.DiagnosticActionPath(target))
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		var diagnostic api.DiagnosticActionResponse
		if err := json.NewDecoder(resp.Body).Decode(&diagnostic); err == nil {
			return diagnostic, "runtime", nil
		}
	}

	setupStatus, err := inspectStartupPreflight(cfg, currentCommandName())
	if err != nil {
		return api.DiagnosticActionResponse{}, "", err
	}
	aiExtractor := buildLocalAIExtractor(cfg)

	switch target {
	case api.DiagnosticTargetBrowser:
		return api.BuildBrowserDiagnosticResponse(cfg), "local", nil
	case api.DiagnosticTargetAI:
		return api.BuildAIDiagnosticResponse(ctx, cfg, aiExtractor), "local", nil
	case api.DiagnosticTargetProxyPool:
		state := api.ProxyPoolRuntimeUnavailable
		if setupStatus != nil {
			state = api.ProxyPoolRuntimeSetupMode
		}
		return api.BuildProxyPoolDiagnosticResponse(cfg, state), "local", nil
	default:
		return api.DiagnosticActionResponse{}, "", fmt.Errorf("unsupported diagnostic target %q", target)
	}
}

func loadHealthSnapshot(ctx context.Context, cfg config.Config, commandName string) (api.HealthResponse, string, error) {
	url := fmt.Sprintf("http://localhost:%s/healthz", cfg.Port)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		var health api.HealthResponse
		if err := json.NewDecoder(resp.Body).Decode(&health); err == nil {
			return health, "runtime", nil
		}
	}
	return buildLocalHealthResponse(ctx, cfg, commandName)
}

func buildLocalHealthResponse(ctx context.Context, cfg config.Config, commandName string) (api.HealthResponse, string, error) {
	setupStatus, err := inspectStartupPreflight(cfg, commandName)
	if err != nil {
		return api.HealthResponse{}, "", err
	}

	res := api.HealthResponse{
		Status:     "ok",
		Version:    buildinfo.Version,
		Components: make(map[string]api.ComponentStatus),
	}

	if setupStatus != nil {
		res.Status = "setup_required"
		res.Setup = setupStatus
		res.Components["database"] = api.ComponentStatus{
			Status:  "setup_required",
			Message: setupStatus.Message,
			Details: map[string]any{
				"dataDir":       setupStatus.DataDir,
				"schemaVersion": setupStatus.SchemaVersion,
			},
			Actions: setupStatus.Actions,
		}
		res.Components["queue"] = api.ComponentStatus{
			Status:  "setup_required",
			Message: "Job processing stays unavailable until setup is completed.",
		}
		res.Components[api.DiagnosticTargetBrowser] = api.BuildBrowserComponentStatus(cfg)
		res.Components[api.DiagnosticTargetAI] = api.BuildAIComponentStatus(ctx, cfg, buildLocalAIExtractor(cfg))
		res.Components[api.DiagnosticTargetProxyPool] = api.BuildProxyPoolComponentStatus(cfg, api.ProxyPoolRuntimeSetupMode)
		res.Notices = append(res.Notices,
			api.RuntimeNotice{
				ID:       setupStatus.Code,
				Scope:    "setup",
				Severity: "error",
				Title:    setupStatus.Title,
				Message:  setupStatus.Message,
				Actions:  setupStatus.Actions,
			},
			api.RuntimeNotice{
				ID:       "server_offline_setup_mode",
				Scope:    "runtime",
				Severity: "warning",
				Title:    "Server is not running",
				Message:  "Showing local setup diagnostics because /healthz is unavailable.",
				Actions:  buildOfflineServerActions(commandName),
			},
		)
		res.Notices = append(res.Notices, api.BuildConfigRuntimeNotices(cfg.StartupNotices)...)
		return res, "local", nil
	}

	hasError := false
	hasDegraded := false
	st, openErr := store.Open(cfg.DataDir)
	if openErr != nil {
		res.Components["database"] = api.ComponentStatus{
			Status:  "error",
			Message: openErr.Error(),
		}
		hasError = true
	} else {
		defer st.Close()
		dbStatus := api.ComponentStatus{Status: "ok", Message: "Local data store is reachable."}
		if err := st.Ping(ctx); err != nil {
			dbStatus.Status = "error"
			dbStatus.Message = err.Error()
			hasError = true
		}
		res.Components["database"] = dbStatus
	}

	queueStatus := buildOfflineQueueComponentStatus(commandName)
	res.Components["queue"] = queueStatus
	hasDegraded = true

	browserStatus := api.BuildBrowserComponentStatus(cfg)
	res.Components[api.DiagnosticTargetBrowser] = browserStatus
	if browserStatus.Status == "degraded" {
		hasDegraded = true
	}

	aiStatus := api.BuildAIComponentStatus(ctx, cfg, buildLocalAIExtractor(cfg))
	res.Components[api.DiagnosticTargetAI] = aiStatus
	if aiStatus.Status == "degraded" {
		hasDegraded = true
	}

	proxyStatus := api.BuildProxyPoolComponentStatus(cfg, api.ProxyPoolRuntimeUnavailable)
	res.Components[api.DiagnosticTargetProxyPool] = proxyStatus
	if proxyStatus.Status == "degraded" {
		hasDegraded = true
	}

	res.Notices = append(res.Notices,
		api.RuntimeNotice{
			ID:       "server_offline",
			Scope:    "runtime",
			Severity: "warning",
			Title:    "Server is not running",
			Message:  "Showing local prerequisite checks because /healthz is unavailable.",
			Actions:  buildOfflineServerActions(commandName),
		},
	)
	res.Notices = append(res.Notices, api.BuildConfigRuntimeNotices(cfg.StartupNotices)...)

	switch {
	case hasError:
		res.Status = "error"
	case hasDegraded || len(res.Notices) > 0:
		res.Status = "degraded"
	default:
		res.Status = "ok"
	}

	return res, "local", nil
}

func buildLocalAIExtractor(cfg config.Config) api.AIHealthChecker {
	if !extract.IsAIEnabled(cfg.AI) {
		return nil
	}
	aiExtractor, err := extract.NewAIExtractor(cfg.AI)
	if err != nil {
		return nil
	}
	return aiExtractor
}

func buildOfflineQueueComponentStatus(commandName string) api.ComponentStatus {
	return api.ComponentStatus{
		Status:  "degraded",
		Message: "Job processing is unavailable until the local Spartan server is running.",
		Actions: buildOfflineServerActions(commandName),
	}
}

func buildOfflineServerActions(commandName string) []api.RecommendedAction {
	startCommand := fmt.Sprintf("%s server", commandName)
	return []api.RecommendedAction{
		{
			Label: "Start the Spartan runtime",
			Kind:  api.ActionKindCommand,
			Value: startCommand,
		},
		{
			Label: "Copy server start command",
			Kind:  api.ActionKindCopy,
			Value: startCommand,
		},
	}
}

func renderHealthResponse(health api.HealthResponse, commandName string, source string) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Health: %s\n", renderStatusLabel(health.Status)))
	builder.WriteString(fmt.Sprintf("Version: %s\n", health.Version))
	if source == "local" {
		builder.WriteString("Source: local fallback (server not responding)\n")
	}
	builder.WriteString("\n")

	if health.Setup != nil && health.Setup.Required {
		builder.WriteString("Setup required\n")
		builder.WriteString(fmt.Sprintf("  %s\n", health.Setup.Title))
		if message := strings.TrimSpace(health.Setup.Message); message != "" {
			builder.WriteString(fmt.Sprintf("  %s\n", message))
		}
		common.WriteRecommendedActions(&builder, "  ", health.Setup.Actions, commandName)
		builder.WriteString("\n")
	}

	builder.WriteString("Components\n")
	for _, key := range orderedComponentKeys(health.Components) {
		component := health.Components[key]
		builder.WriteString(fmt.Sprintf("  %s %s — %s\n", renderComponentMarker(component.Status), renderComponentName(key), renderStatusLabel(component.Status)))
		if message := strings.TrimSpace(component.Message); message != "" {
			builder.WriteString(fmt.Sprintf("    %s\n", message))
		}
		common.WriteRecommendedActions(&builder, "    ", component.Actions, commandName)
	}

	if len(health.Notices) > 0 {
		builder.WriteString("\nNotices\n")
		for _, notice := range health.Notices {
			builder.WriteString(fmt.Sprintf("  - [%s/%s] %s\n", strings.ToUpper(strings.TrimSpace(notice.Scope)), strings.ToUpper(strings.TrimSpace(notice.Severity)), notice.Title))
			if message := strings.TrimSpace(notice.Message); message != "" {
				builder.WriteString(fmt.Sprintf("    %s\n", message))
			}
			common.WriteRecommendedActions(&builder, "    ", notice.Actions, commandName)
		}
	}

	return builder.String()
}

func renderDiagnosticResponse(target string, response api.DiagnosticActionResponse, commandName string, source string) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Diagnostic: %s (%s)\n", renderComponentName(target), renderStatusLabel(response.Status)))
	if source == "local" {
		builder.WriteString("Source: local fallback (server not responding)\n")
	}
	if title := strings.TrimSpace(response.Title); title != "" {
		builder.WriteString(fmt.Sprintf("%s\n", title))
	}
	if message := strings.TrimSpace(response.Message); message != "" {
		builder.WriteString(fmt.Sprintf("%s\n", message))
	}
	common.WriteRecommendedActions(&builder, "  ", response.Actions, commandName)
	return builder.String()
}

func orderedComponentKeys(components map[string]api.ComponentStatus) []string {
	preferred := []string{"database", "queue", api.DiagnosticTargetBrowser, api.DiagnosticTargetAI, api.DiagnosticTargetProxyPool}
	keys := make([]string, 0, len(components))
	seen := make(map[string]bool, len(components))
	for _, key := range preferred {
		if _, ok := components[key]; ok {
			keys = append(keys, key)
			seen[key] = true
		}
	}
	remaining := make([]string, 0, len(components))
	for key := range components {
		if !seen[key] {
			remaining = append(remaining, key)
		}
	}
	sort.Strings(remaining)
	return append(keys, remaining...)
}

func renderComponentName(key string) string {
	switch key {
	case "database":
		return "Database"
	case "queue":
		return "Queue"
	case api.DiagnosticTargetBrowser:
		return "Browser"
	case api.DiagnosticTargetAI:
		return "AI"
	case api.DiagnosticTargetProxyPool:
		return "Proxy pool"
	default:
		words := strings.Fields(strings.ReplaceAll(key, "_", " "))
		for i, word := range words {
			if word == "" {
				continue
			}
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
		return strings.Join(words, " ")
	}
}

func renderComponentMarker(status string) string {
	switch status {
	case "ok":
		return "✓"
	case "disabled":
		return "○"
	case "setup_required":
		return "!"
	default:
		return "!"
	}
}

func renderStatusLabel(status string) string {
	switch strings.TrimSpace(status) {
	case "ok":
		return "OK"
	case "disabled":
		return "DISABLED"
	case "setup_required":
		return "SETUP REQUIRED"
	case "degraded":
		return "DEGRADED"
	case "error":
		return "ERROR"
	default:
		return strings.ToUpper(strings.TrimSpace(status))
	}
}

func printJSON(v any) {
	payload, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(payload))
}
