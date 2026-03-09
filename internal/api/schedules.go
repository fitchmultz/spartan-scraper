// Package api provides HTTP handlers for scheduled job management endpoints.
//
// Purpose:
// - Manage recurring scrape, crawl, and research schedules over HTTP.
//
// Responsibilities:
// - Validate schedule creation inputs.
// - Translate API requests into scheduler storage records.
// - Return consistent JSON responses for list/create/delete operations.
//
// Scope:
// - Direct schedule CRUD handlers in the API package.
//
// Usage:
// - Mounted under `/v1/schedules` and `/v1/schedules/{id}` by the API router.
//
// Invariants/Assumptions:
// - Schedule creation requests use strict JSON decoding via shared helpers.
// - Supported kinds are scrape, crawl, and research.
package api

import (
	"net/http"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
)

func (s *Server) handleSchedules(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		schedules, err := scheduler.List(s.cfg.DataDir)
		if err != nil {
			writeError(w, r, err)
			return
		}
		response := make([]ScheduleResponse, len(schedules))
		for i, sched := range schedules {
			response[i] = toScheduleResponse(sched)
		}
		writeCollectionJSON(w, "schedules", response)
		return
	}
	if r.Method == http.MethodPost {
		var req ScheduleRequest
		if err := decodeJSONBody(w, r, &req); err != nil {
			writeError(w, r, err)
			return
		}
		if req.Kind == "" {
			writeError(w, r, apperrors.Validation("kind is required"))
			return
		}
		if req.IntervalSeconds <= 0 {
			writeError(w, r, apperrors.Validation("intervalSeconds must be positive"))
			return
		}
		if req.Kind != "scrape" && req.Kind != "crawl" && req.Kind != "research" {
			writeError(w, r, apperrors.Validation("kind must be scrape, crawl, or research"))
			return
		}

		schedule := scheduler.Schedule{
			Kind:            model.Kind(req.Kind),
			IntervalSeconds: req.IntervalSeconds,
			Params:          scheduleParamsFromRequest(req),
		}

		addedSchedule, err := scheduler.Add(s.cfg.DataDir, schedule)
		if err != nil {
			writeError(w, r, err)
			return
		}

		writeCreatedJSON(w, toScheduleResponse(*addedSchedule))
		return
	}
	writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
}

func (s *Server) handleSchedule(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path, "schedules")
	if id == "" {
		writeError(w, r, apperrors.Validation("id required"))
		return
	}
	if r.Method != http.MethodDelete {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}
	if err := scheduler.Delete(s.cfg.DataDir, id); err != nil {
		writeError(w, r, err)
		return
	}
	writeOKStatus(w)
}

func toScheduleResponse(schedule scheduler.Schedule) ScheduleResponse {
	return ScheduleResponse{
		ID:              schedule.ID,
		Kind:            string(schedule.Kind),
		IntervalSeconds: schedule.IntervalSeconds,
		NextRun:         schedule.NextRun.Format(time.RFC3339),
		Params:          schedule.Params,
	}
}

func scheduleParamsFromRequest(req ScheduleRequest) map[string]interface{} {
	params := map[string]interface{}{
		"headless": req.Headless,
		"timeout":  req.TimeoutSeconds,
	}
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
	if req.Playwright != nil {
		params["playwright"] = *req.Playwright
	}
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
	if req.Incremental != nil && req.Kind != "research" {
		params["incremental"] = *req.Incremental
	}
	if req.SitemapURL != nil && req.Kind == "crawl" {
		params["sitemapURL"] = *req.SitemapURL
	}
	if req.SitemapOnly != nil && req.Kind == "crawl" {
		params["sitemapOnly"] = *req.SitemapOnly
	}
	if req.Screenshot != nil {
		params["screenshot"] = *req.Screenshot
	}
	if req.Device != nil {
		params["device"] = *req.Device
	}
	return params
}
