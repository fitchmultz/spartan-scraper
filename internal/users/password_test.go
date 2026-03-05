// Package users provides user management, workspace membership, and RBAC.
//
// This file tests password hashing and verification.
package users

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "mysecretpassword",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
		},
		{
			name:     "long password",
			password: "thisisaverylongpasswordthatexceedsnormallengths123456789",
			wantErr:  false,
		},
		{
			name:     "password with special chars",
			password: "p@$$w0rd!#$%^&*()",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("HashPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && hash == "" {
				t.Error("HashPassword() returned empty hash for valid password")
			}
		})
	}
}

func TestCheckPassword(t *testing.T) {
	// Create a valid hash
	password := "mysecretpassword"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		want     bool
	}{
		{
			name:     "correct password",
			password: password,
			hash:     hash,
			want:     true,
		},
		{
			name:     "incorrect password",
			password: "wrongpassword",
			hash:     hash,
			want:     false,
		},
		{
			name:     "empty password",
			password: "",
			hash:     hash,
			want:     false,
		},
		{
			name:     "empty hash",
			password: password,
			hash:     "",
			want:     false,
		},
		{
			name:     "both empty",
			password: "",
			hash:     "",
			want:     false,
		},
		{
			name:     "invalid hash",
			password: password,
			hash:     "invalidhash",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckPassword(tt.password, tt.hash)
			if got != tt.want {
				t.Errorf("CheckPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPasswordHashingRoundTrip(t *testing.T) {
	password := "testpassword123"

	// Hash the password
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}

	// Verify the hash is different from the password
	if hash == password {
		t.Error("Hash should be different from the original password")
	}

	// Verify the password matches
	if !CheckPassword(password, hash) {
		t.Error("CheckPassword() should return true for correct password")
	}

	// Verify a different password doesn't match
	if CheckPassword("differentpassword", hash) {
		t.Error("CheckPassword() should return false for incorrect password")
	}
}
