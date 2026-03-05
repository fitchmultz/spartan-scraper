// Package fetch provides tests for render profile management.
//
// Tests cover:
// - Load/Save round-trip
// - CRUD operations (List, Get, Upsert, Delete)
// - Validation (duplicate names, invalid fields, unknown JSON fields)
// - Edge cases (missing file, empty profiles)
//
// Does NOT test:
// - Runtime profile matching (see render_profiles_store_test.go)
// - File system permissions (assumes writable temp dir)
package fetch

import (
	"os"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

func TestLoadRenderProfilesFile_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	file, err := LoadRenderProfilesFile(tmpDir)
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if len(file.Profiles) != 0 {
		t.Errorf("expected empty profiles, got %d", len(file.Profiles))
	}
}

func TestLoadRenderProfilesFile_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := RenderProfilesPath(tmpDir)
	if err := os.WriteFile(path, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	_, err := LoadRenderProfilesFile(tmpDir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadRenderProfilesFile_UnknownField(t *testing.T) {
	tmpDir := t.TempDir()
	path := RenderProfilesPath(tmpDir)
	content := `{"profiles": [{"name": "test", "hostPatterns": ["example.com"], "unknownField": "value"}]}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	_, err := LoadRenderProfilesFile(tmpDir)
	if err == nil {
		t.Error("expected error for unknown field")
	}
}

func TestLoadRenderProfilesFile_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	path := RenderProfilesPath(tmpDir)
	content := `{
		"profiles": [
			{
				"name": "test-profile",
				"hostPatterns": ["example.com", "*.example.com"],
				"forceEngine": "chromedp"
			}
		]
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	file, err := LoadRenderProfilesFile(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(file.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(file.Profiles))
	}
	if file.Profiles[0].Name != "test-profile" {
		t.Errorf("expected name 'test-profile', got '%s'", file.Profiles[0].Name)
	}
}

func TestSaveRenderProfilesFile_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	file := RenderProfilesFile{
		Profiles: []RenderProfile{
			{
				Name:         "profile1",
				HostPatterns: []string{"example.com"},
				ForceEngine:  RenderEngineHTTP,
			},
			{
				Name:         "profile2",
				HostPatterns: []string{"*.other.com"},
				ForceEngine:  RenderEngineChromedp,
			},
		},
	}

	if err := SaveRenderProfilesFile(tmpDir, file); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	loaded, err := LoadRenderProfilesFile(tmpDir)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	if len(loaded.Profiles) != 2 {
		t.Fatalf("expected 2 profiles, got %d", len(loaded.Profiles))
	}
	if loaded.Profiles[0].Name != "profile1" {
		t.Errorf("expected profile1 first, got %s", loaded.Profiles[0].Name)
	}
	if loaded.Profiles[1].Name != "profile2" {
		t.Errorf("expected profile2 second, got %s", loaded.Profiles[1].Name)
	}
}

func TestListRenderProfileNames(t *testing.T) {
	tmpDir := t.TempDir()
	file := RenderProfilesFile{
		Profiles: []RenderProfile{
			{Name: "zebra", HostPatterns: []string{"z.com"}},
			{Name: "alpha", HostPatterns: []string{"a.com"}},
			{Name: "beta", HostPatterns: []string{"b.com"}},
		},
	}
	if err := SaveRenderProfilesFile(tmpDir, file); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	names, err := ListRenderProfileNames(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []string{"alpha", "beta", "zebra"}
	if len(names) != len(expected) {
		t.Fatalf("expected %d names, got %d", len(expected), len(names))
	}
	for i, name := range names {
		if name != expected[i] {
			t.Errorf("expected name[%d] = %s, got %s", i, expected[i], name)
		}
	}
}

func TestGetRenderProfile(t *testing.T) {
	tmpDir := t.TempDir()
	file := RenderProfilesFile{
		Profiles: []RenderProfile{
			{Name: "exists", HostPatterns: []string{"example.com"}},
		},
	}
	if err := SaveRenderProfilesFile(tmpDir, file); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Test found
	profile, found, err := GetRenderProfile(tmpDir, "exists")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Error("expected profile to be found")
	}
	if profile.Name != "exists" {
		t.Errorf("expected name 'exists', got '%s'", profile.Name)
	}

	// Test not found
	_, found, err = GetRenderProfile(tmpDir, "not-exists")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Error("expected profile to not be found")
	}
}

func TestUpsertRenderProfile_Create(t *testing.T) {
	tmpDir := t.TempDir()
	profile := RenderProfile{
		Name:         "new-profile",
		HostPatterns: []string{"example.com"},
		ForceEngine:  RenderEngineHTTP,
	}

	if err := UpsertRenderProfile(tmpDir, profile); err != nil {
		t.Fatalf("failed to upsert: %v", err)
	}

	loaded, found, err := GetRenderProfile(tmpDir, "new-profile")
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if !found {
		t.Fatal("expected profile to exist")
	}
	if loaded.Name != "new-profile" {
		t.Errorf("expected name 'new-profile', got '%s'", loaded.Name)
	}
}

func TestUpsertRenderProfile_Update(t *testing.T) {
	tmpDir := t.TempDir()
	profile := RenderProfile{
		Name:         "profile",
		HostPatterns: []string{"example.com"},
		ForceEngine:  RenderEngineHTTP,
	}
	if err := UpsertRenderProfile(tmpDir, profile); err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	// Update
	profile.ForceEngine = RenderEngineChromedp
	profile.HostPatterns = []string{"updated.com"}
	if err := UpsertRenderProfile(tmpDir, profile); err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	loaded, _, _ := GetRenderProfile(tmpDir, "profile")
	if loaded.ForceEngine != RenderEngineChromedp {
		t.Errorf("expected engine chromedp, got %s", loaded.ForceEngine)
	}
	if len(loaded.HostPatterns) != 1 || loaded.HostPatterns[0] != "updated.com" {
		t.Errorf("expected hostPatterns ['updated.com'], got %v", loaded.HostPatterns)
	}
}

func TestUpsertRenderProfile_PreservesOrder(t *testing.T) {
	tmpDir := t.TempDir()
	profiles := []RenderProfile{
		{Name: "first", HostPatterns: []string{"a.com"}},
		{Name: "second", HostPatterns: []string{"b.com"}},
		{Name: "third", HostPatterns: []string{"c.com"}},
	}
	for _, p := range profiles {
		if err := UpsertRenderProfile(tmpDir, p); err != nil {
			t.Fatalf("failed to create: %v", err)
		}
	}

	// Update second profile
	profiles[1].HostPatterns = []string{"updated.com"}
	if err := UpsertRenderProfile(tmpDir, profiles[1]); err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	file, _ := LoadRenderProfilesFile(tmpDir)
	if len(file.Profiles) != 3 {
		t.Fatalf("expected 3 profiles, got %d", len(file.Profiles))
	}
	if file.Profiles[0].Name != "first" {
		t.Errorf("expected first at index 0, got %s", file.Profiles[0].Name)
	}
	if file.Profiles[1].Name != "second" {
		t.Errorf("expected second at index 1, got %s", file.Profiles[1].Name)
	}
	if file.Profiles[1].HostPatterns[0] != "updated.com" {
		t.Errorf("expected updated host pattern, got %v", file.Profiles[1].HostPatterns)
	}
	if file.Profiles[2].Name != "third" {
		t.Errorf("expected third at index 2, got %s", file.Profiles[2].Name)
	}
}

func TestDeleteRenderProfile(t *testing.T) {
	tmpDir := t.TempDir()
	profiles := []RenderProfile{
		{Name: "keep", HostPatterns: []string{"a.com"}},
		{Name: "delete", HostPatterns: []string{"b.com"}},
	}
	for _, p := range profiles {
		UpsertRenderProfile(tmpDir, p)
	}

	if err := DeleteRenderProfile(tmpDir, "delete"); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	file, _ := LoadRenderProfilesFile(tmpDir)
	if len(file.Profiles) != 1 {
		t.Fatalf("expected 1 profile, got %d", len(file.Profiles))
	}
	if file.Profiles[0].Name != "keep" {
		t.Errorf("expected 'keep', got %s", file.Profiles[0].Name)
	}
}

func TestDeleteRenderProfile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	err := DeleteRenderProfile(tmpDir, "not-exists")
	if err == nil {
		t.Fatal("expected error for non-existent profile")
	}
	if !apperrors.IsKind(err, apperrors.KindNotFound) {
		t.Errorf("expected not_found error, got %v", err)
	}
}

