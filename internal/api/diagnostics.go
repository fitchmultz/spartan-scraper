// Package api provides read-only recovery diagnostics for setup and degraded runtime states.
//
// Purpose:
// - Expose safe one-click diagnostic endpoints and action builders that help operators recover inside the product.
//
// Responsibilities:
// - Route browser, AI, and proxy-pool re-checks onto the shared diagnostic builders.
// - Keep setup-mode and normal-mode diagnostic responses consistent.
// - Preserve read-only semantics for repeated operator troubleshooting.
//
// Scope:
// - HTTP diagnostic endpoint handling only; shared message construction lives in sibling helpers.
//
// Usage:
// - Mounted by `Server.Routes()` for setup mode and normal mode surfaces.
//
// Invariants/Assumptions:
// - Endpoints remain safe to call repeatedly.
// - Recovery actions should favor copy-ready commands plus browser-safe links.
package api

import (
	"net/http"
)

func allowDiagnosticMethod(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodGet || r.Method == http.MethodPost {
		return true
	}
	writeJSONStatus(w, http.StatusMethodNotAllowed, ErrorResponse{Error: "method not allowed"})
	return false
}

func (s *Server) handleBrowserDiagnostic(w http.ResponseWriter, r *http.Request) {
	if !allowDiagnosticMethod(w, r) {
		return
	}
	writeJSONStatus(w, http.StatusOK, BuildBrowserDiagnosticResponse(s.cfg))
}

func (s *Server) handleAIDiagnostic(w http.ResponseWriter, r *http.Request) {
	if !allowDiagnosticMethod(w, r) {
		return
	}
	writeJSONStatus(w, http.StatusOK, BuildAIDiagnosticResponse(r.Context(), s.cfg, s.aiExtractor))
}

func (s *Server) handleProxyPoolDiagnostic(w http.ResponseWriter, r *http.Request) {
	if !allowDiagnosticMethod(w, r) {
		return
	}
	writeJSONStatus(w, http.StatusOK, BuildProxyPoolDiagnosticResponse(s.cfg, s.proxyPoolRuntimeState()))
}
