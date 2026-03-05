// Package fetch provides HTTP and headless browser content fetching capabilities.
//
// This file handles headless browser authentication and session management.
// It provides login form detection, automated login flows, and cookie/session
// persistence. Does NOT handle auth profile management (see internal/auth).
package fetch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// performLogin performs a headless login using the provided auth options.
func (f *ChromedpFetcher) performLogin(ctx context.Context, auth AuthOptions) error {
	// If auto-detect is enabled, detect form fields first
	if auth.LoginAutoDetect {
		detected, err := f.detectLoginForm(ctx, auth.LoginURL)
		if err != nil {
			return fmt.Errorf("failed to detect login form: %w", err)
		}
		if detected == nil {
			return errors.New("could not detect login form on page")
		}

		// Use detected selectors
		auth.LoginUserSelector = detected.UserField.Selector
		auth.LoginPassSelector = detected.PassField.Selector
		auth.LoginSubmitSelector = detected.SubmitField.Selector

		slog.Info("login form auto-detected",
			"userSelector", auth.LoginUserSelector,
			"passSelector", auth.LoginPassSelector,
			"submitSelector", auth.LoginSubmitSelector,
			"confidence", detected.Score)
	}

	// Validate selectors are present
	if auth.LoginUserSelector == "" || auth.LoginPassSelector == "" || auth.LoginSubmitSelector == "" {
		return errors.New("login selectors are required (provide manually or use auto-detect)")
	}

	return chromedp.Run(ctx,
		chromedp.Navigate(auth.LoginURL),
		chromedp.WaitVisible(auth.LoginUserSelector),
		chromedp.SendKeys(auth.LoginUserSelector, auth.LoginUser),
		chromedp.SendKeys(auth.LoginPassSelector, auth.LoginPass),
		chromedp.Click(auth.LoginSubmitSelector),
		chromedp.WaitReady("body", chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond),
	)
}

// detectLoginForm uses form detection to analyze the login page and find form fields.
func (f *ChromedpFetcher) detectLoginForm(ctx context.Context, loginURL string) (*DetectedForm, error) {
	// Navigate to login page
	if err := chromedp.Run(ctx, chromedp.Navigate(loginURL)); err != nil {
		return nil, err
	}

	// Wait for body to be ready
	if err := chromedp.Run(ctx, chromedp.WaitReady("body", chromedp.ByQuery)); err != nil {
		return nil, err
	}

	// Extract page HTML
	var html string
	if err := chromedp.Run(ctx, chromedp.OuterHTML("html", &html)); err != nil {
		return nil, err
	}

	// Use form detector
	detector := NewFormDetector()
	form, err := detector.DetectLoginForm(html)
	if err != nil {
		return nil, err
	}

	return form, nil
}

// extractAndSaveSession extracts cookies from the browser and saves them as a session.
func (f *ChromedpFetcher) extractAndSaveSession(ctx context.Context, req Request) error {
	var cookies []*network.Cookie
	err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		cookies, err = network.GetCookies().Do(ctx)
		return err
	}))
	if err != nil {
		return fmt.Errorf("failed to get cookies: %w", err)
	}

	// Convert network.Cookie to sessionCookie
	sessionCookies := make([]sessionCookie, 0, len(cookies))
	for _, c := range cookies {
		sessionCookies = append(sessionCookies, sessionCookie{
			Name:     c.Name,
			Value:    c.Value,
			Domain:   c.Domain,
			Path:     c.Path,
			Secure:   c.Secure,
			HttpOnly: c.HTTPOnly,
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

// saveSession saves a session to the sessions.json file.
func saveSession(dataDir string, newSession session) error {
	sessions, err := loadSessions(dataDir)
	if err != nil {
		return err
	}

	// Update existing or append new
	found := false
	for i := range sessions {
		if sessions[i].ID == newSession.ID {
			sessions[i] = newSession
			found = true
			break
		}
	}
	if !found {
		sessions = append(sessions, newSession)
	}

	return writeSessions(dataDir, sessions)
}

// writeSessions writes all sessions to the sessions.json file.
func writeSessions(dataDir string, sessions []session) error {
	if dataDir == "" {
		dataDir = ".data"
	}

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0o700); err != nil {
		return err
	}

	path := filepath.Join(dataDir, "sessions.json")
	payload, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, payload, 0o600)
}

// extractDomain extracts the domain from a URL.
func extractDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}
