// Package sessions provides session management for web UI authentication.
//
// This file tests session creation, validation, and management.
package sessions

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/store"
)

func setupTestStore(t *testing.T) (*store.Store, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "sessions_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	s, err := store.Open(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to open store: %v", err)
	}

	cleanup := func() {
		s.Close()
		os.RemoveAll(tmpDir)
	}

	return s, cleanup
}

func TestStore_CreateSession(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	store := NewStore(s)
	ctx := context.Background()

	tests := []struct {
		name      string
		userID    string
		ipAddress string
		userAgent string
		duration  time.Duration
		wantErr   bool
	}{
		{
			name:      "valid session",
			userID:    "user-123",
			ipAddress: "127.0.0.1",
			userAgent: "Mozilla/5.0",
			duration:  time.Hour,
			wantErr:   false,
		},
		{
			name:      "empty user id",
			userID:    "",
			ipAddress: "127.0.0.1",
			userAgent: "Mozilla/5.0",
			duration:  time.Hour,
			wantErr:   true,
		},
		{
			name:      "default duration",
			userID:    "user-456",
			ipAddress: "127.0.0.1",
			userAgent: "Mozilla/5.0",
			duration:  0, // Should use default
			wantErr:   false,
		},
		{
			name:      "long duration",
			userID:    "user-789",
			ipAddress: "127.0.0.1",
			userAgent: "Mozilla/5.0",
			duration:  30 * 24 * time.Hour, // 30 days
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, token, err := store.CreateSession(ctx, tt.userID, tt.ipAddress, tt.userAgent, tt.duration)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSession() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if session == nil {
					t.Error("CreateSession() returned nil session")
					return
				}
				if token == "" {
					t.Error("CreateSession() returned empty token")
				}
				if session.UserID != tt.userID {
					t.Errorf("CreateSession() userID = %v, want %v", session.UserID, tt.userID)
				}
				if session.ID == "" {
					t.Error("CreateSession() session ID is empty")
				}
				// Check expiration is in the future
				if session.ExpiresAt.Before(time.Now()) {
					t.Error("CreateSession() expires_at should be in the future")
				}
			}
		})
	}
}

func TestStore_ValidateSession(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	store := NewStore(s)
	ctx := context.Background()

	// Create a valid session
	session, token, err := store.CreateSession(ctx, "user-123", "127.0.0.1", "Mozilla/5.0", time.Hour)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "valid token",
			token:   token,
			wantErr: false,
		},
		{
			name:    "empty token",
			token:   "",
			wantErr: true,
		},
		{
			name:    "invalid token",
			token:   "invalid-token-12345",
			wantErr: true,
		},
		{
			name:    "wrong token format",
			token:   "not-a-hex-token-!!!",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validated, err := store.ValidateSession(ctx, tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSession() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if validated.ID != session.ID {
					t.Errorf("ValidateSession() session ID = %v, want %v", validated.ID, session.ID)
				}
				if validated.UserID != session.UserID {
					t.Errorf("ValidateSession() userID = %v, want %v", validated.UserID, session.UserID)
				}
			}
		})
	}
}

func TestStore_ValidateSession_Expired(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	store := NewStore(s)
	ctx := context.Background()

	// Create a session that expires immediately
	_, token, err := store.CreateSession(ctx, "user-123", "127.0.0.1", "Mozilla/5.0", time.Millisecond)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Wait for session to expire
	time.Sleep(10 * time.Millisecond)

	// Try to validate expired session
	_, err = store.ValidateSession(ctx, token)
	if err == nil {
		t.Error("ValidateSession() should return error for expired session")
	}
}

func TestStore_DeleteSession(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	store := NewStore(s)
	ctx := context.Background()

	// Create a session
	_, token, err := store.CreateSession(ctx, "user-123", "127.0.0.1", "Mozilla/5.0", time.Hour)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Verify session is valid
	_, err = store.ValidateSession(ctx, token)
	if err != nil {
		t.Fatalf("Session should be valid: %v", err)
	}

	// Delete the session
	err = store.DeleteSession(ctx, token)
	if err != nil {
		t.Errorf("DeleteSession() error = %v", err)
	}

	// Verify session is no longer valid
	_, err = store.ValidateSession(ctx, token)
	if err == nil {
		t.Error("ValidateSession() should return error after deletion")
	}
}

