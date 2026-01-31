// Package store provides SQLite-backed persistent storage for users and workspaces.
//
// This file is responsible for:
// - User CRUD operations
// - Workspace CRUD operations
// - Workspace membership management
// - Role-based access control queries
//
// This file does NOT handle:
// - Business logic (see internal/users/)
// - Password hashing (see internal/users/password.go)
// - Session management (see internal/sessions/)
package store

import (
	"context"
	"database/sql"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// CreateUser creates a new user with an optional password hash.
func (s *Store) CreateUser(ctx context.Context, user *model.User, passwordHash string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO users (id, email, password_hash, name, avatar_url, auth_provider, auth_provider_id, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, user.ID, user.Email, passwordHash, user.Name, user.AvatarURL, user.AuthProvider, nil, user.IsActive, user.CreatedAt.Format(time.RFC3339), user.UpdatedAt.Format(time.RFC3339))

	if err != nil {
		if isUniqueConstraintError(err) {
			return apperrors.Validation("user with this email already exists")
		}
		return apperrors.Wrap(apperrors.KindInternal, "failed to create user", err)
	}

	return nil
}

// GetUser retrieves a user by ID.
func (s *Store) GetUser(ctx context.Context, id string) (*model.User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, avatar_url, auth_provider, is_active, created_at, updated_at
		FROM users WHERE id = ?
	`, id)

	user := &model.User{}
	var createdAtStr, updatedAtStr string
	err := row.Scan(&user.ID, &user.Email, &user.Name, &user.AvatarURL, &user.AuthProvider, &user.IsActive, &createdAtStr, &updatedAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.NotFound("user not found")
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to get user", err)
	}

	user.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	user.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	return user, nil
}

// GetUserByEmail retrieves a user by email address.
func (s *Store) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, email, name, avatar_url, auth_provider, is_active, created_at, updated_at
		FROM users WHERE email = ?
	`, email)

	user := &model.User{}
	var createdAtStr, updatedAtStr string
	err := row.Scan(&user.ID, &user.Email, &user.Name, &user.AvatarURL, &user.AuthProvider, &user.IsActive, &createdAtStr, &updatedAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.NotFound("user not found")
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to get user by email", err)
	}

	user.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	user.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	return user, nil
}

// GetUserWithPassword retrieves a user with their password hash for authentication.
func (s *Store) GetUserWithPassword(ctx context.Context, email string) (*model.User, string, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, name, avatar_url, auth_provider, is_active, created_at, updated_at
		FROM users WHERE email = ?
	`, email)

	user := &model.User{}
	var passwordHash string
	var createdAtStr, updatedAtStr string
	err := row.Scan(&user.ID, &user.Email, &passwordHash, &user.Name, &user.AvatarURL, &user.AuthProvider, &user.IsActive, &createdAtStr, &updatedAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", apperrors.NotFound("user not found")
		}
		return nil, "", apperrors.Wrap(apperrors.KindInternal, "failed to get user", err)
	}

	user.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	user.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	return user, passwordHash, nil
}

// UpdateUser updates a user's profile information.
func (s *Store) UpdateUser(ctx context.Context, user *model.User) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET name = ?, avatar_url = ?, is_active = ?, updated_at = ?
		WHERE id = ?
	`, user.Name, user.AvatarURL, user.IsActive, user.UpdatedAt.Format(time.RFC3339), user.ID)

	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to update user", err)
	}

	return nil
}

