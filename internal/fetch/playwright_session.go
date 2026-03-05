// Package fetch provides HTTP and headless browser content fetching capabilities.
//
// This file provides session/cookie extraction for Playwright fetcher.
// It handles extracting cookies from browser contexts and saving them as sessions
// for later reuse. Does NOT handle session loading or authentication flows.
package fetch

import (
	"fmt"
	"log/slog"

	"github.com/playwright-community/playwright-go"
)

// extractAndSaveSession extracts cookies from the browser context and saves them as a session.
func (f *PlaywrightFetcher) extractAndSaveSession(browserCtx playwright.BrowserContext, req Request) error {
	cookies, err := browserCtx.Cookies()
	if err != nil {
		return fmt.Errorf("failed to get cookies: %w", err)
	}

	// Convert playwright.Cookie to sessionCookie
	sessionCookies := make([]sessionCookie, 0, len(cookies))
	for _, c := range cookies {
		sessionCookies = append(sessionCookies, sessionCookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HttpOnly: c.HttpOnly,
		})
	}

	// Save session
	sess := session{
		ID:      req.SessionID,
		Name:    req.SessionID,
		Domain:  extractDomain(req.URL),
		Cookies: sessionCookies,
	}

	if err := saveSession(req.DataDir, sess); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	slog.Info("saved session cookies", "sessionID", req.SessionID, "domain", sess.Domain, "count", len(sessionCookies))
	return nil
}
