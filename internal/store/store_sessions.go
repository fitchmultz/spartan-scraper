// Package store provides SQLite-backed persistent storage for user sessions.
//
// This file is responsible for:
// - Session CRUD operations
// - Session token lookups
// - Session expiration cleanup
//
// This file does NOT handle:
// - Token generation (see internal/sessions/)
// - Session validation logic (see internal/sessions/)
package store

import (
	"context"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// CreateSession creates a new user session.
func (s *Store) CreateSession(ctx context.Context, session *model.UserSession) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_sessions (id, user_id, token_hash, expires_at, created_at, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, session.ID, session.UserID, session.TokenHash, session.ExpiresAt.Format(time.RFC3339), session.CreatedAt.Format(time.RFC3339), session.IPAddress, session.UserAgent)

	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create session", err)
	}

	return nil
}

// GetSessionByTokenHash retrieves a session by its token hash.
func (s *Store) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*model.UserSession, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, token_hash, expires_at, created_at, ip_address, user_agent
		FROM user_sessions WHERE token_hash = ?
	`, tokenHash)

	session := &model.UserSession{}
	var expiresAtStr, createdAtStr string
	err := row.Scan(&session.ID, &session.UserID, &session.TokenHash, &expiresAtStr, &createdAtStr, &session.IPAddress, &session.UserAgent)
	if err != nil {
		return nil, wrapScanError(err, "session not found", "failed to get session")
	}

	session.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAtStr)
	session.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)

	return session, nil
}

// GetSession retrieves a session by ID.
func (s *Store) GetSession(ctx context.Context, id string) (*model.UserSession, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, user_id, token_hash, expires_at, created_at, ip_address, user_agent
		FROM user_sessions WHERE id = ?
	`, id)

	session := &model.UserSession{}
	var expiresAtStr, createdAtStr string
	err := row.Scan(&session.ID, &session.UserID, &session.TokenHash, &expiresAtStr, &createdAtStr, &session.IPAddress, &session.UserAgent)
	if err != nil {
		return nil, wrapScanError(err, "session not found", "failed to get session")
	}

	session.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAtStr)
	session.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)

	return session, nil
}

// DeleteSession deletes a session by ID.
func (s *Store) DeleteSession(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE id = ?`, id)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to delete session", err)
	}
	return nil
}

// DeleteUserSessions deletes all sessions for a user.
func (s *Store) DeleteUserSessions(ctx context.Context, userID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE user_id = ?`, userID)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to delete user sessions", err)
	}
	return nil
}

// CleanupExpiredSessions deletes all expired sessions.
func (s *Store) CleanupExpiredSessions(ctx context.Context) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `DELETE FROM user_sessions WHERE expires_at < ?`, now)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to cleanup expired sessions", err)
	}
	return nil
}

// ListUserSessions retrieves all sessions for a user.
func (s *Store) ListUserSessions(ctx context.Context, userID string) ([]*model.UserSession, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, user_id, token_hash, expires_at, created_at, ip_address, user_agent
		FROM user_sessions WHERE user_id = ? ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to list user sessions", err)
	}
	defer rows.Close()

	var sessions []*model.UserSession
	for rows.Next() {
		session := &model.UserSession{}
		var expiresAtStr, createdAtStr string
		err := rows.Scan(&session.ID, &session.UserID, &session.TokenHash, &expiresAtStr, &createdAtStr, &session.IPAddress, &session.UserAgent)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to scan session", err)
		}
		session.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAtStr)
		session.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		sessions = append(sessions, session)
	}

	return sessions, rows.Err()
}
