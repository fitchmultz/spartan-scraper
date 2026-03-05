// Package api provides HTTP handlers for job chain management endpoints.
//
// This file is responsible for:
// - Chain CRUD operations (create, list, get, delete)
// - Chain submission (instantiating jobs from a chain)
//
// This file does NOT handle:
// - Chain execution (jobs package handles this)
// - Chain validation (model package handles this)
//
// Invariants:
// - Chain definitions are validated before creation
// - Chain names must be unique
// - Only chains with no active jobs can be deleted
package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// ChainCreateRequest represents a request to create a new chain.
type ChainCreateRequest struct {
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Definition  model.ChainDefinition `json:"definition"`
}

// ChainSubmitRequest represents a request to submit/instantiate a chain.
type ChainSubmitRequest struct {
	Overrides map[string]json.RawMessage `json:"overrides,omitempty"`
}

// ChainSubmitResponse represents the response from submitting a chain.
type ChainSubmitResponse struct {
	Jobs []model.Job `json:"jobs"`
}

func (s *Server) handleChains(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListChains(w, r)
	case http.MethodPost:
		s.handleCreateChain(w, r)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

func (s *Server) handleListChains(w http.ResponseWriter, r *http.Request) {
	chains, err := s.manager.ListChains(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, map[string]interface{}{"chains": chains})
}

func (s *Server) handleCreateChain(w http.ResponseWriter, r *http.Request) {
	var req ChainCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, apperrors.Validation("invalid request body: "+err.Error()))
		return
	}

	if req.Name == "" {
		writeError(w, r, apperrors.Validation("name is required"))
		return
	}

	if len(req.Definition.Nodes) == 0 {
		writeError(w, r, apperrors.Validation("chain must have at least one node"))
		return
	}

	chain, err := s.manager.CreateChain(r.Context(), req.Name, req.Description, req.Definition)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, chain)
}

func (s *Server) handleChain(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Check if this is a submit request: /v1/chains/{id}/submit
	if strings.HasSuffix(path, "/submit") {
		if r.Method == http.MethodPost {
			s.handleChainSubmit(w, r)
		} else {
			writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		}
		return
	}

	id := extractID(path, "chains")
	if id == "" {
		writeError(w, r, apperrors.Validation("chain ID is required"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetChain(w, r, id)
	case http.MethodDelete:
		s.handleDeleteChain(w, r, id)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

func (s *Server) handleGetChain(w http.ResponseWriter, r *http.Request, id string) {
	chain, err := s.manager.GetChain(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, chain)
}

func (s *Server) handleDeleteChain(w http.ResponseWriter, r *http.Request, id string) {
	if err := s.manager.DeleteChain(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleChainSubmit(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	// Extract chain ID from /v1/chains/{chainId}/submit
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 4 || parts[0] != "v1" || parts[1] != "chains" {
		writeError(w, r, apperrors.Validation("invalid path"))
		return
	}
	chainID := parts[2]

	var req ChainSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, apperrors.Validation("invalid request body: "+err.Error()))
		return
	}

	jobs, err := s.manager.SubmitChain(r.Context(), chainID, req.Overrides)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, ChainSubmitResponse{Jobs: jobs})
}