// UpdateUserPassword updates a user's password hash.
func (s *Store) UpdateUserPassword(ctx context.Context, userID, passwordHash string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users SET password_hash = ?, updated_at = ?
		WHERE id = ?
	`, passwordHash, time.Now().UTC().Format(time.RFC3339), userID)

	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to update password", err)
	}

	return nil
}

// DeleteUser deletes a user and their associated data.
func (s *Store) DeleteUser(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to delete user", err)
	}
	return nil
}

// ListUsers retrieves a limited list of users.
func (s *Store) ListUsers(ctx context.Context, limit int) ([]*model.User, error) {
	if limit == 0 {
		limit = 100
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, email, name, avatar_url, auth_provider, is_active, created_at, updated_at
		FROM users ORDER BY created_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to list users", err)
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		user := &model.User{}
		var createdAtStr, updatedAtStr string
		err := rows.Scan(&user.ID, &user.Email, &user.Name, &user.AvatarURL, &user.AuthProvider, &user.IsActive, &createdAtStr, &updatedAtStr)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to scan user", err)
		}
		user.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		user.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		users = append(users, user)
	}

	return users, rows.Err()
}

// CreateWorkspace creates a new workspace.
func (s *Store) CreateWorkspace(ctx context.Context, workspace *model.Workspace) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO workspaces (id, name, slug, description, is_personal, owner_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, workspace.ID, workspace.Name, workspace.Slug, workspace.Description, workspace.IsPersonal, workspace.OwnerID, workspace.CreatedAt.Format(time.RFC3339), workspace.UpdatedAt.Format(time.RFC3339))

	if err != nil {
		if isUniqueConstraintError(err) {
			return apperrors.Validation("workspace with this slug already exists")
		}
		return apperrors.Wrap(apperrors.KindInternal, "failed to create workspace", err)
	}

	return nil
}

// GetWorkspace retrieves a workspace by ID.
func (s *Store) GetWorkspace(ctx context.Context, id string) (*model.Workspace, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, description, is_personal, owner_id, created_at, updated_at
		FROM workspaces WHERE id = ?
	`, id)

	workspace := &model.Workspace{}
	var createdAtStr, updatedAtStr string
	err := row.Scan(&workspace.ID, &workspace.Name, &workspace.Slug, &workspace.Description, &workspace.IsPersonal, &workspace.OwnerID, &createdAtStr, &updatedAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.NotFound("workspace not found")
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to get workspace", err)
	}

	workspace.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	workspace.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	return workspace, nil
}

// GetWorkspaceBySlug retrieves a workspace by its slug.
func (s *Store) GetWorkspaceBySlug(ctx context.Context, slug string) (*model.Workspace, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, description, is_personal, owner_id, created_at, updated_at
		FROM workspaces WHERE slug = ?
	`, slug)

	workspace := &model.Workspace{}
	var createdAtStr, updatedAtStr string
	err := row.Scan(&workspace.ID, &workspace.Name, &workspace.Slug, &workspace.Description, &workspace.IsPersonal, &workspace.OwnerID, &createdAtStr, &updatedAtStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, apperrors.NotFound("workspace not found")
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to get workspace by slug", err)
	}

	workspace.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
	workspace.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)

	return workspace, nil
}

// ListWorkspacesByUser retrieves all workspaces where the user is a member.
func (s *Store) ListWorkspacesByUser(ctx context.Context, userID string) ([]*model.Workspace, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT w.id, w.name, w.slug, w.description, w.is_personal, w.owner_id, w.created_at, w.updated_at
		FROM workspaces w
		JOIN workspace_members wm ON w.id = wm.workspace_id
		WHERE wm.user_id = ?
		ORDER BY w.created_at DESC
	`, userID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to list user workspaces", err)
	}
	defer rows.Close()

	var workspaces []*model.Workspace
	for rows.Next() {
		workspace := &model.Workspace{}
		var createdAtStr, updatedAtStr string
		err := rows.Scan(&workspace.ID, &workspace.Name, &workspace.Slug, &workspace.Description, &workspace.IsPersonal, &workspace.OwnerID, &createdAtStr, &updatedAtStr)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to scan workspace", err)
		}
		workspace.CreatedAt, _ = time.Parse(time.RFC3339, createdAtStr)
		workspace.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAtStr)
		workspaces = append(workspaces, workspace)
	}

	return workspaces, rows.Err()
}

// UpdateWorkspace updates a workspace's information.
func (s *Store) UpdateWorkspace(ctx context.Context, workspace *model.Workspace) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE workspaces SET name = ?, description = ?, updated_at = ?
		WHERE id = ?
	`, workspace.Name, workspace.Description, workspace.UpdatedAt.Format(time.RFC3339), workspace.ID)

	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to update workspace", err)
	}

	return nil
}

// DeleteWorkspace deletes a workspace.
func (s *Store) DeleteWorkspace(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM workspaces WHERE id = ?`, id)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to delete workspace", err)
	}
	return nil
}

