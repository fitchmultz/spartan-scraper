// Package api implements the REST API server for Spartan Scraper.
// It provides endpoints for enqueuing jobs, managing auth profiles,
// and retrieving job status and results.
package api

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/aiauthoring"
	"github.com/fitchmultz/spartan-scraper/internal/analytics"
	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/extract"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/fitchmultz/spartan-scraper/internal/webhook"
	"github.com/gobwas/ws"
)

type Server struct {
	manager            *jobs.Manager
	store              *store.Store
	cfg                config.Config
	wsHub              *Hub
	metricsCollector   *MetricsCollector
	webhookDispatcher  *webhook.Dispatcher
	analyticsCollector *analytics.Collector
	analyticsService   *analytics.Service
	aiExtractor        *extract.AIExtractor
	aiAuthoring        *aiauthoring.Service
	ctx                context.Context
	cancel             context.CancelFunc
}

func NewServer(manager *jobs.Manager, store *store.Store, cfg config.Config) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize AI extractor if configured
	var aiExtractor *extract.AIExtractor
	if extract.IsAIEnabled(cfg.AI) {
		var err error
		aiExtractor, err = extract.NewAIExtractor(cfg.AI)
		if err != nil {
			slog.Warn("Failed to initialize AI extractor", "error", err)
		}
	}

	s := &Server{
		manager:          manager,
		store:            store,
		cfg:              cfg,
		wsHub:            NewHub(),
		metricsCollector: NewMetricsCollector(),
		analyticsService: analytics.NewService(store),
		aiExtractor:      aiExtractor,
		ctx:              ctx,
		cancel:           cancel,
	}

	// Start WebSocket hub
	go s.wsHub.Run()

	// Subscribe hub to job manager events
	go s.subscribeToJobEvents()

	// Start periodic metrics broadcasting
	go s.startMetricsBroadcast()

	// Set up metrics callback for fetch operations.
	// The fetch layer emits a zero-duration start marker before completion.
	s.manager.SetMetricsCallback(s.metricsCollector.Callback())

	// Initialize webhook dispatcher if configured
	if cfg.Webhook.Enabled || cfg.Webhook.Secret != "" {
		// Create webhook store for delivery tracking
		webhookStore := webhook.NewStore(cfg.DataDir)
		if err := webhookStore.Load(); err != nil {
			slog.Warn("failed to load webhook delivery store", "error", err)
		}

		dispatcher := webhook.NewDispatcherWithStore(webhook.Config{
			Secret:                  cfg.Webhook.Secret,
			MaxRetries:              cfg.Webhook.MaxRetries,
			BaseDelay:               cfg.Webhook.BaseDelay,
			MaxDelay:                cfg.Webhook.MaxDelay,
			Timeout:                 cfg.Webhook.Timeout,
			AllowInternal:           cfg.Webhook.AllowInternal,
			MaxConcurrentDispatches: cfg.Webhook.MaxConcurrentDispatches,
		}, webhookStore)
		s.webhookDispatcher = dispatcher
		s.manager.SetWebhookDispatcher(dispatcher)
	}

	// Initialize analytics collector with adapter
	metricsAdapter := &metricsCollectorAdapter{collector: s.metricsCollector}
	s.analyticsCollector = analytics.NewCollector(store, metricsAdapter)
	s.analyticsCollector.Start()

	return s
}

// metricsCollectorAdapter adapts api.MetricsCollector to analytics.MetricsCollector
type metricsCollectorAdapter struct {
	collector *MetricsCollector
}

func (a *metricsCollectorAdapter) GetSnapshot() analytics.MetricsSnapshot {
	snapshot := a.collector.GetSnapshot()
	return analytics.MetricsSnapshot{
		RequestsPerSec:  snapshot.RequestsPerSec,
		SuccessRate:     snapshot.SuccessRate,
		AvgResponseTime: snapshot.AvgResponseTime,
		ActiveRequests:  snapshot.ActiveRequests,
		TotalRequests:   snapshot.TotalRequests,
		FetcherUsage: struct {
			HTTP       uint64
			Chromedp   uint64
			Playwright uint64
		}{
			HTTP:       snapshot.FetcherUsage.HTTP,
			Chromedp:   snapshot.FetcherUsage.Chromedp,
			Playwright: snapshot.FetcherUsage.Playwright,
		},
		JobThroughput:  snapshot.JobThroughput,
		AvgJobDuration: snapshot.AvgJobDuration,
		Timestamp:      snapshot.Timestamp,
	}
}

// GetMetricsCollector returns the server's metrics collector for external registration
func (s *Server) GetMetricsCollector() *MetricsCollector {
	return s.metricsCollector
}

// Stop gracefully shuts down the server's background services.
// This should be called during application shutdown to ensure
// analytics data is properly flushed to storage.
func (s *Server) Stop() {
	// Cancel context to signal goroutines to stop
	if s.cancel != nil {
		s.cancel()
	}

	// Stop WebSocket hub and wait for it to exit
	if s.wsHub != nil {
		s.wsHub.Stop()
		s.wsHub.Wait()
	}

	// Stop analytics collector
	if s.analyticsCollector != nil {
		s.analyticsCollector.Stop()
	}
}

