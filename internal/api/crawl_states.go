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
		writeError(w, apperrors.MethodNotAllowed("method not allowed"))
	}
}

func (s *Server) handleCrawlStatesList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit := parseIntParam(query.Get("limit"), 100)
	offset := parseIntParam(query.Get("offset"), 0)
	opts := store.ListCrawlStatesOptions{Limit: limit, Offset: offset}

	states, err := s.store.ListCrawlStates(r.Context(), opts)
	if err != nil {
		writeError(w, err)
		return
	}

	total, err := s.store.CountCrawlStates(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}

	w.Header().Set("X-Total-Count", strconv.Itoa(total))
	writeJSON(w, map[string]interface{}{"crawlStates": states})
}

func (s *Server) handleCrawlStatesDelete(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")

	if url != "" {
		if err := s.store.DeleteCrawlState(r.Context(), url); err != nil {
			writeError(w, err)
			return
		}
	} else {
		if err := s.store.DeleteAllCrawlStates(r.Context()); err != nil {
			writeError(w, err)
			return
		}
	}

	writeJSON(w, StatusResponse{Status: "ok", RequestID: contextRequestID(r.Context())})
}
