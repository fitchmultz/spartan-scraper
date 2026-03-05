// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/hostmatch"
)

type RenderProfileStore struct {
	path        string
	lastModTime time.Time
	profiles    []RenderProfile
	mu          sync.RWMutex
}

// NewRenderProfileStore initializes a new store. It attempts to load profiles immediately.
func NewRenderProfileStore(dataDir string) *RenderProfileStore {
	s := &RenderProfileStore{
		path:     RenderProfilesPath(dataDir),
		profiles: []RenderProfile{},
	}
	_ = s.Reload()
	return s
}

// Reload loads profiles from disk. If file is missing, profiles become empty. Idempotent.
func (s *RenderProfileStore) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			s.profiles = []RenderProfile{}
			s.lastModTime = time.Time{}
			return nil
		}
		return err
	}

	info, statErr := os.Stat(s.path)
	if statErr != nil {
		// Should not happen if ReadFile succeeded, but handle gracefully
		s.lastModTime = time.Time{}
	} else {
		s.lastModTime = info.ModTime()
	}

	var pf RenderProfilesFile
	if err := json.Unmarshal(data, &pf); err != nil {
		return err
	}

	s.profiles = pf.Profiles
	return nil
}

// Profiles returns a copy of all loaded profiles.
func (s *RenderProfileStore) Profiles() []RenderProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]RenderProfile, len(s.profiles))
	copy(out, s.profiles)
	return out
}

// ReloadIfChanged checks file modification time and reloads if necessary.
func (s *RenderProfileStore) ReloadIfChanged() error {
	s.mu.RLock()
	currentMod := s.lastModTime
	s.mu.RUnlock()

	info, err := os.Stat(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist. If we had profiles, clear them.
			s.mu.Lock()
			if len(s.profiles) > 0 {
				s.profiles = []RenderProfile{}
				s.lastModTime = time.Time{}
			}
			s.mu.Unlock()
			return nil
		}
		return err
	}

	if info.ModTime().Equal(currentMod) {
		return nil // No change
	}

	return s.Reload()
}

// MatchURL returns the highest-precedence matching profile for a given URL.
// Precedence: first match in file order (user-controlled), deterministic.
func (s *RenderProfileStore) MatchURL(rawURL string) (*RenderProfile, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	host := hostmatch.HostFromURL(rawURL)
	if host == "" {
		return nil, false, nil
	}

	for _, profile := range s.profiles {
		if HostMatchesAnyPattern(host, profile.HostPatterns) {
			return &profile, true, nil
		}
	}
	return nil, false, nil
}

// GetRateLimitsForURL returns the rate limit configuration for a given URL.
// Returns (0, 0) if no matching profile or if profile has no rate limits set.
func (s *RenderProfileStore) GetRateLimitsForURL(rawURL string) (qps int, burst int) {
	profile, found, _ := s.MatchURL(rawURL)
	if !found || profile == nil {
		return 0, 0
	}
	return profile.RateLimitQPS, profile.RateLimitBurst
}
