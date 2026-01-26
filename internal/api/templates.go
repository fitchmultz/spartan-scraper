// Package api provides HTTP handlers for extraction template listing endpoints.
// Template handlers support listing available extraction templates for use
// in scrape, crawl, and research jobs.
package api

import (
	"net/http"

	"spartan-scraper/internal/extract"
)

func (s *Server) handleTemplates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	names, err := extract.ListTemplateNames(s.cfg.DataDir)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, map[string]interface{}{"templates": names})
}
