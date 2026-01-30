// Package auth provides OAuth 2.0 and OIDC authentication support.
// This file implements OAuth state and token storage.
package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/fsutil"
)

const (
	oauthStateFilename  = "oauth_state.json"
	oauthTokensFilename = "oauth_tokens.json"
	stateExpiry         = 10 * time.Minute
)

// OAuthStore manages OAuth 2.0 state and token persistence.
// It provides thread-safe access to OAuth flow state and stored tokens.
type OAuthStore struct {
	dataDir string
	mu      sync.RWMutex
}

// NewOAuthStore creates a new OAuth store.
func NewOAuthStore(dataDir string) *OAuthStore {
	return &OAuthStore{dataDir: dataDir}
}

func (s *OAuthStore) statePath() string {
	return filepath.Join(s.dataDir, oauthStateFilename)
}

func (s *OAuthStore) tokensPath() string {
	return filepath.Join(s.dataDir, oauthTokensFilename)
}

// loadStates loads all OAuth states from disk.
func (s *OAuthStore) loadStates() (map[string]OAuth2State, error) {
	path := s.statePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]OAuth2State), nil
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to read OAuth state file", err)
	}

	var states map[string]OAuth2State
	if err := json.Unmarshal(data, &states); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to decode OAuth state", err)
	}

	return states, nil
}

// saveStates saves all OAuth states to disk.
func (s *OAuthStore) saveStates(states map[string]OAuth2State) error {
	if err := fsutil.EnsureDataDir(s.dataDir); err != nil {
		return err
	}

	path := s.statePath()
	payload, err := json.MarshalIndent(states, "", "  ")
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to encode OAuth state", err)
	}

	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to write OAuth state file", err)
	}

	return nil
}

// loadTokens loads all OAuth tokens from disk.
func (s *OAuthStore) loadTokens() (map[string]OAuth2Token, error) {
	path := s.tokensPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]OAuth2Token), nil
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to read OAuth tokens file", err)
	}

	var tokens map[string]OAuth2Token
	if err := json.Unmarshal(data, &tokens); err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to decode OAuth tokens", err)
	}

	return tokens, nil
}

// saveTokens saves all OAuth tokens to disk.
func (s *OAuthStore) saveTokens(tokens map[string]OAuth2Token) error {
	if err := fsutil.EnsureDataDir(s.dataDir); err != nil {
		return err
	}

	path := s.tokensPath()
	payload, err := json.MarshalIndent(tokens, "", "  ")
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to encode OAuth tokens", err)
	}

	if err := os.WriteFile(path, payload, 0o600); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to write OAuth tokens file", err)
	}

	return nil
}

// SaveState persists an OAuth2State (for CSRF protection).
func (s *OAuthStore) SaveState(state OAuth2State) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	states, err := s.loadStates()
	if err != nil {
		return err
	}

	states[state.State] = state
	return s.saveStates(states)
}

// LoadState retrieves and validates an OAuth2State by state parameter.
// Returns the state, a boolean indicating if found and valid, and any error.
func (s *OAuthStore) LoadState(state string) (OAuth2State, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	states, err := s.loadStates()
	if err != nil {
		return OAuth2State{}, false, err
	}

	oauthState, ok := states[state]
	if !ok {
		return OAuth2State{}, false, nil
	}

	// Check if state has expired
	if time.Now().After(oauthState.ExpiresAt) {
		return OAuth2State{}, false, nil
	}

	return oauthState, true, nil
}

// DeleteState removes a state after use.
func (s *OAuthStore) DeleteState(state string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	states, err := s.loadStates()
	if err != nil {
		return err
	}

	delete(states, state)
	return s.saveStates(states)
}

// SaveToken persists OAuth2 tokens for a profile.
func (s *OAuthStore) SaveToken(profileName string, token OAuth2Token) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokens, err := s.loadTokens()
	if err != nil {
		return err
	}

	tokens[profileName] = token
	return s.saveTokens(tokens)
}

// LoadToken retrieves OAuth2 tokens for a profile.
// Returns the token, a boolean indicating if found, and any error.
func (s *OAuthStore) LoadToken(profileName string) (OAuth2Token, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tokens, err := s.loadTokens()
	if err != nil {
		return OAuth2Token{}, false, err
	}

	token, ok := tokens[profileName]
	if !ok {
		return OAuth2Token{}, false, nil
	}

	return token, true, nil
}

// DeleteToken removes OAuth2 tokens for a profile.
func (s *OAuthStore) DeleteToken(profileName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokens, err := s.loadTokens()
	if err != nil {
		return err
	}

	delete(tokens, profileName)
	return s.saveTokens(tokens)
}

// CleanupExpiredStates removes expired state entries.
// Returns the number of states removed.
func (s *OAuthStore) CleanupExpiredStates() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	states, err := s.loadStates()
	if err != nil {
		return 0, err
	}

	now := time.Now()
	removed := 0

	for stateKey, state := range states {
		if now.After(state.ExpiresAt) {
			delete(states, stateKey)
			removed++
		}
	}

	if removed > 0 {
		if err := s.saveStates(states); err != nil {
			return removed, err
		}
	}

	return removed, nil
}

// ListTokens returns a list of profile names that have stored tokens.
func (s *OAuthStore) ListTokens() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tokens, err := s.loadTokens()
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(tokens))
	for name := range tokens {
		names = append(names, name)
	}

	return names, nil
}

// CreateOAuthState creates and stores a new OAuth state for CSRF protection.
// Returns the generated state string and any error.
func (s *OAuthStore) CreateOAuthState(profileName string, redirectURI string, usePKCE bool) (string, string, error) {
	state, err := GenerateOAuthState()
	if err != nil {
		return "", "", err
	}

	oauthState := OAuth2State{
		State:       state,
		ProfileName: profileName,
		RedirectURI: redirectURI,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(stateExpiry),
	}

	if usePKCE {
		verifier, _, _, err := GeneratePKCE()
		if err != nil {
			return "", "", err
		}
		oauthState.CodeVerifier = verifier
	}

	if err := s.SaveState(oauthState); err != nil {
		return "", "", err
	}

	return state, oauthState.CodeVerifier, nil
}
