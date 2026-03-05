// Package users provides user management, workspace membership, and RBAC.
//
// This file tests user CRUD operations (Create, Get, Update, Delete).
package users

import (
	"context"
	"strings"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/model"
)

func TestService_CreateUser(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	svc := NewService(s)
	ctx := context.Background()

	tests := []struct {
		testName string
		email    string
		userName string
		password string
		wantErr  bool
		errType  string
	}{
		{
			testName: "valid user",
			email:    "test@example.com",
			userName: "Test User",
			password: "password123",
			wantErr:  false,
		},
		{
			testName: "empty email",
			email:    "",
			userName: "Test User",
			password: "password123",
			wantErr:  true,
			errType:  "validation",
		},
		{
			testName: "invalid email",
			email:    "notanemail",
			userName: "Test User",
			password: "password123",
			wantErr:  true,
			errType:  "validation",
		},
		{
			testName: "empty name",
			email:    "test2@example.com",
			userName: "",
			password: "password123",
			wantErr:  true,
			errType:  "validation",
		},
		{
			testName: "duplicate email",
			email:    "test@example.com",
			userName: "Another User",
			password: "password123",
			wantErr:  true,
			errType:  "validation",
		},
		{
			testName: "OAuth user without password",
			email:    "oauth@example.com",
			userName: "OAuth User",
			password: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			user, err := svc.CreateUser(ctx, tt.email, tt.userName, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if user == nil {
					t.Error("CreateUser() returned nil user")
					return
				}
				if user.Email != strings.ToLower(tt.email) {
					t.Errorf("CreateUser() email = %v, want %v", user.Email, strings.ToLower(tt.email))
				}
				if user.Name != tt.userName {
					t.Errorf("CreateUser() name = %v, want %v", user.Name, tt.userName)
				}
				if user.ID == "" {
					t.Error("CreateUser() user ID is empty")
				}
				if !user.IsActive {
					t.Error("CreateUser() user should be active")
				}
				if user.AuthProvider != "local" {
					t.Errorf("CreateUser() authProvider = %v, want local", user.AuthProvider)
				}
			}
		})
	}
}

func TestService_GetUser(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	svc := NewService(s)
	ctx := context.Background()

	// Create a test user
	created, err := svc.CreateUser(ctx, "getuser@example.com", "Get User", "password123")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "existing user",
			id:      created.ID,
			wantErr: false,
		},
		{
			name:    "non-existent user",
			id:      "non-existent-id",
			wantErr: true,
		},
		{
			name:    "empty id",
			id:      "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := svc.GetUser(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && user.ID != tt.id {
				t.Errorf("GetUser() id = %v, want %v", user.ID, tt.id)
			}
		})
	}
}

func TestService_GetUserByEmail(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	svc := NewService(s)
	ctx := context.Background()

	// Create a test user
	email := "byemail@example.com"
	_, err := svc.CreateUser(ctx, email, "By Email", "password123")
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "existing user",
			email:   email,
			wantErr: false,
		},
		{
			name:    "uppercase email",
			email:   strings.ToUpper(email),
			wantErr: false,
		},
		{
			name:    "non-existent user",
			email:   "nonexistent@example.com",
			wantErr: true,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := svc.GetUserByEmail(ctx, tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUserByEmail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.EqualFold(user.Email, tt.email) {
				t.Errorf("GetUserByEmail() email = %v, want %v", user.Email, tt.email)
			}
		})
	}
}

func TestService_UpdateUser(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	svc := NewService(s)
	ctx := context.Background()

	user, _ := svc.CreateUser(ctx, "update@example.com", "Original Name", "password123")

	tests := []struct {
		name    string
		user    *model.User
		wantErr bool
	}{
		{
			name: "valid update",
			user: &model.User{
				ID:        user.ID,
				Name:      "Updated Name",
				AvatarURL: "https://example.com/avatar.png",
			},
			wantErr: false,
		},
		{
			name:    "nil user",
			user:    nil,
			wantErr: true,
		},
		{
			name: "empty user id",
			user: &model.User{
				ID:   "",
				Name: "Name",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.UpdateUser(ctx, tt.user)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateUser() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Verify the update persisted
	updated, _ := svc.GetUser(ctx, user.ID)
	if updated.Name != "Updated Name" {
		t.Errorf("UpdateUser() name not updated, got %v", updated.Name)
	}
	if updated.AvatarURL != "https://example.com/avatar.png" {
		t.Errorf("UpdateUser() avatarURL not updated, got %v", updated.AvatarURL)
	}
	if updated.UpdatedAt.Equal(user.UpdatedAt) {
		t.Error("UpdateUser() updated_at should be different from original")
	}
}

func TestService_DeleteUser(t *testing.T) {
	s, cleanup := setupTestStore(t)
	defer cleanup()

	svc := NewService(s)
	ctx := context.Background()

	user, _ := svc.CreateUser(ctx, "delete@example.com", "Delete Me", "password123")

	tests := []struct {
		name    string
		id      string
		wantErr bool
	}{
		{
			name:    "existing user",
			id:      user.ID,
			wantErr: false,
		},
		{
			name:    "empty id",
			id:      "",
			wantErr: true,
		},
		{
			name:    "already deleted user",
			id:      user.ID,
			wantErr: false, // Deleting non-existent user doesn't error in current implementation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.DeleteUser(ctx, tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteUser() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	// Verify user is deleted
	_, err := svc.GetUser(ctx, user.ID)
	if err == nil {
		t.Error("GetUser() should return error for deleted user")
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "valid email",
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with plus",
			email:   "test+tag@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with subdomain",
			email:   "test@mail.example.com",
			wantErr: false,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: true,
		},
		{
			name:    "whitespace email",
			email:   "   ",
			wantErr: true,
		},
		{
			name:    "missing @",
			email:   "testexample.com",
			wantErr: true,
		},
		{
			name:    "missing domain",
			email:   "test@",
			wantErr: true,
		},
		{
			name:    "missing local part",
			email:   "@example.com",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEmail(tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
