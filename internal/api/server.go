package api

import (
	"encoding/json"
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

type Server struct {
	manager *jobs.Manager
	store   *store.Store
	cfg     config.Config
}

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

func NewServer(manager *jobs.Manager, store *store.Store, cfg config.Config) *Server {
	return &Server{manager: manager, store: store, cfg: cfg}
}

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

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleScrape(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "content-type must be application/json", http.StatusUnsupportedMediaType)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req ScrapeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}
	if !isValidURL(req.URL) {
		http.Error(w, "invalid url: must be http or https and have a host", http.StatusBadRequest)
		return
	}
	if !isValidProfileName(req.AuthProfile) {
		http.Error(w, "invalid authProfile: only alphanumeric, hyphens, and underscores allowed", http.StatusBadRequest)
		return
	}
	if req.TimeoutSeconds != 0 && (req.TimeoutSeconds < 5 || req.TimeoutSeconds > 300) {
		http.Error(w, "timeoutSeconds must be between 5 and 300", http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	job, err := s.manager.CreateScrapeJob(r.Context(), req.URL, req.Headless, usePlaywright, authOptions, timeout, extractOpts, pipelineOpts, incremental)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.manager.Enqueue(job); err != nil {
		http.Error(w, "failed to enqueue job: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	writeJSON(w, job)
}

func (s *Server) handleCrawl(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "content-type must be application/json", http.StatusUnsupportedMediaType)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req CrawlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.URL == "" {
		http.Error(w, "url is required", http.StatusBadRequest)
		return
	}
	if !isValidURL(req.URL) {
		http.Error(w, "invalid url: must be http or https and have a host", http.StatusBadRequest)
		return
	}
	if !isValidProfileName(req.AuthProfile) {
		http.Error(w, "invalid authProfile: only alphanumeric, hyphens, and underscores allowed", http.StatusBadRequest)
		return
	}
	if req.TimeoutSeconds != 0 && (req.TimeoutSeconds < 5 || req.TimeoutSeconds > 300) {
		http.Error(w, "timeoutSeconds must be between 5 and 300", http.StatusBadRequest)
		return
	}
	if req.MaxDepth != 0 && (req.MaxDepth < 1 || req.MaxDepth > 10) {
		http.Error(w, "maxDepth must be between 1 and 10", http.StatusBadRequest)
		return
	}
	if req.MaxPages != 0 && (req.MaxPages < 1 || req.MaxPages > 10000) {
		http.Error(w, "maxPages must be between 1 and 10000", http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	job, err := s.manager.CreateCrawlJob(r.Context(), req.URL, req.MaxDepth, req.MaxPages, req.Headless, usePlaywright, authOptions, timeout, extractOpts, pipelineOpts, incremental)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.manager.Enqueue(job); err != nil {
		http.Error(w, "failed to enqueue job: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	writeJSON(w, job)
}

func (s *Server) handleResearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "content-type must be application/json", http.StatusUnsupportedMediaType)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var req ResearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Query == "" || len(req.URLs) == 0 {
		http.Error(w, "query and urls are required", http.StatusBadRequest)
		return
	}
	for _, u := range req.URLs {
		if !isValidURL(u) {
			http.Error(w, fmt.Sprintf("invalid url in list: %s (must be http or https and have a host)", u), http.StatusBadRequest)
			return
		}
	}
	if !isValidProfileName(req.AuthProfile) {
		http.Error(w, "invalid authProfile: only alphanumeric, hyphens, and underscores allowed", http.StatusBadRequest)
		return
	}
	if req.TimeoutSeconds != 0 && (req.TimeoutSeconds < 5 || req.TimeoutSeconds > 300) {
		http.Error(w, "timeoutSeconds must be between 5 and 300", http.StatusBadRequest)
		return
	}
	if req.MaxDepth != 0 && (req.MaxDepth < 1 || req.MaxDepth > 10) {
		http.Error(w, "maxDepth must be between 1 and 10", http.StatusBadRequest)
		return
	}
	if req.MaxPages != 0 && (req.MaxPages < 1 || req.MaxPages > 10000) {
		http.Error(w, "maxPages must be between 1 and 10000", http.StatusBadRequest)
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	job, err := s.manager.CreateResearchJob(r.Context(), req.Query, req.URLs, req.MaxDepth, req.MaxPages, req.Headless, usePlaywright, authOptions, timeout, extractOpts, pipelineOpts, incremental)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := s.manager.Enqueue(job); err != nil {
		http.Error(w, "failed to enqueue job: "+err.Error(), http.StatusServiceUnavailable)
		return
	}

	writeJSON(w, job)
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	jobsList, err := s.store.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]interface{}{"jobs": jobsList})
}

func (s *Server) handleJob(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if strings.HasSuffix(path, "/results") {
		s.handleJobResults(w, r)
		return
	}
	id := filepath.Base(path)
	if id == "" || id == "jobs" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		job, err := s.store.Get(r.Context(), id)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		writeJSON(w, job)
	case http.MethodDelete:
		if err := s.manager.CancelJob(r.Context(), id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]string{"status": "canceled"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleJobResults(w http.ResponseWriter, r *http.Request) {
	id := filepath.Base(strings.TrimSuffix(r.URL.Path, "/results"))
	if id == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	job, err := s.store.Get(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	if job.ResultPath == "" {
		http.Error(w, "no results", http.StatusNotFound)
		return
	}
	http.ServeFile(w, r, job.ResultPath)
}

func (s *Server) handleAuthProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	vault, err := auth.LoadVault(s.cfg.DataDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]interface{}{"profiles": vault.Profiles})
}

func (s *Server) handleAuthProfile(w http.ResponseWriter, r *http.Request) {
	name := filepath.Base(r.URL.Path)
	if name == "" || name == "profiles" {
		http.Error(w, "name required", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodPut:
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			http.Error(w, "content-type must be application/json", http.StatusUnsupportedMediaType)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
		var profile auth.Profile
		if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
			http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
			return
		}
		if profile.Name == "" {
			profile.Name = name
		}
		if profile.Name != name {
			http.Error(w, "profile name mismatch", http.StatusBadRequest)
			return
		}
		if !isValidProfileName(profile.Name) {
			http.Error(w, "invalid profile name: only alphanumeric, hyphens, and underscores allowed", http.StatusBadRequest)
			return
		}
		if err := auth.UpsertProfile(s.cfg.DataDir, profile); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, profile)
	case http.MethodDelete:
		if err := auth.DeleteProfile(s.cfg.DataDir, name); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]string{"status": "ok"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAuthImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "content-type must be application/json", http.StatusUnsupportedMediaType)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var payload struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if payload.Path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}
	if err := auth.ImportVault(s.cfg.DataDir, payload.Path); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleAuthExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		http.Error(w, "content-type must be application/json", http.StatusUnsupportedMediaType)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	var payload struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid json: "+err.Error(), http.StatusBadRequest)
		return
	}
	if payload.Path == "" {
		http.Error(w, "path is required", http.StatusBadRequest)
		return
	}
	if err := auth.ExportVault(s.cfg.DataDir, payload.Path); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
