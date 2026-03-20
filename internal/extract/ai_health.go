// Package extract models capability-scoped AI health for operator-facing diagnostics.
//
// Purpose:
// - Translate bridge and config state into stable per-capability AI health snapshots.
//
// Responsibilities:
// - Classify capabilities as ok, degraded, or disabled.
// - Preserve configured, available, and per-route status details for shared diagnostics.
// - Summarize overall AI readiness without collapsing partial degradation into total failure.
//
// Scope:
// - Health modeling only; bridge I/O and request execution live in internal/ai.
//
// Usage:
// - Built by AI providers and consumed by API, CLI, and MCP diagnostic surfaces.
//
// Invariants/Assumptions:
// - Explicit empty route lists mean a capability is intentionally disabled.
// - Enabled capabilities degrade when they have configured routes but zero available routes.
package extract

import (
	"fmt"
	"sort"
	"strings"

	piai "github.com/fitchmultz/spartan-scraper/internal/ai"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

type AIRouteHealth struct {
	RouteID        string `json:"routeId"`
	Provider       string `json:"provider,omitempty"`
	Model          string `json:"model,omitempty"`
	Status         string `json:"status"`
	Message        string `json:"message,omitempty"`
	ModelFound     bool   `json:"modelFound"`
	AuthConfigured bool   `json:"authConfigured"`
}

type AICapabilityHealth struct {
	Status           string          `json:"status"`
	Message          string          `json:"message,omitempty"`
	ConfiguredRoutes []string        `json:"configuredRoutes,omitempty"`
	AvailableRoutes  []string        `json:"availableRoutes,omitempty"`
	RouteStatus      []AIRouteHealth `json:"routeStatus,omitempty"`
}

type AIHealthSnapshot struct {
	Status       string                        `json:"status"`
	Mode         string                        `json:"mode,omitempty"`
	Message      string                        `json:"message,omitempty"`
	Capabilities map[string]AICapabilityHealth `json:"capabilities,omitempty"`
	LoadError    string                        `json:"loadError,omitempty"`
	AuthErrors   []string                      `json:"authErrors,omitempty"`
}

func BuildConfiguredAIHealth(cfg config.AIConfig) AIHealthSnapshot {
	resolved := make(map[string][]string, len(config.AllAICapabilities()))
	for _, capability := range config.AllAICapabilities() {
		routes := cfg.Routing.RoutesFor(capability)
		if routes == nil {
			continue
		}
		resolved[capability] = append([]string(nil), routes...)
	}
	return BuildAIHealthSnapshot(cfg, piai.HealthResponse{
		Mode:     cfg.Mode,
		Resolved: resolved,
	})
}

func BuildAIHealthSnapshot(cfg config.AIConfig, health piai.HealthResponse) AIHealthSnapshot {
	snapshot := AIHealthSnapshot{
		Status:       "ok",
		Mode:         firstNonEmpty(strings.TrimSpace(health.Mode), strings.TrimSpace(cfg.Mode)),
		Capabilities: make(map[string]AICapabilityHealth, len(config.AllAICapabilities())),
		LoadError:    strings.TrimSpace(health.LoadError),
		AuthErrors:   compactStrings(health.AuthErrors),
	}

	var ready, degraded, disabled []string
	for _, capability := range config.AllAICapabilities() {
		configured := configuredRoutesForCapability(cfg, health, capability)
		available := trimRoutes(health.Available[capability])
		routeStatus := convertRouteStatuses(health.RouteStatus[capability])

		item := AICapabilityHealth{
			ConfiguredRoutes: configured,
			AvailableRoutes:  available,
			RouteStatus:      routeStatus,
		}

		switch {
		case len(configured) == 0:
			item.Status = "disabled"
			item.Message = "Capability intentionally disabled."
			disabled = append(disabled, capability)
		case len(available) > 0:
			item.Status = "ok"
			item.Message = fmt.Sprintf("%d route(s) ready.", len(available))
			ready = append(ready, capability)
		default:
			item.Status = "degraded"
			item.Message = degradedCapabilityMessage(capability, routeStatus)
			degraded = append(degraded, capability)
		}

		snapshot.Capabilities[capability] = item
	}

	switch {
	case len(ready) == 0 && len(degraded) == 0:
		snapshot.Status = "disabled"
		snapshot.Message = "All AI capabilities are intentionally disabled."
	case len(degraded) > 0:
		snapshot.Status = "degraded"
		snapshot.Message = formatCapabilitySummary("AI helpers are partially available.", ready, degraded, disabled)
	default:
		snapshot.Status = "ok"
		if len(disabled) == 0 {
			snapshot.Message = "AI helpers are ready."
		} else {
			snapshot.Message = formatCapabilitySummary("AI helpers are ready for enabled capabilities.", ready, nil, disabled)
		}
	}

	return snapshot
}

func configuredRoutesForCapability(cfg config.AIConfig, health piai.HealthResponse, capability string) []string {
	if health.Resolved != nil {
		if routes, ok := health.Resolved[capability]; ok {
			return trimRoutes(routes)
		}
	}
	routes := cfg.Routing.RoutesFor(capability)
	if routes == nil {
		return nil
	}
	return append([]string(nil), routes...)
}

func trimRoutes(in []string) []string {
	if in == nil {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, route := range in {
		if trimmed := strings.TrimSpace(route); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return []string{}
	}
	return out
}

func convertRouteStatuses(in []piai.HealthRouteStatus) []AIRouteHealth {
	if len(in) == 0 {
		return nil
	}
	out := make([]AIRouteHealth, 0, len(in))
	for _, item := range in {
		out = append(out, AIRouteHealth{
			RouteID:        item.RouteID,
			Provider:       item.Provider,
			Model:          item.Model,
			Status:         item.Status,
			Message:        item.Message,
			ModelFound:     item.ModelFound,
			AuthConfigured: item.AuthConfigured,
		})
	}
	return out
}

func degradedCapabilityMessage(capability string, statuses []AIRouteHealth) string {
	if len(statuses) == 0 {
		return fmt.Sprintf("No routes are currently available for %s.", capability)
	}
	parts := make([]string, 0, len(statuses))
	for _, status := range statuses {
		label := status.RouteID
		if label == "" {
			label = "<unknown-route>"
		}
		if status.Message != "" {
			parts = append(parts, fmt.Sprintf("%s (%s)", label, status.Message))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s (%s)", label, status.Status))
	}
	return strings.Join(parts, ", ")
}

func formatCapabilitySummary(prefix string, ready, degraded, disabled []string) string {
	chunks := []string{prefix}
	if len(ready) > 0 {
		chunks = append(chunks, "Ready: "+joinSorted(ready)+".")
	}
	if len(degraded) > 0 {
		chunks = append(chunks, "Degraded: "+joinSorted(degraded)+".")
	}
	if len(disabled) > 0 {
		chunks = append(chunks, "Disabled: "+joinSorted(disabled)+".")
	}
	return strings.Join(chunks, " ")
}

func joinSorted(in []string) string {
	items := append([]string(nil), in...)
	sort.Strings(items)
	return strings.Join(items, ", ")
}

func compactStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, item := range in {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
