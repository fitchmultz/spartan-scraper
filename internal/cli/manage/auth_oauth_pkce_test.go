// Package manage contains CLI commands for configuration/data management.
//
// Purpose:
// - Verify PKCE derivation helpers.
//
// Responsibilities:
// - Cover PKCE challenge derivation edge cases and determinism.
//
// Scope:
// - OAuth CLI behavior only.
//
// Usage:
// - Run with `go test ./internal/cli/manage`.
//
// Invariants/Assumptions:
// - Tests use temp data dirs and mocked HTTP servers only.
package manage

import (
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/auth"
)

func TestDerivePKCEChallengeS256(t *testing.T) {
	// Generate a verifier first
	verifier, _, _, err := auth.GeneratePKCE()
	if err != nil {
		t.Fatalf("GeneratePKCE failed: %v", err)
	}

	// Derive challenge from verifier
	challenge, method, err := auth.DerivePKCEChallengeS256(verifier)
	if err != nil {
		t.Fatalf("DerivePKCEChallengeS256 failed: %v", err)
	}

	if challenge == "" {
		t.Error("challenge should not be empty")
	}
	if method != auth.PKCEMethodS256 {
		t.Errorf("expected method '%s', got '%s'", auth.PKCEMethodS256, method)
	}
}

// TestDerivePKCEChallengeS256_EmptyVerifier tests with empty verifier.
func TestDerivePKCEChallengeS256_EmptyVerifier(t *testing.T) {
	_, _, err := auth.DerivePKCEChallengeS256("")
	if err == nil {
		t.Error("expected error for empty verifier")
	}
}

// TestDerivePKCEChallengeS256_Consistency tests that the same verifier produces the same challenge.
func TestDerivePKCEChallengeS256_Consistency(t *testing.T) {
	verifier := "test-verifier-string"

	challenge1, method1, err1 := auth.DerivePKCEChallengeS256(verifier)
	challenge2, method2, err2 := auth.DerivePKCEChallengeS256(verifier)

	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: %v, %v", err1, err2)
	}

	if challenge1 != challenge2 {
		t.Error("same verifier should produce same challenge")
	}
	if method1 != method2 {
		t.Error("same verifier should produce same method")
	}
}
