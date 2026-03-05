// Package api provides REST API handlers for session management.
package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/auth"
)

// handleListSessions handles GET /v1/auth/sessions
func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	store := auth.NewSessionStore(s.cfg.DataDir)
	sessions, err := store.List()
	if err != nil {
		writeError(w, r, apperrors.Internal("failed to list sessions"))
		return
	}

	writeJSON(w, map[string]interface{}{
		"sessions": sessions,
	})
}

// handleGetSession handles GET /v1/auth/sessions/{id}
func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/auth/sessions/")
	if id == "" {
		writeError(w, r, apperrors.Validation("session ID is required"))
		return
	}

	store := auth.NewSessionStore(s.cfg.DataDir)
	session, found, err := store.Get(id)
	if err != nil {
		writeError(w, r, apperrors.Internal("failed to get session"))
		return
	}
	if !found {
		writeError(w, r, apperrors.NotFound("session not found"))
		return
	}

	writeJSON(w, map[string]interface{}{
		"session": session,
	})
}

// handleCreateSession handles POST /v1/auth/sessions
func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var input auth.Session
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, r, apperrors.Validation("invalid request body"))
		return
	}

	if strings.TrimSpace(input.ID) == "" {
		writeError(w, r, apperrors.Validation("session ID is required"))
		return
	}
	if strings.TrimSpace(input.Domain) == "" {
		writeError(w, r, apperrors.Validation("domain is required"))
		return
	}

	store := auth.NewSessionStore(s.cfg.DataDir)
	if err := store.Upsert(input); err != nil {
		writeError(w, r, apperrors.Internal("failed to save session"))
		return
	}

	writeJSON(w, map[string]interface{}{
		"session": input,
	})
}

// handleDeleteSession handles DELETE /v1/auth/sessions/{id}
func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/auth/sessions/")
	if id == "" {
		writeError(w, r, apperrors.Validation("session ID is required"))
		return
	}

	store := auth.NewSessionStore(s.cfg.DataDir)
	if err := store.Delete(id); err != nil {
		writeError(w, r, apperrors.Internal("failed to delete session"))
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
