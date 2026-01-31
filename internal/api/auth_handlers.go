// Package api implements the REST API server for Spartan Scraper.
//
// This file handles session-based authentication for web UI:
// - Login/logout endpoints
// - User registration
// - Session cookie management
//
// This file does NOT handle:
// - Target website authentication (see auth.go)
// - API key authentication (see middleware.go)
package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"github.com/fitchmultz/spartan-scraper/internal/model"
	"github.com/fitchmultz/spartan-scraper/internal/sessions"
	"github.com/fitchmultz/spartan-scraper/internal/users"
)

const (
	// SessionCookieName is the name of the session cookie
	SessionCookieName = "spartan_session"
	// SessionHeaderName is the header for session token (alternative to cookie)
	SessionHeaderName = "X-Session-Token"
)

// LoginRequest represents a login request
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginResponse represents a successful login response
type LoginResponse struct {
	User       *model.User        `json:"user"`
	Workspaces []*model.Workspace `json:"workspaces"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// handleAuthLogin handles POST /v1/auth/login
func (s *Server) handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, apperrors.Validation("invalid request body"))
		return
	}

	if req.Email == "" || req.Password == "" {
		writeError(w, r, apperrors.Validation("email and password are required"))
		return
	}

	// Get user with password hash
	user, passwordHash, err := s.store.GetUserWithPassword(r.Context(), req.Email)
	if err != nil {
		// Don't reveal if email exists or not
		writeError(w, r, apperrors.Permission("invalid credentials"))
		return
	}

	// Verify password
	if !users.CheckPassword(req.Password, passwordHash) {
		writeError(w, r, apperrors.Permission("invalid credentials"))
		return
	}

	// Check if user is active
	if !user.IsActive {
		writeError(w, r, apperrors.Permission("account is disabled"))
		return
	}

	// Create session
	sessionStore := sessions.NewStore(s.store)
	ipAddress := r.RemoteAddr
	userAgent := r.UserAgent()
	session, token, err := sessionStore.CreateSession(r.Context(), user.ID, ipAddress, userAgent, 24*time.Hour)
	if err != nil {
		writeError(w, r, apperrors.Wrap(apperrors.KindInternal, "failed to create session", err))
		return
	}

	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil, // Only secure if HTTPS
		SameSite: http.SameSiteStrictMode,
		Expires:  session.ExpiresAt,
	})

	// Get user's workspaces
	userService := users.NewService(s.store)
	workspaces, err := userService.ListUserWorkspaces(r.Context(), user.ID)
	if err != nil {
		workspaces = []*model.Workspace{}
	}

	// Create audit log
	s.createAuditLog(r.Context(), "", user.ID, model.AuditActionUserLogin, model.AuditResourceUser, user.ID, nil)

	w.WriteHeader(http.StatusOK)
	writeJSON(w, LoginResponse{
		User:       user,
		Workspaces: workspaces,
	})
}

// handleAuthLogout handles POST /v1/auth/logout
func (s *Server) handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	// Get session token from cookie or header
	token := getSessionToken(r)
	if token != "" {
		sessionStore := sessions.NewStore(s.store)
		session, err := sessionStore.ValidateSession(r.Context(), token)
		if err == nil {
			// Create audit log before deleting session
			s.createAuditLog(r.Context(), "", session.UserID, model.AuditActionUserLogout, model.AuditResourceSession, session.ID, nil)
		}

		// Delete session
		_ = sessionStore.DeleteSession(r.Context(), token)
	}

	// Clear session cookie
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	w.WriteHeader(http.StatusOK)
	writeJSON(w, map[string]string{"status": "logged out"})
}

// handleAuthRegister handles POST /v1/auth/register
func (s *Server) handleAuthRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, apperrors.Validation("invalid request body"))
		return
	}

	if req.Email == "" || req.Password == "" || req.Name == "" {
		writeError(w, r, apperrors.Validation("email, password, and name are required"))
		return
	}

	// Validate password strength
	if len(req.Password) < 8 {
		writeError(w, r, apperrors.Validation("password must be at least 8 characters"))
		return
	}

	// Create user
	userService := users.NewService(s.store)
	user, err := userService.CreateUser(r.Context(), req.Email, req.Name, req.Password)
	if err != nil {
		if apperrors.IsKind(err, apperrors.KindValidation) {
			writeError(w, r, err)
			return
		}
		writeError(w, r, apperrors.Wrap(apperrors.KindInternal, "failed to create user", err))
		return
	}

	// Create audit log
	s.createAuditLog(r.Context(), "", user.ID, model.AuditActionUserRegister, model.AuditResourceUser, user.ID, map[string]any{
		"email": user.Email,
		"name":  user.Name,
	})

	w.WriteHeader(http.StatusCreated)
	writeJSON(w, user)
}

// handleAuthMe handles GET /v1/auth/me - returns current user info
func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, r, apperrors.MethodNotAllowed("method not allowed"))
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, ok := GetUserIDFromContext(r.Context())
	if !ok {
		writeError(w, r, apperrors.Permission("not authenticated"))
		return
	}

	// Get user
	userService := users.NewService(s.store)
	user, err := userService.GetUser(r.Context(), userID)
	if err != nil {
		writeError(w, r, err)
		return
	}

	// Get user's workspaces
	workspaces, err := userService.ListUserWorkspaces(r.Context(), user.ID)
	if err != nil {
		workspaces = []*model.Workspace{}
	}

	w.WriteHeader(http.StatusOK)
	writeJSON(w, LoginResponse{
		User:       user,
		Workspaces: workspaces,
	})
}

// getSessionToken extracts the session token from cookie or header
func getSessionToken(r *http.Request) string {
	// Try cookie first
	cookie, err := r.Cookie(SessionCookieName)
	if err == nil && cookie.Value != "" {
		return cookie.Value
	}

	// Try header
	return r.Header.Get(SessionHeaderName)
}

// createAuditLog creates an audit log entry
func (s *Server) createAuditLog(ctx context.Context, workspaceID, userID, action, resourceType, resourceID string, metadata map[string]any) {
	log := &model.AuditLog{
		ID:           generateID(),
		WorkspaceID:  workspaceID,
		UserID:       userID,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Metadata:     metadata,
		CreatedAt:    time.Now().UTC(),
	}
	_ = s.store.CreateAuditLog(ctx, log)
}

// generateID generates a unique ID for audit logs
func generateID() string {
	return time.Now().Format(time.RFC3339Nano) + "-" + randomString(8)
}

// randomString generates a random string of the given length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
