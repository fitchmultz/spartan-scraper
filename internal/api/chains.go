// Package api provides HTTP handlers for job chain management endpoints.
//
// Purpose:
// - Expose chain CRUD and submission operations over the REST API.
//
// Responsibilities:
// - Validate chain definitions before creation.
// - Normalize node request payloads onto the canonical submission contract.
// - Submit chain nodes through the shared operator-facing request-to-spec conversion layer.
//
// Scope:
// - `/v1/chains` CRUD and submission routes only.
//
// Usage:
// - Mounted by Server.Routes for chain management.
//
// Invariants/Assumptions:
// - Chain node requests must stay on the same request contract as live job submissions.
// - Only chains with no active jobs can be deleted.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/submission"
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

	writeCollectionJSON(w, "chains", chains)
}

func (s *Server) handleCreateChain(w http.ResponseWriter, r *http.Request) {
	var req ChainCreateRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
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

	for i := range req.Definition.Nodes {
		node := &req.Definition.Nodes[i]
		if len(node.Request) == 0 {
			writeError(w, r, apperrors.Validation(fmt.Sprintf("definition.nodes[%d].request is required", i)))
			return
		}
		if _, _, err := submission.JobSpecFromRawRequest(s.cfg, s.nonResolvingSubmissionDefaults(), node.Kind, node.Request); err != nil {
			writeError(w, r, apperrors.Validation("invalid chain node request for "+node.ID+": "+err.Error()))
			return
		}
		normalizedRequest, err := submission.NormalizeRawRequest(node.Kind, node.Request)
		if err != nil {
			writeError(w, r, apperrors.Validation("invalid chain node request for "+node.ID+": "+err.Error()))
			return
		}
		node.Request = normalizedRequest
	}

	chain, err := s.manager.CreateChain(r.Context(), req.Name, req.Description, req.Definition)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeCreatedJSON(w, chain)
}

func (s *Server) handleChain(w http.ResponseWriter, r *http.Request) {
	if handlePathSuffix(r.URL.Path, "/submit", func() {
		if r.Method == http.MethodPost {
			s.handleChainSubmit(w, r)
		} else {
			writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		}
	}) {
		return
	}

	id, err := requireResourceID(r, "chains", "chain id")
	if err != nil {
		writeError(w, r, err)
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

	writeNoContent(w)
}

func (s *Server) handleChainSubmit(w http.ResponseWriter, r *http.Request) {
	chainID, err := requireResourceID(r, "chains", "chain id")
	if err != nil {
		writeError(w, r, err)
		return
	}

	var req ChainSubmitRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	jobs, err := s.manager.SubmitChain(r.Context(), chainID, req.Overrides, func(kind model.Kind, rawRequest json.RawMessage) (jobs.JobSpec, error) {
		spec, _, err := submission.JobSpecFromRawRequest(s.cfg, s.requestSubmissionDefaults(r), kind, rawRequest)
		return spec, err
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeCreatedJSON(w, ChainSubmitResponse{Jobs: jobs})
}
