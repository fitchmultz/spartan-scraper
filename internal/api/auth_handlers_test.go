// Package api provides tests for auth handlers (login, register, logout, me).
// Tests cover security constraints including request body size limits.
package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleAuthLoginBodySizeLimit(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "valid login request",
			body:           `{"email":"test@example.com","password":"password123"}`,
			expectedStatus: http.StatusForbidden, // Invalid credentials, but not 413
		},
		{
			name:           "oversized request body",
			body:           `{"email":"test@example.com","password":"` + strings.Repeat("a", 2*1024*1024) + `"}`,
			expectedStatus: http.StatusRequestEntityTooLarge, // 413
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/auth/login", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v, body: %s", status, tt.expectedStatus, rr.Body.String())
			}
		})
	}
}

func TestHandleAuthRegisterBodySizeLimit(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "valid register request",
			body:           `{"email":"newuser@example.com","password":"password123","name":"New User"}`,
			expectedStatus: http.StatusCreated, // Successfully created
		},
		{
			name:           "oversized request body",
			body:           `{"email":"test@example.com","password":"` + strings.Repeat("a", 2*1024*1024) + `","name":"Test"}`,
			expectedStatus: http.StatusRequestEntityTooLarge, // 413
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/v1/auth/register", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			srv.Routes().ServeHTTP(rr, req)

			if status := rr.Code; status != tt.expectedStatus {
				t.Errorf("handler returned wrong status code: got %v want %v, body: %s", status, tt.expectedStatus, rr.Body.String())
			}
		})
	}
}
