// Package ai manages the bridge process used for pi-backed LLM operations.
//
// Purpose:
// - Evaluate bridge startup health and summarize readiness diagnostics.
//
// Responsibilities:
// - Preserve startup health snapshots on failure, log health summaries,
// - and normalize route-status reporting for operator-facing diagnostics.
//
// Scope:
// - Bridge health helpers only; request execution and process lifecycle live in adjacent files.
//
// Usage:
// - Called during bridge startup and explicit health checks.
//
// Invariants/Assumptions:
// - Startup should only fail when enabled capabilities have no auth-ready routes.
package ai

import (
	"fmt"
	"log/slog"
	"strings"
)

func (e *BridgeHealthError) Error() string {
	return e.Err.Error()
}

func (e *BridgeHealthError) Unwrap() error {
	return e.Err
}

func logBridgeHealth(health HealthResponse) {
	if health.Mode == "" && len(health.Resolved) == 0 && len(health.Available) == 0 && len(health.RouteStatus) == 0 && health.LoadError == "" && len(health.AuthErrors) == 0 {
		return
	}

	degraded := make(map[string]string)
	disabled := make([]string, 0)
	for capability, routes := range health.Resolved {
		resolved := trimHealthRoutes(routes)
		switch {
		case len(resolved) == 0:
			disabled = append(disabled, capability)
		case len(trimHealthRoutes(health.Available[capability])) == 0:
			degraded[capability] = formatRouteStatuses(health.RouteStatus[capability], resolved)
		}
	}

	slog.Info(
		"pi bridge startup health",
		"mode", health.Mode,
		"ready_routes", summarizeRouteCounts(health.Available),
		"configured_routes", summarizeRouteCounts(health.Resolved),
		"degraded_capabilities", degraded,
		"disabled_capabilities", disabled,
	)

	if len(degraded) > 0 {
		slog.Warn("pi bridge capability degradation", "capabilities", degraded)
	}
	if health.LoadError != "" || len(health.AuthErrors) > 0 {
		slog.Warn(
			"pi bridge startup diagnostics",
			"models_error", health.LoadError,
			"auth_errors", health.AuthErrors,
		)
	}
}

func validateBridgeHealth(health HealthResponse) error {
	enabled := 0
	ready := 0
	issues := make([]string, 0)
	for capability, routes := range health.Resolved {
		resolved := trimHealthRoutes(routes)
		if len(resolved) == 0 {
			continue
		}
		enabled++
		available := trimHealthRoutes(health.Available[capability])
		if len(available) > 0 {
			ready++
			continue
		}
		issues = append(issues, fmt.Sprintf("%s: %s", capability, formatRouteStatuses(health.RouteStatus[capability], resolved)))
	}
	if enabled == 0 || ready > 0 {
		return nil
	}

	parts := make([]string, 0, 2+len(health.AuthErrors))
	parts = append(parts, "no auth-ready pi routes available for any enabled capability: "+strings.Join(issues, "; "))
	if health.LoadError != "" {
		parts = append(parts, "models.json: "+health.LoadError)
	}
	for _, authErr := range health.AuthErrors {
		if strings.TrimSpace(authErr) == "" {
			continue
		}
		parts = append(parts, "auth: "+authErr)
	}
	return fmt.Errorf("bridge startup diagnostics: %s", strings.Join(parts, " | "))
}

func trimHealthRoutes(routes []string) []string {
	if routes == nil {
		return nil
	}
	trimmed := make([]string, 0, len(routes))
	for _, route := range routes {
		if value := strings.TrimSpace(route); value != "" {
			trimmed = append(trimmed, value)
		}
	}
	if len(trimmed) == 0 {
		return []string{}
	}
	return trimmed
}

func summarizeRouteCounts(routes map[string][]string) map[string]string {
	if len(routes) == 0 {
		return nil
	}
	counts := make(map[string]string, len(routes))
	for capability, entries := range routes {
		counts[capability] = fmt.Sprintf("%d", len(entries))
	}
	return counts
}

func formatRouteStatuses(statuses []HealthRouteStatus, fallbackRoutes []string) string {
	if len(statuses) == 0 {
		if len(fallbackRoutes) == 0 {
			return "no routes configured"
		}
		return strings.Join(fallbackRoutes, ", ")
	}

	parts := make([]string, 0, len(statuses))
	for _, status := range statuses {
		label := status.RouteID
		if strings.TrimSpace(label) == "" {
			label = "<unknown-route>"
		}
		if strings.TrimSpace(status.Message) != "" {
			parts = append(parts, fmt.Sprintf("%s (%s)", label, status.Message))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s (%s)", label, status.Status))
	}
	return strings.Join(parts, ", ")
}
