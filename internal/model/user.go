// Package model defines shared domain types for users, workspaces, and RBAC.
// It handles type definitions for User, Workspace, WorkspaceMember, Role,
// UserSession, and AuditLog.
// It does NOT handle user persistence, authentication, or session management.
package model

import "time"

// User represents an application user with authentication details.
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	AvatarURL    string    `json:"avatarUrl,omitempty"`
	AuthProvider string    `json:"authProvider"`
	IsActive     bool      `json:"isActive"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// Workspace represents a team workspace or personal workspace.
type Workspace struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description,omitempty"`
	IsPersonal  bool      `json:"isPersonal"`
	OwnerID     string    `json:"ownerId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// WorkspaceMember represents a user's membership in a workspace with a specific role.
type WorkspaceMember struct {
	WorkspaceID string    `json:"workspaceId"`
	UserID      string    `json:"userId"`
	Role        Role      `json:"role"`
	InvitedBy   string    `json:"invitedBy,omitempty"`
	JoinedAt    time.Time `json:"joinedAt"`
	// User is populated when fetching members with user details
	User *User `json:"user,omitempty"`
}

// Role represents the permission level of a workspace member.
type Role string

const (
	// RoleAdmin has full workspace control including member management.
	RoleAdmin Role = "admin"
	// RoleEditor can create and edit jobs, view all workspace jobs.
	RoleEditor Role = "editor"
	// RoleViewer has read-only access to workspace jobs.
	RoleViewer Role = "viewer"
)

// IsValid returns true if the role is a recognized value.
func (r Role) IsValid() bool {
	switch r {
	case RoleAdmin, RoleEditor, RoleViewer:
		return true
	}
	return false
}

// CanCreateJob returns true if the role can create new jobs.
func (r Role) CanCreateJob() bool {
	return r == RoleAdmin || r == RoleEditor
}

// CanDeleteJob returns true if the role can delete jobs.
func (r Role) CanDeleteJob() bool {
	return r == RoleAdmin || r == RoleEditor
}

// CanManageWorkspace returns true if the role can manage workspace settings and members.
func (r Role) CanManageWorkspace() bool {
	return r == RoleAdmin
}

// CanInviteMembers returns true if the role can invite new members.
func (r Role) CanInviteMembers() bool {
	return r == RoleAdmin
}

// UserSession represents a web UI session for authentication.
type UserSession struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	TokenHash string    `json:"-"` // Never expose in JSON
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
	IPAddress string    `json:"ipAddress,omitempty"`
	UserAgent string    `json:"userAgent,omitempty"`
}

// IsExpired returns true if the session has expired.
func (s *UserSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// AuditLog represents an audit log entry for tracking user actions.
type AuditLog struct {
	ID           string                 `json:"id"`
	WorkspaceID  string                 `json:"workspaceId,omitempty"`
	UserID       string                 `json:"userId,omitempty"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resourceType"`
	ResourceID   string                 `json:"resourceId,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt    time.Time              `json:"createdAt"`
}

// Common audit log actions.
const (
	AuditActionJobCreate       = "job.create"
	AuditActionJobUpdate       = "job.update"
	AuditActionJobDelete       = "job.delete"
	AuditActionJobCancel       = "job.cancel"
	AuditActionUserLogin       = "user.login"
	AuditActionUserLogout      = "user.logout"
	AuditActionUserRegister    = "user.register"
	AuditActionWorkspaceCreate = "workspace.create"
	AuditActionWorkspaceUpdate = "workspace.update"
	AuditActionWorkspaceDelete = "workspace.delete"
	AuditActionMemberAdd       = "member.add"
	AuditActionMemberRemove    = "member.remove"
	AuditActionMemberUpdate    = "member.update"
)

// Common resource types for audit logs.
const (
	AuditResourceJob       = "job"
	AuditResourceUser      = "user"
	AuditResourceWorkspace = "workspace"
	AuditResourceMember    = "member"
	AuditResourceSession   = "session"
)
