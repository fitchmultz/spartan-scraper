// Package pipeline provides tests for JavaScript script management.
//
// Tests cover:
// - Load/Save round-trip
// - CRUD operations (List, Get, Upsert, Delete)
// - Validation (duplicate names, invalid fields, unknown JSON fields)
// - Edge cases (missing file, empty scripts)
//
// Does NOT test:
// - JavaScript execution
// - Script matching at runtime (see js_test.go)
package pipeline

import (
	"os"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

func TestLoadJSRegistryStrict_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	registry, err := LoadJSRegistryStrict(tmpDir)
	if err != nil {
		t.Fatalf("expected no error for missing file, got %v", err)
	}
	if len(registry.Scripts) != 0 {
		t.Errorf("expected empty scripts, got %d", len(registry.Scripts))
	}
}

func TestLoadJSRegistryStrict_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := jsRegistryPath(tmpDir)
	if err := os.WriteFile(path, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	_, err := LoadJSRegistryStrict(tmpDir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadJSRegistryStrict_UnknownField(t *testing.T) {
	tmpDir := t.TempDir()
	path := jsRegistryPath(tmpDir)
	content := `{"scripts": [{"name": "test", "hostPatterns": ["example.com"], "unknownField": "value"}]}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	_, err := LoadJSRegistryStrict(tmpDir)
	if err == nil {
		t.Error("expected error for unknown field")
	}
}

func TestLoadJSRegistryStrict_Valid(t *testing.T) {
	tmpDir := t.TempDir()
	path := jsRegistryPath(tmpDir)
	content := `{
		"scripts": [
			{
				"name": "test-script",
				"hostPatterns": ["example.com", "*.example.com"],
				"engine": "chromedp",
				"preNav": "console.log('before')",
				"postNav": "console.log('after')",
				"selectors": [".class1", "#id1"]
			}
		]
	}`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	registry, err := LoadJSRegistryStrict(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(registry.Scripts) != 1 {
		t.Fatalf("expected 1 script, got %d", len(registry.Scripts))
	}
	if registry.Scripts[0].Name != "test-script" {
		t.Errorf("expected name 'test-script', got '%s'", registry.Scripts[0].Name)
	}
}

func TestSaveJSRegistry_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	registry := JSRegistry{
		Scripts: []JSTargetScript{
			{
				Name:         "script1",
				HostPatterns: []string{"example.com"},
				Engine:       EngineChromedp,
				PreNav:       "console.log('pre1')",
			},
			{
				Name:         "script2",
				HostPatterns: []string{"*.other.com"},
				Engine:       EnginePlaywright,
				PostNav:      "console.log('post2')",
			},
		},
	}

	if err := SaveJSRegistry(tmpDir, registry); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	loaded, err := LoadJSRegistryStrict(tmpDir)
	if err != nil {
		t.Fatalf("failed to load: %v", err)
	}

	if len(loaded.Scripts) != 2 {
		t.Fatalf("expected 2 scripts, got %d", len(loaded.Scripts))
	}
	if loaded.Scripts[0].Name != "script1" {
		t.Errorf("expected script1 first, got %s", loaded.Scripts[0].Name)
	}
	if loaded.Scripts[1].Name != "script2" {
		t.Errorf("expected script2 second, got %s", loaded.Scripts[1].Name)
	}
}

func TestListJSScriptNames(t *testing.T) {
	tmpDir := t.TempDir()
	registry := JSRegistry{
		Scripts: []JSTargetScript{
			{Name: "zebra", HostPatterns: []string{"z.com"}},
			{Name: "alpha", HostPatterns: []string{"a.com"}},
			{Name: "beta", HostPatterns: []string{"b.com"}},
		},
	}
	if err := SaveJSRegistry(tmpDir, registry); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	names, err := ListJSScriptNames(tmpDir)
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

func TestGetJSScript(t *testing.T) {
	tmpDir := t.TempDir()
	registry := JSRegistry{
		Scripts: []JSTargetScript{
			{Name: "exists", HostPatterns: []string{"example.com"}},
		},
	}
	if err := SaveJSRegistry(tmpDir, registry); err != nil {
		t.Fatalf("failed to save: %v", err)
	}

	// Test found
	script, found, err := GetJSScript(tmpDir, "exists")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !found {
		t.Error("expected script to be found")
	}
	if script.Name != "exists" {
		t.Errorf("expected name 'exists', got '%s'", script.Name)
	}

	// Test not found
	_, found, err = GetJSScript(tmpDir, "not-exists")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Error("expected script to not be found")
	}
}

func TestUpsertJSScript_Create(t *testing.T) {
	tmpDir := t.TempDir()
	script := JSTargetScript{
		Name:         "new-script",
		HostPatterns: []string{"example.com"},
		Engine:       EngineChromedp,
	}

	if err := UpsertJSScript(tmpDir, script); err != nil {
		t.Fatalf("failed to upsert: %v", err)
	}

	loaded, found, err := GetJSScript(tmpDir, "new-script")
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if !found {
		t.Fatal("expected script to exist")
	}
	if loaded.Name != "new-script" {
		t.Errorf("expected name 'new-script', got '%s'", loaded.Name)
	}
}

func TestUpsertJSScript_Update(t *testing.T) {
	tmpDir := t.TempDir()
	script := JSTargetScript{
		Name:         "script",
		HostPatterns: []string{"example.com"},
		Engine:       EngineChromedp,
	}
	if err := UpsertJSScript(tmpDir, script); err != nil {
		t.Fatalf("failed to create: %v", err)
	}

	// Update
	script.Engine = EnginePlaywright
	script.HostPatterns = []string{"updated.com"}
	if err := UpsertJSScript(tmpDir, script); err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	loaded, _, _ := GetJSScript(tmpDir, "script")
	if loaded.Engine != EnginePlaywright {
		t.Errorf("expected engine playwright, got %s", loaded.Engine)
	}
	if len(loaded.HostPatterns) != 1 || loaded.HostPatterns[0] != "updated.com" {
		t.Errorf("expected hostPatterns ['updated.com'], got %v", loaded.HostPatterns)
	}
}

func TestUpsertJSScript_PreservesOrder(t *testing.T) {
	tmpDir := t.TempDir()
	scripts := []JSTargetScript{
		{Name: "first", HostPatterns: []string{"a.com"}},
		{Name: "second", HostPatterns: []string{"b.com"}},
		{Name: "third", HostPatterns: []string{"c.com"}},
	}
	for _, s := range scripts {
		if err := UpsertJSScript(tmpDir, s); err != nil {
			t.Fatalf("failed to create: %v", err)
		}
	}

	// Update second script
	scripts[1].HostPatterns = []string{"updated.com"}
	if err := UpsertJSScript(tmpDir, scripts[1]); err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	registry, _ := LoadJSRegistryStrict(tmpDir)
	if len(registry.Scripts) != 3 {
		t.Fatalf("expected 3 scripts, got %d", len(registry.Scripts))
	}
	if registry.Scripts[0].Name != "first" {
		t.Errorf("expected first at index 0, got %s", registry.Scripts[0].Name)
	}
	if registry.Scripts[1].Name != "second" {
		t.Errorf("expected second at index 1, got %s", registry.Scripts[1].Name)
	}
	if registry.Scripts[1].HostPatterns[0] != "updated.com" {
		t.Errorf("expected updated host pattern, got %v", registry.Scripts[1].HostPatterns)
	}
	if registry.Scripts[2].Name != "third" {
		t.Errorf("expected third at index 2, got %s", registry.Scripts[2].Name)
	}
}

func TestDeleteJSScript(t *testing.T) {
	tmpDir := t.TempDir()
	scripts := []JSTargetScript{
		{Name: "keep", HostPatterns: []string{"a.com"}},
		{Name: "delete", HostPatterns: []string{"b.com"}},
	}
	for _, s := range scripts {
		UpsertJSScript(tmpDir, s)
	}

	if err := DeleteJSScript(tmpDir, "delete"); err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	registry, _ := LoadJSRegistryStrict(tmpDir)
	if len(registry.Scripts) != 1 {
		t.Fatalf("expected 1 script, got %d", len(registry.Scripts))
	}
	if registry.Scripts[0].Name != "keep" {
		t.Errorf("expected 'keep', got %s", registry.Scripts[0].Name)
	}
}

func TestDeleteJSScript_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	err := DeleteJSScript(tmpDir, "not-exists")
	if err == nil {
		t.Fatal("expected error for non-existent script")
	}
	if !apperrors.IsKind(err, apperrors.KindNotFound) {
		t.Errorf("expected not_found error, got %v", err)
	}
}

func TestValidateJSTargetScript(t *testing.T) {
	tests := []struct {
		name    string
		script  JSTargetScript
		wantErr bool
	}{
		{
			name: "valid minimal",
			script: JSTargetScript{
				Name:         "test",
				HostPatterns: []string{"example.com"},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			script: JSTargetScript{
				HostPatterns: []string{"example.com"},
			},
			wantErr: true,
		},
		{
			name: "empty name",
			script: JSTargetScript{
				Name:         "",
				HostPatterns: []string{"example.com"},
			},
			wantErr: true,
		},
		{
			name: "whitespace name",
			script: JSTargetScript{
				Name:         "   ",
				HostPatterns: []string{"example.com"},
			},
			wantErr: true,
		},
		{
			name: "missing hostPatterns",
			script: JSTargetScript{
				Name: "test",
			},
			wantErr: true,
		},
		{
			name: "empty hostPatterns",
			script: JSTargetScript{
				Name:         "test",
				HostPatterns: []string{},
			},
			wantErr: true,
		},
		{
			name: "invalid engine",
			script: JSTargetScript{
				Name:         "test",
				HostPatterns: []string{"example.com"},
				Engine:       "invalid",
			},
			wantErr: true,
		},
		{
			name: "valid engine chromedp",
			script: JSTargetScript{
				Name:         "test",
				HostPatterns: []string{"example.com"},
				Engine:       EngineChromedp,
			},
			wantErr: false,
		},
		{
			name: "valid engine playwright",
			script: JSTargetScript{
				Name:         "test",
				HostPatterns: []string{"example.com"},
				Engine:       EnginePlaywright,
			},
			wantErr: false,
		},
		{
			name: "valid engine case insensitive",
			script: JSTargetScript{
				Name:         "test",
				HostPatterns: []string{"example.com"},
				Engine:       "CHROMEDP",
			},
			wantErr: false,
		},
		{
			name: "valid with all fields",
			script: JSTargetScript{
				Name:         "test",
				HostPatterns: []string{"example.com", "*.example.com"},
				Engine:       EngineChromedp,
				PreNav:       "console.log('pre')",
				PostNav:      "console.log('post')",
				Selectors:    []string{".class", "#id"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJSTargetScript(tt.script)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJSTargetScript() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateJSRegistry_DuplicateNames(t *testing.T) {
	registry := JSRegistry{
		Scripts: []JSTargetScript{
			{Name: "dup", HostPatterns: []string{"a.com"}},
			{Name: "unique", HostPatterns: []string{"b.com"}},
			{Name: "dup", HostPatterns: []string{"c.com"}},
		},
	}
	err := ValidateJSRegistry(registry)
	if err == nil {
		t.Error("expected error for duplicate names")
	}
}