// startMetricsBroadcast periodically broadcasts metrics via WebSocket
func (s *Server) startMetricsBroadcast() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.syncHostLimiters()
			snapshot := s.metricsCollector.GetSnapshot()
			s.wsHub.BroadcastMetrics(snapshot)
		case <-s.ctx.Done():
			return
		}
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

	for {
		select {
		case event := <-eventCh:
			s.wsHub.BroadcastJobEvent(JobEvent{
				Type:       JobEventType(event.Type),
				Job:        event.Job,
				PrevStatus: event.PrevStatus,
			})
		case <-s.ctx.Done():
			return
		}
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
	mux.HandleFunc("/v1/export-schedules", s.handleExportSchedules)
	mux.HandleFunc("/v1/export-schedules/", s.handleExportScheduleDetail)
	mux.HandleFunc("/v1/watch", s.handleWatches)
	mux.HandleFunc("/v1/watch/", s.handleWatchCheckWrapper)
	mux.HandleFunc("/v1/templates", s.handleTemplates)
	mux.HandleFunc("/v1/templates/", s.handleTemplate)
	mux.HandleFunc("/v1/template-preview", s.handleTemplatePreview)
	mux.HandleFunc("/v1/template-preview/test-selector", s.handleTestSelector)
	mux.HandleFunc("/v1/crawl-states", s.handleCrawlStates)
	mux.HandleFunc("/v1/metrics", s.handleMetrics)
	mux.HandleFunc("/v1/ws", s.handleWebSocket)
	mux.HandleFunc("/v1/chains", s.handleChains)
	mux.HandleFunc("/v1/chains/", s.handleChain)
	mux.HandleFunc("/v1/proxy-pool/status", s.handleProxyPoolStatus)
	mux.HandleFunc("/v1/transform/validate", s.handleValidateTransform)
	mux.HandleFunc("/v1/webhooks/deliveries", s.handleWebhookDeliveries)
	mux.HandleFunc("/v1/webhooks/deliveries/", s.handleWebhookDeliveryDetail)
	mux.HandleFunc("/v1/analytics/metrics", s.handleAnalyticsMetrics)
	mux.HandleFunc("/v1/analytics/hosts", s.handleAnalyticsHosts)
	mux.HandleFunc("/v1/analytics/trends", s.handleAnalyticsTrends)
	mux.HandleFunc("/v1/analytics/dashboard", s.handleAnalyticsDashboard)

	// AI authoring endpoints
	mux.HandleFunc("/v1/ai/extract-preview", s.handleAIExtractPreview)
	mux.HandleFunc("/v1/ai/template-generate", s.handleAITemplateGenerate)
	mux.HandleFunc("/v1/ai/template-debug", s.handleAITemplateDebug)
	mux.HandleFunc("/v1/ai/render-profile-generate", s.handleAIRenderProfileGenerate)
	mux.HandleFunc("/v1/ai/render-profile-debug", s.handleAIRenderProfileDebug)
	mux.HandleFunc("/v1/ai/pipeline-js-generate", s.handleAIPipelineJSGenerate)
	mux.HandleFunc("/v1/ai/pipeline-js-debug", s.handleAIPipelineJSDebug)
	mux.HandleFunc("/v1/ai/research-refine", s.handleAIResearchRefine)
	mux.HandleFunc("/v1/ai/export-shape", s.handleAIExportShape)

	// Deduplication endpoints
	mux.HandleFunc("/v1/dedup/duplicates", s.handleDedupDuplicates)
	mux.HandleFunc("/v1/dedup/history", s.handleDedupHistory)
	mux.HandleFunc("/v1/dedup/stats", s.handleDedupStats)
	mux.HandleFunc("/v1/dedup/job/", s.handleDedupJobDelete)

	// Retention endpoints
	mux.HandleFunc("/v1/retention/", s.handleRetention)

	// Render profiles endpoints
	mux.HandleFunc("/v1/render-profiles", s.handleRenderProfiles)
	mux.HandleFunc("/v1/render-profiles/", s.handleRenderProfile)

	// Pipeline JS endpoints
	mux.HandleFunc("/v1/pipeline-js", s.handlePipelineJS)
	mux.HandleFunc("/v1/pipeline-js/", s.handlePipelineJSScript)

	// Build middleware chain
	handler := requestIDMiddleware(loggingMiddleware(recoveryMiddleware(mux)))

	// Add auth middleware if enabled or if bind address is not localhost
	if s.cfg.APIAuthEnabled || !isLocalhost(s.cfg.BindAddr) {
		handler = apiKeyAuthMiddleware(s.cfg, handler)
	}

	return handler
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	if !isAllowedWebSocketOrigin(r.Header.Get("Origin")) {
		http.Error(w, "forbidden websocket origin", http.StatusForbidden)
		return
	}

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

// isAllowedWebSocketOrigin validates browser-originated WebSocket upgrades.
// Empty Origin is allowed for non-browser clients.
func isAllowedWebSocketOrigin(origin string) bool {
	if origin == "" {
		return true
	}

	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return false
	}

	return isLocalhost(parsed.Hostname())
}
