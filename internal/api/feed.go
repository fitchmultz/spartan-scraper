// Package api provides HTTP handlers for feed monitoring endpoints.
// Feed handlers support CRUD operations and manual check triggering
// for RSS/Atom feed monitoring.
package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/feed"
)

// FeedRequest represents the request body for creating/updating a feed.
type FeedRequest struct {
	URL             string            `json:"url"`
	FeedType        string            `json:"feedType,omitempty"`
	IntervalSeconds int               `json:"intervalSeconds"`
	Enabled         bool              `json:"enabled"`
	AutoScrape      bool              `json:"autoScrape"`
	ExtractOptions  map[string]string `json:"extractOptions,omitempty"`
}

// FeedResponse represents a feed in API responses.
type FeedResponse struct {
	ID                  string            `json:"id"`
	URL                 string            `json:"url"`
	FeedType            string            `json:"feedType"`
	IntervalSeconds     int               `json:"intervalSeconds"`
	Enabled             bool              `json:"enabled"`
	AutoScrape          bool              `json:"autoScrape"`
	ExtractOptions      map[string]string `json:"extractOptions,omitempty"`
	CreatedAt           time.Time         `json:"createdAt"`
	LastCheckedAt       time.Time         `json:"lastCheckedAt,omitempty"`
	LastError           string            `json:"lastError,omitempty"`
	ConsecutiveFailures int               `json:"consecutiveFailures"`
	Status              string            `json:"status"`
}

// FeedItemResponse represents a feed item in API responses.
type FeedItemResponse struct {
	GUID        string    `json:"guid"`
	Title       string    `json:"title"`
	Link        string    `json:"link"`
	Description string    `json:"description,omitempty"`
	PubDate     time.Time `json:"pubDate,omitempty"`
	Author      string    `json:"author,omitempty"`
	Categories  []string  `json:"categories,omitempty"`
	SeenAt      time.Time `json:"seenAt"`
}

// FeedCheckResponse represents the result of a feed check.
type FeedCheckResponse struct {
	FeedID     string             `json:"feedId"`
	CheckedAt  time.Time          `json:"checkedAt"`
	NewItems   []FeedItemResponse `json:"newItems"`
	TotalItems int                `json:"totalItems"`
	FeedTitle  string             `json:"feedTitle,omitempty"`
	FeedDesc   string             `json:"feedDesc,omitempty"`
	Error      string             `json:"error,omitempty"`
}

// toFeedResponse converts a feed.Feed to FeedResponse.
func toFeedResponse(f feed.Feed) FeedResponse {
	return FeedResponse{
		ID:                  f.ID,
		URL:                 f.URL,
		FeedType:            string(f.FeedType),
		IntervalSeconds:     f.IntervalSeconds,
		Enabled:             f.Enabled,
		AutoScrape:          f.AutoScrape,
		ExtractOptions:      f.ExtractOptions,
		CreatedAt:           f.CreatedAt,
		LastCheckedAt:       f.LastCheckedAt,
		LastError:           f.LastError,
		ConsecutiveFailures: f.ConsecutiveFailures,
		Status:              f.GetStatus(),
	}
}

// toFeedItemResponse converts a feed.SeenItem to FeedItemResponse.
func toFeedItemResponse(item feed.SeenItem) FeedItemResponse {
	return FeedItemResponse{
		GUID:   item.GUID,
		Title:  item.Title,
		Link:   item.Link,
		SeenAt: item.SeenAt,
	}
}

