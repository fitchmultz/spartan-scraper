// Package api provides HTTP handlers for watch monitoring endpoints.
//
// Purpose:
// - Expose CRUD and manual-check endpoints for configured watches.
//
// Responsibilities:
// - Decode and normalize watch API requests.
// - Map storage not-found errors to stable HTTP responses.
// - Return consistent watch and check-result payloads.
//
// Scope:
// - `/v1/watch` and nested watch detail routes only.
//
// Usage:
// - Mounted by Server.Routes for watch management and manual checks.
//
// Invariants/Assumptions:
// - Create requests apply schema defaults for omitted optionals.
// - Update requests preserve existing optional values when they are omitted.
package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/submission"
	"github.com/fitchmultz/spartan-scraper/internal/watch"
)

// WatchRequest represents the request body for creating/updating a watch.
type WatchRequest struct {
	URL                 string                  `json:"url"`
	Selector            string                  `json:"selector,omitempty"`
	IntervalSeconds     int                     `json:"intervalSeconds"`
	Enabled             *bool                   `json:"enabled,omitempty"`
	DiffFormat          string                  `json:"diffFormat,omitempty"`
	WebhookConfig       *model.WebhookSpec      `json:"webhookConfig,omitempty"`
	NotifyOnChange      bool                    `json:"notifyOnChange"`
	MinChangeSize       int                     `json:"minChangeSize,omitempty"`
	IgnorePatterns      []string                `json:"ignorePatterns,omitempty"`
	Headless            bool                    `json:"headless"`
	UsePlaywright       bool                    `json:"usePlaywright"`
	ExtractMode         string                  `json:"extractMode,omitempty"`
	ScreenshotEnabled   bool                    `json:"screenshotEnabled,omitempty"`
	ScreenshotConfig    *fetch.ScreenshotConfig `json:"screenshotConfig,omitempty"`
	VisualDiffThreshold *float64                `json:"visualDiffThreshold,omitempty"`
	JobTrigger          *watch.JobTrigger       `json:"jobTrigger,omitempty"`
}

// WatchResponse represents a watch in API responses.
type WatchResponse struct {
	ID                  string                  `json:"id"`
	URL                 string                  `json:"url"`
	Selector            string                  `json:"selector,omitempty"`
	IntervalSeconds     int                     `json:"intervalSeconds"`
	Enabled             bool                    `json:"enabled"`
	CreatedAt           time.Time               `json:"createdAt"`
	LastCheckedAt       time.Time               `json:"lastCheckedAt,omitempty"`
	LastChangedAt       time.Time               `json:"lastChangedAt,omitempty"`
	ChangeCount         int                     `json:"changeCount"`
	DiffFormat          string                  `json:"diffFormat"`
	WebhookConfig       *model.WebhookSpec      `json:"webhookConfig,omitempty"`
	NotifyOnChange      bool                    `json:"notifyOnChange"`
	MinChangeSize       int                     `json:"minChangeSize,omitempty"`
	IgnorePatterns      []string                `json:"ignorePatterns,omitempty"`
	Headless            bool                    `json:"headless"`
	UsePlaywright       bool                    `json:"usePlaywright"`
	ExtractMode         string                  `json:"extractMode,omitempty"`
	Status              string                  `json:"status"`
	ScreenshotEnabled   bool                    `json:"screenshotEnabled,omitempty"`
	ScreenshotConfig    *fetch.ScreenshotConfig `json:"screenshotConfig,omitempty"`
	VisualDiffThreshold float64                 `json:"visualDiffThreshold,omitempty"`
	JobTrigger          *watch.JobTrigger       `json:"jobTrigger,omitempty"`
}

// WatchCheckResponse represents the result of a watch check.
type WatchCheckResponse struct {
	WatchID            string                  `json:"watchId"`
	URL                string                  `json:"url"`
	CheckedAt          time.Time               `json:"checkedAt"`
	Changed            bool                    `json:"changed"`
	PreviousHash       string                  `json:"previousHash,omitempty"`
	CurrentHash        string                  `json:"currentHash,omitempty"`
	DiffText           string                  `json:"diffText,omitempty"`
	DiffHTML           string                  `json:"diffHtml,omitempty"`
	Error              string                  `json:"error,omitempty"`
	Selector           string                  `json:"selector,omitempty"`
	Artifacts          []WatchArtifactResponse `json:"artifacts,omitempty"`
	VisualHash         string                  `json:"visualHash,omitempty"`
	PreviousVisualHash string                  `json:"previousVisualHash,omitempty"`
	VisualChanged      bool                    `json:"visualChanged"`
	VisualSimilarity   float64                 `json:"visualSimilarity,omitempty"`
	TriggeredJobs      []string                `json:"triggeredJobs,omitempty"`
}

