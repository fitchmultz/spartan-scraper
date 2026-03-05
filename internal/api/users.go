// Package api implements the REST API server for Spartan Scraper.
//
// This file handles user management endpoints:
// - List users (admin only)
// - Get user by ID
// - Update user profile
// - Delete user
package api

import (
	"encoding/json"
	"net/http"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/users"
)

// handleUsers handles GET /v1/users - list users (admin only in future)
func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	// Get current user ID
	currentUserID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, r, apperrors.Permission("not authenticated"))
		return
	}

	// For now, just return the current user
	userService := users.NewService(s.store)
	user, err := userService.GetUser(r.Context(), currentUserID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	writeJSON(w, map[string]any{"users": []*model.User{user}})
}

// handleUser handles GET/PUT/DELETE /v1/users/{id}
func (s *Server) handleUser(w http.ResponseWriter, r *http.Request) {
	// Extract user ID from path
	id := extractID(r.URL.Path, "users")
	if id == "" {
		writeError(w, r, apperrors.Validation("user ID required"))
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
	// Users can only get their own profile for now
	currentUserID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, r, apperrors.Permission("not authenticated"))
		return
	}

	if id != "me" && id != currentUserID {
		writeError(w, r, apperrors.Permission("can only view your own profile"))
		return
	}

	userService := users.NewService(s.store)
	user, err := userService.GetUser(r.Context(), currentUserID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	writeJSON(w, user)
}

// handleUpdateUser handles PUT /v1/users/{id}
func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request, id string) {
	// Users can only update their own profile
	currentUserID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, r, apperrors.Permission("not authenticated"))
		return
	}

	if id != "me" && id != currentUserID {
		writeError(w, r, apperrors.Permission("can only update your own profile"))
		return
	}

	var req struct {
		Name      string `json:"name"`
		AvatarURL string `json:"avatarUrl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, apperrors.Validation("invalid request body"))
		return
	}

	userService := users.NewService(s.store)
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

	w.WriteHeader(http.StatusOK)
	writeJSON(w, user)
}

// handleDeleteUser handles DELETE /v1/users/{id}
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request, id string) {
	// Users can only delete their own account
	currentUserID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, r, apperrors.Permission("not authenticated"))
		return
	}

	if id != "me" && id != currentUserID {
		writeError(w, r, apperrors.Permission("can only delete your own account"))
		return
	}

	userService := users.NewService(s.store)
	if err := userService.DeleteUser(r.Context(), currentUserID); err != nil {
		writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
