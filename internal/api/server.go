// Package api implements the REST API server for Spartan Scraper.
// It provides endpoints for enqueuing jobs, managing auth profiles,
// and retrieving job status and results.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"spartan-scraper/internal/auth"
	"spartan-scraper/internal/config"
	"spartan-scraper/internal/extract"
	"spartan-scraper/internal/fetch"
	"spartan-scraper/internal/jobs"
	"spartan-scraper/internal/pipeline"
	"spartan-scraper/internal/store"
)

const maxRequestBodySize = 1024 * 1024 // 1MB

// Server implements the HTTP API for Spartan Scraper.
type Server struct {
	manager *jobs.Manager
	store   *store.Store
	cfg     config.Config
}

// ComponentStatus represents the health of a single system component.
type ComponentStatus struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

// HealthResponse represents the overall health of the system.
type HealthResponse struct {
	Status     string                     `json:"status"`
	Components map[string]ComponentStatus `json:"components"`
}

// ErrorResponse represents a standard error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// ScrapeRequest represents a request to scrape a single page.
type ScrapeRequest struct {
	URL            string                  `json:"url"`
	Headless       bool                    `json:"headless"`
	Playwright     *bool                   `json:"playwright"`
	TimeoutSeconds int                     `json:"timeoutSeconds"`
	AuthProfile    string                  `json:"authProfile,omitempty"`
	Auth           *fetch.AuthOptions      `json:"auth"`
	Extract        *extract.ExtractOptions `json:"extract"`
	Pipeline       *pipeline.Options       `json:"pipeline"`
	Incremental    *bool                   `json:"incremental"`
}

// CrawlRequest represents a request to crawl a website.
type CrawlRequest struct {
	URL            string                  `json:"url"`
	MaxDepth       int                     `json:"maxDepth"`
	MaxPages       int                     `json:"maxPages"`
	Headless       bool                    `json:"headless"`
	Playwright     *bool                   `json:"playwright"`
	TimeoutSeconds int                     `json:"timeoutSeconds"`
	AuthProfile    string                  `json:"authProfile,omitempty"`
	Auth           *fetch.AuthOptions      `json:"auth"`
	Extract        *extract.ExtractOptions `json:"extract"`
	Pipeline       *pipeline.Options       `json:"pipeline"`
	Incremental    *bool                   `json:"incremental"`
}

// ResearchRequest represents a request to perform deep research across multiple URLs.
type ResearchRequest struct {
	Query          string                  `json:"query"`
	URLs           []string                `json:"urls"`
	MaxDepth       int                     `json:"maxDepth"`
	MaxPages       int                     `json:"maxPages"`
	Headless       bool                    `json:"headless"`
	Playwright     *bool                   `json:"playwright"`
	TimeoutSeconds int                     `json:"timeoutSeconds"`
	AuthProfile    string                  `json:"authProfile,omitempty"`
	Auth           *fetch.AuthOptions      `json:"auth"`
	Extract        *extract.ExtractOptions `json:"extract"`
	Pipeline       *pipeline.Options       `json:"pipeline"`
	Incremental    *bool                   `json:"incremental"`
}

// NewServer creates a new API server.
func NewServer(manager *jobs.Manager, store *store.Store, cfg config.Config) *Server {
	return &Server{manager: manager, store: store, cfg: cfg}
}

