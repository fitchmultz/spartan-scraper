// Package crawl provides functionality for crawling multiple pages of a website.
// This file implements robots.txt parsing and caching for optional compliance.
package crawl

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Cache provides thread-safe caching of robots.txt data per host with TTL.
type Cache struct {
	client *http.Client
	ttl    time.Duration
	mu     sync.RWMutex
	data   map[string]*cacheEntry
}

type cacheEntry struct {
	ruleset   *Ruleset
	fetchedAt time.Time
}

// Ruleset represents parsed robots.txt rules for a specific user-agent.
type Ruleset struct {
	Rules      []Rule
	CrawlDelay time.Duration
	groups     []agentGroup // Internal storage for user-agent matching
}

// Rule represents a single allow/disallow rule.
type Rule struct {
	Pattern     string
	Allowed     bool
	IsPrefix    bool // true if pattern ends with /
	HasWildcard bool // true if pattern contains *
}

// NewCache creates a new robots.txt cache with the given HTTP client and TTL.
// If client is nil, a default http.Client with 30s timeout is used.
// If ttl is 0, a default of 1 hour is used.
func NewCache(client *http.Client, ttl time.Duration) *Cache {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	if ttl <= 0 {
		ttl = time.Hour
	}
	return &Cache{
		client: client,
		ttl:    ttl,
		data:   make(map[string]*cacheEntry),
	}
}

// IsAllowed checks if a URL is allowed for the given user-agent according to robots.txt.
// Returns true if allowed, false if disallowed. On fetch errors, returns true (fail-open).
func (c *Cache) IsAllowed(rawURL, userAgent string) (bool, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return true, fmt.Errorf("invalid URL: %w", err)
	}

	host := parsedURL.Host
	if host == "" {
		return true, fmt.Errorf("URL has no host")
	}

	ruleset, err := c.getRuleset(host, userAgent)
	if err != nil {
		// Fail-open: allow on error
		slog.Debug("robots.txt fetch failed, allowing URL", "host", host, "error", err)
		return true, nil
	}

	if ruleset == nil {
		// No rules for this user-agent, allow all
		return true, nil
	}

	urlPath := parsedURL.Path
	if urlPath == "" {
		urlPath = "/"
	}

	return ruleset.IsAllowed(urlPath), nil
}

// GetCrawlDelay returns the crawl delay for the given host and user-agent.
// Returns 0 if no crawl delay is specified.
func (c *Cache) GetCrawlDelay(host, userAgent string) time.Duration {
	ruleset, err := c.getRuleset(host, userAgent)
	if err != nil || ruleset == nil {
		return 0
	}
	return ruleset.CrawlDelay
}

// getRuleset retrieves or fetches the ruleset for a host and user-agent.
func (c *Cache) getRuleset(host, userAgent string) (*Ruleset, error) {
	// Check cache first
	c.mu.RLock()
	entry, exists := c.data[host]
	c.mu.RUnlock()

	if exists && time.Since(entry.fetchedAt) < c.ttl {
		return entry.ruleset.ForUserAgent(userAgent), nil
	}

	// Need to fetch
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	entry, exists = c.data[host]
	if exists && time.Since(entry.fetchedAt) < c.ttl {
		return entry.ruleset.ForUserAgent(userAgent), nil
	}

	// Fetch robots.txt
	rawRuleset, err := c.fetchRobotsTxt(host)
	if err != nil {
		// Cache the error to avoid repeated failed fetches
		c.data[host] = &cacheEntry{
			ruleset:   nil,
			fetchedAt: time.Now(),
		}
		return nil, err
	}

	c.data[host] = &cacheEntry{
		ruleset:   rawRuleset,
		fetchedAt: time.Now(),
	}

	return rawRuleset.ForUserAgent(userAgent), nil
}

// fetchRobotsTxt fetches and parses robots.txt for the given host.
func (c *Cache) fetchRobotsTxt(host string) (*Ruleset, error) {
	robotsURL := fmt.Sprintf("https://%s/robots.txt", host)
	resp, err := c.client.Get(robotsURL)
	if err != nil {
		// Try HTTP as fallback
		robotsURL = fmt.Sprintf("http://%s/robots.txt", host)
		resp, err = c.client.Get(robotsURL)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch robots.txt: %w", err)
		}
	}
	defer resp.Body.Close()

	// 404 means no robots.txt, which means allow all
	if resp.StatusCode == http.StatusNotFound {
		return &Ruleset{}, nil
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("robots.txt returned status %d", resp.StatusCode)
	}

	return parseRobotsTxt(resp.Body)
}

// parseRobotsTxt parses robots.txt content into a Ruleset.
func parseRobotsTxt(r io.Reader) (*Ruleset, error) {
	scanner := bufio.NewScanner(r)
	var groups []agentGroup
	var currentGroup *agentGroup

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Remove inline comments
		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}

		// Parse directive
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue // Invalid line, skip
		}

		directive := strings.TrimSpace(strings.ToLower(line[:colonIdx]))
		value := strings.TrimSpace(line[colonIdx+1:])

		switch directive {
		case "user-agent":
			// Start a new group if we were already building one with rules
			if currentGroup != nil && len(currentGroup.rules) > 0 {
				groups = append(groups, *currentGroup)
			}
			currentGroup = &agentGroup{
				agents: []string{strings.ToLower(value)},
			}
		case "disallow":
			if currentGroup == nil {
				currentGroup = &agentGroup{}
			}
			currentGroup.rules = append(currentGroup.rules, Rule{
				Pattern:     value,
				Allowed:     false,
				IsPrefix:    strings.HasSuffix(value, "/"),
				HasWildcard: strings.Contains(value, "*"),
			})
		case "allow":
			if currentGroup == nil {
				currentGroup = &agentGroup{}
			}
			currentGroup.rules = append(currentGroup.rules, Rule{
				Pattern:     value,
				Allowed:     true,
				IsPrefix:    strings.HasSuffix(value, "/"),
				HasWildcard: strings.Contains(value, "*"),
			})
		case "crawl-delay":
			if currentGroup == nil {
				currentGroup = &agentGroup{}
			}
			var delay int
			if _, err := fmt.Sscanf(value, "%d", &delay); err == nil && delay > 0 {
				currentGroup.crawlDelay = time.Duration(delay) * time.Second
			}
		}
	}

	// Add the last group
	if currentGroup != nil {
		groups = append(groups, *currentGroup)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading robots.txt: %w", err)
	}

	return &Ruleset{
		groups: groups,
	}, nil
}

