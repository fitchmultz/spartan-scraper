// Package api provides shared HTTP request helpers for Spartan Scraper's API.
//
// Purpose:
// - Centralize repeated request/response patterns used by API handlers.
//
// Responsibilities:
// - Decode strict JSON request bodies consistently.
// - Write common JSON responses such as 201 Created payloads.
// - Provide reusable named-resource CRUD scaffolding for file-backed API endpoints.
//
// Scope:
// - Helper utilities for handler implementations in this package only.
//
// Usage:
//   - Call decodeJSONBody for JSON request parsing and handleNamedResourceCollection/
//     handleNamedResourceItem for collection-style CRUD endpoints.
//
// Invariants/Assumptions:
// - JSON endpoints require application/json content types.
// - Unknown JSON fields are rejected.
// - Named resources are addressed by the path segment immediately after a stable route prefix.
package api

import (
	"net/http"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

type namedResourceStore[T any] struct {
	pathSegment   string
	singularLabel string
	collectionKey string
	list          func(dataDir string) ([]T, error)
	get           func(dataDir, name string) (T, bool, error)
	upsert        func(dataDir string, value T) error
	delete        func(dataDir, name string) error
	nameOf        func(value T) string
	setName       func(value *T, name string)
}

func writeCreatedJSON(w http.ResponseWriter, payload any) {
	writeJSONStatus(w, http.StatusCreated, payload)
}

func valueOr[T any](ptr *T, fallback T) T {
	if ptr == nil {
		return fallback
	}
	return *ptr
}

func handleNamedResourceCollection[T any](s *Server, w http.ResponseWriter, r *http.Request, store namedResourceStore[T]) {
	switch r.Method {
	case http.MethodGet:
		items, err := store.list(s.cfg.DataDir)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, map[string]any{store.collectionKey: items})
	case http.MethodPost:
		var input T
		if err := decodeJSONBody(w, r, &input); err != nil {
			writeError(w, r, err)
			return
		}
		name := strings.TrimSpace(store.nameOf(input))
		if name == "" {
			writeError(w, r, apperrors.Validation(store.singularLabel+" name is required"))
			return
		}
		if _, found, err := store.get(s.cfg.DataDir, name); err != nil {
			writeError(w, r, err)
			return
		} else if found {
			writeError(w, r, apperrors.Validation(store.singularLabel+" already exists"))
			return
		}
		if err := store.upsert(s.cfg.DataDir, input); err != nil {
			writeError(w, r, err)
			return
		}
		writeCreatedJSON(w, input)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

func handleNamedResourceItem[T any](s *Server, w http.ResponseWriter, r *http.Request, store namedResourceStore[T]) {
	name := extractID(r.URL.Path, store.pathSegment)
	if name == "" {
		writeError(w, r, apperrors.Validation(store.singularLabel+" name is required"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		item, found, err := store.get(s.cfg.DataDir, name)
		if err != nil {
			writeError(w, r, err)
			return
		}
		if !found {
			writeError(w, r, apperrors.NotFound(store.singularLabel+" not found"))
			return
		}
		writeJSON(w, item)
	case http.MethodPut:
		if _, found, err := store.get(s.cfg.DataDir, name); err != nil {
			writeError(w, r, err)
			return
		} else if !found {
			writeError(w, r, apperrors.NotFound(store.singularLabel+" not found"))
			return
		}

		var input T
		if err := decodeJSONBody(w, r, &input); err != nil {
			writeError(w, r, err)
			return
		}
		store.setName(&input, name)
		if err := store.upsert(s.cfg.DataDir, input); err != nil {
			writeError(w, r, err)
			return
		}
		writeJSON(w, input)
	case http.MethodDelete:
		if _, found, err := store.get(s.cfg.DataDir, name); err != nil {
			writeError(w, r, err)
			return
		} else if !found {
			writeError(w, r, apperrors.NotFound(store.singularLabel+" not found"))
			return
		}
		if err := store.delete(s.cfg.DataDir, name); err != nil {
			writeError(w, r, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}
