// Package users provides user management, workspace membership, and RBAC.
//
// This file is responsible for:
// - User CRUD operations
// - Workspace management and ownership
// - Workspace membership and role management
// - RBAC permission checks
//
// This file does NOT handle:
// - Authentication (see internal/api/auth_handlers.go)
// - Session management (see internal/sessions/)
// - Password hashing (see password.go)
//
// Invariants:
// - Every user has a personal workspace created automatically
// - Workspace slugs are unique and URL-friendly
// - Roles are validated before assignment
// - Owner cannot be removed from their workspace
package users

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/store"
	"github.com/google/uuid"
)

// Service provides user and workspace management functionality.
type Service struct {
	store *store.Store
}

// NewService creates a new user service with the given store.
func NewService(store *store.Store) *Service {
	return &Service{store: store}
}

// CreateUser creates a new user with a personal workspace.
// If password is empty, the user is created for OAuth-only authentication.
func (s *Service) CreateUser(ctx context.Context, email, name, password string) (*model.User, error) {
	if err := validateEmail(email); err != nil {
		return nil, err
	}
	if strings.TrimSpace(name) == "" {
		return nil, apperrors.Validation("name is required")
	}

	user := &model.User{
		ID:           uuid.New().String(),
		Email:        strings.ToLower(strings.TrimSpace(email)),
		Name:         strings.TrimSpace(name),
		AuthProvider: "local",
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	var passwordHash string
	if password != "" {
		hash, err := HashPassword(password)
		if err != nil {
			return nil, err
		}
		passwordHash = hash
	}

	if err := s.store.CreateUser(ctx, user, passwordHash); err != nil {
		return nil, err
	}

	// Create personal workspace for the user
	_, err := s.CreateWorkspace(ctx, user.ID, name+"'s Workspace", "Personal workspace")
	if err != nil {
		// Best effort - user is created but workspace creation failed
		// Log this but don't fail the user creation
		return user, nil
	}

	return user, nil
}

// CreateUserWithOAuth creates a new user from OAuth provider data.
func (s *Service) CreateUserWithOAuth(ctx context.Context, email, name, avatarURL, provider, providerID string) (*model.User, error) {
	if err := validateEmail(email); err != nil {
		return nil, err
	}

	user := &model.User{
		ID:           uuid.New().String(),
		Email:        strings.ToLower(strings.TrimSpace(email)),
		Name:         strings.TrimSpace(name),
		AvatarURL:    avatarURL,
		AuthProvider: provider,
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := s.store.CreateUser(ctx, user, ""); err != nil {
		return nil, err
	}

	// Create personal workspace for the user
	_, err := s.CreateWorkspace(ctx, user.ID, name+"'s Workspace", "Personal workspace")
	if err != nil {
		return user, nil
	}

	return user, nil
}

// GetUser retrieves a user by ID.
func (s *Service) GetUser(ctx context.Context, id string) (*model.User, error) {
	if id == "" {
		return nil, apperrors.Validation("user ID is required")
	}
	return s.store.GetUser(ctx, id)
}

// GetUserByEmail retrieves a user by email address.
func (s *Service) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	if email == "" {
		return nil, apperrors.Validation("email is required")
	}
	return s.store.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
}

// UpdateUser updates a user's profile information.
func (s *Service) UpdateUser(ctx context.Context, user *model.User) error {
	if user == nil || user.ID == "" {
		return apperrors.Validation("user is required")
	}
	user.UpdatedAt = time.Now().UTC()
	return s.store.UpdateUser(ctx, user)
}

// DeleteUser deletes a user and all their associated data.
// Note: This will cascade delete workspaces where they are the sole member.
func (s *Service) DeleteUser(ctx context.Context, id string) error {
	if id == "" {
		return apperrors.Validation("user ID is required")
	}
	return s.store.DeleteUser(ctx, id)
}

// UpdatePassword updates a user's password.
func (s *Service) UpdatePassword(ctx context.Context, userID, newPassword string) error {
	if userID == "" {
		return apperrors.Validation("user ID is required")
	}
	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.store.UpdateUserPassword(ctx, userID, hash)
}

// CreateWorkspace creates a new workspace owned by the given user.
func (s *Service) CreateWorkspace(ctx context.Context, ownerID, name, description string) (*model.Workspace, error) {
	if ownerID == "" {
		return nil, apperrors.Validation("owner ID is required")
	}
	if strings.TrimSpace(name) == "" {
		return nil, apperrors.Validation("workspace name is required")
	}

	slug := generateSlug(name)
	workspace := &model.Workspace{
		ID:          uuid.New().String(),
		Name:        strings.TrimSpace(name),
		Slug:        slug,
		Description: strings.TrimSpace(description),
		IsPersonal:  false,
		OwnerID:     ownerID,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := s.store.CreateWorkspace(ctx, workspace); err != nil {
		return nil, err
	}

	// Add owner as admin member
	member := &model.WorkspaceMember{
		WorkspaceID: workspace.ID,
		UserID:      ownerID,
		Role:        model.RoleAdmin,
		JoinedAt:    time.Now().UTC(),
	}
	if err := s.store.AddWorkspaceMember(ctx, member); err != nil {
		// Best effort - workspace created but owner not added as member
		return workspace, nil
	}

	return workspace, nil
}

// GetWorkspace retrieves a workspace by ID.
func (s *Service) GetWorkspace(ctx context.Context, id string) (*model.Workspace, error) {
	if id == "" {
		return nil, apperrors.Validation("workspace ID is required")
	}
	return s.store.GetWorkspace(ctx, id)
}

// GetWorkspaceBySlug retrieves a workspace by its slug.
func (s *Service) GetWorkspaceBySlug(ctx context.Context, slug string) (*model.Workspace, error) {
	if slug == "" {
		return nil, apperrors.Validation("workspace slug is required")
	}
	return s.store.GetWorkspaceBySlug(ctx, slug)
}

// ListUserWorkspaces retrieves all workspaces where the user is a member.
func (s *Service) ListUserWorkspaces(ctx context.Context, userID string) ([]*model.Workspace, error) {
	if userID == "" {
		return nil, apperrors.Validation("user ID is required")
	}
	return s.store.ListWorkspacesByUser(ctx, userID)
}

// UpdateWorkspace updates a workspace's information.
func (s *Service) UpdateWorkspace(ctx context.Context, workspace *model.Workspace) error {
	if workspace == nil || workspace.ID == "" {
		return apperrors.Validation("workspace is required")
	}
	workspace.UpdatedAt = time.Now().UTC()
	return s.store.UpdateWorkspace(ctx, workspace)
}

// DeleteWorkspace deletes a workspace and all its associated data.
func (s *Service) DeleteWorkspace(ctx context.Context, id string) error {
	if id == "" {
		return apperrors.Validation("workspace ID is required")
	}
	return s.store.DeleteWorkspace(ctx, id)
}

// AddWorkspaceMember adds a user to a workspace with the specified role.
func (s *Service) AddWorkspaceMember(ctx context.Context, workspaceID, userID string, role model.Role, invitedBy string) error {
	if workspaceID == "" {
		return apperrors.Validation("workspace ID is required")
	}
	if userID == "" {
		return apperrors.Validation("user ID is required")
	}
	if !role.IsValid() {
		return apperrors.Validation("invalid role: " + string(role))
	}

	member := &model.WorkspaceMember{
		WorkspaceID: workspaceID,
		UserID:      userID,
		Role:        role,
		InvitedBy:   invitedBy,
		JoinedAt:    time.Now().UTC(),
	}
	return s.store.AddWorkspaceMember(ctx, member)
}

// RemoveWorkspaceMember removes a user from a workspace.
func (s *Service) RemoveWorkspaceMember(ctx context.Context, workspaceID, userID string) error {
	if workspaceID == "" {
		return apperrors.Validation("workspace ID is required")
	}
	if userID == "" {
		return apperrors.Validation("user ID is required")
	}

	// Check if user is the owner - cannot remove owner
	workspace, err := s.store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return err
	}
	if workspace.OwnerID == userID {
		return apperrors.Validation("cannot remove workspace owner")
	}

	return s.store.RemoveWorkspaceMember(ctx, workspaceID, userID)
}

// UpdateMemberRole updates a member's role in a workspace.
func (s *Service) UpdateMemberRole(ctx context.Context, workspaceID, userID string, role model.Role) error {
	if workspaceID == "" {
		return apperrors.Validation("workspace ID is required")
	}
	if userID == "" {
		return apperrors.Validation("user ID is required")
	}
	if !role.IsValid() {
		return apperrors.Validation("invalid role: " + string(role))
	}

	// Check if user is the owner - owner must remain admin
	workspace, err := s.store.GetWorkspace(ctx, workspaceID)
	if err != nil {
		return err
	}
	if workspace.OwnerID == userID && role != model.RoleAdmin {
		return apperrors.Validation("workspace owner must remain admin")
	}

	return s.store.UpdateWorkspaceMemberRole(ctx, workspaceID, userID, role)
}

// GetWorkspaceMembers retrieves all members of a workspace.
func (s *Service) GetWorkspaceMembers(ctx context.Context, workspaceID string) ([]*model.WorkspaceMember, error) {
	if workspaceID == "" {
		return nil, apperrors.Validation("workspace ID is required")
	}
	return s.store.GetWorkspaceMembers(ctx, workspaceID)
}

// GetUserWorkspaceRole retrieves the user's role in a workspace.
func (s *Service) GetUserWorkspaceRole(ctx context.Context, userID, workspaceID string) (model.Role, error) {
	if userID == "" || workspaceID == "" {
		return "", apperrors.Validation("user ID and workspace ID are required")
	}
	return s.store.GetUserWorkspaceRole(ctx, userID, workspaceID)
}

// IsWorkspaceMember checks if a user is a member of a workspace.
func (s *Service) IsWorkspaceMember(ctx context.Context, userID, workspaceID string) bool {
	_, err := s.GetUserWorkspaceRole(ctx, userID, workspaceID)
	return err == nil
}

// CanAccessWorkspace checks if a user can access a workspace.
func (s *Service) CanAccessWorkspace(ctx context.Context, userID, workspaceID string) bool {
	return s.IsWorkspaceMember(ctx, userID, workspaceID)
}

// CanCreateJob checks if a user can create jobs in a workspace.
func (s *Service) CanCreateJob(ctx context.Context, userID, workspaceID string) bool {
	role, err := s.GetUserWorkspaceRole(ctx, userID, workspaceID)
	if err != nil {
		return false
	}
	return role.CanCreateJob()
}

// CanDeleteJob checks if a user can delete jobs in a workspace.
func (s *Service) CanDeleteJob(ctx context.Context, userID, workspaceID string) bool {
	role, err := s.GetUserWorkspaceRole(ctx, userID, workspaceID)
	if err != nil {
		return false
	}
	return role.CanDeleteJob()
}

// CanManageWorkspace checks if a user can manage workspace settings.
func (s *Service) CanManageWorkspace(ctx context.Context, userID, workspaceID string) bool {
	role, err := s.GetUserWorkspaceRole(ctx, userID, workspaceID)
	if err != nil {
		return false
	}
	return role.CanManageWorkspace()
}

// CanInviteMembers checks if a user can invite members to a workspace.
func (s *Service) CanInviteMembers(ctx context.Context, userID, workspaceID string) bool {
	role, err := s.GetUserWorkspaceRole(ctx, userID, workspaceID)
	if err != nil {
		return false
	}
	return role.CanInviteMembers()
}

// validateEmail validates an email address format.
func validateEmail(email string) error {
	if strings.TrimSpace(email) == "" {
		return apperrors.Validation("email is required")
	}
	// Simple email validation regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return apperrors.Validation("invalid email format")
	}
	return nil
}

// generateSlug creates a URL-friendly slug from a name.
func generateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)
	// Replace spaces and underscores with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	// Remove non-alphanumeric characters except hyphens
	slug = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(slug, "")
	// Remove consecutive hyphens
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")
	// Trim hyphens from ends
	slug = strings.Trim(slug, "-")
	// Add UUID suffix for uniqueness
	if slug == "" {
		slug = uuid.New().String()[:8]
	} else {
		slug = slug + "-" + uuid.New().String()[:8]
	}
	return slug
}

// GetOrCreateDefaultUser creates a default admin user if no users exist.
// This is used for first-run setup and backward compatibility.
func (s *Service) GetOrCreateDefaultUser(ctx context.Context, email, password string) (*model.User, error) {
	// Check if any users exist
	users, err := s.store.ListUsers(ctx, 1)
	if err != nil {
		return nil, err
	}
	if len(users) > 0 {
		// Users already exist, return nil
		return nil, nil
	}

	// Create default admin user
	if email == "" {
		email = "admin@localhost"
	}
	if password == "" {
		password = "changeme123"
	}

	user, err := s.CreateUser(ctx, email, "Admin", password)
	if err != nil {
		return nil, fmt.Errorf("failed to create default user: %w", err)
	}

	return user, nil
}