func TestStore_DeleteSession_EmptyToken(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	store := NewStore(s)
	ctx := context.Background()

	err := store.DeleteSession(ctx, "")
	if err == nil {
		t.Error("DeleteSession() should return error for empty token")
	}
}

func TestStore_DeleteSession_NonExistent(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	store := NewStore(s)
	ctx := context.Background()

	// Deleting a non-existent session should not error
	err := store.DeleteSession(ctx, "non-existent-token-1234567890abcdef")
	if err != nil {
		t.Errorf("DeleteSession() error = %v for non-existent session", err)
	}
}

func TestStore_DeleteSessionByID(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	store := NewStore(s)
	ctx := context.Background()

	// Create a session
	session, token, err := store.CreateSession(ctx, "user-123", "127.0.0.1", "Mozilla/5.0", time.Hour)
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}

	// Delete by ID
	err = store.DeleteSessionByID(ctx, session.ID)
	if err != nil {
		t.Errorf("DeleteSessionByID() error = %v", err)
	}

	// Verify session is no longer valid
	_, err = store.ValidateSession(ctx, token)
	if err == nil {
		t.Error("ValidateSession() should return error after deletion by ID")
	}
}

func TestStore_DeleteSessionByID_EmptyID(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	store := NewStore(s)
	ctx := context.Background()

	err := store.DeleteSessionByID(ctx, "")
	if err == nil {
		t.Error("DeleteSessionByID() should return error for empty ID")
	}
}

func TestStore_DeleteUserSessions(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	store := NewStore(s)
	ctx := context.Background()

	userID := "user-123"

	// Create multiple sessions for the user
	_, token1, _ := store.CreateSession(ctx, userID, "127.0.0.1", "Mozilla/5.0", time.Hour)
	_, token2, _ := store.CreateSession(ctx, userID, "192.168.1.1", "Chrome/90", time.Hour)

	// Create session for another user
	_, otherToken, _ := store.CreateSession(ctx, "other-user", "127.0.0.1", "Mozilla/5.0", time.Hour)

	// Delete all sessions for user
	err := store.DeleteUserSessions(ctx, userID)
	if err != nil {
		t.Errorf("DeleteUserSessions() error = %v", err)
	}

	// Verify user's sessions are deleted
	_, err = store.ValidateSession(ctx, token1)
	if err == nil {
		t.Error("ValidateSession() should return error for deleted session 1")
	}
	_, err = store.ValidateSession(ctx, token2)
	if err == nil {
		t.Error("ValidateSession() should return error for deleted session 2")
	}

	// Verify other user's session is still valid
	_, err = store.ValidateSession(ctx, otherToken)
	if err != nil {
		t.Errorf("ValidateSession() should succeed for other user's session: %v", err)
	}
}

func TestStore_DeleteUserSessions_EmptyUserID(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	store := NewStore(s)
	ctx := context.Background()

	err := store.DeleteUserSessions(ctx, "")
	if err == nil {
		t.Error("DeleteUserSessions() should return error for empty user ID")
	}
}

func TestStore_CleanupExpiredSessions(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	store := NewStore(s)
	ctx := context.Background()

	// Create an expired session
	_, expiredToken, _ := store.CreateSession(ctx, "user-123", "127.0.0.1", "Mozilla/5.0", time.Millisecond)

	// Create a valid session
	_, validToken, _ := store.CreateSession(ctx, "user-123", "127.0.0.1", "Mozilla/5.0", time.Hour)

	// Wait for first session to expire
	time.Sleep(10 * time.Millisecond)

	// Run cleanup
	err := store.CleanupExpiredSessions(ctx)
	if err != nil {
		t.Errorf("CleanupExpiredSessions() error = %v", err)
	}

	// Verify expired session is gone
	_, err = store.ValidateSession(ctx, expiredToken)
	if err == nil {
		t.Error("ValidateSession() should return error for expired session after cleanup")
	}

	// Verify valid session still works
	_, err = store.ValidateSession(ctx, validToken)
	if err != nil {
		t.Errorf("ValidateSession() should succeed for valid session: %v", err)
	}
}

