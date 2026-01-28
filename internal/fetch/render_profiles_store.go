// Package fetch provides HTTP and headless browser content fetching capabilities.
// It handles request routing, rate limiting, retry logic, and render profiles.
// It does NOT handle content extraction or parsing.
package fetch

import (
	"encoding/json"
	"os"
	"sync"
	"time"
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

	host := extractHost(rawURL)
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

func extractHost(rawURL string) string {
	// Basic parsing to handle "http://example.com", "https://example.com", "example.com"
	rawURL = trimSpaceASCII(rawURL)
	if rawURL == "" {
		return ""
	}

	if idx := indexSubstring(rawURL, "://"); idx >= 0 {
		rawURL = rawURL[idx+3:]
	}
	if idx := indexByte(rawURL, '/'); idx >= 0 {
		rawURL = rawURL[:idx]
	}
	if idx := indexByte(rawURL, '?'); idx >= 0 {
		rawURL = rawURL[:idx]
	}
	return trimSpaceASCII(rawURL)
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func indexSubstring(s, sub string) int {
	if len(sub) == 0 {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func trimSpaceASCII(s string) string {
	start := 0
	for start < len(s) && isSpaceASCII(s[start]) {
		start++
	}
	end := len(s)
	for end > start && isSpaceASCII(s[end-1]) {
		end--
	}
	return s[start:end]
}

func isSpaceASCII(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\r', '\f', '\v':
		return true
	default:
		return false
	}
}
