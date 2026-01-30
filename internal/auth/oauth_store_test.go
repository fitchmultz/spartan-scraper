// Package auth provides OAuth 2.0 and OIDC authentication support.
// This file contains tests for OAuth storage.
package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOAuthStore_SaveAndLoadState(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewOAuthStore(tmpDir)

	state := OAuth2State{
		State:        "test-state-123",
		CodeVerifier: "test-verifier",
		ProfileName:  "test-profile",
		RedirectURI:  "http://localhost:8080/callback",
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(10 * time.Minute),
	}

	// Save state
	err := store.SaveState(state)
	if err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Load state
	loadedState, found, err := store.LoadState("test-state-123")
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}
	if !found {
		t.Fatal("state should be found")
	}

	if loadedState.State != state.State {
		t.Errorf("state mismatch: got %s, want %s", loadedState.State, state.State)
	}
	if loadedState.CodeVerifier != state.CodeVerifier {
		t.Errorf("code_verifier mismatch")
	}
	if loadedState.ProfileName != state.ProfileName {
		t.Errorf("profile_name mismatch")
	}
}

func TestOAuthStore_LoadState_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewOAuthStore(tmpDir)

	_, found, err := store.LoadState("non-existent-state")
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}
	if found {
		t.Error("should not find non-existent state")
	}
}

func TestOAuthStore_LoadState_Expired(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewOAuthStore(tmpDir)

	state := OAuth2State{
		State:       "expired-state",
		ProfileName: "test-profile",
		CreatedAt:   time.Now().Add(-20 * time.Minute),
		ExpiresAt:   time.Now().Add(-10 * time.Minute), // Expired 10 minutes ago
	}

	err := store.SaveState(state)
	if err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	_, found, err := store.LoadState("expired-state")
	if err != nil {
		t.Fatalf("LoadState failed: %v", err)
	}
	if found {
		t.Error("should not find expired state")
	}
}

func TestOAuthStore_DeleteState(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewOAuthStore(tmpDir)

	state := OAuth2State{
		State:       "delete-me",
		ProfileName: "test-profile",
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}

	err := store.SaveState(state)
	if err != nil {
		t.Fatalf("SaveState failed: %v", err)
	}

	// Verify it exists
	_, found, _ := store.LoadState("delete-me")
	if !found {
		t.Fatal("state should exist before deletion")
	}

	// Delete it
	err = store.DeleteState("delete-me")
	if err != nil {
		t.Fatalf("DeleteState failed: %v", err)
	}

	// Verify it's gone
	_, found, _ = store.LoadState("delete-me")
	if found {
		t.Error("state should be deleted")
	}
}

func TestOAuthStore_SaveAndLoadToken(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewOAuthStore(tmpDir)

	expiresAt := time.Now().Add(time.Hour)
	token := OAuth2Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		ExpiresAt:    &expiresAt,
		Scope:        "openid email",
	}

	// Save token
	err := store.SaveToken("test-profile", token)
	if err != nil {
		t.Fatalf("SaveToken failed: %v", err)
	}

	// Load token
	loadedToken, found, err := store.LoadToken("test-profile")
	if err != nil {
		t.Fatalf("LoadToken failed: %v", err)
	}
	if !found {
		t.Fatal("token should be found")
	}

	if loadedToken.AccessToken != token.AccessToken {
		t.Errorf("access_token mismatch")
	}
	if loadedToken.RefreshToken != token.RefreshToken {
		t.Errorf("refresh_token mismatch")
	}
	if loadedToken.TokenType != token.TokenType {
		t.Errorf("token_type mismatch")
	}
	if loadedToken.Scope != token.Scope {
		t.Errorf("scope mismatch")
	}
}

func TestOAuthStore_LoadToken_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewOAuthStore(tmpDir)

	_, found, err := store.LoadToken("non-existent-profile")
	if err != nil {
		t.Fatalf("LoadToken failed: %v", err)
	}
	if found {
		t.Error("should not find token for non-existent profile")
	}
}

func TestOAuthStore_DeleteToken(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewOAuthStore(tmpDir)

	token := OAuth2Token{
		AccessToken: "test-token",
		TokenType:   "Bearer",
	}

	err := store.SaveToken("delete-profile", token)
	if err != nil {
		t.Fatalf("SaveToken failed: %v", err)
	}

	// Verify it exists
	_, found, _ := store.LoadToken("delete-profile")
	if !found {
		t.Fatal("token should exist before deletion")
	}

	// Delete it
	err = store.DeleteToken("delete-profile")
	if err != nil {
		t.Fatalf("DeleteToken failed: %v", err)
	}

	// Verify it's gone
	_, found, _ = store.LoadToken("delete-profile")
	if found {
		t.Error("token should be deleted")
	}
}

