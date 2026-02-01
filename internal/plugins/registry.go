// Package plugins provides a WASM-based plugin system for third-party extensions.
package plugins

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/fitchmultz/spartan-scraper/internal/pipeline"
)

// PluginRegistry extends the pipeline Registry with external plugin support.
type PluginRegistry struct {
	*pipeline.Registry
	loader      *Loader
	runtime     *WASMRuntime
	wasmPlugins map[string]*WASMPlugin
	mu          sync.RWMutex
	dataDir     string
}

// NewPluginRegistry creates a new plugin registry with WASM support.
func NewPluginRegistry(dataDir string) *PluginRegistry {
	return &PluginRegistry{
		Registry:    pipeline.NewRegistry(),
		loader:      NewLoader(dataDir),
		runtime:     NewWASMRuntime(dataDir),
		wasmPlugins: make(map[string]*WASMPlugin),
		dataDir:     dataDir,
	}
}

// Close shuts down the plugin registry and releases resources.
func (r *PluginRegistry) Close(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Close all WASM plugin instances
	for _, plugin := range r.wasmPlugins {
		plugin.Close(ctx)
	}
	r.wasmPlugins = make(map[string]*WASMPlugin)

	// Close the WASM runtime
	return r.runtime.Close(ctx)
}

// LoadExternalPlugins discovers and loads all external plugins from the data directory.
func (r *PluginRegistry) LoadExternalPlugins() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Discover plugins
	infos, err := r.loader.Discover()
	if err != nil {
		return fmt.Errorf("failed to discover plugins: %w", err)
	}

	// Load each enabled plugin
	for _, info := range infos {
		if !info.Enabled {
			continue
		}

		if err := r.loadPlugin(info.Name); err != nil {
			// Log error but continue loading other plugins
			fmt.Fprintf(os.Stderr, "[WARN] Failed to load plugin %s: %v\n", info.Name, err)
			continue
		}
	}

	return nil
}

// loadPlugin loads a single plugin by name.
func (r *PluginRegistry) loadPlugin(name string) error {
	// Check if already loaded
	if _, ok := r.wasmPlugins[name]; ok {
		return nil
	}

	// Load manifest
	manifest, pluginDir, err := r.loader.LoadPlugin(name)
	if err != nil {
		return err
	}

	// Create wrapper
	wrapper := NewWASMPlugin(manifest, r.runtime, pluginDir)

	// Register with pipeline registry
	r.Registry.Register(wrapper)

	// Store reference
	r.wasmPlugins[name] = wrapper

	fmt.Printf("[plugin] Loaded: %s v%s\n", manifest.Name, manifest.Version)

	return nil
}

// RegisterWASMPlugin manually registers a WASM plugin.
func (r *PluginRegistry) RegisterWASMPlugin(plugin *WASMPlugin) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.Registry.Register(plugin)
	r.wasmPlugins[plugin.Name()] = plugin
}

// UnloadPlugin unloads a plugin and removes it from the registry.
func (r *PluginRegistry) UnloadPlugin(ctx context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	plugin, ok := r.wasmPlugins[name]
	if !ok {
		return fmt.Errorf("plugin not found: %s", name)
	}

	// Close the plugin instance
	plugin.Close(ctx)

	// Remove from map
	delete(r.wasmPlugins, name)

	// Note: The pipeline.Registry doesn't support unregistering,
	// so we can't remove it from there. The plugin will just be
	// disabled on the next load.

	return nil
}

// GetPlugin returns a loaded WASM plugin by name.
func (r *PluginRegistry) GetPlugin(name string) (*WASMPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plugin, ok := r.wasmPlugins[name]
	return plugin, ok
}

// ListPlugins returns information about all installed plugins.
func (r *PluginRegistry) ListPlugins() ([]*PluginInfo, error) {
	return r.loader.Discover()
}

// InstallPlugin installs a plugin from a source directory.
func (r *PluginRegistry) InstallPlugin(sourceDir string) (*PluginInfo, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.loader.Install(sourceDir)
}

// UninstallPlugin removes an installed plugin.
func (r *PluginRegistry) UninstallPlugin(ctx context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Unload if currently loaded
	if plugin, ok := r.wasmPlugins[name]; ok {
		plugin.Close(ctx)
		delete(r.wasmPlugins, name)
	}

	return r.loader.Uninstall(name)
}

// EnablePlugin enables a plugin.
func (r *PluginRegistry) EnablePlugin(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.loader.Enable(name)
}

// DisablePlugin disables a plugin.
func (r *PluginRegistry) DisablePlugin(ctx context.Context, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Unload if currently loaded
	if plugin, ok := r.wasmPlugins[name]; ok {
		plugin.Close(ctx)
		delete(r.wasmPlugins, name)
	}

	return r.loader.Disable(name)
}

// ConfigurePlugin updates a plugin's configuration.
func (r *PluginRegistry) ConfigurePlugin(name string, key string, value any) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.loader.Configure(name, key, value)
}

// GetPluginConfig gets a plugin's configuration value.
func (r *PluginRegistry) GetPluginConfig(name string, key string) (any, bool, error) {
	return r.loader.GetConfig(name, key)
}

// GetPluginInfo returns information about a specific plugin.
func (r *PluginRegistry) GetPluginInfo(name string) (*PluginInfo, error) {
	pluginDir := r.loader.GetPluginsDir()
	manifestPath := pluginDir + "/" + name + "/manifest.json"

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, err
	}

	return manifest.ToInfo(pluginDir + "/" + name)
}

// ReloadPlugins reloads all enabled plugins.
func (r *PluginRegistry) ReloadPlugins(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Close all current plugins
	for _, plugin := range r.wasmPlugins {
		plugin.Close(ctx)
	}
	r.wasmPlugins = make(map[string]*WASMPlugin)

	// Clear and re-register built-in plugins only
	// (external plugins will be reloaded by LoadExternalPlugins)
	r.Registry = pipeline.NewRegistry()

	return nil
}

// IsPluginLoaded checks if a plugin is currently loaded.
func (r *PluginRegistry) IsPluginLoaded(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, ok := r.wasmPlugins[name]
	return ok
}
