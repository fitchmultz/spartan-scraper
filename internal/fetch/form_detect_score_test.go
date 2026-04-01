// Package fetch provides fetch functionality for Spartan Scraper.
//
// Purpose:
// - Verify form detect score test behavior for package fetch.
//
// Responsibilities:
// - Define focused Go test coverage, fixtures, and assertions for the package behavior exercised here.
//
// Scope:
// - Automated test coverage only; production behavior stays in non-test package files.
//
// Usage:
// - Run with `go test` for package `fetch` or through `make test-ci`/`make ci`.
//
// Invariants/Assumptions:
// - Tests should remain deterministic and describe the package contract they protect.

package fetch

import (
	"testing"
)

// TestFormDetector_Score tests that confidence scores are reasonable.
func TestFormDetector_Score(t *testing.T) {
	html := `
		<html>
		<body>
			<form>
				<input type="text" name="username" autocomplete="username">
				<input type="password" name="password" autocomplete="current-password">
				<button type="submit">Login</button>
			</form>
		</body>
		</html>
	`

	detector := NewFormDetector()
	forms, err := detector.DetectForms(html)
	if err != nil {
		t.Fatalf("DetectForms() error = %v", err)
	}

	if len(forms) == 0 {
		t.Fatal("expected form to be detected")
	}

	form := forms[0]

	// Score should be reasonable for a well-formed login form
	if form.Score < 0.5 {
		t.Errorf("expected score >= 0.5 for good login form, got %f", form.Score)
	}

	if form.Score > 1.0 {
		t.Errorf("expected score <= 1.0, got %f", form.Score)
	}
}

// TestFormDetector_FieldConfidence tests that field confidence scores are populated.
func TestFormDetector_FieldConfidence(t *testing.T) {
	html := `
		<html>
		<body>
			<form>
				<input type="email" name="email" autocomplete="username">
				<input type="password" name="password" autocomplete="current-password">
				<button type="submit">Login</button>
			</form>
		</body>
		</html>
	`

	detector := NewFormDetector()
	forms, err := detector.DetectForms(html)
	if err != nil {
		t.Fatalf("DetectForms() error = %v", err)
	}

	if len(forms) == 0 {
		t.Fatal("expected form to be detected")
	}

	form := forms[0]

	if form.UserField != nil && form.UserField.Confidence <= 0 {
		t.Error("expected positive confidence for user field")
	}

	if form.PassField != nil && form.PassField.Confidence <= 0 {
		t.Error("expected positive confidence for password field")
	}

	if form.SubmitField != nil && form.SubmitField.Confidence <= 0 {
		t.Error("expected positive confidence for submit field")
	}
}

// TestFormDetector_MatchReasons tests that match reasons are populated.
func TestFormDetector_MatchReasons(t *testing.T) {
	html := `
		<html>
		<body>
			<form>
				<input type="email" name="email" autocomplete="username">
				<input type="password" name="password">
				<button type="submit">Login</button>
			</form>
		</body>
		</html>
	`

	detector := NewFormDetector()
	forms, err := detector.DetectForms(html)
	if err != nil {
		t.Fatalf("DetectForms() error = %v", err)
	}

	if len(forms) == 0 {
		t.Fatal("expected form to be detected")
	}

	form := forms[0]

	if form.UserField != nil && len(form.UserField.MatchReasons) == 0 {
		t.Error("expected match reasons for user field")
	}

	if form.PassField != nil && len(form.PassField.MatchReasons) == 0 {
		t.Error("expected match reasons for password field")
	}
}