// Routes returns the HTTP handler with all API routes configured.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/v1/auth/profiles", s.handleAuthProfiles)
	mux.HandleFunc("/v1/auth/profiles/", s.handleAuthProfile)
	mux.HandleFunc("/v1/auth/import", s.handleAuthImport)
	mux.HandleFunc("/v1/auth/export", s.handleAuthExport)
	mux.HandleFunc("/v1/scrape", s.handleScrape)
	mux.HandleFunc("/v1/crawl", s.handleCrawl)
	mux.HandleFunc("/v1/research", s.handleResearch)
	mux.HandleFunc("/v1/jobs", s.handleJobs)
	mux.HandleFunc("/v1/jobs/", s.handleJob)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	res := HealthResponse{
		Status:     "ok",
		Components: make(map[string]ComponentStatus),
	}
	healthy := true

	// Check Database
	dbStatus := ComponentStatus{Status: "ok"}
	if err := s.store.Ping(ctx); err != nil {
		dbStatus.Status = "error"
		dbStatus.Message = err.Error()
		healthy = false
	}
	res.Components["database"] = dbStatus

	// Check Queue
	qStatus := s.manager.Status()
	res.Components["queue"] = ComponentStatus{
		Status: "ok",
		Details: map[string]int{
			"queued": qStatus.QueuedJobs,
			"active": qStatus.ActiveJobs,
		},
	}

	// Check Browser
	browserStatus := ComponentStatus{Status: "ok"}
	usePlaywright := s.cfg.UsePlaywright
	if err := fetch.CheckBrowserAvailability(usePlaywright); err != nil {
		browserStatus.Status = "error"
		browserStatus.Message = err.Error()
		// Only fail health check if browser is critical and failing
		// healthy = false
	}
	res.Components["browser"] = browserStatus

	if !healthy {
		res.Status = "error"
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	writeJSON(w, res)
}

func (s *Server) handleScrape(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeJSONError(w, http.StatusUnsupportedMediaType, "content-type must be application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req ScrapeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if req.URL == "" {
		writeJSONError(w, http.StatusBadRequest, "url is required")
		return
	}
	if !isValidURL(req.URL) {
		writeJSONError(w, http.StatusBadRequest, "invalid url: must be http or https and have a host")
		return
	}
	if !isValidProfileName(req.AuthProfile) {
		writeJSONError(w, http.StatusBadRequest, "invalid authProfile: only alphanumeric, hyphens, and underscores allowed")
		return
	}
	if req.TimeoutSeconds != 0 && (req.TimeoutSeconds < 5 || req.TimeoutSeconds > 300) {
		writeJSONError(w, http.StatusBadRequest, "timeoutSeconds must be between 5 and 300")
		return
	}

	incremental := false
	if req.Incremental != nil {
		incremental = *req.Incremental
	}

	extractOpts := extract.ExtractOptions{}
	if req.Extract != nil {
		extractOpts = *req.Extract
	}

	pipelineOpts := pipeline.Options{}
	if req.Pipeline != nil {
		pipelineOpts = *req.Pipeline
	}

	timeout := req.TimeoutSeconds
	if timeout <= 0 {
		timeout = s.manager.DefaultTimeoutSeconds()
	}
	usePlaywright := s.manager.DefaultUsePlaywright()
	if req.Playwright != nil {
		usePlaywright = *req.Playwright
	}

	authOptions, err := resolveAuthForRequest(s.cfg, req.URL, req.AuthProfile, req.Auth)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	job, err := s.manager.CreateScrapeJob(r.Context(), req.URL, req.Headless, usePlaywright, authOptions, timeout, extractOpts, pipelineOpts, incremental)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.manager.Enqueue(job); err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, "failed to enqueue job: "+err.Error())
		return
	}

	writeJSON(w, job)
}

