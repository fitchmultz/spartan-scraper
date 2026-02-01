// Package plugins provides a WASM-based plugin system for third-party extensions.
package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

func TestPluginManifest_Validate(t *testing.T) {
	tests := []struct {
		name     string
		manifest PluginManifest
		wantErr  bool
		errKind  apperrors.Kind
	}{
		{
			name: "valid manifest",
			manifest: PluginManifest{
				Name:     "test-plugin",
				Version:  "1.0.0",
				WASMPath: "plugin.wasm",
				Hooks:    []string{"pre_fetch"},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			manifest: PluginManifest{
				Version:  "1.0.0",
				WASMPath: "plugin.wasm",
			},
			wantErr: true,
			errKind: apperrors.KindValidation,
		},
		{
			name: "missing version",
			manifest: PluginManifest{
				Name:     "test-plugin",
				WASMPath: "plugin.wasm",
			},
			wantErr: true,
			errKind: apperrors.KindValidation,
		},
		{
			name: "missing wasm_path",
			manifest: PluginManifest{
				Name:    "test-plugin",
				Version: "1.0.0",
			},
			wantErr: true,
			errKind: apperrors.KindValidation,
		},
		{
			name: "invalid name character",
			manifest: PluginManifest{
				Name:     "test@plugin",
				Version:  "1.0.0",
				WASMPath: "plugin.wasm",
			},
			wantErr: true,
			errKind: apperrors.KindValidation,
		},
		{
			name: "invalid hook",
			manifest: PluginManifest{
				Name:     "test-plugin",
				Version:  "1.0.0",
				WASMPath: "plugin.wasm",
				Hooks:    []string{"invalid_hook"},
			},
			wantErr: true,
			errKind: apperrors.KindValidation,
		},
		{
			name: "invalid permission",
			manifest: PluginManifest{
				Name:        "test-plugin",
				Version:     "1.0.0",
				WASMPath:    "plugin.wasm",
				Permissions: []string{"invalid_perm"},
			},
			wantErr: true,
			errKind: apperrors.KindValidation,
		},
		{
			name: "all valid hooks",
			manifest: PluginManifest{
				Name:     "test-plugin",
				Version:  "1.0.0",
				WASMPath: "plugin.wasm",
				Hooks:    []string{"pre_fetch", "post_fetch", "pre_extract", "post_extract", "pre_output", "post_output"},
			},
			wantErr: false,
		},
		{
			name: "all valid permissions",
			manifest: PluginManifest{
				Name:        "test-plugin",
				Version:     "1.0.0",
				WASMPath:    "plugin.wasm",
				Permissions: []string{"network", "filesystem", "env"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.manifest.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errKind != "" {
				if !apperrors.IsKind(err, tt.errKind) {
					t.Errorf("Validate() error kind = %v, want %v", apperrors.KindOf(err), tt.errKind)
				}
			}
		})
	}
}

func TestPluginManifest_SupportsHook(t *testing.T) {
	m := &PluginManifest{
		Name:     "test-plugin",
		Version:  "1.0.0",
		WASMPath: "plugin.wasm",
		Hooks:    []string{"pre_fetch", "post_extract"},
	}

	tests := []struct {
		hook string
		want bool
	}{
		{"pre_fetch", true},
		{"post_fetch", false},
		{"pre_extract", false},
		{"post_extract", true},
		{"PRE_FETCH", true}, // case insensitive
		{"Post_Extract", true},
	}

	for _, tt := range tests {
		t.Run(tt.hook, func(t *testing.T) {
			if got := m.SupportsHook(tt.hook); got != tt.want {
				t.Errorf("SupportsHook(%q) = %v, want %v", tt.hook, got, tt.want)
			}
		})
	}
}

func TestPluginManifest_HasPermission(t *testing.T) {
	m := &PluginManifest{
		Name:        "test-plugin",
		Version:     "1.0.0",
		WASMPath:    "plugin.wasm",
		Permissions: []string{"network", "env"},
	}

	tests := []struct {
		perm string
		want bool
	}{
		{"network", true},
		{"filesystem", false},
		{"env", true},
		{"NETWORK", true}, // case insensitive
		{"Env", true},
	}

	for _, tt := range tests {
		t.Run(tt.perm, func(t *testing.T) {
			if got := m.HasPermission(tt.perm); got != tt.want {
				t.Errorf("HasPermission(%q) = %v, want %v", tt.perm, got, tt.want)
			}
		})
	}
}

func TestLoadManifest(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	t.Run("valid manifest file", func(t *testing.T) {
		manifestPath := filepath.Join(tmpDir, "valid_manifest.json")
		content := `{
			"name": "test-plugin",
			"version": "1.0.0",
			"description": "A test plugin",
			"author": "Test Author",
			"hooks": ["pre_fetch"],
			"permissions": ["network"],
			"wasm_path": "plugin.wasm",
			"enabled": true,
			"priority": 10
		}`
		if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		manifest, err := LoadManifest(manifestPath)
		if err != nil {
			t.Errorf("LoadManifest() error = %v", err)
			return
		}

		if manifest.Name != "test-plugin" {
			t.Errorf("Name = %q, want %q", manifest.Name, "test-plugin")
		}
		if manifest.Version != "1.0.0" {
			t.Errorf("Version = %q, want %q", manifest.Version, "1.0.0")
		}
		if manifest.WASMPath != "plugin.wasm" {
			t.Errorf("WASMPath = %q, want %q", manifest.WASMPath, "plugin.wasm")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		manifestPath := filepath.Join(tmpDir, "nonexistent.json")
		_, err := LoadManifest(manifestPath)
		if err == nil {
			t.Error("LoadManifest() expected error for missing file")
			return
		}
		if !apperrors.IsKind(err, apperrors.KindNotFound) {
			t.Errorf("Expected NotFound error, got %v", apperrors.KindOf(err))
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		manifestPath := filepath.Join(tmpDir, "invalid.json")
		content := `{invalid json`
		if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file: %v", err)
		}

		_, err := LoadManifest(manifestPath)
		if err == nil {
			t.Error("LoadManifest() expected error for invalid JSON")
			return
		}
		if !apperrors.IsKind(err, apperrors.KindValidation) {
			t.Errorf("Expected Validation error, got %v", apperrors.KindOf(err))
		}
	})
}

func TestSaveManifest(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.json")

	manifest := &PluginManifest{
		Name:        "test-plugin",
		Version:     "1.0.0",
		Description: "A test plugin",
		Author:      "Test Author",
		Hooks:       []string{"pre_fetch"},
		Permissions: []string{"network"},
		WASMPath:    "plugin.wasm",
		Config:      map[string]any{"key": "value"},
		Enabled:     true,
		Priority:    10,
	}

	if err := SaveManifest(manifestPath, manifest); err != nil {
		t.Errorf("SaveManifest() error = %v", err)
		return
	}

	// Verify file was created
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Error("SaveManifest() did not create file")
		return
	}

	// Load and verify
	loaded, err := LoadManifest(manifestPath)
	if err != nil {
		t.Errorf("Failed to load saved manifest: %v", err)
		return
	}

	if loaded.Name != manifest.Name {
		t.Errorf("Loaded Name = %q, want %q", loaded.Name, manifest.Name)
	}
	if loaded.Version != manifest.Version {
		t.Errorf("Loaded Version = %q, want %q", loaded.Version, manifest.Version)
	}
}

func TestPluginManifest_ToInfo(t *testing.T) {
	tmpDir := t.TempDir()
	pluginDir := filepath.Join(tmpDir, "test-plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("Failed to create plugin directory: %v", err)
	}

	// Create a dummy WASM file
	wasmPath := filepath.Join(pluginDir, "plugin.wasm")
	wasmContent := []byte("dummy wasm content")
	if err := os.WriteFile(wasmPath, wasmContent, 0644); err != nil {
		t.Fatalf("Failed to write WASM file: %v", err)
	}

	manifest := &PluginManifest{
		Name:        "test-plugin",
		Version:     "1.0.0",
		Description: "A test plugin",
		Author:      "Test Author",
		Hooks:       []string{"pre_fetch"},
		Permissions: []string{"network"},
		WASMPath:    "plugin.wasm",
		Config:      map[string]any{"key": "value"},
		Enabled:     true,
		Priority:    10,
	}

	info, err := manifest.ToInfo(pluginDir)
	if err != nil {
		t.Errorf("ToInfo() error = %v", err)
		return
	}

	if info.Name != manifest.Name {
		t.Errorf("Info.Name = %q, want %q", info.Name, manifest.Name)
	}
	if info.Version != manifest.Version {
		t.Errorf("Info.Version = %q, want %q", info.Version, manifest.Version)
	}
	if info.WASMSize != int64(len(wasmContent)) {
		t.Errorf("Info.WASMSize = %d, want %d", info.WASMSize, len(wasmContent))
	}
}

func TestHookToStage(t *testing.T) {
	tests := []struct {
		hook string
		want string
	}{
		{"pre_fetch", "pre fetch"},
		{"post_fetch", "post fetch"},
		{"pre_extract", "pre extract"},
		{"invalid", "invalid"},
		{"single", "single"},
	}

	for _, tt := range tests {
		t.Run(tt.hook, func(t *testing.T) {
			if got := HookToStage(tt.hook); got != tt.want {
				t.Errorf("HookToStage(%q) = %q, want %q", tt.hook, got, tt.want)
			}
		})
	}
}

func TestValidatePluginName(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"valid-plugin", false},
		{"valid_plugin", false},
		{"validPlugin123", false},
		{"", true},               // empty
		{"plugin@invalid", true}, // invalid character
		{"builtin", true},        // reserved name
		{"system", true},         // reserved name
		{"spartan", true},        // reserved name
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePluginName(tt.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePluginName(%q) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}
