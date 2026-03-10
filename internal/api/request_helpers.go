// Package api provides shared HTTP request helpers for Spartan Scraper's API.
//
// Purpose:
// - Centralize repeated request/response patterns used by API handlers.
//
// Responsibilities:
// - Decode strict JSON request bodies consistently.
// - Write common JSON responses such as 201 Created payloads.
// - Provide reusable named-resource CRUD scaffolding for file-backed API endpoints.
// - Centralize path, pagination, and collection-mapping helpers for resource handlers.
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

type statusResponse struct {
	Status string `json:"status"`
}

type pageParams struct {
	Limit  int
	Offset int
}

func writeCreatedJSON(w http.ResponseWriter, payload any) {
	writeJSONStatus(w, http.StatusCreated, payload)
}

func writeCollectionJSON(w http.ResponseWriter, collectionKey string, items any) {
	writeJSON(w, map[string]any{collectionKey: items})
}

func writeNamedResourceJSON(w http.ResponseWriter, resourceKey string, item any) {
	writeJSON(w, map[string]any{resourceKey: item})
}

func writeRecordsPageJSON(w http.ResponseWriter, records any, total int, limit int, offset int) {
	writeJSON(w, map[string]any{
		"records": records,
		"total":   total,
		"limit":   limit,
		"offset":  offset,
	})
}

func writeOKStatus(w http.ResponseWriter) {
	writeStatusJSON(w, "ok")
}

func writeStatusJSON(w http.ResponseWriter, status string) {
	writeJSON(w, statusResponse{Status: status})
}

func valueOr[T any](ptr *T, fallback T) T {
	if ptr == nil {
		return fallback
	}
	return *ptr
}

func mapSlice[T any, U any](items []T, fn func(T) U) []U {
	result := make([]U, len(items))
	for i, item := range items {
		result[i] = fn(item)
	}
	return result
}

func getStoredResource[T any](w http.ResponseWriter, r *http.Request, load func() (*T, error), isNotFound func(error) bool, label string) (*T, bool) {
	item, err := load()
	if err == nil {
		return item, true
	}
	if isNotFound(err) {
		writeError(w, r, apperrors.NotFound(label+" not found"))
		return nil, false
	}
	writeError(w, r, err)
	return nil, false
}

func deleteStoredResource(w http.ResponseWriter, r *http.Request, remove func() error, isNotFound func(error) bool, label string) {
	if err := remove(); err != nil {
		if isNotFound(err) {
			writeError(w, r, apperrors.NotFound(label+" not found"))
			return
		}
		writeError(w, r, err)
		return
	}
	writeNoContent(w)
}

func requireResourceID(r *http.Request, pathSegment string, label string) (string, error) {
	id := extractID(r.URL.Path, pathSegment)
	if strings.TrimSpace(id) == "" {
		return "", apperrors.Validation(label + " is required")
	}
	return id, nil
}

func requireJobID(r *http.Request) (string, error) {
	id, err := requireResourceID(r, "jobs", "job id")
	if err != nil {
		return "", err
	}
	if err := validateJobID(id); err != nil {
		return "", err
	}
	return id, nil
}

func handlePathSuffix(path string, suffix string, handler func()) bool {
	if strings.HasSuffix(strings.TrimSuffix(path, "/"), suffix) {
		handler()
		return true
	}
	return false
}

func parsePageParams(r *http.Request, defaultLimit int, maxLimit int) (pageParams, error) {
	query := r.URL.Query()
	limit, err := parseIntParamStrict(query.Get("limit"), "limit")
	if err != nil {
		return pageParams{}, err
	}
	if limit == 0 {
		limit = defaultLimit
	}
	if maxLimit > 0 && limit > maxLimit {
		limit = maxLimit
	}

	offset, err := parseIntParamStrict(query.Get("offset"), "offset")
	if err != nil {
		return pageParams{}, err
	}

	return pageParams{Limit: limit, Offset: offset}, nil
}

func handleNamedResourceCollection[T any](s *Server, w http.ResponseWriter, r *http.Request, store namedResourceStore[T]) {
	switch r.Method {
	case http.MethodGet:
		items, err := store.list(s.cfg.DataDir)
		if err != nil {
			writeError(w, r, err)
			return
		}
		writeCollectionJSON(w, store.collectionKey, items)
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
