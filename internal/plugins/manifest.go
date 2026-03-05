// Package plugins provides a WASM-based plugin system for third-party extensions.
// It supports sandboxed plugin execution with explicit permissions and hooks
// into the pipeline stages (fetch, extract, output).
package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fitchmultz/spartan-scraper/internal/apperrors"
)

// Permission constants for plugin capabilities.
const (
	PermNetwork    = "network"
	PermFilesystem = "filesystem"
	PermEnv        = "env"
)

// ValidHooks contains all supported hook names.
var ValidHooks = []string{
	"pre_fetch", "post_fetch",
	"pre_extract", "post_extract",
	"pre_output", "post_output",
}

// ValidPermissions contains all supported permission names.
var ValidPermissions = []string{
	PermNetwork,
	PermFilesystem,
	PermEnv,
}

// PluginManifest defines the plugin metadata and configuration.
type PluginManifest struct {
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	Description string         `json:"description"`
	Author      string         `json:"author"`
	Hooks       []string       `json:"hooks"`       // ["pre_fetch", "post_extract", ...]
	Permissions []string       `json:"permissions"` // ["network", "filesystem", ...]
	WASMPath    string         `json:"wasm_path"`   // Relative to plugin directory
	Config      map[string]any `json:"config,omitempty"`
	Enabled     bool           `json:"enabled"`
	Priority    int            `json:"priority"` // Execution order (lower = earlier)
}

// Validate checks the manifest for correctness.
func (m *PluginManifest) Validate() error {
	if m.Name == "" {
		return apperrors.Validation("plugin name is required")
	}

	// Validate name format (alphanumeric, hyphens, underscores)
	for _, r := range m.Name {
		if !isValidNameChar(r) {
			return apperrors.Validation(fmt.Sprintf("plugin name contains invalid character: %q", r))
		}
	}

	if m.Version == "" {
		return apperrors.Validation("plugin version is required")
	}

	if m.WASMPath == "" {
		return apperrors.Validation("wasm_path is required")
	}

	// Validate hooks
	for _, hook := range m.Hooks {
		if !isValidHook(hook) {
			return apperrors.Validation(fmt.Sprintf("invalid hook: %q", hook))
		}
	}

	// Validate permissions
	for _, perm := range m.Permissions {
		if !isValidPermission(perm) {
			return apperrors.Validation(fmt.Sprintf("invalid permission: %q", perm))
		}
	}

	return nil
}

// SupportsHook checks if the plugin supports a specific hook.
func (m *PluginManifest) SupportsHook(hook string) bool {
	hook = strings.ToLower(hook)
	for _, h := range m.Hooks {
		if strings.ToLower(h) == hook {
			return true
		}
	}
	return false
}

// HasPermission checks if the plugin has a specific permission.
func (m *PluginManifest) HasPermission(perm string) bool {
	perm = strings.ToLower(perm)
	for _, p := range m.Permissions {
		if strings.ToLower(p) == perm {
			return true
		}
	}
	return false
}

// GetWASMPath returns the absolute path to the WASM binary.
func (m *PluginManifest) GetWASMPath(pluginDir string) string {
	return filepath.Join(pluginDir, m.WASMPath)
}

// LoadManifest loads a plugin manifest from a file.
func LoadManifest(path string) (*PluginManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, apperrors.NotFound(fmt.Sprintf("manifest not found: %s", path))
		}
		return nil, apperrors.Wrap(apperrors.KindInternal, "failed to read manifest", err)
	}

	var manifest PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, apperrors.Wrap(apperrors.KindValidation, "invalid manifest JSON", err)
	}

	if err := manifest.Validate(); err != nil {
		return nil, err
	}

	return &manifest, nil
}

// SaveManifest saves a plugin manifest to a file.
func SaveManifest(path string, manifest *PluginManifest) error {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return apperrors.Wrap(apperrors.KindInternal, "failed to marshal manifest", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return apperrors.Wrap(apperrors.KindPermission, "failed to write manifest", err)
	}

	return nil
}

// PluginInfo contains runtime information about a plugin.
type PluginInfo struct {
	Name        string         `json:"name"`
	Version     string         `json:"version"`
	Description string         `json:"description"`
	Author      string         `json:"author"`
	Hooks       []string       `json:"hooks"`
	Permissions []string       `json:"permissions"`
	Enabled     bool           `json:"enabled"`
	Priority    int            `json:"priority"`
	WASMSize    int64          `json:"wasm_size"`
	Config      map[string]any `json:"config,omitempty"`
}

// ToInfo converts a manifest to PluginInfo with additional runtime data.
func (m *PluginManifest) ToInfo(pluginDir string) (*PluginInfo, error) {
	wasmPath := m.GetWASMPath(pluginDir)
	var wasmSize int64

	info, err := os.Stat(wasmPath)
	if err == nil {
		wasmSize = info.Size()
	}

	return &PluginInfo{
		Name:        m.Name,
		Version:     m.Version,
		Description: m.Description,
		Author:      m.Author,
		Hooks:       m.Hooks,
		Permissions: m.Permissions,
		Enabled:     m.Enabled,
		Priority:    m.Priority,
		WASMSize:    wasmSize,
		Config:      m.Config,
	}, nil
}

// Helper functions

func isValidNameChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '-' || r == '_'
}

func isValidHook(hook string) bool {
	hook = strings.ToLower(hook)
	for _, h := range ValidHooks {
		if h == hook {
			return true
		}
	}
	return false
}

func isValidPermission(perm string) bool {
	perm = strings.ToLower(perm)
	for _, p := range ValidPermissions {
		if p == perm {
			return true
		}
	}
	return false
}

// HookToStage converts a hook name to a display-friendly stage name.
func HookToStage(hook string) string {
	parts := strings.Split(hook, "_")
	if len(parts) != 2 {
		return hook
	}
	return fmt.Sprintf("%s %s", parts[0], parts[1])
}
