// PluginManager.tsx - Web UI component for managing WASM plugins
//
// Responsibilities:
// - Display list of installed plugins with their status and metadata
// - Allow enabling/disabling plugins
// - Allow configuring plugin settings
// - Allow installing and uninstalling plugins
//
// Does NOT handle:
// - Plugin execution (handled by internal/plugins)
// - WASM runtime (handled by internal/plugins/wasm.go)

import { useState, useEffect, useCallback } from "react";
import {
  getV1Plugins,
  getV1PluginsByName,
  postV1Plugins,
  deleteV1PluginsByName,
  postV1PluginsByNameEnable,
  postV1PluginsByNameDisable,
  putV1PluginsByName,
} from "../api/sdk.gen";
import type { PluginInfo } from "../api/types.gen";
import { formatDisplayValue } from "../lib/formatting";

interface PluginManagerProps {
  onError?: (error: string) => void;
}

export function PluginManager({
  onError,
}: PluginManagerProps): React.ReactElement {
  const [plugins, setPlugins] = useState<PluginInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedPlugin, setSelectedPlugin] = useState<PluginInfo | null>(null);
  const [showInstallModal, setShowInstallModal] = useState(false);
  const [showConfigModal, setShowConfigModal] = useState(false);
  const [installPath, setInstallPath] = useState("");
  const [configKey, setConfigKey] = useState("");
  const [configValue, setConfigValue] = useState("");

  const loadPlugins = useCallback(async () => {
    try {
      setLoading(true);
      const response = await getV1Plugins();
      setPlugins((response.data?.plugins as PluginInfo[]) || []);
    } catch (err) {
      onError?.(err instanceof Error ? err.message : "Failed to load plugins");
    } finally {
      setLoading(false);
    }
  }, [onError]);

  useEffect(() => {
    loadPlugins();
  }, [loadPlugins]);

  const handleInstall = async () => {
    if (!installPath.trim()) return;

    try {
      await postV1Plugins({ body: { source: installPath.trim() } });
      setInstallPath("");
      setShowInstallModal(false);
      await loadPlugins();
    } catch (err) {
      onError?.(
        err instanceof Error ? err.message : "Failed to install plugin",
      );
    }
  };

  const handleUninstall = async (name: string) => {
    if (!confirm(`Are you sure you want to uninstall plugin "${name}"?`)) {
      return;
    }

    try {
      await deleteV1PluginsByName({ path: { name } });
      await loadPlugins();
    } catch (err) {
      onError?.(
        err instanceof Error ? err.message : "Failed to uninstall plugin",
      );
    }
  };

  const handleToggleEnabled = async (plugin: PluginInfo) => {
    try {
      if (plugin.enabled) {
        await postV1PluginsByNameDisable({ path: { name: plugin.name } });
      } else {
        await postV1PluginsByNameEnable({ path: { name: plugin.name } });
      }
      await loadPlugins();
    } catch (err) {
      onError?.(err instanceof Error ? err.message : "Failed to toggle plugin");
    }
  };

  const handleConfigure = async () => {
    if (!selectedPlugin || !configKey.trim()) return;

    try {
      let parsedValue: unknown = configValue;

      // Try to parse as JSON (for numbers, booleans, objects)
      try {
        parsedValue = JSON.parse(configValue);
      } catch {
        // Keep as string if not valid JSON
      }

      await putV1PluginsByName({
        path: { name: selectedPlugin.name },
        body: { key: configKey.trim(), value: parsedValue },
      });

      setConfigKey("");
      setConfigValue("");
      setShowConfigModal(false);
      setSelectedPlugin(null);
      await loadPlugins();
    } catch (err) {
      onError?.(
        err instanceof Error ? err.message : "Failed to configure plugin",
      );
    }
  };

  const openConfigModal = async (plugin: PluginInfo) => {
    try {
      // Refresh plugin details to get latest config
      const response = await getV1PluginsByName({
        path: { name: plugin.name },
      });
      const details = response.data;
      setSelectedPlugin(details as PluginInfo);
      setShowConfigModal(true);
    } catch (err) {
      onError?.(
        err instanceof Error ? err.message : "Failed to load plugin details",
      );
    }
  };

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / k ** i).toFixed(2))} ${sizes[i]}`;
  };

  const getHookBadgeClass = (hook: string): string => {
    if (hook.includes("fetch")) return "bg-blue-100 text-blue-800";
    if (hook.includes("extract")) return "bg-green-100 text-green-800";
    if (hook.includes("output")) return "bg-purple-100 text-purple-800";
    return "bg-gray-100 text-gray-800";
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-accent"></div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold">Plugins</h2>
          <p className="text-sm text-gray-500">
            Manage WASM plugins for extending pipeline functionality
          </p>
        </div>
        <button
          type="button"
          onClick={() => setShowInstallModal(true)}
          className="px-4 py-2 bg-accent text-white rounded-md hover:bg-accent/90 transition-colors"
        >
          Install Plugin
        </button>
      </div>

      {/* Plugin List */}
      {plugins.length === 0 ? (
        <div className="bg-surface rounded-lg p-8 text-center border border-stroke">
          <p className="text-gray-500 mb-2">No plugins installed</p>
          <p className="text-sm text-gray-400">
            Install plugins using the button above or the CLI:{" "}
            <code className="bg-gray-100 px-1 py-0.5 rounded text-xs">
              spartan plugin install --path ./my-plugin/
            </code>
          </p>
        </div>
      ) : (
        <div className="grid gap-4">
          {plugins.map((plugin) => (
            <div
              key={plugin.name}
              className="bg-surface rounded-lg p-4 border border-stroke"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <h3 className="font-semibold text-lg">{plugin.name}</h3>
                    <span
                      className={`px-2 py-0.5 rounded text-xs font-medium ${
                        plugin.enabled
                          ? "bg-green-100 text-green-800"
                          : "bg-gray-100 text-gray-800"
                      }`}
                    >
                      {plugin.enabled ? "Enabled" : "Disabled"}
                    </span>
                  </div>

                  <p className="text-sm text-gray-500 mb-2">
                    {plugin.description || "No description"}
                  </p>

                  <div className="flex flex-wrap items-center gap-4 text-xs text-gray-400 mb-3">
                    <span>Version: {plugin.version}</span>
                    {plugin.author && <span>by {plugin.author}</span>}
                    <span>Size: {formatBytes(plugin.wasmSize || 0)}</span>
                    <span>Priority: {plugin.priority}</span>
                  </div>

                  {/* Hooks */}
                  <div className="flex flex-wrap gap-1 mb-3">
                    {plugin.hooks?.map((hook: string) => (
                      <span
                        key={hook}
                        className={`px-2 py-0.5 rounded text-xs ${getHookBadgeClass(hook)}`}
                      >
                        {hook}
                      </span>
                    ))}
                  </div>

                  {/* Permissions */}
                  {plugin.permissions && plugin.permissions.length > 0 && (
                    <div className="flex flex-wrap gap-1">
                      <span className="text-xs text-gray-400 mr-1">
                        Permissions:
                      </span>
                      {plugin.permissions.map((perm: string) => (
                        <span
                          key={perm}
                          className="px-2 py-0.5 rounded text-xs bg-orange-100 text-orange-800"
                        >
                          {perm}
                        </span>
                      ))}
                    </div>
                  )}
                </div>

                {/* Actions */}
                <div className="flex flex-col gap-2 ml-4">
                  <button
                    type="button"
                    onClick={() => handleToggleEnabled(plugin)}
                    className={`px-3 py-1.5 rounded text-sm font-medium transition-colors ${
                      plugin.enabled
                        ? "bg-gray-100 text-gray-700 hover:bg-gray-200"
                        : "bg-accent text-white hover:bg-accent/90"
                    }`}
                  >
                    {plugin.enabled ? "Disable" : "Enable"}
                  </button>
                  <button
                    type="button"
                    onClick={() => openConfigModal(plugin)}
                    className="px-3 py-1.5 rounded text-sm font-medium bg-gray-100 text-gray-700 hover:bg-gray-200 transition-colors"
                  >
                    Configure
                  </button>
                  <button
                    type="button"
                    onClick={() => handleUninstall(plugin.name)}
                    className="px-3 py-1.5 rounded text-sm font-medium bg-red-100 text-red-700 hover:bg-red-200 transition-colors"
                  >
                    Uninstall
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Install Modal */}
      {showInstallModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-surface rounded-lg p-6 w-full max-w-md border border-stroke">
            <h3 className="text-lg font-semibold mb-4">Install Plugin</h3>
            <div className="space-y-4">
              <div>
                <span className="block text-sm font-medium mb-1">
                  Plugin Directory Path
                </span>
                <input
                  type="text"
                  value={installPath}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                    setInstallPath(e.target.value)
                  }
                  placeholder="/path/to/plugin/directory"
                  className="w-full px-3 py-2 border border-stroke rounded-md focus:outline-none focus:ring-2 focus:ring-accent"
                />
                <p className="text-xs text-gray-400 mt-1">
                  Path to a directory containing manifest.json and plugin.wasm
                </p>
              </div>
              <div className="flex justify-end gap-2">
                <button
                  type="button"
                  onClick={() => setShowInstallModal(false)}
                  className="px-4 py-2 rounded text-sm font-medium bg-gray-100 text-gray-700 hover:bg-gray-200 transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={handleInstall}
                  disabled={!installPath.trim()}
                  className="px-4 py-2 rounded text-sm font-medium bg-accent text-white hover:bg-accent/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  Install
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Configure Modal */}
      {showConfigModal && selectedPlugin && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-surface rounded-lg p-6 w-full max-w-md border border-stroke">
            <h3 className="text-lg font-semibold mb-4">
              Configure {selectedPlugin.name}
            </h3>
            <div className="space-y-4">
              {/* Current Config */}
              {selectedPlugin.config &&
                Object.keys(selectedPlugin.config).length > 0 && (
                  <div>
                    <span className="block text-sm font-medium mb-2">
                      Current Configuration
                    </span>
                    <div className="bg-gray-50 rounded p-3 space-y-1">
                      {Object.entries(selectedPlugin.config).map(
                        ([key, value]) => (
                          <div
                            key={key}
                            className="flex justify-between text-sm"
                          >
                            <span className="text-gray-500">{key}:</span>
                            <span className="font-mono text-xs">
                              {formatDisplayValue(value, {
                                emptyLabel: "-",
                                objectLabel: "{...}",
                              })}
                            </span>
                          </div>
                        ),
                      )}
                    </div>
                  </div>
                )}

              {/* New Config Entry */}
              <div>
                <span className="block text-sm font-medium mb-2">
                  Add/Update Configuration
                </span>
                <div className="space-y-2">
                  <input
                    type="text"
                    value={configKey}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                      setConfigKey(e.target.value)
                    }
                    placeholder="Configuration key"
                    className="w-full px-3 py-2 border border-stroke rounded-md focus:outline-none focus:ring-2 focus:ring-accent"
                  />
                  <input
                    type="text"
                    value={configValue}
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) =>
                      setConfigValue(e.target.value)
                    }
                    placeholder="Configuration value (JSON or string)"
                    className="w-full px-3 py-2 border border-stroke rounded-md focus:outline-none focus:ring-2 focus:ring-accent"
                  />
                </div>
                <p className="text-xs text-gray-400 mt-1">
                  Use JSON format for numbers, booleans, or objects
                </p>
              </div>

              <div className="flex justify-end gap-2">
                <button
                  type="button"
                  onClick={() => {
                    setShowConfigModal(false);
                    setSelectedPlugin(null);
                    setConfigKey("");
                    setConfigValue("");
                  }}
                  className="px-4 py-2 rounded text-sm font-medium bg-gray-100 text-gray-700 hover:bg-gray-200 transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={handleConfigure}
                  disabled={!configKey.trim()}
                  className="px-4 py-2 rounded text-sm font-medium bg-accent text-white hover:bg-accent/90 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  Save Configuration
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default PluginManager;
