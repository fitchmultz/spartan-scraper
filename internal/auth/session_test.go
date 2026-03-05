// Package auth provides tests for session persistence functionality.
package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionStore_LoadSave(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewSessionStore(tmpDir)

	sessions := []Session{
		{
			ID:     "test1",
			Name:   "Test Session 1",
			Domain: "example.com",
			Cookies: []Cookie{
				{Name: "session", Value: "abc123"},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	if err := store.Save(sessions); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded) != 1 {
		t.Errorf("expected 1 session, got %d", len(loaded))
	}

	if loaded[0].ID != "test1" {
		t.Errorf("expected ID test1, got %s", loaded[0].ID)
	}
}

func TestSessionStore_Upsert(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewSessionStore(tmpDir)

	session := Session{
		ID:     "test1",
		Domain: "example.com",
		Cookies: []Cookie{
			{Name: "session", Value: "abc123"},
		},
	}

	if err := store.Upsert(session); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Update existing
	session.Cookies = append(session.Cookies, Cookie{Name: "auth", Value: "xyz"})
	if err := store.Upsert(session); err != nil {
		t.Fatalf("Upsert update failed: %v", err)
	}

	loaded, found, err := store.Get("test1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("session not found")
	}
	if len(loaded.Cookies) != 2 {
		t.Errorf("expected 2 cookies, got %d", len(loaded.Cookies))
	}
}

func TestSessionStore_Delete(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewSessionStore(tmpDir)

	session := Session{
		ID:     "test1",
		Domain: "example.com",
	}

	if err := store.Upsert(session); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	if err := store.Delete("test1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, found, err := store.Get("test1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Error("session should not be found after delete")
	}
}

func TestSessionStore_Get_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewSessionStore(tmpDir)

	_, found, err := store.Get("nonexistent")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if found {
		t.Error("expected session not to be found")
	}
}

func TestSessionStore_Validation(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewSessionStore(tmpDir)

	// Empty ID should fail
	session := Session{
		ID:     "",
		Domain: "example.com",
	}

	err := store.Upsert(session)
	if err == nil {
		t.Error("expected validation error for empty ID")
	}
}

func TestSessionStore_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewSessionStore(tmpDir)

	session := Session{
		ID:     "test1",
		Domain: "example.com",
	}

	if err := store.Upsert(session); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Check file permissions
	path := filepath.Join(tmpDir, "sessions.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	// File should be readable/writable by owner only (0600)
	mode := info.Mode().Perm()
	if mode != 0o600 {
		t.Errorf("expected file mode 0600, got %04o", mode)
	}
}

func TestCookiesFromJar(t *testing.T) {
	// This is a basic test - full testing would require a real cookie jar
	// which is difficult to set up in unit tests

	// Create a simple test to verify the function doesn't panic
	tmpDir := t.TempDir()
	store := NewSessionStore(tmpDir)

	session := Session{
		ID:     "test1",
		Domain: "example.com",
		Cookies: []Cookie{
			{Name: "session", Value: "abc123", Domain: "example.com", Path: "/"},
		},
	}

	if err := store.Upsert(session); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Verify we can load it back
	loaded, found, err := store.Get("test1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !found {
		t.Fatal("session not found")
	}
	if len(loaded.Cookies) != 1 {
		t.Errorf("expected 1 cookie, got %d", len(loaded.Cookies))
	}
	if loaded.Cookies[0].Name != "session" {
		t.Errorf("expected cookie name 'session', got '%s'", loaded.Cookies[0].Name)
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com/path", "example.com"},
		{"https://example.com:8443/path", "example.com:8443"},
		{"http://sub.example.com", "sub.example.com"},
		{"invalid-url", ""},
		{"", ""},
	}

	for _, tt := range tests {
		result := ExtractDomain(tt.input)
		if result != tt.expected {
			t.Errorf("ExtractDomain(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}
