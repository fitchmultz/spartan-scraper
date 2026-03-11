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
	"encoding/json"
	"net/http"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/scheduler"
)

func (s *Server) handleSchedules(w http.ResponseWriter, r *http.Request) {
	handleCollectionMethods(w, r, func() {
		s.handleListSchedules(w, r)
	}, func() {
		s.handleCreateSchedule(w, r)
	})
}

func (s *Server) handleSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := requireResourceID(r, "schedules", "schedule id")
	if err != nil {
		writeError(w, r, err)
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

func (s *Server) handleListSchedules(w http.ResponseWriter, r *http.Request) {
	schedules, err := scheduler.List(s.cfg.DataDir)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeCollectionJSON(w, "schedules", mapSlice(schedules, toScheduleResponse))
}

func (s *Server) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	var req ScheduleRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	schedule, err := scheduleFromRequest(req)
	if err != nil {
		writeError(w, r, err)
		return
	}

	addedSchedule, err := scheduler.Add(s.cfg.DataDir, schedule)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeCreatedJSON(w, toScheduleResponse(*addedSchedule))
}

func toScheduleResponse(schedule scheduler.Schedule) ScheduleResponse {
	return ScheduleResponse{
		ID:              schedule.ID,
		Kind:            string(schedule.Kind),
		IntervalSeconds: schedule.IntervalSeconds,
		NextRun:         schedule.NextRun.Format(time.RFC3339),
		SpecVersion:     schedule.SpecVersion,
		Spec:            schedule.Spec,
	}
}

func scheduleFromRequest(req ScheduleRequest) (scheduler.Schedule, error) {
	if req.Kind == "" {
		return scheduler.Schedule{}, apperrors.Validation("kind is required")
	}
	if req.IntervalSeconds <= 0 {
		return scheduler.Schedule{}, apperrors.Validation("intervalSeconds must be positive")
	}
	if req.SpecVersion == 0 {
		return scheduler.Schedule{}, apperrors.Validation("specVersion is required")
	}
	if len(req.Spec) == 0 {
		return scheduler.Schedule{}, apperrors.Validation("spec is required")
	}
	kind := model.Kind(req.Kind)
	switch kind {
	case model.KindScrape, model.KindCrawl, model.KindResearch:
	default:
		return scheduler.Schedule{}, apperrors.Validation("kind must be scrape, crawl, or research")
	}

	spec, err := model.DecodeJobSpec(kind, req.SpecVersion, req.Spec)
	if err != nil {
		return scheduler.Schedule{}, err
	}
	raw, err := json.Marshal(spec)
	if err != nil {
		return scheduler.Schedule{}, apperrors.Wrap(apperrors.KindInternal, "failed to normalize schedule spec", err)
	}
	normalizedSpec, err := model.DecodeJobSpec(kind, req.SpecVersion, raw)
	if err != nil {
		return scheduler.Schedule{}, err
	}

	return scheduler.Schedule{
		Kind:            kind,
		IntervalSeconds: req.IntervalSeconds,
		SpecVersion:     req.SpecVersion,
		Spec:            normalizedSpec,
	}, nil
}
