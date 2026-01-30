// Package api implements the REST API server for Spartan Scraper.
// It provides endpoints for enqueuing jobs, managing auth profiles,
// and retrieving job status and results.
package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
	"github.com/gobwas/ws"
)

type Server struct {
	manager           *jobs.Manager
	store             *store.Store
	cfg               config.Config
	wsHub             *Hub
	metricsCollector  *MetricsCollector
	webhookDispatcher *webhook.Dispatcher
}

func NewServer(manager *jobs.Manager, store *store.Store, cfg config.Config) *Server {
	s := &Server{
		manager:          manager,
		store:            store,
		cfg:              cfg,
		wsHub:            NewHub(),
		metricsCollector: NewMetricsCollector(),
	}

	// Start WebSocket hub
	go s.wsHub.Run()

	// Subscribe hub to job manager events
	go s.subscribeToJobEvents()

	// Start periodic metrics broadcasting
	go s.startMetricsBroadcast()

	// Set up metrics callback for fetch operations
	s.manager.SetMetricsCallback(s.metricsCollector.RecordRequest)

	// Initialize webhook dispatcher if configured
	if cfg.Webhook.Enabled || cfg.Webhook.Secret != "" {
		// Create webhook store for delivery tracking
		webhookStore := webhook.NewStore(cfg.DataDir)
		if err := webhookStore.Load(); err != nil {
			slog.Warn("failed to load webhook delivery store", "error", err)
		}

		dispatcher := webhook.NewDispatcherWithStore(webhook.Config{
			Secret:     cfg.Webhook.Secret,
			MaxRetries: cfg.Webhook.MaxRetries,
			BaseDelay:  cfg.Webhook.BaseDelay,
			MaxDelay:   cfg.Webhook.MaxDelay,
			Timeout:    cfg.Webhook.Timeout,
		}, webhookStore)
		s.webhookDispatcher = dispatcher
		s.manager.SetWebhookDispatcher(dispatcher)
	}

	return s
}

// GetMetricsCollector returns the server's metrics collector for external registration
func (s *Server) GetMetricsCollector() *MetricsCollector {
	return s.metricsCollector
}

// startMetricsBroadcast periodically broadcasts metrics via WebSocket
func (s *Server) startMetricsBroadcast() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.syncHostLimiters()
		snapshot := s.metricsCollector.GetSnapshot()
		s.wsHub.BroadcastMetrics(snapshot)
	}
}

// syncHostLimiters syncs host limiters from the job manager to the metrics collector.
func (s *Server) syncHostLimiters() {
	limiter := s.manager.GetLimiter()
	if limiter == nil {
		return
	}

	// Get all host statuses and register them with the metrics collector
	statuses := limiter.GetHostStatus()
	for _, status := range statuses {
		l := limiter.GetLimiter(status.Host)
		if l != nil {
			s.metricsCollector.RegisterHostLimiter(status.Host, l, status.QPS, status.Burst)
		}
	}
}

// subscribeToJobEvents subscribes the WebSocket hub to job manager events.
func (s *Server) subscribeToJobEvents() {
	eventCh := make(chan jobs.JobEvent, 256)
	s.manager.SubscribeToEvents(eventCh)
	defer s.manager.UnsubscribeFromEvents(eventCh)

	for event := range eventCh {
		s.wsHub.BroadcastJobEvent(JobEvent{
			Type:       JobEventType(event.Type),
			Job:        event.Job,
			PrevStatus: event.PrevStatus,
		})
	}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/v1/auth/profiles", s.handleAuthProfiles)
	mux.HandleFunc("/v1/auth/profiles/", s.handleAuthProfile)
	mux.HandleFunc("/v1/auth/import", s.handleAuthImport)
	mux.HandleFunc("/v1/auth/export", s.handleAuthExport)
	mux.HandleFunc("/v1/auth/oauth/initiate", s.handleOAuthInitiate)
	mux.HandleFunc("/v1/auth/oauth/callback", s.handleOAuthCallback)
	mux.HandleFunc("/v1/auth/oauth/refresh", s.handleOAuthRefresh)
	mux.HandleFunc("/v1/auth/oauth/discover", s.handleOIDCDiscover)
	mux.HandleFunc("/v1/auth/oauth/revoke", s.handleOAuthRevoke)
	mux.HandleFunc("/v1/auth/sessions", s.handleSessions)
	mux.HandleFunc("/v1/auth/sessions/", s.handleSession)
	mux.HandleFunc("/v1/scrape", s.handleScrape)
	mux.HandleFunc("/v1/crawl", s.handleCrawl)
	mux.HandleFunc("/v1/research", s.handleResearch)
	mux.HandleFunc("/v1/jobs", s.handleJobs)
	mux.HandleFunc("/v1/jobs/", s.handleJob)
	mux.HandleFunc("/v1/jobs/batch/scrape", s.handleBatchScrape)
	mux.HandleFunc("/v1/jobs/batch/crawl", s.handleBatchCrawl)
	mux.HandleFunc("/v1/jobs/batch/research", s.handleBatchResearch)
	mux.HandleFunc("/v1/jobs/batch/", s.handleBatchGet)
	mux.HandleFunc("/v1/schedules", s.handleSchedules)
	mux.HandleFunc("/v1/schedules/", s.handleSchedule)
	mux.HandleFunc("/v1/templates", s.handleTemplates)
	mux.HandleFunc("/v1/crawl-states", s.handleCrawlStates)
	mux.HandleFunc("/v1/metrics", s.handleMetrics)
	mux.HandleFunc("/v1/ws", s.handleWebSocket)
	mux.HandleFunc("/v1/chains", s.handleChains)
	mux.HandleFunc("/v1/chains/", s.handleChain)
	mux.HandleFunc("/v1/proxy-pool/status", s.handleProxyPoolStatus)
	mux.HandleFunc("/v1/transform/validate", s.handleValidateTransform)
	mux.HandleFunc("/v1/webhooks/deliveries", s.handleWebhookDeliveries)
	mux.HandleFunc("/v1/webhooks/deliveries/", s.handleWebhookDeliveryDetail)

	// Build middleware chain
	handler := requestIDMiddleware(loggingMiddleware(recoveryMiddleware(mux)))

	// Add auth middleware if enabled or if bind address is not localhost
	if s.cfg.APIAuthEnabled || !isLocalhost(s.cfg.BindAddr) {
		handler = apiKeyAuthMiddleware(s.cfg, handler)
	}

	return handler
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP to WebSocket
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		http.Error(w, "websocket upgrade failed", http.StatusBadRequest)
		return
	}

	// Create client and register with hub
	client := s.wsHub.NewClient(conn)
	s.wsHub.register <- client

	// Start goroutines for the client
	go client.writePump()
	go client.readPump()
}

// handleSessions handles requests to /v1/auth/sessions
func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListSessions(w, r)
	case http.MethodPost:
		s.handleCreateSession(w, r)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

// handleSession handles requests to /v1/auth/sessions/{id}
func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetSession(w, r)
	case http.MethodDelete:
		s.handleDeleteSession(w, r)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}