func (s *Server) handleCrawl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeJSONError(w, http.StatusUnsupportedMediaType, "content-type must be application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req CrawlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if req.URL == "" {
		writeJSONError(w, http.StatusBadRequest, "url is required")
		return
	}
	if !isValidURL(req.URL) {
		writeJSONError(w, http.StatusBadRequest, "invalid url: must be http or https and have a host")
		return
	}
	if !isValidProfileName(req.AuthProfile) {
		writeJSONError(w, http.StatusBadRequest, "invalid authProfile: only alphanumeric, hyphens, and underscores allowed")
		return
	}
	if req.TimeoutSeconds != 0 && (req.TimeoutSeconds < 5 || req.TimeoutSeconds > 300) {
		writeJSONError(w, http.StatusBadRequest, "timeoutSeconds must be between 5 and 300")
		return
	}
	if req.MaxDepth != 0 && (req.MaxDepth < 1 || req.MaxDepth > 10) {
		writeJSONError(w, http.StatusBadRequest, "maxDepth must be between 1 and 10")
		return
	}
	if req.MaxPages != 0 && (req.MaxPages < 1 || req.MaxPages > 10000) {
		writeJSONError(w, http.StatusBadRequest, "maxPages must be between 1 and 10000")
		return
	}

	incremental := false
	if req.Incremental != nil {
		incremental = *req.Incremental
	}

	extractOpts := extract.ExtractOptions{}
	if req.Extract != nil {
		extractOpts = *req.Extract
	}

	pipelineOpts := pipeline.Options{}
	if req.Pipeline != nil {
		pipelineOpts = *req.Pipeline
	}

	timeout := req.TimeoutSeconds
	if timeout <= 0 {
		timeout = s.manager.DefaultTimeoutSeconds()
	}
	usePlaywright := s.manager.DefaultUsePlaywright()
	if req.Playwright != nil {
		usePlaywright = *req.Playwright
	}

	authOptions, err := resolveAuthForRequest(s.cfg, req.URL, req.AuthProfile, req.Auth)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	job, err := s.manager.CreateCrawlJob(r.Context(), req.URL, req.MaxDepth, req.MaxPages, req.Headless, usePlaywright, authOptions, timeout, extractOpts, pipelineOpts, incremental)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.manager.Enqueue(job); err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, "failed to enqueue job: "+err.Error())
		return
	}

	writeJSON(w, job)
}

func (s *Server) handleResearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeJSONError(w, http.StatusUnsupportedMediaType, "content-type must be application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req ResearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if req.Query == "" || len(req.URLs) == 0 {
		writeJSONError(w, http.StatusBadRequest, "query and urls are required")
		return
	}
	for _, u := range req.URLs {
		if !isValidURL(u) {
			writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("invalid url in list: %s (must be http or https and have a host)", u))
			return
		}
	}
	if !isValidProfileName(req.AuthProfile) {
		writeJSONError(w, http.StatusBadRequest, "invalid authProfile: only alphanumeric, hyphens, and underscores allowed")
		return
	}
	if req.TimeoutSeconds != 0 && (req.TimeoutSeconds < 5 || req.TimeoutSeconds > 300) {
		writeJSONError(w, http.StatusBadRequest, "timeoutSeconds must be between 5 and 300")
		return
	}
	if req.MaxDepth != 0 && (req.MaxDepth < 1 || req.MaxDepth > 10) {
		writeJSONError(w, http.StatusBadRequest, "maxDepth must be between 1 and 10")
		return
	}
	if req.MaxPages != 0 && (req.MaxPages < 1 || req.MaxPages > 10000) {
		writeJSONError(w, http.StatusBadRequest, "maxPages must be between 1 and 10000")
		return
	}

	incremental := false
	if req.Incremental != nil {
		incremental = *req.Incremental
	}

	extractOpts := extract.ExtractOptions{}
	if req.Extract != nil {
		extractOpts = *req.Extract
	}

	pipelineOpts := pipeline.Options{}
	if req.Pipeline != nil {
		pipelineOpts = *req.Pipeline
	}

	timeout := req.TimeoutSeconds
	if timeout <= 0 {
		timeout = s.manager.DefaultTimeoutSeconds()
	}
	usePlaywright := s.manager.DefaultUsePlaywright()
	if req.Playwright != nil {
		usePlaywright = *req.Playwright
	}

	targetURL := ""
	if len(req.URLs) > 0 {
		targetURL = req.URLs[0]
	}
	authOptions, err := resolveAuthForRequest(s.cfg, targetURL, req.AuthProfile, req.Auth)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	job, err := s.manager.CreateResearchJob(r.Context(), req.Query, req.URLs, req.MaxDepth, req.MaxPages, req.Headless, usePlaywright, authOptions, timeout, extractOpts, pipelineOpts, incremental)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := s.manager.Enqueue(job); err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, "failed to enqueue job: "+err.Error())
		return
	}

	writeJSON(w, job)
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	query := r.URL.Query()
	limit := parseIntParam(query.Get("limit"), 100)
	offset := parseIntParam(query.Get("offset"), 0)

	opts := store.ListOptions{Limit: limit, Offset: offset}
	jobsList, err := s.store.ListOpts(r.Context(), opts)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"jobs": jobsList})
}