func TestStore_ListUserSessions(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	store := NewStore(s)
	ctx := context.Background()

	userID := "user-123"

	// Create sessions for user
	session1, _, _ := store.CreateSession(ctx, userID, "127.0.0.1", "Mozilla/5.0", time.Hour)
	session2, _, _ := store.CreateSession(ctx, userID, "192.168.1.1", "Chrome/90", time.Hour)

	// Create session for another user
	_, _, _ = store.CreateSession(ctx, "other-user", "127.0.0.1", "Mozilla/5.0", time.Hour)

	sessions, err := store.ListUserSessions(ctx, userID)
	if err != nil {
		t.Errorf("ListUserSessions() error = %v", err)
		return
	}

	if len(sessions) != 2 {
		t.Errorf("ListUserSessions() returned %d sessions, want 2", len(sessions))
	}

	// Check that both sessions are present
	sessionIDs := make(map[string]bool)
	for _, s := range sessions {
		sessionIDs[s.ID] = true
	}
	if !sessionIDs[session1.ID] {
		t.Error("ListUserSessions() missing session1")
	}
	if !sessionIDs[session2.ID] {
		t.Error("ListUserSessions() missing session2")
	}
}

func TestStore_ListUserSessions_EmptyUserID(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	store := NewStore(s)
	ctx := context.Background()

	_, err := store.ListUserSessions(ctx, "")
	if err == nil {
		t.Error("ListUserSessions() should return error for empty user ID")
	}
}

func TestStore_ListUserSessions_ExcludesExpired(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	store := NewStore(s)
	ctx := context.Background()

	userID := "user-123"

	// Create a valid session
	validSession, _, _ := store.CreateSession(ctx, userID, "127.0.0.1", "Mozilla/5.0", time.Hour)

	// Create an expired session
	_, _, _ = store.CreateSession(ctx, userID, "192.168.1.1", "Chrome/90", time.Millisecond)

	// Wait for session to expire
	time.Sleep(10 * time.Millisecond)

	sessions, err := store.ListUserSessions(ctx, userID)
	if err != nil {
		t.Errorf("ListUserSessions() error = %v", err)
		return
	}

	if len(sessions) != 1 {
		t.Errorf("ListUserSessions() returned %d sessions, want 1 (expired excluded)", len(sessions))
	}

	if sessions[0].ID != validSession.ID {
		t.Error("ListUserSessions() returned wrong session")
	}
}

func TestHashToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "simple token",
			token: "simple-token-123",
		},
		{
			name:  "hex token",
			token: "abcdef1234567890abcdef1234567890",
		},
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "long token",
			token: "this-is-a-very-long-token-with-many-characters-1234567890-abcdefghijklmnopqrstuvwxyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := hashToken(tt.token)
			hash2 := hashToken(tt.token)

			// Same token should produce same hash
			if hash1 != hash2 {
				t.Error("hashToken() should produce consistent hashes")
			}

			// Hash should be 64 characters (SHA-256 hex)
			if len(hash1) != 64 {
				t.Errorf("hashToken() returned hash of length %d, want 64", len(hash1))
			}
		})
	}
}

func TestHashToken_DifferentTokens(t *testing.T) {
	token1 := "token-one-1234567890"
	token2 := "token-two-0987654321"

	hash1 := hashToken(token1)
	hash2 := hashToken(token2)

	if hash1 == hash2 {
		t.Error("hashToken() should produce different hashes for different tokens")
	}
}

func TestSession_TokenUniqueness(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	store := NewStore(s)
	ctx := context.Background()

	// Create multiple sessions
	tokens := make(map[string]bool)
	for i := 0; i < 10; i++ {
		_, token, err := store.CreateSession(ctx, "user-123", "127.0.0.1", "Mozilla/5.0", time.Hour)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}
		if tokens[token] {
			t.Error("CreateSession() produced duplicate token")
		}
		tokens[token] = true
	}
}

func TestSession_DifferentUsers(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	store := NewStore(s)
	ctx := context.Background()

	// Create sessions for different users
	session1, token1, _ := store.CreateSession(ctx, "user-1", "127.0.0.1", "Mozilla/5.0", time.Hour)
	session2, token2, _ := store.CreateSession(ctx, "user-2", "127.0.0.1", "Mozilla/5.0", time.Hour)

	if session1.UserID != "user-1" {
		t.Errorf("Session1 userID = %v, want user-1", session1.UserID)
	}
	if session2.UserID != "user-2" {
		t.Errorf("Session2 userID = %v, want user-2", session2.UserID)
	}

	// Validate each session
	validated1, _ := store.ValidateSession(ctx, token1)
	validated2, _ := store.ValidateSession(ctx, token2)

	if validated1.UserID != "user-1" {
		t.Error("Validated session1 has wrong userID")
	}
	if validated2.UserID != "user-2" {
		t.Error("Validated session2 has wrong userID")
	}
}
