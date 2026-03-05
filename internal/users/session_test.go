// Package users provides user management, workspace membership, and RBAC.
//
// This file tests user session functionality.
package users

import (
	"testing"
	"time"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestUserSession_IsExpired(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name     string
		session  model.UserSession
		expected bool
	}{
		{
			name: "not expired",
			session: model.UserSession{
				ExpiresAt: now.Add(time.Hour),
			},
			expected: false,
		},
		{
			name: "expired",
			session: model.UserSession{
				ExpiresAt: now.Add(-time.Hour),
			},
			expected: true,
		},
		{
			name: "expires now",
			session: model.UserSession{
				ExpiresAt: now.Add(-time.Second),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.session.IsExpired(); got != tt.expected {
				t.Errorf("UserSession.IsExpired() = %v, want %v", got, tt.expected)
			}
		})
	}
}