// handleFeeds handles requests to /v1/feeds
func (s *Server) handleFeeds(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListFeeds(w, r)
	case http.MethodPost:
		s.handleCreateFeed(w, r)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

// handleFeedDetailWrapper routes to handleFeedDetail or handleFeedCheck based on path
func (s *Server) handleFeedDetailWrapper(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if strings.HasSuffix(path, "/check") {
		s.handleFeedCheck(w, r)
		return
	}
	if strings.HasSuffix(path, "/items") {
		s.handleFeedItems(w, r)
		return
	}
	s.handleFeedDetail(w, r)
}

// handleListFeeds lists all feeds.
func (s *Server) handleListFeeds(w http.ResponseWriter, r *http.Request) {
	storage := feed.NewFileStorage(s.cfg.DataDir)
	feeds, err := storage.List()
	if err != nil {
		writeError(w, r, err)
		return
	}

	response := make([]FeedResponse, len(feeds))
	for i, f := range feeds {
		response[i] = toFeedResponse(f)
	}
	writeJSON(w, map[string]interface{}{"feeds": response})
}

// handleCreateFeed creates a new feed.
func (s *Server) handleCreateFeed(w http.ResponseWriter, r *http.Request) {
	var req FeedRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	// Set defaults
	if req.IntervalSeconds == 0 {
		req.IntervalSeconds = 3600
	}
	if req.FeedType == "" {
		req.FeedType = "auto"
	}

	newFeed := &feed.Feed{
		URL:             req.URL,
		FeedType:        feed.FeedType(req.FeedType),
		IntervalSeconds: req.IntervalSeconds,
		Enabled:         req.Enabled,
		AutoScrape:      req.AutoScrape,
		ExtractOptions:  req.ExtractOptions,
	}

	if err := newFeed.Validate(); err != nil {
		writeError(w, r, apperrors.Validation(err.Error()))
		return
	}

	storage := feed.NewFileStorage(s.cfg.DataDir)
	created, err := storage.Add(newFeed)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeCreatedJSON(w, toFeedResponse(*created))
}

// handleFeedDetail handles requests to /v1/feeds/{id}
func (s *Server) handleFeedDetail(w http.ResponseWriter, r *http.Request) {
	id := extractID(r.URL.Path, "feeds")
	if id == "" {
		writeError(w, r, apperrors.Validation("id required"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetFeed(w, r, id)
	case http.MethodPut:
		s.handleUpdateFeed(w, r, id)
	case http.MethodDelete:
		s.handleDeleteFeed(w, r, id)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

// handleGetFeed retrieves a single feed.
func (s *Server) handleGetFeed(w http.ResponseWriter, r *http.Request, id string) {
	storage := feed.NewFileStorage(s.cfg.DataDir)
	feedItem, err := storage.Get(id)
	if err != nil {
		if feed.IsNotFoundError(err) {
			writeError(w, r, apperrors.NotFound("feed not found"))
			return
		}
		writeError(w, r, err)
		return
	}
	writeJSON(w, toFeedResponse(*feedItem))
}

// handleUpdateFeed updates an existing feed.
func (s *Server) handleUpdateFeed(w http.ResponseWriter, r *http.Request, id string) {
	var req FeedRequest
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	// Get existing feed to preserve createdAt and other immutable fields
	storage := feed.NewFileStorage(s.cfg.DataDir)
	existing, err := storage.Get(id)
	if err != nil {
		if feed.IsNotFoundError(err) {
			writeError(w, r, apperrors.NotFound("feed not found"))
			return
		}
		writeError(w, r, err)
		return
	}

	// Update fields
	existing.URL = req.URL
	existing.FeedType = feed.FeedType(req.FeedType)
	existing.IntervalSeconds = req.IntervalSeconds
	existing.Enabled = req.Enabled
	existing.AutoScrape = req.AutoScrape
	existing.ExtractOptions = req.ExtractOptions

	if err := existing.Validate(); err != nil {
		writeError(w, r, apperrors.Validation(err.Error()))
		return
	}

	if err := storage.Update(existing); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, toFeedResponse(*existing))
}

// handleDeleteFeed deletes a feed.
func (s *Server) handleDeleteFeed(w http.ResponseWriter, r *http.Request, id string) {
	storage := feed.NewFileStorage(s.cfg.DataDir)
	// Check if feed exists first
	if _, err := storage.Get(id); err != nil {
		if feed.IsNotFoundError(err) {
			writeError(w, r, apperrors.NotFound("feed not found"))
			return
		}
		writeError(w, r, err)
		return
	}
	if err := storage.Delete(id); err != nil {
		writeError(w, r, err)
		return
	}
	writeNoContent(w)
}

// handleFeedCheck handles requests to /v1/feeds/{id}/check
func (s *Server) handleFeedCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	id := extractID(r.URL.Path, "feeds")
	if id == "" {
		writeError(w, r, apperrors.Validation("id required"))
		return
	}

	storage := feed.NewFileStorage(s.cfg.DataDir)
	feedItem, err := storage.Get(id)
	if err != nil {
		if feed.IsNotFoundError(err) {
			writeError(w, r, apperrors.NotFound("feed not found"))
			return
		}
		writeError(w, r, err)
		return
	}

	// Create seen storage and checker
	seenStorage := feed.NewFileSeenStorage(s.cfg.DataDir)
	checker := feed.NewChecker(storage, seenStorage, s.manager)

	// Perform check
	result, err := checker.Check(r.Context(), feedItem)
	if err != nil {
		// Return result even on error (error is in result.Error)
		writeJSON(w, FeedCheckResponse{
			FeedID:     result.FeedID,
			CheckedAt:  result.CheckedAt,
			NewItems:   []FeedItemResponse{},
			TotalItems: result.TotalItems,
			FeedTitle:  result.FeedTitle,
			FeedDesc:   result.FeedDesc,
			Error:      result.Error,
		})
		return
	}

	// Convert new items to response format
	newItems := make([]FeedItemResponse, len(result.NewItems))
	for i, item := range result.NewItems {
		newItems[i] = FeedItemResponse{
			GUID:        item.GUID,
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			PubDate:     item.PubDate,
			Author:      item.Author,
			Categories:  item.Categories,
		}
	}

	writeJSON(w, FeedCheckResponse{
		FeedID:     result.FeedID,
		CheckedAt:  result.CheckedAt,
		NewItems:   newItems,
		TotalItems: result.TotalItems,
		FeedTitle:  result.FeedTitle,
		FeedDesc:   result.FeedDesc,
	})
}

// handleFeedItems handles requests to /v1/feeds/{id}/items
func (s *Server) handleFeedItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	id := extractID(r.URL.Path, "feeds")
	if id == "" {
		writeError(w, r, apperrors.Validation("id required"))
		return
	}

	// Check if feed exists
	storage := feed.NewFileStorage(s.cfg.DataDir)
	if _, err := storage.Get(id); err != nil {
		if feed.IsNotFoundError(err) {
			writeError(w, r, apperrors.NotFound("feed not found"))
			return
		}
		writeError(w, r, err)
		return
	}

	// Get seen items
	seenStorage := feed.NewFileSeenStorage(s.cfg.DataDir)
	items, err := seenStorage.GetSeen(id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	response := make([]FeedItemResponse, len(items))
	for i, item := range items {
		response[i] = toFeedItemResponse(item)
	}

	writeJSON(w, map[string]interface{}{"items": response})
}
