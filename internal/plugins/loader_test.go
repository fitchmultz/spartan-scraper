// Package plugins provides a WASM-based plugin system for third-party extensions.
package plugins

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader("/tmp/test-data")
	if loader == nil {
		t.Error("NewLoader() returned nil")
		return
	}
	if loader.dataDir != "/tmp/test-data" {
		t.Errorf("dataDir = %q, want %q", loader.dataDir, "/tmp/test-data")
	}
}

func TestLoader_GetPluginsDir(t *testing.T) {
	loader := NewLoader("/tmp/test-data")
	want := filepath.Join("/tmp/test-data", "plugins")
	if got := loader.GetPluginsDir(); got != want {
		t.Errorf("GetPluginsDir() = %q, want %q", got, want)
	}
}

func TestLoader_Discover(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader(tmpDir)

	t.Run("empty plugins directory", func(t *testing.T) {
		plugins, err := loader.Discover()
		if err != nil {
			t.Errorf("Discover() error = %v", err)
			return
		}
		if len(plugins) != 0 {
			t.Errorf("Discover() returned %d plugins, want 0", len(plugins))
		}
	})

	t.Run("with valid plugins", func(t *testing.T) {
		// Create plugin directory
		pluginsDir := loader.GetPluginsDir()
		pluginDir := filepath.Join(pluginsDir, "test-plugin")
		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			t.Fatalf("Failed to create plugin directory: %v", err)
		}

		// Create manifest
		manifest := &PluginManifest{
			Name:     "test-plugin",
			Version:  "1.0.0",
			WASMPath: "plugin.wasm",
			Hooks:    []string{"pre_fetch"},
			Enabled:  true,
			Priority: 5,
		}
		manifestPath := filepath.Join(pluginDir, "manifest.json")
		if err := SaveManifest(manifestPath, manifest); err != nil {
			t.Fatalf("Failed to save manifest: %v", err)
		}

		// Create dummy WASM file
		wasmPath := filepath.Join(pluginDir, "plugin.wasm")
		if err := os.WriteFile(wasmPath, []byte("dummy"), 0644); err != nil {
			t.Fatalf("Failed to write WASM file: %v", err)
		}

		plugins, err := loader.Discover()
		if err != nil {
			t.Errorf("Discover() error = %v", err)
			return
		}
		if len(plugins) != 1 {
			t.Errorf("Discover() returned %d plugins, want 1", len(plugins))
			return
		}

		if plugins[0].Name != "test-plugin" {
			t.Errorf("Plugin name = %q, want %q", plugins[0].Name, "test-plugin")
		}
		if plugins[0].Version != "1.0.0" {
			t.Errorf("Plugin version = %q, want %q", plugins[0].Version, "1.0.0")
		}
	})

	t.Run("sorted by priority", func(t *testing.T) {
		// Create another plugin with higher priority (lower number)
		pluginsDir := loader.GetPluginsDir()
		pluginDir := filepath.Join(pluginsDir, "high-priority-plugin")
		if err := os.MkdirAll(pluginDir, 0755); err != nil {
			t.Fatalf("Failed to create plugin directory: %v", err)
		}

		manifest := &PluginManifest{
			Name:     "high-priority-plugin",
			Version:  "1.0.0",
			WASMPath: "plugin.wasm",
			Hooks:    []string{"post_fetch"},
			Enabled:  true,
			Priority: 1, // Lower priority number = earlier
		}
		manifestPath := filepath.Join(pluginDir, "manifest.json")
		if err := SaveManifest(manifestPath, manifest); err != nil {
			t.Fatalf("Failed to save manifest: %v", err)
		}

		wasmPath := filepath.Join(pluginDir, "plugin.wasm")
		if err := os.WriteFile(wasmPath, []byte("dummy"), 0644); err != nil {
			t.Fatalf("Failed to write WASM file: %v", err)
		}

		plugins, err := loader.Discover()
		if err != nil {
			t.Errorf("Discover() error = %v", err)
			return
		}
		if len(plugins) != 2 {
			t.Errorf("Discover() returned %d plugins, want 2", len(plugins))
			return
		}

		// High priority (priority=1) should come first
		if plugins[0].Name != "high-priority-plugin" {
			t.Errorf("First plugin = %q, want %q", plugins[0].Name, "high-priority-plugin")
		}
	})
}

