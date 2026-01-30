// Package auth provides session persistence for authenticated scraping.
// Sessions store cookies from successful logins and can be reused across requests.
package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
)

const sessionsFilename = "sessions.json"

// SessionStore manages persisted sessions.
type SessionStore struct {
	dataDir string
	mu      sync.RWMutex
}

// NewSessionStore creates a new session store for the given data directory.
func NewSessionStore(dataDir string) *SessionStore {
	return &SessionStore{dataDir: dataDir}
}

// Load retrieves all sessions from disk.
func (s *SessionStore) Load() ([]Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.sessionsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Session{}, nil
		}
		return nil, err
	}

	var sessions []Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

// Save persists all sessions to disk.
func (s *SessionStore) Save(sessions []Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := fsutil.EnsureDataDir(s.dataDirOrDefault()); err != nil {
		return err
	}

	path := s.sessionsPath()
	payload, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o600)
}

// Get retrieves a session by ID.
func (s *SessionStore) Get(id string) (Session, bool, error) {
	sessions, err := s.Load()
	if err != nil {
		return Session{}, false, err
	}
	for _, sess := range sessions {
		if sess.ID == id {
			return sess, true, nil
		}
	}
	return Session{}, false, nil
}

// Upsert creates or updates a session.
func (s *SessionStore) Upsert(session Session) error {
	if strings.TrimSpace(session.ID) == "" {
		return apperrors.Validation("session ID is required")
	}

	sessions, err := s.Load()
	if err != nil {
		return err
	}

	now := time.Now()
	session.UpdatedAt = now
	if session.CreatedAt.IsZero() {
		session.CreatedAt = now
	}

	found := false
	for i := range sessions {
		if sessions[i].ID == session.ID {
			sessions[i] = session
			found = true
			break
		}
	}
	if !found {
		sessions = append(sessions, session)
	}

	return s.Save(sessions)
}

// Delete removes a session by ID.
func (s *SessionStore) Delete(id string) error {
	sessions, err := s.Load()
	if err != nil {
		return err
	}

	filtered := make([]Session, 0, len(sessions))
	for _, sess := range sessions {
		if sess.ID != id {
			filtered = append(filtered, sess)
		}
	}

	return s.Save(filtered)
}

// List returns all sessions.
func (s *SessionStore) List() ([]Session, error) {
	return s.Load()
}

func (s *SessionStore) sessionsPath() string {
	return filepath.Join(s.dataDirOrDefault(), sessionsFilename)
}

func (s *SessionStore) dataDirOrDefault() string {
	if s.dataDir != "" {
		return s.dataDir
	}
	return ".data"
}

// CookiesFromJar extracts cookies from a net/http cookiejar for a given domain.
func CookiesFromJar(jar http.CookieJar, domain string) []Cookie {
	// Create a URL to extract cookies for the domain
	u := &url.URL{Scheme: "https", Host: domain}
	httpCookies := jar.Cookies(u)

	cookies := make([]Cookie, 0, len(httpCookies))
	for _, c := range httpCookies {
		var expires *time.Time
		if !c.Expires.IsZero() {
			e := c.Expires
			expires = &e
		}
		cookies = append(cookies, Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Expires:  expires,
			Secure:   c.Secure,
			HttpOnly: c.HttpOnly,
			SameSite: sameSiteToString(c.SameSite),
		})
	}
	return cookies
}

// CookiesToHTTP converts auth.Cookie to http.Cookie.
func CookiesToHTTP(cookies []Cookie) []*http.Cookie {
	result := make([]*http.Cookie, 0, len(cookies))
	for _, c := range cookies {
		httpCookie := &http.Cookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HttpOnly: c.HttpOnly,
		}
		if c.Expires != nil && !c.Expires.IsZero() {
			httpCookie.Expires = *c.Expires
		}
		result = append(result, httpCookie)
	}
	return result
}

// ApplySessionToJar adds session cookies to a cookiejar.
func ApplySessionToJar(jar http.CookieJar, session Session, targetURL string) error {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return err
	}

	httpCookies := CookiesToHTTP(session.Cookies)
	jar.SetCookies(parsed, httpCookies)
	return nil
}

func sameSiteToString(s http.SameSite) string {
	switch s {
	case http.SameSiteLaxMode:
		return "Lax"
	case http.SameSiteStrictMode:
		return "Strict"
	case http.SameSiteNoneMode:
		return "None"
	default:
		return ""
	}
}

// ExtractDomain extracts the domain from a URL.
func ExtractDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}
