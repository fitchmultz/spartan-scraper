// Package plugins provides a WASM-based plugin system for third-party extensions.
package plugins

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// Loader handles plugin discovery and loading from the data directory.
type Loader struct {
	dataDir string
}

// NewLoader creates a new plugin loader for the given data directory.
func NewLoader(dataDir string) *Loader {
	return &Loader{dataDir: dataDir}
}

// GetPluginsDir returns the path to the plugins directory.
func (l *Loader) GetPluginsDir() string {
	return filepath.Join(l.dataDir, "plugins")
}

// Discover finds all plugins in the plugins directory.
func (l *Loader) Discover() ([]*PluginInfo, error) {
	pluginsDir := l.GetPluginsDir()

	// Ensure plugins directory exists
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		return nil, apperrors.Wrap(apperrors.KindPermission, "failed to create plugins directory", err)
	}

	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to read plugins directory", err)
	}

	var plugins []*PluginInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pluginName := entry.Name()
		pluginDir := filepath.Join(pluginsDir, pluginName)

		info, err := l.LoadPluginInfo(pluginDir)
		if err != nil {
			// Log warning but continue with other plugins
			fmt.Fprintf(os.Stderr, "[WARN] Failed to load plugin %s: %v\n", pluginName, err)
			continue
		}

		plugins = append(plugins, info)
	}

	// Sort by priority (lower = earlier), then by name
	sort.Slice(plugins, func(i, j int) bool {
		if plugins[i].Priority != plugins[j].Priority {
			return plugins[i].Priority < plugins[j].Priority
		}
		return plugins[i].Name < plugins[j].Name
	})

	return plugins, nil
}

// LoadPluginInfo loads information about a plugin from its directory.
func (l *Loader) LoadPluginInfo(pluginDir string) (*PluginInfo, error) {
	manifestPath := filepath.Join(pluginDir, "manifest.json")

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, err
	}

	return manifest.ToInfo(pluginDir)
}

// LoadPlugin loads a plugin manifest from a directory.
func (l *Loader) LoadPlugin(pluginName string) (*PluginManifest, string, error) {
	pluginDir := filepath.Join(l.GetPluginsDir(), pluginName)

	// Check if directory exists
	info, err := os.Stat(pluginDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, "", apperrors.NotFound(fmt.Sprintf("plugin not found: %s", pluginName))
		}
		return nil, "", apperrors.Wrap(apperrors.KindInternal, "failed to stat plugin directory", err)
	}

	if !info.IsDir() {
		return nil, "", apperrors.Validation(fmt.Sprintf("plugin path is not a directory: %s", pluginName))
	}

	manifestPath := filepath.Join(pluginDir, "manifest.json")
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, "", err
	}

	return manifest, pluginDir, nil
}

// Install installs a plugin from a source directory.
func (l *Loader) Install(sourceDir string) (*PluginInfo, error) {
	// Validate source directory
	sourceManifestPath := filepath.Join(sourceDir, "manifest.json")
	manifest, err := LoadManifest(sourceManifestPath)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.KindValidation, "invalid plugin source", err)
	}

	// Check if plugin already exists
	pluginDir := filepath.Join(l.GetPluginsDir(), manifest.Name)
	if _, err := os.Stat(pluginDir); err == nil {
		return nil, apperrors.Validation(fmt.Sprintf("plugin already installed: %s", manifest.Name))
	}

	// Create plugin directory
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return nil, apperrors.Wrap(apperrors.KindPermission, "failed to create plugin directory", err)
	}

	// Copy manifest
	destManifestPath := filepath.Join(pluginDir, "manifest.json")
	if err := copyFile(sourceManifestPath, destManifestPath); err != nil {
		os.RemoveAll(pluginDir)
		return nil, err
	}

	// Copy WASM binary
	sourceWASMPath := filepath.Join(sourceDir, manifest.WASMPath)
	destWASMPath := filepath.Join(pluginDir, manifest.WASMPath)
	if err := copyFile(sourceWASMPath, destWASMPath); err != nil {
		os.RemoveAll(pluginDir)
		return nil, err
	}

	// Copy optional config file if present
	sourceConfigPath := filepath.Join(sourceDir, "config.json")
	if _, err := os.Stat(sourceConfigPath); err == nil {
		destConfigPath := filepath.Join(pluginDir, "config.json")
		if err := copyFile(sourceConfigPath, destConfigPath); err != nil {
			os.RemoveAll(pluginDir)
			return nil, err
		}
	}

	return manifest.ToInfo(pluginDir)
}