// AddWorkspaceMember adds a user to a workspace with a specific role.
func (s *Store) AddWorkspaceMember(ctx context.Context, member *model.WorkspaceMember) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role, invited_by, joined_at)
		VALUES (?, ?, ?, ?, ?)
	`, member.WorkspaceID, member.UserID, member.Role, member.InvitedBy, member.JoinedAt.Format(time.RFC3339))

	if err != nil {
		if isUniqueConstraintError(err) {
			return apperrors.Validation("user is already a member of this workspace")
		}
		return apperrors.Wrap(apperrors.KindInternal, "failed to add workspace member", err)
	}

	return nil
}

// RemoveWorkspaceMember removes a user from a workspace.
func (s *Store) RemoveWorkspaceMember(ctx context.Context, workspaceID, userID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM workspace_members WHERE workspace_id = ? AND user_id = ?
	`, workspaceID, userID)

	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to remove workspace member", err)
	}

	return nil
}

// UpdateWorkspaceMemberRole updates a member's role in a workspace.
func (s *Store) UpdateWorkspaceMemberRole(ctx context.Context, workspaceID, userID string, role model.Role) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE workspace_members SET role = ? WHERE workspace_id = ? AND user_id = ?
	`, role, workspaceID, userID)

	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to update member role", err)
	}

	return nil
}

// GetWorkspaceMembers retrieves all members of a workspace with their details.
func (s *Store) GetWorkspaceMembers(ctx context.Context, workspaceID string) ([]*model.WorkspaceMember, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT wm.workspace_id, wm.user_id, wm.role, wm.invited_by, wm.joined_at,
			u.id, u.email, u.name, u.avatar_url, u.auth_provider, u.is_active, u.created_at, u.updated_at
		FROM workspace_members wm
		JOIN users u ON wm.user_id = u.id
		WHERE wm.workspace_id = ?
		ORDER BY wm.joined_at ASC
	`, workspaceID)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to get workspace members", err)
	}
	defer rows.Close()

	var members []*model.WorkspaceMember
	for rows.Next() {
		member := &model.WorkspaceMember{}
		user := &model.User{}
		var joinedAtStr string
		var userCreatedAtStr, userUpdatedAtStr string

		err := rows.Scan(
			&member.WorkspaceID, &member.UserID, &member.Role, &member.InvitedBy, &joinedAtStr,
			&user.ID, &user.Email, &user.Name, &user.AvatarURL, &user.AuthProvider, &user.IsActive, &userCreatedAtStr, &userUpdatedAtStr,
		)
		if err != nil {
			return nil, apperrors.Wrap(apperrors.KindInternal, "failed to scan member", err)
		}

		member.JoinedAt, _ = time.Parse(time.RFC3339, joinedAtStr)
		user.CreatedAt, _ = time.Parse(time.RFC3339, userCreatedAtStr)
		user.UpdatedAt, _ = time.Parse(time.RFC3339, userUpdatedAtStr)
		member.User = user

		members = append(members, member)
	}

	return members, rows.Err()
}

// GetUserWorkspaceRole retrieves the user's role in a workspace.
func (s *Store) GetUserWorkspaceRole(ctx context.Context, userID, workspaceID string) (model.Role, error) {
	var role string
	err := s.db.QueryRowContext(ctx, `
		SELECT role FROM workspace_members WHERE workspace_id = ? AND user_id = ?
	`, workspaceID, userID).Scan(&role)

	if err != nil {
		if err == sql.ErrNoRows {
			return "", apperrors.NotFound("user is not a member of this workspace")
		}
		return "", apperrors.Wrap(apperrors.KindInternal, "failed to get user role", err)
	}

	return model.Role(role), nil
}

// isUniqueConstraintError checks if an error is a unique constraint violation.
func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	// SQLite unique constraint error messages
	errStr := err.Error()
	return contains(errStr, "UNIQUE constraint failed") ||
		contains(errStr, "unique constraint")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
