// Package users provides user management, workspace membership, and RBAC.
//
// This file tests workspace management operations.
package users

import (
	"context"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestService_CreateWorkspace(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	svc := NewService(s)
	ctx := context.Background()

	// Create a test user
	user, err := svc.CreateUser(ctx, "workspace@example.com", "Workspace User", "password123")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name          string
		ownerID       string
		workspaceName string
		description   string
		wantErr       bool
	}{
		{
			name:          "valid workspace",
			ownerID:       user.ID,
			workspaceName: "Test Workspace",
			description:   "A test workspace",
			wantErr:       false,
		},
		{
			name:          "empty owner id",
			ownerID:       "",
			workspaceName: "Test Workspace",
			description:   "A test workspace",
			wantErr:       true,
		},
		{
			name:          "empty name",
			ownerID:       user.ID,
			workspaceName: "",
			description:   "A test workspace",
			wantErr:       true,
		},
		{
			name:          "whitespace name",
			ownerID:       user.ID,
			workspaceName: "   ",
			description:   "A test workspace",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws, err := svc.CreateWorkspace(ctx, tt.ownerID, tt.workspaceName, tt.description)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateWorkspace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if ws == nil {
					t.Error("CreateWorkspace() returned nil workspace")
					return
				}
				if ws.Name != strings.TrimSpace(tt.workspaceName) {
					t.Errorf("CreateWorkspace() name = %v, want %v", ws.Name, tt.workspaceName)
				}
				if ws.OwnerID != tt.ownerID {
					t.Errorf("CreateWorkspace() ownerID = %v, want %v", ws.OwnerID, tt.ownerID)
				}
				if ws.ID == "" {
					t.Error("CreateWorkspace() workspace ID is empty")
				}
				if ws.Slug == "" {
					t.Error("CreateWorkspace() workspace slug is empty")
				}
				if ws.CreatedAt.IsZero() {
					t.Error("CreateWorkspace() created_at is zero")
				}
			}
		})
	}
}

func TestService_WorkspaceMembership(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	svc := NewService(s)
	ctx := context.Background()

	// Create test users
	owner, err := svc.CreateUser(ctx, "owner@example.com", "Owner", "password123")
	if err != nil {
		t.Fatalf("Failed to create owner: %v", err)
	}

	member, err := svc.CreateUser(ctx, "member@example.com", "Member", "password123")
	if err != nil {
		t.Fatalf("Failed to create member: %v", err)
	}

	// Create a workspace
	ws, err := svc.CreateWorkspace(ctx, owner.ID, "Test Workspace", "Description")
	if err != nil {
		t.Fatalf("Failed to create workspace: %v", err)
	}

	t.Run("add member to workspace", func(t *testing.T) {
		err := svc.AddWorkspaceMember(ctx, ws.ID, member.ID, model.RoleEditor, owner.ID)
		if err != nil {
			t.Errorf("AddWorkspaceMember() error = %v", err)
		}

		// Verify member can access workspace
		if !svc.IsWorkspaceMember(ctx, member.ID, ws.ID) {
			t.Error("IsWorkspaceMember() should return true for added member")
		}

		// Verify role
		role, err := svc.GetUserWorkspaceRole(ctx, member.ID, ws.ID)
		if err != nil {
			t.Errorf("GetUserWorkspaceRole() error = %v", err)
		}
		if role != model.RoleEditor {
			t.Errorf("GetUserWorkspaceRole() = %v, want %v", role, model.RoleEditor)
		}
	})

	t.Run("list workspace members", func(t *testing.T) {
		members, err := svc.GetWorkspaceMembers(ctx, ws.ID)
		if err != nil {
			t.Errorf("GetWorkspaceMembers() error = %v", err)
			return
		}
		if len(members) != 2 { // owner + member
			t.Errorf("GetWorkspaceMembers() returned %d members, want 2", len(members))
		}
	})

	t.Run("update member role", func(t *testing.T) {
		err := svc.UpdateMemberRole(ctx, ws.ID, member.ID, model.RoleViewer)
		if err != nil {
			t.Errorf("UpdateMemberRole() error = %v", err)
		}

		role, _ := svc.GetUserWorkspaceRole(ctx, member.ID, ws.ID)
		if role != model.RoleViewer {
			t.Errorf("Role after update = %v, want %v", role, model.RoleViewer)
		}
	})

	t.Run("cannot remove owner", func(t *testing.T) {
		err := svc.RemoveWorkspaceMember(ctx, ws.ID, owner.ID)
		if err == nil {
			t.Error("RemoveWorkspaceMember() should fail for owner")
		}
	})

	t.Run("remove member from workspace", func(t *testing.T) {
		err := svc.RemoveWorkspaceMember(ctx, ws.ID, member.ID)
		if err != nil {
			t.Errorf("RemoveWorkspaceMember() error = %v", err)
		}

		if svc.IsWorkspaceMember(ctx, member.ID, ws.ID) {
			t.Error("IsWorkspaceMember() should return false after removal")
		}
	})
}

func TestService_ListUserWorkspaces(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	svc := NewService(s)
	ctx := context.Background()

	user, _ := svc.CreateUser(ctx, "listws@example.com", "List Workspaces", "password123")

	// Create multiple workspaces
	ws1, _ := svc.CreateWorkspace(ctx, user.ID, "Workspace 1", "First")
	ws2, _ := svc.CreateWorkspace(ctx, user.ID, "Workspace 2", "Second")

	tests := []struct {
		name    string
		userID  string
		wantErr bool
		count   int
	}{
		{
			name:    "user with workspaces",
			userID:  user.ID,
			wantErr: false,
			count:   2, // Personal workspace + 2 created
		},
		{
			name:    "empty user id",
			userID:  "",
			wantErr: true,
			count:   0,
		},
		{
			name:    "non-existent user",
			userID:  "non-existent",
			wantErr: false,
			count:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			workspaces, err := svc.ListUserWorkspaces(ctx, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListUserWorkspaces() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// Note: count includes personal workspace created automatically
			_ = ws1
			_ = ws2
			if !tt.wantErr && tt.count > 0 && len(workspaces) < tt.count {
				t.Errorf("ListUserWorkspaces() returned %d workspaces, expected at least %d", len(workspaces), tt.count)
			}
		})
	}
}

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple name",
			input:    "My Workspace",
			expected: "my-workspace",
		},
		{
			name:     "name with special chars",
			input:    "My @#$%^&*() Workspace!",
			expected: "my-workspace",
		},
		{
			name:     "name with underscores",
			input:    "my_workspace_name",
			expected: "my-workspace-name",
		},
		{
			name:     "name with multiple spaces",
			input:    "My   Workspace   Name",
			expected: "my-workspace-name",
		},
		{
			name:     "empty name",
			input:    "",
			expected: "",
		},
		{
			name:     "only special chars",
			input:    "@#$%^&*()",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slug := generateSlug(tt.input)
			// Slug will have UUID suffix, so check prefix
			if tt.expected != "" && !strings.HasPrefix(slug, tt.expected) {
				t.Errorf("generateSlug() = %v, expected prefix %v", slug, tt.expected)
			}
			// Verify slug is not empty for non-empty expected
			if tt.expected != "" && slug == "" {
				t.Error("generateSlug() returned empty string for valid input")
			}
		})
	}
}
