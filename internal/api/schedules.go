// Package api provides HTTP handlers for scheduled job management endpoints.
//
// Purpose:
// - Manage recurring scrape, crawl, and research schedules over HTTP.
//
// Responsibilities:
// - Validate schedule creation inputs.
// - Translate operator-facing job requests into scheduler storage records.
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
	responses := make([]ScheduleResponse, 0, len(schedules))
	for _, item := range schedules {
		resp, err := toScheduleResponse(item)
		if err != nil {
			writeError(w, r, err)
			return
		}
		responses = append(responses, resp)
	}
	writeCollectionJSON(w, "schedules", responses)
}

func (s *Server) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	var req ScheduleRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	schedule, err := s.scheduleFromRequest(req)
	if err != nil {
		writeError(w, r, err)
		return
	}

	addedSchedule, err := scheduler.Add(s.cfg.DataDir, schedule)
	if err != nil {
		writeError(w, r, err)
		return
	}

	resp, err := toScheduleResponse(*addedSchedule)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeCreatedJSON(w, resp)
}

func toScheduleResponse(schedule scheduler.Schedule) (ScheduleResponse, error) {
	request, err := requestFromSchedule(schedule)
	if err != nil {
		return ScheduleResponse{}, err
	}
	return ScheduleResponse{
		ID:              schedule.ID,
		Kind:            string(schedule.Kind),
		IntervalSeconds: schedule.IntervalSeconds,
		NextRun:         schedule.NextRun.Format(time.RFC3339),
		Request:         request,
	}, nil
}

func (s *Server) scheduleFromRequest(req ScheduleRequest) (scheduler.Schedule, error) {
	if req.Kind == "" {
		return scheduler.Schedule{}, apperrors.Validation("kind is required")
	}
	if req.IntervalSeconds <= 0 {
		return scheduler.Schedule{}, apperrors.Validation("intervalSeconds must be positive")
	}
	if len(req.Request) == 0 {
		return scheduler.Schedule{}, apperrors.Validation("request is required")
	}
	kind := model.Kind(req.Kind)
	switch kind {
	case model.KindScrape, model.KindCrawl, model.KindResearch:
	default:
		return scheduler.Schedule{}, apperrors.Validation("kind must be scrape, crawl, or research")
	}

	_, specVersion, typedSpec, err := convertScheduleRequestToTypedSpec(s, kind, req.Request)
	if err != nil {
		return scheduler.Schedule{}, err
	}

	return scheduler.Schedule{
		Kind:            kind,
		IntervalSeconds: req.IntervalSeconds,
		SpecVersion:     specVersion,
		Spec:            typedSpec,
	}, nil
}
