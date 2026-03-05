// Package api provides REST API handlers for pipeline JavaScript management.
// This file implements CRUD endpoints for /v1/pipeline-js.
//
// Responsibilities:
// - List, get, create, update, and delete pipeline JavaScript scripts
// - Validate script configuration
// - Return appropriate HTTP status codes
//
// This file does NOT:
// - Execute JavaScript code
// - Handle script matching at runtime
//
// Endpoints:
//   GET    /v1/pipeline-js       - List all scripts
//   POST   /v1/pipeline-js       - Create a new script
//   GET    /v1/pipeline-js/{name} - Get a specific script
//   PUT    /v1/pipeline-js/{name} - Update a script
//   DELETE /v1/pipeline-js/{name} - Delete a script

package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

// handlePipelineJS handles requests to /v1/pipeline-js
func (s *Server) handlePipelineJS(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListPipelineJS(w, r)
	case http.MethodPost:
		s.handleCreatePipelineJS(w, r)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

// handlePipelineJSScript handles requests to /v1/pipeline-js/{name}
func (s *Server) handlePipelineJSScript(w http.ResponseWriter, r *http.Request) {
	// Extract script name from URL path
	path := strings.TrimPrefix(r.URL.Path, "/v1/pipeline-js/")
	name := strings.Split(path, "/")[0]

	if name == "" {
		writeError(w, r, apperrors.Validation("script name is required"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetPipelineJS(w, r, name)
	case http.MethodPut:
		s.handleUpdatePipelineJS(w, r, name)
	case http.MethodDelete:
		s.handleDeletePipelineJS(w, r, name)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

func (s *Server) handleListPipelineJS(w http.ResponseWriter, r *http.Request) {
	registry, err := pipeline.LoadJSRegistryStrict(s.cfg.DataDir)
	if err != nil {
		writeError(w, r, err)
		return
	}

	response := map[string]interface{}{
		"scripts": registry.Scripts,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		writeError(w, r, apperrors.Internal("failed to encode response"))
	}
}

func (s *Server) handleGetPipelineJS(w http.ResponseWriter, r *http.Request, name string) {
	script, found, err := pipeline.GetJSScript(s.cfg.DataDir, name)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if !found {
		writeError(w, r, apperrors.NotFound("script not found"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(script); err != nil {
		writeError(w, r, apperrors.Internal("failed to encode response"))
	}
}

func (s *Server) handleCreatePipelineJS(w http.ResponseWriter, r *http.Request) {
	var input pipeline.JSTargetScript
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, r, apperrors.Validation("invalid JSON: "+err.Error()))
		return
	}

	// Check if script already exists
	_, found, err := pipeline.GetJSScript(s.cfg.DataDir, input.Name)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if found {
		writeError(w, r, apperrors.Validation("script already exists"))
		return
	}

	if err := pipeline.UpsertJSScript(s.cfg.DataDir, input); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(input); err != nil {
		writeError(w, r, apperrors.Internal("failed to encode response"))
	}
}

func (s *Server) handleUpdatePipelineJS(w http.ResponseWriter, r *http.Request, name string) {
	// Check if script exists
	_, found, err := pipeline.GetJSScript(s.cfg.DataDir, name)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if !found {
		writeError(w, r, apperrors.NotFound("script not found"))
		return
	}

	var input pipeline.JSTargetScript
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, r, apperrors.Validation("invalid JSON: "+err.Error()))
		return
	}

	// Ensure the name matches the URL parameter
	input.Name = name

	if err := pipeline.UpsertJSScript(s.cfg.DataDir, input); err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(input); err != nil {
		writeError(w, r, apperrors.Internal("failed to encode response"))
	}
}

func (s *Server) handleDeletePipelineJS(w http.ResponseWriter, r *http.Request, name string) {
	if err := pipeline.DeleteJSScript(s.cfg.DataDir, name); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
