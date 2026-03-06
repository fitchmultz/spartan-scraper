/**
 * Spartan Scraper Extension - Storage Helpers
 *
 * Chrome storage API wrappers for settings and cached data.
 */

import { DEFAULT_SETTINGS, type ExtensionSettings } from "./types.js";

const STORAGE_KEYS = {
  SETTINGS: "spartan_settings",
  CACHED_TEMPLATES: "spartan_cached_templates",
  TEMPLATES_CACHE_TIME: "spartan_templates_cache_time",
} as const;

const CACHE_TTL_MS = 5 * 60 * 1000; // 5 minutes

type SettingsStorage = {
  [STORAGE_KEYS.SETTINGS]?: ExtensionSettings;
};

type TemplateCacheStorage = {
  [STORAGE_KEYS.CACHED_TEMPLATES]?: string[];
  [STORAGE_KEYS.TEMPLATES_CACHE_TIME]?: number;
};

/**
 * Get extension settings from chrome.storage.sync
 * Returns default settings if none are stored.
 */
export async function getSettings(): Promise<ExtensionSettings> {
  try {
    const result = await chrome.storage.sync.get<SettingsStorage>(
      STORAGE_KEYS.SETTINGS,
    );
    const stored = result[STORAGE_KEYS.SETTINGS];

    if (stored) {
      // Merge with defaults to handle new fields
      return { ...DEFAULT_SETTINGS, ...stored };
    }
  } catch (err) {
    console.error("Failed to load settings:", err);
  }

  return DEFAULT_SETTINGS;
}

/**
 * Save extension settings to chrome.storage.sync
 */
export async function saveSettings(
  settings: ExtensionSettings,
): Promise<void> {
  try {
    await chrome.storage.sync.set({
      [STORAGE_KEYS.SETTINGS]: settings,
    });
  } catch (err) {
    console.error("Failed to save settings:", err);
    throw new Error("Failed to save settings");
  }
}

/**
 * Get the stored API key
 */
export async function getApiKey(): Promise<string> {
  const settings = await getSettings();
  return settings.apiKey;
}

/**
 * Get the configured API base URL
 */
export async function getApiUrl(): Promise<string> {
  const settings = await getSettings();
  return settings.apiUrl;
}

/**
 * Cache templates list in chrome.storage.local
 */
export async function cacheTemplates(templates: string[]): Promise<void> {
  try {
    await chrome.storage.local.set({
      [STORAGE_KEYS.CACHED_TEMPLATES]: templates,
      [STORAGE_KEYS.TEMPLATES_CACHE_TIME]: Date.now(),
    });
  } catch (err) {
    console.error("Failed to cache templates:", err);
  }
}

/**
 * Get cached templates if they haven't expired
 */
export async function getCachedTemplates(): Promise<string[] | null> {
  try {
    const result = await chrome.storage.local.get<TemplateCacheStorage>([
      STORAGE_KEYS.CACHED_TEMPLATES,
      STORAGE_KEYS.TEMPLATES_CACHE_TIME,
    ]);

    const templates = result[STORAGE_KEYS.CACHED_TEMPLATES];
    const cacheTime = result[STORAGE_KEYS.TEMPLATES_CACHE_TIME];

    if (templates && cacheTime) {
      const age = Date.now() - cacheTime;
      if (age < CACHE_TTL_MS) {
        return templates;
      }
    }
  } catch (err) {
    console.error("Failed to get cached templates:", err);
  }

  return null;
}

/**
 * Clear cached templates
 */
export async function clearCachedTemplates(): Promise<void> {
  try {
    await chrome.storage.local.remove([
      STORAGE_KEYS.CACHED_TEMPLATES,
      STORAGE_KEYS.TEMPLATES_CACHE_TIME,
    ]);
  } catch (err) {
    console.error("Failed to clear template cache:", err);
  }
}