func TestLoader_LoadPlugin(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader(tmpDir)

	// Create plugin
	pluginsDir := loader.GetPluginsDir()
	pluginDir := filepath.Join(pluginsDir, "test-plugin")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("Failed to create plugin directory: %v", err)
	}

	manifest := &PluginManifest{
		Name:     "test-plugin",
		Version:  "1.0.0",
		WASMPath: "plugin.wasm",
		Hooks:    []string{"pre_fetch"},
		Enabled:  true,
		Priority: 10,
	}
	manifestPath := filepath.Join(pluginDir, "manifest.json")
	if err := SaveManifest(manifestPath, manifest); err != nil {
		t.Fatalf("Failed to save manifest: %v", err)
	}

	t.Run("existing plugin", func(t *testing.T) {
		loadedManifest, loadedDir, err := loader.LoadPlugin("test-plugin")
		if err != nil {
			t.Errorf("LoadPlugin() error = %v", err)
			return
		}
		if loadedManifest.Name != "test-plugin" {
			t.Errorf("Manifest.Name = %q, want %q", loadedManifest.Name, "test-plugin")
		}
		if loadedDir != pluginDir {
			t.Errorf("PluginDir = %q, want %q", loadedDir, pluginDir)
		}
	})

	t.Run("non-existent plugin", func(t *testing.T) {
		_, _, err := loader.LoadPlugin("non-existent")
		if err == nil {
			t.Error("LoadPlugin() expected error for non-existent plugin")
			return
		}
		if !apperrors.IsKind(err, apperrors.KindNotFound) {
			t.Errorf("Expected NotFound error, got %v", apperrors.KindOf(err))
		}
	})
}

func TestLoader_Install(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader(tmpDir)

	// Create source directory
	sourceDir := filepath.Join(tmpDir, "source-plugin")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source directory: %v", err)
	}

	// Create source manifest
	manifest := &PluginManifest{
		Name:     "install-test",
		Version:  "1.0.0",
		WASMPath: "plugin.wasm",
		Hooks:    []string{"pre_fetch"},
		Enabled:  false,
		Priority: 10,
	}
	manifestPath := filepath.Join(sourceDir, "manifest.json")
	if err := SaveManifest(manifestPath, manifest); err != nil {
		t.Fatalf("Failed to save manifest: %v", err)
	}

	// Create source WASM file
	wasmPath := filepath.Join(sourceDir, "plugin.wasm")
	if err := os.WriteFile(wasmPath, []byte("dummy wasm"), 0644); err != nil {
		t.Fatalf("Failed to write WASM file: %v", err)
	}

	t.Run("successful install", func(t *testing.T) {
		info, err := loader.Install(sourceDir)
		if err != nil {
			t.Errorf("Install() error = %v", err)
			return
		}
		if info.Name != "install-test" {
			t.Errorf("Info.Name = %q, want %q", info.Name, "install-test")
		}

		// Verify files were copied
		pluginDir := filepath.Join(loader.GetPluginsDir(), "install-test")
		if _, err := os.Stat(filepath.Join(pluginDir, "manifest.json")); os.IsNotExist(err) {
			t.Error("Install() did not copy manifest.json")
		}
		if _, err := os.Stat(filepath.Join(pluginDir, "plugin.wasm")); os.IsNotExist(err) {
			t.Error("Install() did not copy plugin.wasm")
		}
	})

	t.Run("already installed", func(t *testing.T) {
		_, err := loader.Install(sourceDir)
		if err == nil {
			t.Error("Install() expected error for already installed plugin")
			return
		}
		if !apperrors.IsKind(err, apperrors.KindValidation) {
			t.Errorf("Expected Validation error, got %v", apperrors.KindOf(err))
		}
	})

	t.Run("invalid source", func(t *testing.T) {
		_, err := loader.Install("/nonexistent/path")
		if err == nil {
			t.Error("Install() expected error for invalid source")
		}
	})
}

func TestLoader_Uninstall(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader(tmpDir)

	// Create plugin
	pluginsDir := loader.GetPluginsDir()
	pluginDir := filepath.Join(pluginsDir, "to-uninstall")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("Failed to create plugin directory: %v", err)
	}

	manifest := &PluginManifest{
		Name:     "to-uninstall",
		Version:  "1.0.0",
		WASMPath: "plugin.wasm",
	}
	manifestPath := filepath.Join(pluginDir, "manifest.json")
	if err := SaveManifest(manifestPath, manifest); err != nil {
		t.Fatalf("Failed to save manifest: %v", err)
	}

	t.Run("successful uninstall", func(t *testing.T) {
		err := loader.Uninstall("to-uninstall")
		if err != nil {
			t.Errorf("Uninstall() error = %v", err)
			return
		}

		// Verify directory was removed
		if _, err := os.Stat(pluginDir); !os.IsNotExist(err) {
			t.Error("Uninstall() did not remove plugin directory")
		}
	})

	t.Run("non-existent plugin", func(t *testing.T) {
		err := loader.Uninstall("non-existent")
		if err == nil {
			t.Error("Uninstall() expected error for non-existent plugin")
			return
		}
		if !apperrors.IsKind(err, apperrors.KindNotFound) {
			t.Errorf("Expected NotFound error, got %v", apperrors.KindOf(err))
		}
	})
}

