// Package api implements the REST API server for Spartan Scraper.
//
// This file handles workspace management endpoints:
// - List workspaces for current user
// - Create new workspace
// - Get workspace details
// - Update workspace
// - Delete workspace
// - Manage workspace members
package api

import (
	"net/http"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// handleWorkspaces handles GET/POST /v1/workspaces
func (s *Server) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListWorkspaces(w, r)
	case http.MethodPost:
		s.handleCreateWorkspace(w, r)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

// handleListWorkspaces handles GET /v1/workspaces
func (s *Server) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
	userID, err := currentUserID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	workspaces, err := s.userService().ListUserWorkspaces(r.Context(), userID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeCollectionJSON(w, "workspaces", workspaces)
}

// handleCreateWorkspace handles POST /v1/workspaces
func (s *Server) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	userID, err := currentUserID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	if req.Name == "" {
		writeError(w, r, apperrors.Validation("name is required"))
		return
	}

	userService := s.userService()
	workspace, err := userService.CreateWorkspace(r.Context(), userID, req.Name, req.Description)
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Create audit log
	s.createAuditLog(r.Context(), workspace.ID, userID, model.AuditActionWorkspaceCreate, model.AuditResourceWorkspace, workspace.ID, map[string]any{
		"name": workspace.Name,
	})

	writeCreatedJSON(w, workspace)
}

// handleWorkspace handles GET/PUT/DELETE /v1/workspaces/{id} and members sub-routes
func (s *Server) handleWorkspace(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	// Check if this is a members sub-route
	if strings.Contains(path, "/members") {
		s.handleWorkspaceMembers(w, r)
		return
	}

	// Extract workspace ID from path
	id := extractID(path, "workspaces")
	if id == "" {
		writeError(w, r, apperrors.Validation("workspace ID required"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetWorkspace(w, r, id)
	case http.MethodPut:
		s.handleUpdateWorkspace(w, r, id)
	case http.MethodDelete:
		s.handleDeleteWorkspace(w, r, id)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

// handleGetWorkspace handles GET /v1/workspaces/{id}
func (s *Server) handleGetWorkspace(w http.ResponseWriter, r *http.Request, id string) {
	userID, err := currentUserID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	userService := s.userService()

	// Check if user has access to workspace
	if !userService.CanAccessWorkspace(r.Context(), userID, id) {
		writeError(w, r, apperrors.Permission("access denied"))
		return
	}

	workspace, err := userService.GetWorkspace(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, workspace)
}

// handleUpdateWorkspace handles PUT /v1/workspaces/{id}
func (s *Server) handleUpdateWorkspace(w http.ResponseWriter, r *http.Request, id string) {
	userID, err := currentUserID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	userService := s.userService()

	// Check if user can manage workspace
	if !userService.CanManageWorkspace(r.Context(), userID, id) {
		writeError(w, r, apperrors.Permission("admin access required"))
		return
	}

	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	workspace, err := userService.GetWorkspace(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	if req.Name != "" {
		workspace.Name = req.Name
	}
	workspace.Description = req.Description

	if err := userService.UpdateWorkspace(r.Context(), workspace); err != nil {
		writeError(w, r, err)
		return
	}

	// Create audit log
	s.createAuditLog(r.Context(), id, userID, model.AuditActionWorkspaceUpdate, model.AuditResourceWorkspace, id, map[string]any{
		"name": workspace.Name,
	})

	writeJSON(w, workspace)
}

// handleDeleteWorkspace handles DELETE /v1/workspaces/{id}
func (s *Server) handleDeleteWorkspace(w http.ResponseWriter, r *http.Request, id string) {
	userID, err := currentUserID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	userService := s.userService()

	// Check if user can manage workspace
	if !userService.CanManageWorkspace(r.Context(), userID, id) {
		writeError(w, r, apperrors.Permission("admin access required"))
		return
	}

	// Create audit log before deletion
	s.createAuditLog(r.Context(), id, userID, model.AuditActionWorkspaceDelete, model.AuditResourceWorkspace, id, nil)

	if err := userService.DeleteWorkspace(r.Context(), id); err != nil {
		writeError(w, r, err)
		return
	}

	writeNoContent(w)
}

// handleWorkspaceMembers handles GET/POST /v1/workspaces/{id}/members
func (s *Server) handleWorkspaceMembers(w http.ResponseWriter, r *http.Request) {
	// Extract workspace ID from path
	workspaceID := extractID(r.URL.Path, "workspaces")
	if workspaceID == "" {
		writeError(w, r, apperrors.Validation("workspace ID required"))
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleListWorkspaceMembers(w, r, workspaceID)
	case http.MethodPost:
		s.handleAddWorkspaceMember(w, r, workspaceID)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

// handleListWorkspaceMembers handles GET /v1/workspaces/{id}/members
func (s *Server) handleListWorkspaceMembers(w http.ResponseWriter, r *http.Request, workspaceID string) {
	userID, err := currentUserID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	userService := s.userService()

	// Check if user has access to workspace
	if !userService.CanAccessWorkspace(r.Context(), userID, workspaceID) {
		writeError(w, r, apperrors.Permission("access denied"))
		return
	}

	members, err := userService.GetWorkspaceMembers(r.Context(), workspaceID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeCollectionJSON(w, "members", members)
}

// handleAddWorkspaceMember handles POST /v1/workspaces/{id}/members
func (s *Server) handleAddWorkspaceMember(w http.ResponseWriter, r *http.Request, workspaceID string) {
	userID, err := currentUserID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	userService := s.userService()

	// Check if user can invite members
	if !userService.CanInviteMembers(r.Context(), userID, workspaceID) {
		writeError(w, r, apperrors.Permission("admin access required"))
		return
	}

	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	if req.Email == "" || req.Role == "" {
		writeError(w, r, apperrors.Validation("email and role are required"))
		return
	}

	// Find user by email
	targetUser, err := userService.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		writeError(w, r, apperrors.NotFound("user not found"))
		return
	}

	role := model.Role(req.Role)
	if !role.IsValid() {
		writeError(w, r, apperrors.Validation("invalid role"))
		return
	}

	if err := userService.AddWorkspaceMember(r.Context(), workspaceID, targetUser.ID, role, userID); err != nil {
		writeError(w, r, err)
		return
	}

	// Create audit log
	s.createAuditLog(r.Context(), workspaceID, userID, model.AuditActionMemberAdd, model.AuditResourceMember, targetUser.ID, map[string]any{
		"email": req.Email,
		"role":  req.Role,
	})

	writeCreatedJSON(w, map[string]string{"status": "member added"})
}

// handleWorkspaceMember handles DELETE /v1/workspaces/{id}/members/{userId}
func (s *Server) handleWorkspaceMember(w http.ResponseWriter, r *http.Request) {
	// This would handle removing/updating members
	// For now, return method not allowed
	writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
}