func TestValidateRenderProfile(t *testing.T) {
	tests := []struct {
		name    string
		profile RenderProfile
		wantErr bool
	}{
		{
			name: "valid minimal",
			profile: RenderProfile{
				Name:         "test",
				HostPatterns: []string{"example.com"},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			profile: RenderProfile{
				HostPatterns: []string{"example.com"},
			},
			wantErr: true,
		},
		{
			name: "empty name",
			profile: RenderProfile{
				Name:         "",
				HostPatterns: []string{"example.com"},
			},
			wantErr: true,
		},
		{
			name: "whitespace name",
			profile: RenderProfile{
				Name:         "   ",
				HostPatterns: []string{"example.com"},
			},
			wantErr: true,
		},
		{
			name: "missing hostPatterns",
			profile: RenderProfile{
				Name: "test",
			},
			wantErr: true,
		},
		{
			name: "empty hostPatterns",
			profile: RenderProfile{
				Name:         "test",
				HostPatterns: []string{},
			},
			wantErr: true,
		},
		{
			name: "invalid engine",
			profile: RenderProfile{
				Name:         "test",
				HostPatterns: []string{"example.com"},
				ForceEngine:  "invalid",
			},
			wantErr: true,
		},
		{
			name: "valid engine http",
			profile: RenderProfile{
				Name:         "test",
				HostPatterns: []string{"example.com"},
				ForceEngine:  RenderEngineHTTP,
			},
			wantErr: false,
		},
		{
			name: "valid engine chromedp",
			profile: RenderProfile{
				Name:         "test",
				HostPatterns: []string{"example.com"},
				ForceEngine:  RenderEngineChromedp,
			},
			wantErr: false,
		},
		{
			name: "valid engine playwright",
			profile: RenderProfile{
				Name:         "test",
				HostPatterns: []string{"example.com"},
				ForceEngine:  RenderEnginePlaywright,
			},
			wantErr: false,
		},
		{
			name: "jsHeavyThreshold too high",
			profile: RenderProfile{
				Name:             "test",
				HostPatterns:     []string{"example.com"},
				JSHeavyThreshold: 1.5,
			},
			wantErr: true,
		},
		{
			name: "jsHeavyThreshold negative",
			profile: RenderProfile{
				Name:             "test",
				HostPatterns:     []string{"example.com"},
				JSHeavyThreshold: -0.5,
			},
			wantErr: true,
		},
		{
			name: "rateLimitQPS negative",
			profile: RenderProfile{
				Name:         "test",
				HostPatterns: []string{"example.com"},
				RateLimitQPS: -1,
			},
			wantErr: true,
		},
		{
			name: "rateLimitBurst negative",
			profile: RenderProfile{
				Name:           "test",
				HostPatterns:   []string{"example.com"},
				RateLimitBurst: -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRenderProfile(tt.profile)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRenderProfile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRenderProfilesFile_DuplicateNames(t *testing.T) {
	file := RenderProfilesFile{
		Profiles: []RenderProfile{
			{Name: "dup", HostPatterns: []string{"a.com"}},
			{Name: "unique", HostPatterns: []string{"b.com"}},
			{Name: "dup", HostPatterns: []string{"c.com"}},
		},
	}
	err := ValidateRenderProfilesFile(file)
	if err == nil {
		t.Error("expected error for duplicate names")
	}
}
