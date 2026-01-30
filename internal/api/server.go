// Package api implements the REST API server for Spartan Scraper.
// It provides endpoints for enqueuing jobs, managing auth profiles,
// and retrieving job status and results.
package api

import (
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/config"
	"github.com/fitchmultz/spartan-scraper/internal/jobs"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/gobwas/ws"
)

type Server struct {
	manager *jobs.Manager
	store   *store.Store
	cfg     config.Config
	wsHub   *Hub
}

func NewServer(manager *jobs.Manager, store *store.Store, cfg config.Config) *Server {
	s := &Server{
		manager: manager,
		store:   store,
		cfg:     cfg,
		wsHub:   NewHub(),
	}

	// Start WebSocket hub
	go s.wsHub.Run()

	// Subscribe hub to job manager events
	go s.subscribeToJobEvents()

	return s
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
	mux.HandleFunc("/v1/scrape", s.handleScrape)
	mux.HandleFunc("/v1/crawl", s.handleCrawl)
	mux.HandleFunc("/v1/research", s.handleResearch)
	mux.HandleFunc("/v1/jobs", s.handleJobs)
	mux.HandleFunc("/v1/jobs/", s.handleJob)
	mux.HandleFunc("/v1/schedules", s.handleSchedules)
	mux.HandleFunc("/v1/schedules/", s.handleSchedule)
	mux.HandleFunc("/v1/templates", s.handleTemplates)
	mux.HandleFunc("/v1/crawl-states", s.handleCrawlStates)
	mux.HandleFunc("/v1/ws", s.handleWebSocket)

	handler := requestIDMiddleware(loggingMiddleware(recoveryMiddleware(mux)))
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
