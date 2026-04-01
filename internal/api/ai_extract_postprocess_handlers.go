// Package api provides HTTP handlers for bounded AI authoring endpoints.
//
// Purpose:
// - Handle AI research refinement, export shaping, transform generation, and shared AI authoring helpers.
//
// Responsibilities:
// - Validate post-processing authoring requests, load job-backed inputs, invoke the shared AI authoring service,
// - and emit consistent AI response headers and completion logs.
//
// Scope:
// - Research/export/transform handlers and helper functions only.
//
// Usage:
// - Mounted under `/v1/ai/*` by the API server.
//
// Invariants/Assumptions:
// - Job-backed handlers require an existing stored result file.
// - Shared AI response headers remain consistent across all AI endpoints.
package api

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/aiauthoring"
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func (s *Server) handleAIResearchRefine(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req AIResearchRefineRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	result, err := s.aiAuthoringService().RefineResearch(r.Context(), aiauthoring.ResearchRefineRequest{
		Result:       req.Result,
		Instructions: req.Instructions,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp := AIResearchRefineResponse{
		Issues:      result.Issues,
		InputStats:  result.InputStats,
		Refined:     result.Refined,
		Markdown:    result.Markdown,
		Explanation: result.Explanation,
		RouteID:     result.RouteID,
		Provider:    result.Provider,
		Model:       result.Model,
	}
	setAIResponseHeaders(w, result.RouteID, result.Provider, result.Model)
	logAIRequestCompletion("research_refine", "", result.RouteID, result.Provider, result.Model, false)
	writeJSON(w, resp)
}

func (s *Server) handleAIExportShape(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req AIExportShapeRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	job, rawResult, err := s.loadAIJobResult(r.Context(), strings.TrimSpace(req.JobID))
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := s.aiAuthoringService().GenerateExportShape(r.Context(), aiauthoring.ExportShapeRequest{
		JobKind:      job.Kind,
		Format:       strings.TrimSpace(req.Format),
		RawResult:    rawResult,
		CurrentShape: req.CurrentShape,
		Instructions: req.Instructions,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	resp := AIExportShapeResponse{
		Issues:      result.Issues,
		InputStats:  result.InputStats,
		Shape:       result.Shape,
		Explanation: result.Explanation,
		RouteID:     result.RouteID,
		Provider:    result.Provider,
		Model:       result.Model,
	}
	setAIResponseHeaders(w, result.RouteID, result.Provider, result.Model)
	logAIRequestCompletion("export_shape", "", result.RouteID, result.Provider, result.Model, false)
	writeJSON(w, resp)
}

func (s *Server) handleAITransformGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req AITransformGenerateRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}
	job, rawResult, err := s.loadAIJobResult(r.Context(), strings.TrimSpace(req.JobID))
	if err != nil {
		writeError(w, r, err)
		return
	}
	result, err := s.aiAuthoringService().GenerateTransform(r.Context(), aiauthoring.TransformRequest{
		JobKind:           job.Kind,
		RawResult:         rawResult,
		CurrentTransform:  req.CurrentTransform,
		PreferredLanguage: strings.TrimSpace(req.PreferredLanguage),
		Instructions:      req.Instructions,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}
	resp := AITransformGenerateResponse{
		Issues:      result.Issues,
		InputStats:  result.InputStats,
		Transform:   result.Transform,
		Preview:     result.Preview,
		Explanation: result.Explanation,
		RouteID:     result.RouteID,
		Provider:    result.Provider,
		Model:       result.Model,
	}
	setAIResponseHeaders(w, result.RouteID, result.Provider, result.Model)
	logAIRequestCompletion("transform_generate", "", result.RouteID, result.Provider, result.Model, false)
	writeJSON(w, resp)
}

func (s *Server) loadAIJobResult(ctx context.Context, jobID string) (*model.Job, []byte, error) {
	if jobID == "" {
		return nil, nil, apperrors.Validation("job_id is required")
	}
	job, err := s.store.Get(ctx, jobID)
	if err != nil {
		return nil, nil, apperrors.Wrap(apperrors.KindNotFound, "job not found", err)
	}
	if strings.TrimSpace(job.ResultPath) == "" {
		return nil, nil, apperrors.NotFound("job has no result file")
	}
	data, err := os.ReadFile(job.ResultPath)
	if err != nil {
		return nil, nil, apperrors.Wrap(apperrors.KindInternal, "read job result file", err)
	}
	return &job, data, nil
}

func (s *Server) aiAuthoringService() *aiauthoring.Service {
	if s.aiAuthoring != nil {
		return s.aiAuthoring
	}
	return aiauthoring.NewService(s.cfg, s.aiExtractor, !s.cfg.APIAuthEnabled && isLocalhost(s.cfg.BindAddr))
}

func (s *Server) fetchHTMLForAI(ctx context.Context, pageURL string, headless bool, usePlaywright bool) (fetch.Result, error) {
	return s.aiAuthoringService().FetchHTML(ctx, pageURL, headless, usePlaywright)
}

func setAIResponseHeaders(w http.ResponseWriter, routeID string, provider string, model string) {
	if strings.TrimSpace(routeID) != "" {
		w.Header().Set("X-Spartan-AI-Route", routeID)
	}
	if strings.TrimSpace(provider) != "" {
		w.Header().Set("X-Spartan-AI-Provider", provider)
	}
	if strings.TrimSpace(model) != "" {
		w.Header().Set("X-Spartan-AI-Model", model)
	}
}

func logAIRequestCompletion(operation string, requestURL string, routeID string, provider string, model string, cached bool) {
	slog.Info("AI request completed",
		"operation", operation,
		"url", apperrors.SanitizeURL(requestURL),
		"route_id", routeID,
		"provider", provider,
		"model", model,
		"cached", cached,
	)
}