func TestOAuthStore_CleanupExpiredStates(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewOAuthStore(tmpDir)

	// Create an expired state
	expiredState := OAuth2State{
		State:       "expired-state",
		ProfileName: "test",
		CreatedAt:   time.Now().Add(-20 * time.Minute),
		ExpiresAt:   time.Now().Add(-10 * time.Minute),
	}

	// Create a valid state
	validState := OAuth2State{
		State:       "valid-state",
		ProfileName: "test",
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}

	store.SaveState(expiredState)
	store.SaveState(validState)

	// Cleanup
	removed, err := store.CleanupExpiredStates()
	if err != nil {
		t.Fatalf("CleanupExpiredStates failed: %v", err)
	}

	if removed != 1 {
		t.Errorf("expected 1 expired state removed, got %d", removed)
	}

	// Verify expired state is gone
	_, found, _ := store.LoadState("expired-state")
	if found {
		t.Error("expired state should be removed")
	}

	// Verify valid state still exists
	_, found, _ = store.LoadState("valid-state")
	if !found {
		t.Error("valid state should still exist")
	}
}

func TestOAuthStore_ListTokens(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewOAuthStore(tmpDir)

	// Initially empty
	tokens, err := store.ListTokens()
	if err != nil {
		t.Fatalf("ListTokens failed: %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("expected 0 tokens, got %d", len(tokens))
	}

	// Add some tokens
	store.SaveToken("profile1", OAuth2Token{AccessToken: "token1"})
	store.SaveToken("profile2", OAuth2Token{AccessToken: "token2"})

	tokens, err = store.ListTokens()
	if err != nil {
		t.Fatalf("ListTokens failed: %v", err)
	}
	if len(tokens) != 2 {
		t.Errorf("expected 2 tokens, got %d", len(tokens))
	}

	// Check that both profiles are listed
	hasProfile1 := false
	hasProfile2 := false
	for _, name := range tokens {
		if name == "profile1" {
			hasProfile1 = true
		}
		if name == "profile2" {
			hasProfile2 = true
		}
	}
	if !hasProfile1 || !hasProfile2 {
		t.Error("expected both profiles in token list")
	}
}

func TestOAuthStore_CreateOAuthState(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewOAuthStore(tmpDir)

	// Create state without PKCE
	state, verifier, err := store.CreateOAuthState("test-profile", "http://localhost/callback", false)
	if err != nil {
		t.Fatalf("CreateOAuthState failed: %v", err)
	}

	if state == "" {
		t.Error("state should not be empty")
	}
	if verifier != "" {
		t.Error("verifier should be empty when PKCE is not used")
	}

	// Verify state was saved
	loadedState, found, _ := store.LoadState(state)
	if !found {
		t.Fatal("created state should be saved")
	}
	if loadedState.ProfileName != "test-profile" {
		t.Error("profile name mismatch")
	}

	// Create state with PKCE
	state2, verifier2, err := store.CreateOAuthState("test-profile2", "http://localhost/callback", true)
	if err != nil {
		t.Fatalf("CreateOAuthState with PKCE failed: %v", err)
	}

	if verifier2 == "" {
		t.Error("verifier should not be empty when PKCE is used")
	}

	// Verify verifier was saved
	loadedState2, _, _ := store.LoadState(state2)
	if loadedState2.CodeVerifier != verifier2 {
		t.Error("code verifier should be saved with state")
	}
}

func TestOAuthStore_Persistence(t *testing.T) {
	tmpDir := t.TempDir()

	// Create store and save data
	store1 := NewOAuthStore(tmpDir)
	token := OAuth2Token{AccessToken: "persistent-token"}
	store1.SaveToken("persistent-profile", token)

	// Create new store instance pointing to same directory
	store2 := NewOAuthStore(tmpDir)

	// Data should persist
	loadedToken, found, err := store2.LoadToken("persistent-profile")
	if err != nil {
		t.Fatalf("LoadToken failed: %v", err)
	}
	if !found {
		t.Error("token should persist across store instances")
	}
	if loadedToken.AccessToken != "persistent-token" {
		t.Error("token data should be preserved")
	}
}

func TestOAuthStore_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewOAuthStore(tmpDir)

	// Save some data
	store.SaveToken("test", OAuth2Token{AccessToken: "test"})

	// Check file permissions
	tokensPath := filepath.Join(tmpDir, oauthTokensFilename)
	info, err := os.Stat(tokensPath)
	if err != nil {
		t.Fatalf("failed to stat tokens file: %v", err)
	}

	// File should be readable/writable by owner only (0o600)
	mode := info.Mode().Perm()
	if mode != 0o600 {
		t.Errorf("expected file mode 0o600, got 0o%o", mode)
	}
}
