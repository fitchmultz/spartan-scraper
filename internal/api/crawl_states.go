// Package api provides HTTP handlers for crawl state listing endpoints.
// Crawl state handlers support listing in-progress and completed crawl states
// with pagination support.
package api

import (
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func (s *Server) handleCrawlStates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	query := r.URL.Query()
	limit := parseIntParam(query.Get("limit"), 100)
	offset := parseIntParam(query.Get("offset"), 0)
	opts := store.ListCrawlStatesOptions{Limit: limit, Offset: offset}

	states, err := s.store.ListCrawlStates(r.Context(), opts)
	if err != nil {
		writeError(w, err)
		return
	}

	writeJSON(w, map[string]interface{}{"crawlStates": states})
}
