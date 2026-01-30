// Package api implements the REST API server for Spartan Scraper.
package api

import (
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

// ProxyPoolStatusResponse represents the response for the proxy pool status endpoint.
type ProxyPoolStatusResponse struct {
	Strategy       string        `json:"strategy"`
	TotalProxies   int           `json:"total_proxies"`
	HealthyProxies int           `json:"healthy_proxies"`
	Proxies        []ProxyStatus `json:"proxies"`
}

// ProxyStatus represents the status of a single proxy.
type ProxyStatus struct {
	ID               string   `json:"id"`
	Region           string   `json:"region,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	IsHealthy        bool     `json:"is_healthy"`
	RequestCount     uint64   `json:"request_count"`
	SuccessCount     uint64   `json:"success_count"`
	FailureCount     uint64   `json:"failure_count"`
	SuccessRate      float64  `json:"success_rate"`
	AvgLatencyMs     int64    `json:"avg_latency_ms"`
	ConsecutiveFails int      `json:"consecutive_fails"`
}

// ProxyPoolProvider is an interface for providing proxy pool status.
type ProxyPoolProvider interface {
	GetProxyPool() *fetch.ProxyPool
}

// handleProxyPoolStatus handles requests to the /v1/proxy-pool/status endpoint.
func (s *Server) handleProxyPoolStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	// Get proxy pool from manager if available
	proxyPool := s.manager.GetProxyPool()
	if proxyPool == nil {
		writeJSON(w, ProxyPoolStatusResponse{
			Strategy:       "none",
			TotalProxies:   0,
			HealthyProxies: 0,
			Proxies:        []ProxyStatus{},
		})
		return
	}

	stats := proxyPool.GetStats()
	strategy := proxyPool.GetStrategy()

	proxies := make([]ProxyStatus, 0, len(stats))
	healthyCount := 0

	entries := proxyPool.GetEntries()
	entryMap := make(map[string]fetch.ProxyEntry)
	for _, entry := range entries {
		entryMap[entry.ID] = entry
	}

	for id, stat := range stats {
		entry := entryMap[id]

		if stat.IsHealthy {
			healthyCount++
		}

		proxies = append(proxies, ProxyStatus{
			ID:               id,
			Region:           entry.Region,
			Tags:             entry.Tags,
			IsHealthy:        stat.IsHealthy,
			RequestCount:     stat.RequestCount,
			SuccessCount:     stat.SuccessCount,
			FailureCount:     stat.FailureCount,
			SuccessRate:      stat.SuccessRate(),
			AvgLatencyMs:     stat.AvgLatencyMs,
			ConsecutiveFails: stat.ConsecutiveFails,
		})
	}

	response := ProxyPoolStatusResponse{
		Strategy:       strategy.String(),
		TotalProxies:   proxyPool.GetTotalProxyCount(),
		HealthyProxies: healthyCount,
		Proxies:        proxies,
	}

	writeJSON(w, response)
}
