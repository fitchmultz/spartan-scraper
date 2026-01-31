// Package sessions provides session management for web UI authentication.
//
// This file is responsible for:
// - Creating and validating user sessions
// - Session token generation and hashing
// - Session expiration and cleanup
// - Database-backed session storage
//
// This file does NOT handle:
// - HTTP cookie management (see internal/api/auth_handlers.go)
// - User authentication (see internal/users/)
//
// Invariants:
// - Session tokens are cryptographically random (32 bytes)
// - Token hashes (not tokens) are stored in the database
// - Sessions have a configurable expiration time
// - Expired sessions are cleaned up periodically
package sessions

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/google/uuid"
)

// Store provides session management functionality.
type Store struct {
	store *store.Store
}

// NewStore creates a new session store with the given database store.
func NewStore(store *store.Store) *Store {
	return &Store{store: store}
}

// Session represents an active user session.
type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// CreateSession creates a new session for a user.
// Returns the session and the plain token (which must be stored in a cookie).
// The token is only returned once - it cannot be retrieved later.
func (s *Store) CreateSession(ctx context.Context, userID, ipAddress, userAgent string, duration time.Duration) (*Session, string, error) {
	if userID == "" {
		return nil, "", apperrors.Validation("user ID is required")
	}

	if duration == 0 {
		duration = 24 * time.Hour // Default 24 hours
	}

	// Generate cryptographically random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, "", apperrors.Wrap(apperrors.KindInternal, "failed to generate session token", err)
	}
	token := hex.EncodeToString(tokenBytes)

	// Hash the token for storage
	tokenHash := hashToken(token)

	userSession := &model.UserSession{
		ID:        uuid.New().String(),
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().UTC().Add(duration),
		CreatedAt: time.Now().UTC(),
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}

	if err := s.store.CreateSession(ctx, userSession); err != nil {
		return nil, "", err
	}

	session := &Session{
		ID:        userSession.ID,
		UserID:    userSession.UserID,
		ExpiresAt: userSession.ExpiresAt,
		CreatedAt: userSession.CreatedAt,
	}

	return session, token, nil
}

// ValidateSession validates a session token and returns the session if valid.
func (s *Store) ValidateSession(ctx context.Context, token string) (*Session, error) {
	if token == "" {
		return nil, apperrors.Permission("session token is required")
	}

	tokenHash := hashToken(token)
	userSession, err := s.store.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, apperrors.Permission("invalid session")
	}

	if userSession.IsExpired() {
		// Clean up expired session
		_ = s.store.DeleteSession(ctx, userSession.ID)
		return nil, apperrors.Permission("session expired")
	}

	return &Session{
		ID:        userSession.ID,
		UserID:    userSession.UserID,
		ExpiresAt: userSession.ExpiresAt,
		CreatedAt: userSession.CreatedAt,
	}, nil
}

// DeleteSession invalidates a session by its token.
func (s *Store) DeleteSession(ctx context.Context, token string) error {
	if token == "" {
		return apperrors.Validation("session token is required")
	}

	tokenHash := hashToken(token)
	userSession, err := s.store.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		// Session doesn't exist or already deleted
		return nil
	}

	return s.store.DeleteSession(ctx, userSession.ID)
}

// DeleteSessionByID invalidates a session by its ID.
func (s *Store) DeleteSessionByID(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return apperrors.Validation("session ID is required")
	}

	return s.store.DeleteSession(ctx, sessionID)
}

// DeleteUserSessions invalidates all sessions for a user.
func (s *Store) DeleteUserSessions(ctx context.Context, userID string) error {
	if userID == "" {
		return apperrors.Validation("user ID is required")
	}

	return s.store.DeleteUserSessions(ctx, userID)
}

// CleanupExpiredSessions removes all expired sessions from the database.
func (s *Store) CleanupExpiredSessions(ctx context.Context) error {
	return s.store.CleanupExpiredSessions(ctx)
}

// ListUserSessions retrieves all active sessions for a user.
func (s *Store) ListUserSessions(ctx context.Context, userID string) ([]*Session, error) {
	if userID == "" {
		return nil, apperrors.Validation("user ID is required")
	}

	sessions, err := s.store.ListUserSessions(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]*Session, 0, len(sessions))
	for _, userSession := range sessions {
		if !userSession.IsExpired() {
			result = append(result, &Session{
				ID:        userSession.ID,
				UserID:    userSession.UserID,
				ExpiresAt: userSession.ExpiresAt,
				CreatedAt: userSession.CreatedAt,
			})
		}
	}

	return result, nil
}

// hashToken creates a hash of the session token for storage.
// We use a simple hash here since we're just preventing token enumeration
// in case of database leaks. The actual security comes from the token's entropy.
func hashToken(token string) string {
	// Use SHA-256 for fast hashing of the token
	// This is not for password storage - just to prevent direct token exposure in DB
	hash := sha256.Sum256([]byte(token))
	_ = hash
	return hex.EncodeToString(hash[:])
}