// parseIntParam parses an integer parameter with a default value.
// Invalid values (negative or non-numeric) result in the default.
func parseIntParam(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	var val int
	if _, err := fmt.Sscanf(s, "%d", &val); err != nil || val < 0 {
		return defaultVal
	}
	return val
}

func (s *Server) handleJob(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if strings.HasSuffix(path, "/results") {
		s.handleJobResults(w, r)
		return
	}
	id := filepath.Base(path)
	if id == "" || id == "jobs" {
		writeJSONError(w, http.StatusBadRequest, "id required")
		return
	}
	switch r.Method {
	case http.MethodGet:
		job, err := s.store.Get(r.Context(), id)
		if err != nil {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}
		writeJSON(w, job)
	case http.MethodDelete:
		// Check if this is a force delete (query param)
		if r.URL.Query().Get("force") == "true" {
			// Permanent delete
			if err := s.store.Delete(r.Context(), id); err != nil {
				writeJSONError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, map[string]string{"status": "deleted"})
		} else {
			// Cancel only (existing behavior)
			if err := s.manager.CancelJob(r.Context(), id); err != nil {
				writeJSONError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, map[string]string{"status": "canceled"})
		}
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleJobResults(w http.ResponseWriter, r *http.Request) {
	id := filepath.Base(strings.TrimSuffix(r.URL.Path, "/results"))
	if id == "" {
		writeJSONError(w, http.StatusBadRequest, "id required")
		return
	}
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	job, err := s.store.Get(r.Context(), id)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	if job.ResultPath == "" {
		writeJSONError(w, http.StatusNotFound, "no results")
		return
	}

	ext := filepath.Ext(job.ResultPath)
	if ct := contentTypeForExtension(ext); ct != "" {
		w.Header().Set("Content-Type", ct)
	}

	http.ServeFile(w, r, job.ResultPath)
}

func (s *Server) handleAuthProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	vault, err := auth.LoadVault(s.cfg.DataDir)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"profiles": vault.Profiles})
}

func (s *Server) handleAuthProfile(w http.ResponseWriter, r *http.Request) {
	name := filepath.Base(r.URL.Path)
	if name == "" || name == "profiles" {
		writeJSONError(w, http.StatusBadRequest, "name required")
		return
	}
	switch r.Method {
	case http.MethodPut:
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			writeJSONError(w, http.StatusUnsupportedMediaType, "content-type must be application/json")
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		var profile auth.Profile
		if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid json: "+err.Error())
			return
		}
		if profile.Name == "" {
			profile.Name = name
		}
		if profile.Name != name {
			writeJSONError(w, http.StatusBadRequest, "profile name mismatch")
			return
		}
		if !isValidProfileName(profile.Name) {
			writeJSONError(w, http.StatusBadRequest, "invalid profile name: only alphanumeric, hyphens, and underscores allowed")
			return
		}
		if err := auth.UpsertProfile(s.cfg.DataDir, profile); err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, profile)
	case http.MethodDelete:
		if err := auth.DeleteProfile(s.cfg.DataDir, name); err != nil {
			writeJSONError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, map[string]string{"status": "ok"})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleAuthImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeJSONError(w, http.StatusUnsupportedMediaType, "content-type must be application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var payload struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if err := auth.ImportVault(s.cfg.DataDir, payload.Path); err != nil {
		if errors.Is(err, auth.ErrInvalidPath) || err.Error() == "path is required" {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleAuthExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		writeJSONError(w, http.StatusUnsupportedMediaType, "content-type must be application/json")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var payload struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return
	}
	if err := auth.ExportVault(s.cfg.DataDir, payload.Path); err != nil {
		if errors.Is(err, auth.ErrInvalidPath) || err.Error() == "path is required" {
			writeJSONError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("failed to encode json response", "error", err)
	}
}

// writeJSONError writes a JSON error response with the given status code.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	errResp := ErrorResponse{Error: message}
	if err := json.NewEncoder(w).Encode(errResp); err != nil {
		slog.Error("failed to encode json error response", "error", err)
	}
}