// agentGroup represents a group of user-agents with shared rules.
type agentGroup struct {
	agents     []string
	rules      []Rule
	crawlDelay time.Duration
}

// ForUserAgent returns the ruleset for a specific user-agent.
// Implements the "most specific match" rule from the robots.txt spec.
func (r *Ruleset) ForUserAgent(userAgent string) *Ruleset {
	if r == nil || len(r.groups) == 0 {
		return &Ruleset{}
	}

	ua := strings.ToLower(userAgent)
	var bestMatch *agentGroup
	bestMatchLen := -1

	for i := range r.groups {
		group := &r.groups[i]
		for _, agent := range group.agents {
			// Exact match
			if agent == ua {
				return &Ruleset{
					Rules:      group.rules,
					CrawlDelay: group.crawlDelay,
				}
			}
			// Wildcard match
			if agent == "*" {
				if bestMatchLen < 0 {
					bestMatch = group
					bestMatchLen = 0
				}
				continue
			}
			// Partial match (e.g., "Googlebot" matches "Googlebot-News")
			if strings.HasPrefix(ua, agent) {
				if len(agent) > bestMatchLen {
					bestMatch = group
					bestMatchLen = len(agent)
				}
			}
		}
	}

	if bestMatch != nil {
		return &Ruleset{
			Rules:      bestMatch.rules,
			CrawlDelay: bestMatch.crawlDelay,
		}
	}

	return &Ruleset{}
}

// IsAllowed checks if a path is allowed according to the rules.
// Implements the "longest match" rule from the robots.txt spec.
func (r *Ruleset) IsAllowed(urlPath string) bool {
	if r == nil || len(r.Rules) == 0 {
		return true
	}

	// Ensure path starts with /
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}

	// Decode the path for matching
	decodedPath, err := url.PathUnescape(urlPath)
	if err != nil {
		decodedPath = urlPath
	}

	var longestMatch *Rule
	longestMatchLen := -1

	for i := range r.Rules {
		rule := &r.Rules[i]
		if matchRule(decodedPath, rule.Pattern) {
			matchLen := len(rule.Pattern)
			if matchLen > longestMatchLen {
				longestMatch = rule
				longestMatchLen = matchLen
			}
		}
	}

	if longestMatch != nil {
		return longestMatch.Allowed
	}

	return true
}

// matchRule checks if a path matches a robots.txt pattern.
// Supports * (matches any sequence) and $ (matches end of path).
func matchRule(urlPath, pattern string) bool {
	if pattern == "" {
		return true // Empty pattern matches everything
	}

	// Handle end-of-path anchor
	endAnchor := false
	if strings.HasSuffix(pattern, "$") {
		endAnchor = true
		pattern = pattern[:len(pattern)-1]
	}

	// If pattern has wildcards, use wildcard matching
	if strings.Contains(pattern, "*") {
		return matchWildcard(urlPath, pattern, endAnchor)
	}

	// Simple prefix matching
	if endAnchor {
		return urlPath == pattern
	}

	// Check if urlPath starts with pattern
	if !strings.HasPrefix(urlPath, pattern) {
		return false
	}

	// If pattern ends with /, it's a directory prefix match
	if strings.HasSuffix(pattern, "/") {
		return true
	}

	// Otherwise, it's a complete segment match or the next char must be /
	nextIdx := len(pattern)
	if nextIdx < len(urlPath) && urlPath[nextIdx] != '/' {
		return false
	}

	return true
}

// matchWildcard matches a path against a pattern containing * wildcards.
// In robots.txt, * matches any sequence of characters (including empty).
// This is the standard behavior used by Google and others.
func matchWildcard(urlPath, pattern string, endAnchor bool) bool {
	// Handle edge cases
	if pattern == "" {
		return !endAnchor || urlPath == ""
	}
	if pattern == "*" {
		return true
	}

	// Split pattern by * and match parts sequentially
	parts := strings.Split(pattern, "*")

	pos := 0
	firstPart := true
	for _, part := range parts {
		if part == "" {
			continue // Skip empty parts from consecutive * or leading/trailing *
		}

		if firstPart {
			// First non-empty part must match at the beginning
			if !strings.HasPrefix(urlPath, part) {
				return false
			}
			pos = len(part)
			firstPart = false
		} else {
			// Subsequent parts must match somewhere after previous match
			// * can match empty, so we allow the part to match at the current position
			idx := strings.Index(urlPath[pos:], part)
			if idx == -1 {
				return false
			}
			pos += idx + len(part)
		}
	}

	// If end anchor is set, we must be at the end
	if endAnchor && pos != len(urlPath) {
		return false
	}

	return true
}
