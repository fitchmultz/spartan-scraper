// Package api provides HTTP handlers for bounded AI authoring endpoints.
//
// Purpose:
// - Handle AI render-profile and pipeline-JS authoring requests.
//
// Responsibilities:
// - Validate automation authoring/debug requests, invoke the shared AI authoring service,
// - and adapt service results into stable API responses.
//
// Scope:
// - Render-profile and pipeline-JS generate/debug handlers only.
//
// Usage:
// - Mounted under `/v1/ai/*` by the API server.
//
// Invariants/Assumptions:
// - These handlers only accept POST JSON requests.
package api

import (
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/aiauthoring"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

func (s *Server) handleAIRenderProfileGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req AIRenderProfileGenerateRequest
	if err := decodeJSONBodyWithLimit(w, r, &req, maxAIAuthoringRequestBodySize); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := s.aiAuthoringService().GenerateRenderProfile(r.Context(), aiauthoring.RenderProfileRequest{
		URL:           req.URL,
		Name:          req.Name,
		HostPatterns:  req.HostPatterns,
		Instructions:  req.Instructions,
		Images:        req.Images,
		Headless:      req.Headless,
		UsePlaywright: req.UsePlaywright,
		Visual:        req.Visual,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := AIRenderProfileGenerateResponse{
		Profile:           result.Profile,
		ResolvedGoal:      result.ResolvedGoal,
		Explanation:       result.Explanation,
		RouteID:           result.RouteID,
		Provider:          result.Provider,
		Model:             result.Model,
		VisualContextUsed: result.VisualContextUsed,
	}
	setAIResponseHeaders(w, result.RouteID, result.Provider, result.Model)
	logAIRequestCompletion("render_profile_generate", req.URL, result.RouteID, result.Provider, result.Model, false)
	writeJSON(w, resp)
}

func (s *Server) handleAIPipelineJSGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req AIPipelineJSGenerateRequest
	if err := decodeJSONBodyWithLimit(w, r, &req, maxAIAuthoringRequestBodySize); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := s.aiAuthoringService().GeneratePipelineJS(r.Context(), aiauthoring.PipelineJSRequest{
		URL:           req.URL,
		Name:          req.Name,
		HostPatterns:  req.HostPatterns,
		Instructions:  req.Instructions,
		Images:        req.Images,
		Headless:      req.Headless,
		UsePlaywright: req.UsePlaywright,
		Visual:        req.Visual,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := AIPipelineJSGenerateResponse{
		Script:            result.Script,
		ResolvedGoal:      result.ResolvedGoal,
		Explanation:       result.Explanation,
		RouteID:           result.RouteID,
		Provider:          result.Provider,
		Model:             result.Model,
		VisualContextUsed: result.VisualContextUsed,
	}
	setAIResponseHeaders(w, result.RouteID, result.Provider, result.Model)
	logAIRequestCompletion("pipeline_js_generate", req.URL, result.RouteID, result.Provider, result.Model, false)
	writeJSON(w, resp)
}

func (s *Server) handleAIRenderProfileDebug(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req AIRenderProfileDebugRequest
	if err := decodeJSONBodyWithLimit(w, r, &req, maxAIAuthoringRequestBodySize); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := s.aiAuthoringService().DebugRenderProfile(r.Context(), aiauthoring.RenderProfileDebugRequest{
		URL:           req.URL,
		Profile:       req.Profile,
		Instructions:  req.Instructions,
		Images:        req.Images,
		Headless:      req.Headless,
		UsePlaywright: req.UsePlaywright,
		Visual:        req.Visual,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := AIRenderProfileDebugResponse{
		Issues:            result.Issues,
		ResolvedGoal:      result.ResolvedGoal,
		Explanation:       result.Explanation,
		SuggestedProfile:  result.SuggestedProfile,
		RouteID:           result.RouteID,
		Provider:          result.Provider,
		Model:             result.Model,
		VisualContextUsed: result.VisualContextUsed,
		RecheckStatus:     result.RecheckStatus,
		RecheckEngine:     result.RecheckEngine,
		RecheckError:      result.RecheckError,
	}
	setAIResponseHeaders(w, result.RouteID, result.Provider, result.Model)
	logAIRequestCompletion("render_profile_debug", req.URL, result.RouteID, result.Provider, result.Model, false)
	writeJSON(w, resp)
}

func (s *Server) handleAIPipelineJSDebug(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req AIPipelineJSDebugRequest
	if err := decodeJSONBodyWithLimit(w, r, &req, maxAIAuthoringRequestBodySize); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := s.aiAuthoringService().DebugPipelineJS(r.Context(), aiauthoring.PipelineJSDebugRequest{
		URL:           req.URL,
		Script:        req.Script,
		Instructions:  req.Instructions,
		Images:        req.Images,
		Headless:      req.Headless,
		UsePlaywright: req.UsePlaywright,
		Visual:        req.Visual,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := AIPipelineJSDebugResponse{
		Issues:            result.Issues,
		ResolvedGoal:      result.ResolvedGoal,
		Explanation:       result.Explanation,
		SuggestedScript:   result.SuggestedScript,
		RouteID:           result.RouteID,
		Provider:          result.Provider,
		Model:             result.Model,
		VisualContextUsed: result.VisualContextUsed,
		RecheckStatus:     result.RecheckStatus,
		RecheckEngine:     result.RecheckEngine,
		RecheckError:      result.RecheckError,
	}
	setAIResponseHeaders(w, result.RouteID, result.Provider, result.Model)
	logAIRequestCompletion("pipeline_js_debug", req.URL, result.RouteID, result.Provider, result.Model, false)
	writeJSON(w, resp)
}