// Uninstall removes a plugin.
func (l *Loader) Uninstall(pluginName string) error {
	pluginDir := filepath.Join(l.GetPluginsDir(), pluginName)

	// Check if plugin exists
	if _, err := os.Stat(pluginDir); err != nil {
		if os.IsNotExist(err) {
			return apperrors.NotFound(fmt.Sprintf("plugin not found: %s", pluginName))
		}
		return apperrors.Wrap(apperrors.KindInternal, "failed to stat plugin directory", err)
	}

	// Remove plugin directory
	if err := os.RemoveAll(pluginDir); err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to remove plugin directory", err)
	}

	return nil
}

// Enable enables a plugin.
func (l *Loader) Enable(pluginName string) error {
	return l.setEnabled(pluginName, true)
}

// Disable disables a plugin.
func (l *Loader) Disable(pluginName string) error {
	return l.setEnabled(pluginName, false)
}

// setEnabled updates the enabled status of a plugin.
func (l *Loader) setEnabled(pluginName string, enabled bool) error {
	pluginDir := filepath.Join(l.GetPluginsDir(), pluginName)
	manifestPath := filepath.Join(pluginDir, "manifest.json")

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return err
	}

	manifest.Enabled = enabled

	if err := SaveManifest(manifestPath, manifest); err != nil {
		return err
	}

	return nil
}

// Configure updates a plugin's configuration.
func (l *Loader) Configure(pluginName string, key string, value any) error {
	pluginDir := filepath.Join(l.GetPluginsDir(), pluginName)
	manifestPath := filepath.Join(pluginDir, "manifest.json")

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return err
	}

	if manifest.Config == nil {
		manifest.Config = make(map[string]any)
	}

	manifest.Config[key] = value

	if err := SaveManifest(manifestPath, manifest); err != nil {
		return err
	}

	return nil
}

// GetConfig gets a plugin's configuration value.
func (l *Loader) GetConfig(pluginName string, key string) (any, bool, error) {
	pluginDir := filepath.Join(l.GetPluginsDir(), pluginName)
	manifestPath := filepath.Join(pluginDir, "manifest.json")

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, false, err
	}

	value, ok := manifest.Config[key]
	return value, ok, nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to read source file", err)
	}

	// Ensure destination directory exists
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return apperrors.Wrap(apperrors.KindPermission, "failed to create destination directory", err)
	}

	if err := os.WriteFile(dst, data, 0644); err != nil {
		return apperrors.Wrap(apperrors.KindPermission, "failed to write destination file", err)
	}

	return nil
}

// ValidatePluginName checks if a plugin name is valid.
func ValidatePluginName(name string) error {
	if name == "" {
		return apperrors.Validation("plugin name is required")
	}

	// Check for valid characters
	for _, r := range name {
		if !isValidNameChar(r) {
			return apperrors.Validation(fmt.Sprintf("plugin name contains invalid character: %q", r))
		}
	}

	// Check for reserved names
	reserved := []string{"builtin", "internal", "system", "spartan"}
	lowerName := strings.ToLower(name)
	for _, r := range reserved {
		if lowerName == r {
			return apperrors.Validation(fmt.Sprintf("plugin name is reserved: %s", name))
		}
	}

	return nil
}
