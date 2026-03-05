// Package api provides REST API handlers for render profile management.
// This file implements CRUD endpoints for /v1/render-profiles.
//
// Responsibilities:
// - List, get, create, update, and delete render profiles
// - Validate profile configuration
// - Return appropriate HTTP status codes
//
// This file does NOT:
// - Handle runtime profile matching
// - Execute fetches
//
// Endpoints:
//   GET    /v1/render-profiles       - List all profiles
//   POST   /v1/render-profiles       - Create a new profile
//   GET    /v1/render-profiles/{name} - Get a specific profile
//   PUT    /v1/render-profiles/{name} - Update a profile
//   DELETE /v1/render-profiles/{name} - Delete a profile

package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

// handleRenderProfiles handles requests to /v1/render-profiles
func (s *Server) handleRenderProfiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListRenderProfiles(w, r)
	case http.MethodPost:
		s.handleCreateRenderProfile(w, r)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

// handleRenderProfile handles requests to /v1/render-profiles/{name}
func (s *Server) handleRenderProfile(w http.ResponseWriter, r *http.Request) {
	// Extract profile name from URL path
	path := strings.TrimPrefix(r.URL.Path, "/v1/render-profiles/")
	name := strings.Split(path, "/")[0]

	if name == "" {
		writeError(w, r, apperrors.Validation("profile name is required"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetRenderProfile(w, r, name)
	case http.MethodPut:
		s.handleUpdateRenderProfile(w, r, name)
	case http.MethodDelete:
		s.handleDeleteRenderProfile(w, r, name)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

func (s *Server) handleListRenderProfiles(w http.ResponseWriter, r *http.Request) {
	file, err := fetch.LoadRenderProfilesFile(s.cfg.DataDir)
	if err != nil {
		writeError(w, r, err)
		return
	}

	response := map[string]interface{}{
		"profiles": file.Profiles,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		writeError(w, r, apperrors.Internal("failed to encode response"))
	}
}

func (s *Server) handleGetRenderProfile(w http.ResponseWriter, r *http.Request, name string) {
	profile, found, err := fetch.GetRenderProfile(s.cfg.DataDir, name)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if !found {
		writeError(w, r, apperrors.NotFound("profile not found"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(profile); err != nil {
		writeError(w, r, apperrors.Internal("failed to encode response"))
	}
}

func (s *Server) handleCreateRenderProfile(w http.ResponseWriter, r *http.Request) {
	var input fetch.RenderProfile
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, r, apperrors.Validation("invalid JSON: "+err.Error()))
		return
	}

	// Check if profile already exists
	_, found, err := fetch.GetRenderProfile(s.cfg.DataDir, input.Name)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if found {
		writeError(w, r, apperrors.Validation("profile already exists"))
		return
	}

	if err := fetch.UpsertRenderProfile(s.cfg.DataDir, input); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(input); err != nil {
		writeError(w, r, apperrors.Internal("failed to encode response"))
	}
}

func (s *Server) handleUpdateRenderProfile(w http.ResponseWriter, r *http.Request, name string) {
	// Check if profile exists
	_, found, err := fetch.GetRenderProfile(s.cfg.DataDir, name)
	if err != nil {
		writeError(w, r, err)
		return
	}
	if !found {
		writeError(w, r, apperrors.NotFound("profile not found"))
		return
	}

	var input fetch.RenderProfile
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, r, apperrors.Validation("invalid JSON: "+err.Error()))
		return
	}

	// Ensure the name matches the URL parameter
	input.Name = name

	if err := fetch.UpsertRenderProfile(s.cfg.DataDir, input); err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(input); err != nil {
		writeError(w, r, apperrors.Internal("failed to encode response"))
	}
}

func (s *Server) handleDeleteRenderProfile(w http.ResponseWriter, r *http.Request, name string) {
	if err := fetch.DeleteRenderProfile(s.cfg.DataDir, name); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
