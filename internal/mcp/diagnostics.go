// Package mcp exposes structured runtime diagnostics over the MCP tool surface.
//
// Purpose:
// - Carry the same health and recovery model used by REST and CLI into MCP.
//
// Responsibilities:
// - Build MCP-facing health payloads for normal runtime and setup mode.
// - Translate one-click recovery actions into MCP tool commands.
// - Execute read-only browser, AI, and proxy-pool diagnostic checks.
//
// Scope:
// - Diagnostics helpers for MCP only; JSON-RPC transport and unrelated tool handlers live elsewhere.
//
// Usage:
// - Called by `health_status` and `diagnostic_check` tool handlers.
//
// Invariants/Assumptions:
// - Setup mode should still expose actionable diagnostics.
// - MCP action values should reference MCP-native commands instead of raw HTTP endpoints.
package mcp

import (
	"context"

	"github.com/fitchmultz/spartan-scraper/internal/api"
	"github.com/fitchmultz/spartan-scraper/internal/buildinfo"
)

func (s *Server) buildHealthStatus(ctx context.Context) api.HealthResponse {
	res := api.HealthResponse{
		Status:     "ok",
		Version:    buildinfo.Version,
		Components: make(map[string]api.ComponentStatus),
	}

	if s.setup != nil {
		setup := *s.setup
		res.Status = "setup_required"
		res.Setup = translateSetupForMCP(setup)
		res.Components["database"] = api.ComponentStatus{
			Status:  "setup_required",
			Message: setup.Message,
			Details: map[string]any{
				"dataDir":       setup.DataDir,
				"schemaVersion": setup.SchemaVersion,
			},
			Actions: api.MCPRecommendedActions(setup.Actions),
		}
		res.Components["queue"] = api.ComponentStatus{
			Status:  "setup_required",
			Message: "Job processing stays unavailable until setup is completed.",
		}
		res.Components[api.DiagnosticTargetBrowser] = translateComponentForMCP(api.BuildBrowserComponentStatus(s.cfg))
		res.Components[api.DiagnosticTargetAI] = translateComponentForMCP(api.BuildAIComponentStatus(ctx, s.cfg, s.aiExtractor))
		res.Components[api.DiagnosticTargetProxyPool] = translateComponentForMCP(api.BuildProxyPoolComponentStatus(s.cfg, api.ProxyPoolRuntimeSetupMode))
		res.Notices = append(res.Notices,
			translateNoticeForMCP(api.RuntimeNotice{
				ID:       setup.Code,
				Scope:    "setup",
				Severity: "error",
				Title:    setup.Title,
				Message:  setup.Message,
				Actions:  setup.Actions,
			}),
		)
		res.Notices = append(res.Notices, translateNoticesForMCP(api.BuildConfigRuntimeNotices(s.cfg.StartupNotices))...)
		return res
	}

	hasError := false
	hasDegraded := false
	if s.store == nil {
		res.Components["database"] = api.ComponentStatus{Status: "error", Message: "store is not initialized"}
		hasError = true
	} else {
		dbStatus := api.ComponentStatus{Status: "ok"}
		if err := s.store.Ping(ctx); err != nil {
			dbStatus.Status = "error"
			dbStatus.Message = err.Error()
			hasError = true
		}
		res.Components["database"] = dbStatus
	}

	if s.manager == nil {
		res.Components["queue"] = api.ComponentStatus{Status: "error", Message: "job manager is not initialized"}
		hasError = true
	} else {
		status := s.manager.Status()
		res.Components["queue"] = api.ComponentStatus{
			Status: "ok",
			Details: map[string]int{
				"queued": status.QueuedJobs,
				"active": status.ActiveJobs,
			},
		}
	}

	browserStatus := translateComponentForMCP(api.BuildBrowserComponentStatus(s.cfg))
	res.Components[api.DiagnosticTargetBrowser] = browserStatus
	if browserStatus.Status == "degraded" {
		hasDegraded = true
	}

	aiStatus := translateComponentForMCP(api.BuildAIComponentStatus(ctx, s.cfg, s.aiExtractor))
	res.Components[api.DiagnosticTargetAI] = aiStatus
	if aiStatus.Status == "degraded" {
		hasDegraded = true
	}

	proxyStatus := translateComponentForMCP(api.BuildProxyPoolComponentStatus(s.cfg, s.proxyPoolRuntimeState()))
	res.Components[api.DiagnosticTargetProxyPool] = proxyStatus
	if proxyStatus.Status == "degraded" {
		hasDegraded = true
	}

	res.Notices = append(res.Notices, translateNoticesForMCP(api.BuildConfigRuntimeNotices(s.cfg.StartupNotices))...)

	switch {
	case hasError:
		res.Status = "error"
	case hasDegraded || len(res.Notices) > 0:
		res.Status = "degraded"
	}
	return res
}

func (s *Server) runDiagnostic(ctx context.Context, target string) api.DiagnosticActionResponse {
	switch target {
	case api.DiagnosticTargetBrowser:
		return translateDiagnosticForMCP(api.BuildBrowserDiagnosticResponse(s.cfg))
	case api.DiagnosticTargetAI:
		return translateDiagnosticForMCP(api.BuildAIDiagnosticResponse(ctx, s.cfg, s.aiExtractor))
	case api.DiagnosticTargetProxyPool:
		return translateDiagnosticForMCP(api.BuildProxyPoolDiagnosticResponse(s.cfg, s.proxyPoolRuntimeState()))
	default:
		return api.DiagnosticActionResponse{Status: "error", Message: "unknown diagnostic target"}
	}
}

func (s *Server) proxyPoolRuntimeState() api.ProxyPoolRuntimeState {
	switch {
	case s.setup != nil:
		return api.ProxyPoolRuntimeSetupMode
	case s.manager == nil:
		return api.ProxyPoolRuntimeUnavailable
	case s.manager.GetProxyPool() == nil:
		return api.ProxyPoolRuntimeUnloaded
	default:
		return api.ProxyPoolRuntimeLoaded
	}
}

func translateSetupForMCP(setup api.SetupStatus) *api.SetupStatus {
	translated := setup
	translated.Actions = api.MCPRecommendedActions(setup.Actions)
	return &translated
}

func translateComponentForMCP(component api.ComponentStatus) api.ComponentStatus {
	component.Actions = api.MCPRecommendedActions(component.Actions)
	return component
}

func translateDiagnosticForMCP(response api.DiagnosticActionResponse) api.DiagnosticActionResponse {
	response.Actions = api.MCPRecommendedActions(response.Actions)
	return response
}

func translateNoticeForMCP(notice api.RuntimeNotice) api.RuntimeNotice {
	notice.Actions = api.MCPRecommendedActions(notice.Actions)
	return notice
}

func translateNoticesForMCP(notices []api.RuntimeNotice) []api.RuntimeNotice {
	translated := make([]api.RuntimeNotice, 0, len(notices))
	for _, notice := range notices {
		translated = append(translated, translateNoticeForMCP(notice))
	}
	return translated
}