func resolveAuthForRequest(cfg config.Config, url string, profile string, override *fetch.AuthOptions) (fetch.AuthOptions, error) {
	input := auth.ResolveInput{
		ProfileName: profile,
		URL:         url,
		Env:         &cfg.AuthOverrides,
	}
	if override != nil {
		input.Headers = toHeaderKVs(override.Headers)
		input.Cookies = toCookies(override.Cookies)
		input.Tokens = tokensFromOverride(*override)
		if login := loginFromOverride(*override); login != nil {
			input.Login = login
		}
	}
	resolved, err := auth.Resolve(cfg.DataDir, input)
	if err != nil {
		return fetch.AuthOptions{}, err
	}
	return auth.ToFetchOptions(resolved), nil
}

func toHeaderKVs(headers map[string]string) []auth.HeaderKV {
	if len(headers) == 0 {
		return nil
	}
	out := make([]auth.HeaderKV, 0, len(headers))
	for key, value := range headers {
		if strings.TrimSpace(key) == "" {
			continue
		}
		out = append(out, auth.HeaderKV{Key: key, Value: value})
	}
	return out
}

func toCookies(cookies []string) []auth.Cookie {
	if len(cookies) == 0 {
		return nil
	}
	out := make([]auth.Cookie, 0, len(cookies))
	for _, raw := range cookies {
		parts := strings.SplitN(strings.TrimSpace(raw), "=", 2)
		if len(parts) != 2 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if name == "" {
			continue
		}
		out = append(out, auth.Cookie{Name: name, Value: value})
	}
	return out
}

func tokensFromOverride(override fetch.AuthOptions) []auth.Token {
	out := []auth.Token{}
	if override.Basic != "" {
		out = append(out, auth.Token{Kind: auth.TokenBasic, Value: override.Basic})
	}
	for key, value := range override.Query {
		out = append(out, auth.Token{Kind: auth.TokenApiKey, Value: value, Query: key})
	}
	return out
}

func loginFromOverride(override fetch.AuthOptions) *auth.LoginFlow {
	if override.LoginURL == "" && override.LoginUserSelector == "" && override.LoginPassSelector == "" && override.LoginSubmitSelector == "" && override.LoginUser == "" && override.LoginPass == "" {
		return nil
	}
	return &auth.LoginFlow{
		URL:            override.LoginURL,
		UserSelector:   override.LoginUserSelector,
		PassSelector:   override.LoginPassSelector,
		SubmitSelector: override.LoginSubmitSelector,
		Username:       override.LoginUser,
		Password:       override.LoginPass,
	}
}

var profileNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func isValidURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}
	if u.Host == "" {
		return false
	}
	return true
}

func isValidProfileName(name string) bool {
	if name == "" {
		return true
	}
	return profileNameRegex.MatchString(name)
}

func contentTypeForExtension(ext string) string {
	switch strings.ToLower(ext) {
	case ".jsonl":
		return "application/x-ndjson"
	case ".json":
		return "application/json"
	case ".csv":
		return "text/csv"
	case ".xml":
		return "application/xml"
	case ".txt":
		return "text/plain; charset=utf-8"
	default:
		return ""
	}
}
