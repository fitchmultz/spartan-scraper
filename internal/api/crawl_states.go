// Package api provides HTTP handlers for crawl state listing endpoints.
// Crawl state handlers support listing in-progress and completed crawl states
// with pagination support.
package api

import (
	"net/http"
	"strconv"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func (s *Server) handleCrawlStates(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleCrawlStatesList(w, r)
	case http.MethodDelete:
		s.handleCrawlStatesDelete(w, r)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

func (s *Server) handleCrawlStatesList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, err := parseIntParamStrict(query.Get("limit"), "limit")
	if err != nil {
		writeError(w, r, err)
		return
	}
	if limit == 0 {
		limit = 100
	}

	offset, err := parseIntParamStrict(query.Get("offset"), "offset")
	if err != nil {
		writeError(w, r, err)
		return
	}
	opts := store.ListCrawlStatesOptions{Limit: limit, Offset: offset}

	states, err := s.store.ListCrawlStates(r.Context(), opts)
	if err != nil {
		writeError(w, r, err)
		return
	}

	total, err := s.store.CountCrawlStates(r.Context())
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	writeCollectionJSON(w, "crawlStates", states)
}

func (s *Server) handleCrawlStatesDelete(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")

	if url != "" {
		if err := s.store.DeleteCrawlState(r.Context(), url); err != nil {
			writeError(w, r, err)
			return
		}
	} else {
		if err := s.store.DeleteAllCrawlStates(r.Context()); err != nil {
			writeError(w, r, err)
			return
		}
	}

	writeJSON(w, StatusResponse{Status: "ok", RequestID: contextRequestID(r.Context())})
}