func TestLoader_EnableDisable(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader(tmpDir)

	// Create plugin
	pluginsDir := loader.GetPluginsDir()
	pluginDir := filepath.Join(pluginsDir, "toggle-test")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("Failed to create plugin directory: %v", err)
	}

	manifest := &PluginManifest{
		Name:     "toggle-test",
		Version:  "1.0.0",
		WASMPath: "plugin.wasm",
		Enabled:  true,
	}
	manifestPath := filepath.Join(pluginDir, "manifest.json")
	if err := SaveManifest(manifestPath, manifest); err != nil {
		t.Fatalf("Failed to save manifest: %v", err)
	}

	t.Run("disable plugin", func(t *testing.T) {
		err := loader.Disable("toggle-test")
		if err != nil {
			t.Errorf("Disable() error = %v", err)
			return
		}

		// Verify
		loaded, _, err := loader.LoadPlugin("toggle-test")
		if err != nil {
			t.Fatalf("Failed to load plugin: %v", err)
		}
		if loaded.Enabled {
			t.Error("Disable() did not disable plugin")
		}
	})

	t.Run("enable plugin", func(t *testing.T) {
		err := loader.Enable("toggle-test")
		if err != nil {
			t.Errorf("Enable() error = %v", err)
			return
		}

		// Verify
		loaded, _, err := loader.LoadPlugin("toggle-test")
		if err != nil {
			t.Fatalf("Failed to load plugin: %v", err)
		}
		if !loaded.Enabled {
			t.Error("Enable() did not enable plugin")
		}
	})

	t.Run("enable non-existent", func(t *testing.T) {
		err := loader.Enable("non-existent")
		if err == nil {
			t.Error("Enable() expected error for non-existent plugin")
		}
	})
}

func TestLoader_Configure(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader(tmpDir)

	// Create plugin
	pluginsDir := loader.GetPluginsDir()
	pluginDir := filepath.Join(pluginsDir, "config-test")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("Failed to create plugin directory: %v", err)
	}

	manifest := &PluginManifest{
		Name:     "config-test",
		Version:  "1.0.0",
		WASMPath: "plugin.wasm",
		Config:   map[string]any{},
	}
	manifestPath := filepath.Join(pluginDir, "manifest.json")
	if err := SaveManifest(manifestPath, manifest); err != nil {
		t.Fatalf("Failed to save manifest: %v", err)
	}

	t.Run("set config value", func(t *testing.T) {
		err := loader.Configure("config-test", "apiKey", "secret123")
		if err != nil {
			t.Errorf("Configure() error = %v", err)
			return
		}

		// Verify
		value, ok, err := loader.GetConfig("config-test", "apiKey")
		if err != nil {
			t.Fatalf("GetConfig() error = %v", err)
		}
		if !ok {
			t.Error("GetConfig() did not find key")
		}
		if value != "secret123" {
			t.Errorf("Config value = %v, want %v", value, "secret123")
		}
	})

	t.Run("update config value", func(t *testing.T) {
		err := loader.Configure("config-test", "apiKey", "newSecret")
		if err != nil {
			t.Errorf("Configure() error = %v", err)
			return
		}

		// Verify
		value, _, _ := loader.GetConfig("config-test", "apiKey")
		if value != "newSecret" {
			t.Errorf("Config value = %v, want %v", value, "newSecret")
		}
	})

	t.Run("configure non-existent", func(t *testing.T) {
		err := loader.Configure("non-existent", "key", "value")
		if err == nil {
			t.Error("Configure() expected error for non-existent plugin")
		}
	})
}

func TestLoader_GetConfig(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader(tmpDir)

	// Create plugin
	pluginsDir := loader.GetPluginsDir()
	pluginDir := filepath.Join(pluginsDir, "get-config-test")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		t.Fatalf("Failed to create plugin directory: %v", err)
	}

	manifest := &PluginManifest{
		Name:     "get-config-test",
		Version:  "1.0.0",
		WASMPath: "plugin.wasm",
		Config: map[string]any{
			"existing": "value",
		},
	}
	manifestPath := filepath.Join(pluginDir, "manifest.json")
	if err := SaveManifest(manifestPath, manifest); err != nil {
		t.Fatalf("Failed to save manifest: %v", err)
	}

	t.Run("existing key", func(t *testing.T) {
		value, ok, err := loader.GetConfig("get-config-test", "existing")
		if err != nil {
			t.Errorf("GetConfig() error = %v", err)
			return
		}
		if !ok {
			t.Error("GetConfig() should find existing key")
		}
		if value != "value" {
			t.Errorf("GetConfig() value = %v, want %v", value, "value")
		}
	})

	t.Run("non-existent key", func(t *testing.T) {
		_, ok, err := loader.GetConfig("get-config-test", "nonexistent")
		if err != nil {
			t.Errorf("GetConfig() error = %v", err)
			return
		}
		if ok {
			t.Error("GetConfig() should not find non-existent key")
		}
	})

	t.Run("non-existent plugin", func(t *testing.T) {
		_, _, err := loader.GetConfig("non-existent", "key")
		if err == nil {
			t.Error("GetConfig() expected error for non-existent plugin")
		}
	})
}