// toWatchResponse converts a watch.Watch to WatchResponse.
func toWatchResponse(w watch.Watch) WatchResponse {
	status := "active"
	if !w.Enabled {
		status = "disabled"
	}
	return WatchResponse{
		ID:                  w.ID,
		URL:                 w.URL,
		Selector:            w.Selector,
		IntervalSeconds:     w.IntervalSeconds,
		Enabled:             w.Enabled,
		CreatedAt:           w.CreatedAt,
		LastCheckedAt:       w.LastCheckedAt,
		LastChangedAt:       w.LastChangedAt,
		ChangeCount:         w.ChangeCount,
		DiffFormat:          w.DiffFormat,
		WebhookConfig:       w.WebhookConfig,
		NotifyOnChange:      w.NotifyOnChange,
		MinChangeSize:       w.MinChangeSize,
		IgnorePatterns:      w.IgnorePatterns,
		Headless:            w.Headless,
		UsePlaywright:       w.UsePlaywright,
		ExtractMode:         w.ExtractMode,
		Status:              status,
		ScreenshotEnabled:   w.ScreenshotEnabled,
		ScreenshotConfig:    w.ScreenshotConfig,
		VisualDiffThreshold: w.VisualDiffThreshold,
		JobTrigger:          w.JobTrigger,
	}
}

func buildWatchFromRequest(req WatchRequest) *watch.Watch {
	if req.IntervalSeconds <= 0 {
		req.IntervalSeconds = 3600
	}
	if strings.TrimSpace(req.DiffFormat) == "" {
		req.DiffFormat = "unified"
	}
	threshold := valueOr(req.VisualDiffThreshold, 0.1)

	return &watch.Watch{
		URL:                 req.URL,
		Selector:            req.Selector,
		IntervalSeconds:     req.IntervalSeconds,
		Enabled:             valueOr(req.Enabled, true),
		DiffFormat:          req.DiffFormat,
		WebhookConfig:       req.WebhookConfig,
		NotifyOnChange:      req.NotifyOnChange,
		MinChangeSize:       req.MinChangeSize,
		IgnorePatterns:      req.IgnorePatterns,
		Headless:            req.Headless,
		UsePlaywright:       req.UsePlaywright,
		ExtractMode:         req.ExtractMode,
		ScreenshotEnabled:   req.ScreenshotEnabled,
		ScreenshotConfig:    req.ScreenshotConfig,
		VisualDiffThreshold: threshold,
		JobTrigger:          req.JobTrigger,
	}
}

func validateWatchJobTrigger(cfg config.Config, defaults submission.Defaults, trigger *watch.JobTrigger) error {
	if trigger == nil {
		return nil
	}
	if len(trigger.Request) == 0 {
		return apperrors.Validation("jobTrigger.request is required when jobTrigger is set")
	}
	normalizedRequest, err := submission.NormalizeRawRequest(trigger.Kind, trigger.Request)
	if err != nil {
		return err
	}
	if _, _, err := submission.JobSpecFromRawRequest(cfg, defaults, trigger.Kind, normalizedRequest); err != nil {
		return err
	}
	trigger.Request = normalizedRequest
	return nil
}

