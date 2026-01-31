// Package users provides user management, workspace membership, and RBAC.
//
// This file handles password hashing and verification using bcrypt.
// It does NOT handle authentication (see internal/api/auth_handlers.go).
// It does NOT handle session management (see internal/sessions/).
package users

import (
	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
	"golang.org/x/crypto/bcrypt"
)

// bcryptCost is the cost parameter for bcrypt hashing.
// Higher values are more secure but slower. 12 is a good balance.
const bcryptCost = 12

// HashPassword hashes a plaintext password using bcrypt.
// Returns an error if the password is empty or hashing fails.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", apperrors.Validation("password cannot be empty")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", apperrors.Wrap(apperrors.KindInternal, "failed to hash password", err)
	}

	return string(hash), nil
}

// CheckPassword verifies a plaintext password against a bcrypt hash.
// Returns true if the password matches, false otherwise.
func CheckPassword(password, hash string) bool {
	if password == "" || hash == "" {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
