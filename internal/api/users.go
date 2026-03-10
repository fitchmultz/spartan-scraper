// Package api implements the REST API server for Spartan Scraper.
//
// This file handles user management endpoints:
// - List users (admin only)
// - Get user by ID
// - Update user profile
// - Delete user
package api

import (
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
)

// handleUsers handles GET /v1/users - list users (admin only in future)
func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	currentUserID, err := currentUserID(r)
	if err != nil {
		writeError(w, r, err)
		return
	}

	user, err := s.userService().GetUser(r.Context(), currentUserID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeCollectionJSON(w, "users", []*model.User{user})
}

// handleUser handles GET/PUT/DELETE /v1/users/{id}
func (s *Server) handleUser(w http.ResponseWriter, r *http.Request) {
	id, err := requireResourceID(r, "users", "user id")
	if err != nil {
		writeError(w, r, err)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetUser(w, r, id)
	case http.MethodPut:
		s.handleUpdateUser(w, r, id)
	case http.MethodDelete:
		s.handleDeleteUser(w, r, id)
	default:
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
	}
}

// handleGetUser handles GET /v1/users/{id}
func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request, id string) {
	currentUserID, err := currentUserIDForResource(r, id, "view")
	if err != nil {
		writeError(w, r, err)
		return
	}

	user, err := s.userService().GetUser(r.Context(), currentUserID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, user)
}

// handleUpdateUser handles PUT /v1/users/{id}
func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request, id string) {
	currentUserID, err := currentUserIDForResource(r, id, "update")
	if err != nil {
		writeError(w, r, err)
		return
	}

	var req struct {
		Name      string `json:"name"`
		AvatarURL string `json:"avatarUrl"`
	}
	if err := decodeJSONBody(w, r, &req); err != nil {
		writeError(w, r, err)
		return
	}

	userService := s.userService()
	user, err := userService.GetUser(r.Context(), currentUserID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.AvatarURL != "" {
		user.AvatarURL = req.AvatarURL
	}

	if err := userService.UpdateUser(r.Context(), user); err != nil {
		writeError(w, r, err)
		return
	}

	writeJSON(w, user)
}

// handleDeleteUser handles DELETE /v1/users/{id}
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request, id string) {
	currentUserID, err := currentUserIDForResource(r, id, "delete")
	if err != nil {
		writeError(w, r, err)
		return
	}

	if err := s.userService().DeleteUser(r.Context(), currentUserID); err != nil {
		writeError(w, r, err)
		return
	}

	writeNoContent(w)
}
