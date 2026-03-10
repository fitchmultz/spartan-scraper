// Package api provides REST API handlers for session management.
//
// Purpose:
// - Expose CRUD-style HTTP handlers for persisted auth sessions.
//
// Responsibilities:
// - Validate session identifiers and required fields.
// - Return consistent JSON envelopes for session list/detail responses.
// - Map store errors onto API-friendly error responses.
//
// Scope:
// - `/v1/auth/sessions` route handling only.
//
// Usage:
// - Mounted by Server.Routes for session management endpoints.
//
// Invariants/Assumptions:
// - Request bodies are decoded with the shared strict JSON helper.
// - Missing session deletes return 404 instead of silently succeeding.
package api

import (
	"errors"
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

	writeCollectionJSON(w, "sessions", sessions)
}

// handleGetSession handles GET /v1/auth/sessions/{id}
func (s *Server) handleGetSession(w http.ResponseWriter, r *http.Request) {
	id, err := requireResourceID(r, "sessions", "session id")
	if err != nil {
		writeError(w, r, err)
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

	writeNamedResourceJSON(w, "session", session)
}

// handleCreateSession handles POST /v1/auth/sessions
func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var input auth.Session
	if err := decodeJSONBody(w, r, &input); err != nil {
		writeError(w, r, err)
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
	if strings.TrimSpace(input.Name) == "" {
		input.Name = input.ID
	}

	store := auth.NewSessionStore(s.cfg.DataDir)
	if err := store.Upsert(input); err != nil {
		writeError(w, r, apperrors.Internal("failed to save session"))
		return
	}

	writeNamedResourceJSON(w, "session", input)
}

// handleDeleteSession handles DELETE /v1/auth/sessions/{id}
func (s *Server) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	id, err := requireResourceID(r, "sessions", "session id")
	if err != nil {
		writeError(w, r, err)
		return
	}

	store := auth.NewSessionStore(s.cfg.DataDir)
	if err := store.Delete(id); err != nil {
		if errors.Is(err, auth.ErrSessionNotFound) {
			writeError(w, r, apperrors.NotFound("session not found"))
			return
		}
		writeError(w, r, apperrors.Internal("failed to delete session"))
		return
	}

	writeNoContent(w)
}
