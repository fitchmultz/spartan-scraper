// Package api provides HTTP handlers for scheduled job management endpoints.
// Schedule handlers support creating, listing, retrieving, updating, and deleting
// recurring scrape, crawl, and research jobs with configurable intervals.
package api

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"spartan-scraper/internal/model"
	"spartan-scraper/internal/scheduler"
)

func (s *Server) handleSchedules(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		schedules, err := scheduler.List(s.cfg.DataDir)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		response := make([]ScheduleResponse, len(schedules))
		for i, sched := range schedules {
			response[i] = ScheduleResponse{
				ID:              sched.ID,
				Kind:            string(sched.Kind),
				IntervalSeconds: sched.IntervalSeconds,
				NextRun:         sched.NextRun.Format(time.RFC3339),
				Params:          sched.Params,
			}
		}
		writeJSON(w, map[string]interface{}{"schedules": response})
		return
	}
	if r.Method == http.MethodPost {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			writeJSONError(w, http.StatusUnsupportedMediaType, "content-type must be application/json")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		var req ScheduleRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid json: "+err.Error())
			return
		}
		if req.Kind == "" {
			writeJSONError(w, http.StatusBadRequest, "kind is required")
			return
		}
		if req.IntervalSeconds <= 0 {
			writeJSONError(w, http.StatusBadRequest, "intervalSeconds must be positive")
			return
		}
		if req.Kind != "scrape" && req.Kind != "crawl" && req.Kind != "research" {
			writeJSONError(w, http.StatusBadRequest, "kind must be scrape, crawl, or research")
			return
		}

		params := make(map[string]interface{})
		if req.URL != nil {
			params["url"] = *req.URL
		}
		if req.Query != nil {
			params["query"] = *req.Query
		}
		if len(req.URLs) > 0 {
			params["urls"] = req.URLs
		}
		if req.MaxDepth != nil {
			params["maxDepth"] = *req.MaxDepth
		}
		if req.MaxPages != nil {
			params["maxPages"] = *req.MaxPages
		}
		params["headless"] = req.Headless
		if req.Playwright != nil {
			params["playwright"] = *req.Playwright
		}
		params["timeout"] = req.TimeoutSeconds
		if req.AuthProfile != nil {
			params["authProfile"] = *req.AuthProfile
		}
		if req.Auth != nil {
			params["headers"] = toHeaderKVs(req.Auth.Headers)
			params["cookies"] = toCookies(req.Auth.Cookies)
			params["tokens"] = tokensFromOverride(*req.Auth)
			if login := loginFromOverride(*req.Auth); login != nil {
				params["login"] = login
			}
		}
		if req.Extract != nil {
			params["extractTemplate"] = req.Extract.Template
			params["extractValidate"] = req.Extract.Validate
		}
		if req.Pipeline != nil {
			params["pipeline"] = *req.Pipeline
		}
		if req.Incremental != nil {
			params["incremental"] = *req.Incremental
		}

		schedule := scheduler.Schedule{
			Kind:            model.Kind(req.Kind),
			IntervalSeconds: req.IntervalSeconds,
			Params:          params,
		}

		addedSchedule, err := scheduler.Add(s.cfg.DataDir, schedule)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeJSON(w, ScheduleResponse{
			ID:              addedSchedule.ID,
			Kind:            string(addedSchedule.Kind),
			IntervalSeconds: addedSchedule.IntervalSeconds,
			NextRun:         addedSchedule.NextRun.Format(time.RFC3339),
			Params:          addedSchedule.Params,
		})
		return
	}
	writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
}

func (s *Server) handleSchedule(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSuffix(r.URL.Path, "/")
	id := filepath.Base(path)
	if id == "" || id == "schedules" {
		writeJSONError(w, http.StatusBadRequest, "id required")
		return
	}
	if r.Method != http.MethodDelete {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if err := scheduler.Delete(s.cfg.DataDir, id); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}