func applyWatchRequest(existing *watch.Watch, req WatchRequest) {
	existing.URL = req.URL
	existing.Selector = req.Selector
	if req.IntervalSeconds > 0 {
		existing.IntervalSeconds = req.IntervalSeconds
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if strings.TrimSpace(req.DiffFormat) != "" {
		existing.DiffFormat = req.DiffFormat
	}
	existing.WebhookConfig = req.WebhookConfig
	existing.NotifyOnChange = req.NotifyOnChange
	existing.MinChangeSize = req.MinChangeSize
	existing.IgnorePatterns = req.IgnorePatterns
	existing.Headless = req.Headless
	existing.UsePlaywright = req.UsePlaywright
	existing.ExtractMode = req.ExtractMode
	existing.ScreenshotEnabled = req.ScreenshotEnabled
	existing.ScreenshotConfig = req.ScreenshotConfig
	if req.VisualDiffThreshold != nil {
		existing.VisualDiffThreshold = *req.VisualDiffThreshold
	}
	existing.JobTrigger = req.JobTrigger
}

// handleWatchCheckWrapper routes nested watch detail paths.
func (s *Server) handleWatchCheckWrapper(w http.ResponseWriter, r *http.Request) {
	if strings.Contains(strings.TrimSuffix(r.URL.Path, "/"), "/artifacts/") {
		s.handleWatchArtifact(w, r)
		return
	}
	if handlePathSuffix(r.URL.Path, "/check", func() {
		s.handleWatchCheck(w, r)
	}) {
		return
	}
	s.handleWatch(w, r)
}

// handleWatches handles requests to /v1/watch
func (s *Server) handleWatches(w http.ResponseWriter, r *http.Request) {
	handleCollectionMethods(w, r, func() {
		s.handleListWatches(w, r)
	}, func() {
		s.handleCreateWatch(w, r)
	})
}

// handleListWatches lists all watches.
func (s *Server) handleListWatches(w http.ResponseWriter, r *http.Request) {
	storage := watch.NewFileStorage(s.cfg.DataDir)
	watches, err := storage.List()
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeCollectionJSON(w, "watches", mapSlice(watches, toWatchResponse))
}

// handleCreateWatch creates a new watch.
func (s *Server) handleCreateWatch(w http.ResponseWriter, r *http.Request) {
	var req WatchRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	newWatch := buildWatchFromRequest(req)
	if err := validateWatchJobTrigger(s.cfg, s.nonResolvingSubmissionDefaults(), newWatch.JobTrigger); err != nil {
		writeError(w, r, err)
		return
	}

	if err := newWatch.Validate(); err != nil {
		writeError(w, r, apperrors.Validation(err.Error()))
		return
	}

	storage := watch.NewFileStorage(s.cfg.DataDir)
	created, err := storage.Add(newWatch)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeCreatedJSON(w, toWatchResponse(*created))
}

// handleWatch handles requests to /v1/watch/{id}
func (s *Server) handleWatch(w http.ResponseWriter, r *http.Request) {
	id, err := requireResourceID(r, "watch", "watch id")
	if err != nil {
		writeError(w, r, err)
		return
	}
	handleDetailMethods(w, r, func() {
		s.handleGetWatch(w, r, id)
	}, func() {
		s.handleUpdateWatch(w, r, id)
	}, func() {
		s.handleDeleteWatch(w, r, id)
	})
}

// handleGetWatch retrieves a single watch.
func (s *Server) handleGetWatch(w http.ResponseWriter, r *http.Request, id string) {
	storage := watch.NewFileStorage(s.cfg.DataDir)
	watchItem, ok := getStoredResource(w, r, func() (*watch.Watch, error) {
		return storage.Get(id)
	}, watch.IsNotFoundError, "watch")
	if !ok {
		return
	}
	writeJSON(w, toWatchResponse(*watchItem))
}

// handleUpdateWatch updates an existing watch.
func (s *Server) handleUpdateWatch(w http.ResponseWriter, r *http.Request, id string) {
	var req WatchRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	storage := watch.NewFileStorage(s.cfg.DataDir)
	existing, ok := getStoredResource(w, r, func() (*watch.Watch, error) {
		return storage.Get(id)
	}, watch.IsNotFoundError, "watch")
	if !ok {
		return
	}

	applyWatchRequest(existing, req)
	if err := validateWatchJobTrigger(s.cfg, s.nonResolvingSubmissionDefaults(), existing.JobTrigger); err != nil {
		writeError(w, r, err)
		return
	}

	if err := existing.Validate(); err != nil {
		writeError(w, r, apperrors.Validation(err.Error()))
		return
	}

	if err := storage.Update(existing); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, toWatchResponse(*existing))
}

// handleDeleteWatch deletes a watch.
func (s *Server) handleDeleteWatch(w http.ResponseWriter, r *http.Request, id string) {
	storage := watch.NewFileStorage(s.cfg.DataDir)
	deleteStoredResource(w, r, func() error { return storage.Delete(id) }, watch.IsNotFoundError, "watch")
}

// handleWatchCheck handles requests to /v1/watch/{id}/check
func (s *Server) handleWatchCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	id, err := requireResourceID(r, "watch", "watch id")
	if err != nil {
		writeError(w, r, err)
		return
	}

	storage := watch.NewFileStorage(s.cfg.DataDir)
	watchItem, ok := getStoredResource(w, r, func() (*watch.Watch, error) {
		return storage.Get(id)
	}, watch.IsNotFoundError, "watch")
	if !ok {
		return
	}

	// Open store for crawl state
	stateStore, err := store.Open(s.cfg.DataDir)
	if err != nil {
		writeError(w, r, apperrors.Internal("failed to open state store"))
		return
	}
	defer stateStore.Close()

	// Create watcher with optional webhook dispatcher and shared job-trigger runtime
	watcher := watch.NewWatcher(storage, stateStore, s.cfg.DataDir, s.webhookDispatcher, &watch.TriggerRuntime{
		Config:  s.cfg,
		Manager: s.manager,
	})

	// Perform check
	result, err := watcher.Check(r.Context(), watchItem)
	if err != nil {
		// Return result even on error (error is in result.Error)
		writeJSON(w, toWatchCheckResponse(result))
		return
	}

	writeJSON(w, toWatchCheckResponse(result))
}
