// Package store provides SQLite-backed persistent storage for audit logs.
//
// This file is responsible for:
// - Audit log CRUD operations
// - Audit log querying by workspace and user
//
// This file does NOT handle:
// - Audit log generation (see business logic layer)
// - Audit log analysis or reporting
package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// CreateAuditLog creates a new audit log entry.
func (s *Store) CreateAuditLog(ctx context.Context, log *model.AuditLog) error {
	var metadataJSON []byte
	var err error
	if log.Metadata != nil {
		metadataJSON, err = json.Marshal(log.Metadata)
		if err != nil {
			return apperrors.Wrap(apperrors.KindInternal, "failed to marshal audit log metadata", err)
		}
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO audit_logs (id, workspace_id, user_id, action, resource_type, resource_id, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, log.ID, log.WorkspaceID, log.UserID, log.Action, log.ResourceType, log.ResourceID, string(metadataJSON), log.CreatedAt.Format(time.RFC3339))

	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to create audit log", err)
	}

	return nil
}

// GetAuditLog retrieves an audit log by ID.
func (s *Store) GetAuditLog(ctx context.Context, id string) (*model.AuditLog, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, user_id, action, resource_type, resource_id, metadata, created_at
		FROM audit_logs WHERE id = ?
	`, id)

	log := &model.AuditLog{}
	var metadataJSON string
	var createdAtStr string
	err := row.Scan(
		&log.ID, &log.WorkspaceID, &log.UserID, &log.Action, &log.ResourceType, &log.ResourceID, &metadataJSON, &createdAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.NotFound("audit log not found")
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to get audit log", err)
	}

	if metadataJSON != "" {
		_ = json.Unmarshal([]byte(metadataJSON), &log.Metadata)
	}
	log.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)

	return log, nil
}

// ListAuditLogsByWorkspace retrieves audit logs for a workspace with pagination.
func (s *Store) ListAuditLogsByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]*model.AuditLog, error) {
	if limit == 0 {
		limit = 100
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, user_id, action, resource_type, resource_id, metadata, created_at
		FROM audit_logs WHERE workspace_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, workspaceID, limit, offset)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to list audit logs", err)
	}
	defer rows.Close()

	var logs []*model.AuditLog
	for rows.Next() {
		log := &model.AuditLog{}
		var metadataJSON string
		var createdAtStr string
		err := rows.Scan(
			&log.ID, &log.WorkspaceID, &log.UserID, &log.Action, &log.ResourceType, &log.ResourceID, &metadataJSON, &createdAtStr)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to scan audit log", err)
		}
		if metadataJSON != "" {
			_ = json.Unmarshal([]byte(metadataJSON), &log.Metadata)
		}
		log.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// ListAuditLogsByUser retrieves audit logs for a user with pagination.
func (s *Store) ListAuditLogsByUser(ctx context.Context, userID string, limit, offset int) ([]*model.AuditLog, error) {
	if limit == 0 {
		limit = 100
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, user_id, action, resource_type, resource_id, metadata, created_at
		FROM audit_logs WHERE user_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, userID, limit, offset)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to list user audit logs", err)
	}
	defer rows.Close()

	var logs []*model.AuditLog
	for rows.Next() {
		log := &model.AuditLog{}
		var metadataJSON string
		var createdAtStr string
		err := rows.Scan(
			&log.ID, &log.WorkspaceID, &log.UserID, &log.Action, &log.ResourceType, &log.ResourceID, &metadataJSON, &createdAtStr)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to scan audit log", err)
		}
		if metadataJSON != "" {
			_ = json.Unmarshal([]byte(metadataJSON), &log.Metadata)
		}
		log.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// ListAuditLogsByAction retrieves audit logs for a specific action with pagination.
func (s *Store) ListAuditLogsByAction(ctx context.Context, action string, limit, offset int) ([]*model.AuditLog, error) {
	if limit == 0 {
		limit = 100
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, workspace_id, user_id, action, resource_type, resource_id, metadata, created_at
		FROM audit_logs WHERE action = ? ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, action, limit, offset)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to list action audit logs", err)
	}
	defer rows.Close()

	var logs []*model.AuditLog
	for rows.Next() {
		log := &model.AuditLog{}
		var metadataJSON string
		var createdAtStr string
		err := rows.Scan(
			&log.ID, &log.WorkspaceID, &log.UserID, &log.Action, &log.ResourceType, &log.ResourceID, &metadataJSON, &createdAtStr)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to scan audit log", err)
		}
		if metadataJSON != "" {
			_ = json.Unmarshal([]byte(metadataJSON), &log.Metadata)
		}
		log.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		logs = append(logs, log)
	}

	return logs, rows.Err()
}

// CountAuditLogsByWorkspace counts audit logs for a workspace.
func (s *Store) CountAuditLogsByWorkspace(ctx context.Context, workspaceID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM audit_logs WHERE workspace_id = ?
	`, workspaceID).Scan(&count)
	if err != nil {
		return 0, apperrors.Wrap(apperrors.KindInternal, "failed to count audit logs", err)
	}
	return count, nil
}

// PurgeOldAuditLogs deletes audit logs older than the specified duration.
func (s *Store) PurgeOldAuditLogs(ctx context.Context, olderThan time.Duration) error {
	cutoff := time.Now().UTC().Add(-olderThan).Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `DELETE FROM audit_logs WHERE created_at < ?`, cutoff)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to purge old audit logs", err)
	}
	return nil
}
