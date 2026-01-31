// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"math"
	"slices"
	"sync/atomic"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// Select returns a proxy based on the configured rotation strategy.
// Returns an error if no healthy proxies are available.
func (p *ProxyPool) Select(hints ProxySelectionHints) (ProxyEntry, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Filter proxies based on hints
	candidates := p.filterProxies(hints)
	if len(candidates) == 0 {
		return ProxyEntry{}, apperrors.NotFound("no healthy proxies available matching selection criteria")
	}

	// Select based on strategy
	var selected ProxyEntry
	switch p.strategy {
	case RotationRoundRobin:
		selected = p.selectRoundRobin(candidates)
	case RotationRandom:
		selected = p.selectRandom(candidates)
	case RotationLeastUsed:
		selected = p.selectLeastUsed(candidates)
	case RotationWeighted:
		selected = p.selectWeighted(candidates)
	case RotationLeastLatency:
		selected = p.selectLeastLatency(candidates)
	default:
		selected = p.selectRoundRobin(candidates)
	}

	return selected, nil
}

// filterProxies returns proxies matching the selection hints.
func (p *ProxyPool) filterProxies(hints ProxySelectionHints) []ProxyEntry {
	var candidates []ProxyEntry

	for _, proxy := range p.entries {
		stats := p.stats[proxy.ID]

		// Skip unhealthy proxies
		if stats != nil && !stats.IsHealthy {
			continue
		}

		// Skip excluded proxies
		if containsString(hints.ExcludeProxyIDs, proxy.ID) {
			continue
		}

		// Check region preference
		if hints.PreferredRegion != "" && proxy.Region != hints.PreferredRegion {
			continue
		}

		// Check required tags
		if len(hints.RequiredTags) > 0 {
			if !hasAllTags(proxy.Tags, hints.RequiredTags) {
				continue
			}
		}

		candidates = append(candidates, proxy)
	}

	return candidates
}

// selectRoundRobin selects the next proxy in round-robin order.
func (p *ProxyPool) selectRoundRobin(candidates []ProxyEntry) ProxyEntry {
	if len(candidates) == 1 {
		return candidates[0]
	}

	idx := atomic.AddUint64(&p.rrIndex, 1) % uint64(len(candidates))
	return candidates[idx]
}

// selectRandom selects a random proxy from candidates.
func (p *ProxyPool) selectRandom(candidates []ProxyEntry) ProxyEntry {
	if len(candidates) == 1 {
		return candidates[0]
	}

	// Use a simple pseudo-random selection based on time
	// In production, consider using crypto/rand for better randomness
	idx := time.Now().UnixNano() % int64(len(candidates))
	return candidates[idx]
}

// selectLeastUsed selects the proxy with the lowest request count.
func (p *ProxyPool) selectLeastUsed(candidates []ProxyEntry) ProxyEntry {
	if len(candidates) == 1 {
		return candidates[0]
	}

	var selected ProxyEntry
	minRequests := ^uint64(0) // Max uint64

	for _, proxy := range candidates {
		stats := p.stats[proxy.ID]
		if stats.RequestCount < minRequests {
			minRequests = stats.RequestCount
			selected = proxy
		}
	}

	return selected
}

// selectWeighted performs weighted random selection.
func (p *ProxyPool) selectWeighted(candidates []ProxyEntry) ProxyEntry {
	if len(candidates) == 1 {
		return candidates[0]
	}

	// Calculate total weight
	totalWeight := 0
	for _, proxy := range candidates {
		weight := proxy.Weight
		if weight <= 0 {
			weight = 1 // Default weight
		}
		totalWeight += weight
	}

	if totalWeight <= 0 {
		// Fallback to random if no weights set
		return p.selectRandom(candidates)
	}

	// Select based on weight
	// Use time-based pseudo-random for simplicity
	r := int(time.Now().UnixNano() % int64(totalWeight))
	cumulativeWeight := 0

	for _, proxy := range candidates {
		weight := proxy.Weight
		if weight <= 0 {
			weight = 1
		}
		cumulativeWeight += weight
		if r < cumulativeWeight {
			return proxy
		}
	}

	// Fallback to last candidate
	return candidates[len(candidates)-1]
}

// selectLeastLatency selects the proxy with the lowest average latency.
func (p *ProxyPool) selectLeastLatency(candidates []ProxyEntry) ProxyEntry {
	if len(candidates) == 1 {
		return candidates[0]
	}

	if len(candidates) == 0 {
		return ProxyEntry{}
	}

	var selected ProxyEntry
	minLatency := int64(math.MaxInt64) // Max int64

	for _, proxy := range candidates {
		stats, ok := p.stats[proxy.ID]
		if !ok {
			continue
		}
		// If no latency data yet, treat as high latency to prefer measured proxies
		latency := stats.AvgLatencyMs
		if latency == 0 {
			latency = 10000 // 10 seconds default for unmeasured
		}
		if latency < minLatency {
			minLatency = latency
			selected = proxy
		}
	}

	return selected
}

// Helper functions

func containsString(slice []string, s string) bool {
	return slices.Contains(slice, s)
}

func hasAllTags(proxyTags, requiredTags []string) bool {
	for _, required := range requiredTags {
		if !slices.Contains(proxyTags, required) {
			return false
		}
	}
	return true
}
