// Package api provides HTTP handlers for health and diagnostic endpoints.
//
// Purpose:
// - Expose structured setup, runtime, and optional-subsystem diagnostics for web, CLI, and automation clients.
//
// Responsibilities:
// - Report overall service health and setup-mode state.
// - Classify component status as ok, degraded, disabled, setup_required, or error.
// - Surface non-fatal startup notices so operators can recover inside the product.
//
// Scope:
// - Health response construction only; component initialization happens elsewhere.
//
// Usage:
// - Mounted at `/healthz` by `Server.Routes()`.
//
// Invariants/Assumptions:
// - Optional subsystems should degrade health instead of blocking startup by default.
// - Setup-mode servers must still answer `/healthz` with actionable recovery metadata.
package api

import (
	"context"
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/buildinfo"
	"github.com/fitchmultz/spartan-scraper/internal/config"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	res := HealthResponse{
		Status:     "ok",
		Version:    buildinfo.Version,
		Components: make(map[string]ComponentStatus),
	}

	if s.setup != nil {
		res.Status = "setup_required"
		res.Setup = s.setup
		res.Components["database"] = ComponentStatus{
			Status:  "setup_required",
			Message: s.setup.Message,
			Details: map[string]any{
				"dataDir":       s.setup.DataDir,
				"schemaVersion": s.setup.SchemaVersion,
			},
			Actions: s.setup.Actions,
		}
		res.Components["queue"] = ComponentStatus{
			Status:  "setup_required",
			Message: "Job processing stays unavailable until setup is completed.",
		}
		res.Components["webhook"] = s.webhookHealthStatus()
		res.Components["browser"] = s.browserHealthStatus()
		res.Components["ai"] = s.aiHealthStatus(ctx)
		res.Components["proxy_pool"] = s.proxyPoolHealthStatus()
		res.Notices = append(res.Notices, RuntimeNotice{
			ID:       s.setup.Code,
			Scope:    "setup",
			Severity: "error",
			Title:    s.setup.Title,
			Message:  s.setup.Message,
			Actions:  s.setup.Actions,
		})
		res.Notices = append(res.Notices, mapConfigNotices(s.cfg.StartupNotices)...)
		writeJSONStatus(w, http.StatusOK, res)
		return
	}

	hasError := false
	hasDegraded := false

	if s.store == nil {
		res.Components["database"] = ComponentStatus{
			Status:  "error",
			Message: "store is not initialized",
		}
		hasError = true
	} else {
		dbStatus := ComponentStatus{Status: "ok"}
		if err := s.store.Ping(ctx); err != nil {
			dbStatus.Status = "error"
			dbStatus.Message = err.Error()
			hasError = true
		}
		res.Components["database"] = dbStatus
	}

	if s.manager == nil {
		res.Components["queue"] = ComponentStatus{
			Status:  "error",
			Message: "job manager is not initialized",
		}
		hasError = true
	} else {
		qStatus := s.manager.Status()
		res.Components["queue"] = ComponentStatus{
			Status: "ok",
			Details: map[string]int{
				"queued": qStatus.QueuedJobs,
				"active": qStatus.ActiveJobs,
			},
		}
	}

	webhookStatus := s.webhookHealthStatus()
	if webhookStatus.Status == "degraded" {
		hasDegraded = true
	}
	res.Components["webhook"] = webhookStatus

	browserStatus := s.browserHealthStatus()
	if browserStatus.Status == "degraded" {
		hasDegraded = true
	}
	res.Components["browser"] = browserStatus

	aiStatus := s.aiHealthStatus(ctx)
	if aiStatus.Status == "degraded" {
		hasDegraded = true
	}
	res.Components["ai"] = aiStatus

	proxyStatus := s.proxyPoolHealthStatus()
	if proxyStatus.Status == "degraded" {
		hasDegraded = true
	}
	res.Components["proxy_pool"] = proxyStatus

	res.Notices = append(res.Notices, mapConfigNotices(s.cfg.StartupNotices)...)

	switch {
	case hasError:
		res.Status = "error"
	case hasDegraded || len(res.Notices) > 0:
		res.Status = "degraded"
	default:
		res.Status = "ok"
	}

	writeJSONStatus(w, http.StatusOK, res)
}

func (s *Server) browserHealthStatus() ComponentStatus {
	return BuildBrowserComponentStatus(s.cfg)
}

func (s *Server) webhookHealthStatus() ComponentStatus {
	if s.webhookDispatcher == nil {
		return ComponentStatus{
			Status:  "disabled",
			Message: "Webhook delivery is disabled.",
		}
	}
	stats := s.webhookDispatcher.Stats()
	status := "ok"
	message := "Webhook dispatcher is ready."
	if stats.Dropped > 0 {
		status = "degraded"
		message = "Webhook dispatcher has dropped deliveries due to queue backpressure."
	}
	return ComponentStatus{
		Status:  status,
		Message: message,
		Details: stats,
	}
}

func (s *Server) aiHealthStatus(ctx context.Context) ComponentStatus {
	return BuildAIComponentStatus(ctx, s.cfg, s.aiExtractor)
}

func (s *Server) proxyPoolHealthStatus() ComponentStatus {
	return BuildProxyPoolComponentStatus(s.cfg, s.proxyPoolRuntimeState())
}

func (s *Server) proxyPoolRuntimeState() ProxyPoolRuntimeState {
	switch {
	case s.setup != nil:
		return ProxyPoolRuntimeSetupMode
	case s.manager == nil:
		return ProxyPoolRuntimeUnavailable
	case s.manager.GetProxyPool() == nil:
		return ProxyPoolRuntimeUnloaded
	default:
		return ProxyPoolRuntimeLoaded
	}
}

func mapConfigNotices(in []config.StartupNotice) []RuntimeNotice {
	return BuildConfigRuntimeNotices(in)
}
