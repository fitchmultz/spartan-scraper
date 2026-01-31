// Package users provides user management, workspace membership, and RBAC.
//
// This file tests role-based access control (RBAC) and role methods.
package users

import (
	"context"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestService_RBAC(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	svc := NewService(s)
	ctx := context.Background()

	// Create test users with different roles
	admin, _ := svc.CreateUser(ctx, "admin@example.com", "Admin", "password123")
	editor, _ := svc.CreateUser(ctx, "editor@example.com", "Editor", "password123")
	viewer, _ := svc.CreateUser(ctx, "viewer@example.com", "Viewer", "password123")
	nonMember, _ := svc.CreateUser(ctx, "nonmember@example.com", "NonMember", "password123")

	// Create workspace
	ws, _ := svc.CreateWorkspace(ctx, admin.ID, "RBAC Test", "Description")
	svc.AddWorkspaceMember(ctx, ws.ID, editor.ID, model.RoleEditor, admin.ID)
	svc.AddWorkspaceMember(ctx, ws.ID, viewer.ID, model.RoleViewer, admin.ID)

	tests := []struct {
		name       string
		userID     string
		checkAdmin bool
		checkEdit  bool
		checkView  bool
	}{
		{
			name:       "admin can do everything",
			userID:     admin.ID,
			checkAdmin: true,
			checkEdit:  true,
			checkView:  true,
		},
		{
			name:       "editor can edit and view",
			userID:     editor.ID,
			checkAdmin: false,
			checkEdit:  true,
			checkView:  true,
		},
		{
			name:       "viewer can only view",
			userID:     viewer.ID,
			checkAdmin: false,
			checkEdit:  false,
			checkView:  true,
		},
		{
			name:       "non-member has no access",
			userID:     nonMember.ID,
			checkAdmin: false,
			checkEdit:  false,
			checkView:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := svc.CanManageWorkspace(ctx, tt.userID, ws.ID); got != tt.checkAdmin {
				t.Errorf("CanManageWorkspace() = %v, want %v", got, tt.checkAdmin)
			}
			if got := svc.CanCreateJob(ctx, tt.userID, ws.ID); got != tt.checkEdit {
				t.Errorf("CanCreateJob() = %v, want %v", got, tt.checkEdit)
			}
			if got := svc.CanAccessWorkspace(ctx, tt.userID, ws.ID); got != tt.checkView {
				t.Errorf("CanAccessWorkspace() = %v, want %v", got, tt.checkView)
			}
		})
	}
}

func TestRole_CanCreateJob(t *testing.T) {
	tests := []struct {
		role string
		want bool
	}{
		{"admin", true},
		{"editor", true},
		{"viewer", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			r := model.Role(tt.role)
			if got := r.CanCreateJob(); got != tt.want {
				t.Errorf("Role(%s).CanCreateJob() = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestRole_CanDeleteJob(t *testing.T) {
	tests := []struct {
		role string
		want bool
	}{
		{"admin", true},
		{"editor", true},
		{"viewer", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			r := model.Role(tt.role)
			if got := r.CanDeleteJob(); got != tt.want {
				t.Errorf("Role(%s).CanDeleteJob() = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestRole_CanManageWorkspace(t *testing.T) {
	tests := []struct {
		role string
		want bool
	}{
		{"admin", true},
		{"editor", false},
		{"viewer", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			r := model.Role(tt.role)
			if got := r.CanManageWorkspace(); got != tt.want {
				t.Errorf("Role(%s).CanManageWorkspace() = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestRole_CanInviteMembers(t *testing.T) {
	tests := []struct {
		role string
		want bool
	}{
		{"admin", true},
		{"editor", false},
		{"viewer", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			r := model.Role(tt.role)
			if got := r.CanInviteMembers(); got != tt.want {
				t.Errorf("Role(%s).CanInviteMembers() = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestRole_IsValid(t *testing.T) {
	tests := []struct {
		role string
		want bool
	}{
		{"admin", true},
		{"editor", true},
		{"viewer", true},
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			r := model.Role(tt.role)
			if got := r.IsValid(); got != tt.want {
				t.Errorf("Role(%s).IsValid() = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}
