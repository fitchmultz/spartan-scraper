// Package api implements the REST API server for Spartan Scraper.
package api

import (
	"net/http"
	"sort"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fetch"
)

// ProxyPoolStatusResponse represents the response for the proxy pool status endpoint.
type ProxyPoolStatusResponse struct {
	Strategy       string        `json:"strategy"`
	TotalProxies   int           `json:"total_proxies"`
	HealthyProxies int           `json:"healthy_proxies"`
	Regions        []string      `json:"regions"`
	Tags           []string      `json:"tags"`
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

// BuildProxyPoolStatusResponse converts a loaded proxy pool into a stable status payload.
func BuildProxyPoolStatusResponse(proxyPool *fetch.ProxyPool) ProxyPoolStatusResponse {
	if proxyPool == nil {
		return ProxyPoolStatusResponse{
			Strategy:       "none",
			TotalProxies:   0,
			HealthyProxies: 0,
			Regions:        []string{},
			Tags:           []string{},
			Proxies:        []ProxyStatus{},
		}
	}

	stats := proxyPool.GetStats()
	entries := proxyPool.GetEntries()
	entryMap := make(map[string]fetch.ProxyEntry, len(entries))
	regionSet := make(map[string]struct{})
	tagSet := make(map[string]struct{})
	for _, entry := range entries {
		entryMap[entry.ID] = entry
		if entry.Region != "" {
			regionSet[entry.Region] = struct{}{}
		}
		for _, tag := range entry.Tags {
			if tag != "" {
				tagSet[tag] = struct{}{}
			}
		}
	}

	proxies := make([]ProxyStatus, 0, len(stats))
	for id, stat := range stats {
		entry := entryMap[id]
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
	sort.Slice(proxies, func(i, j int) bool { return proxies[i].ID < proxies[j].ID })

	regions := make([]string, 0, len(regionSet))
	for region := range regionSet {
		regions = append(regions, region)
	}
	sort.Strings(regions)

	tags := make([]string, 0, len(tagSet))
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	sort.Strings(tags)

	return ProxyPoolStatusResponse{
		Strategy:       proxyPool.GetStrategy().String(),
		TotalProxies:   proxyPool.GetTotalProxyCount(),
		HealthyProxies: proxyPool.GetHealthyProxyCount(),
		Regions:        regions,
		Tags:           tags,
		Proxies:        proxies,
	}
}

// handleProxyPoolStatus handles requests to the /v1/proxy-pool/status endpoint.
func (s *Server) handleProxyPoolStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	writeJSON(w, BuildProxyPoolStatusResponse(s.manager.GetProxyPool()))
}
