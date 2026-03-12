/**
 * API client configuration module.
 * Provides the base URL for API requests based on environment configuration.
 */

import { client } from "../api/client.gen";

// Import from Vite's env (available at build time)
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL as string | undefined;

/**
 * Get the API base URL for requests.
 *
 * - Empty string: Use relative paths (works with Vite dev proxy or same-origin deployment)
 * - Full URL: Use absolute URL for cross-origin requests (production with separate API server)
 *
 * @returns The base URL to use for API requests
 */
export function getApiBaseUrl(): string {
  // If env var is set, use it; otherwise default to empty string (relative paths)
  return API_BASE_URL ?? "";
}

let configuredApiBaseUrl: string | null = null;

/**
 * Configure the generated API client to use the shared runtime base URL.
 *
 * This keeps every SDK call on the same-origin/proxy path unless the user
 * explicitly opts into a browser-visible cross-origin base URL.
 */
export function configureApiClient(): string {
  const baseUrl = getApiBaseUrl();

  if (configuredApiBaseUrl === baseUrl) {
    return baseUrl;
  }

  client.setConfig({ baseUrl });
  configuredApiBaseUrl = baseUrl;
  return baseUrl;
}

/**
 * Build a full API URL by combining the base URL with a path.
 *
 * If the base URL is empty (development mode with proxy), returns the path directly.
 * If the base URL is set (production with separate API server), returns the full URL.
 *
 * Handles trailing slashes on the base URL to avoid double slashes in the final URL.
 *
 * @param path - The API path (e.g., "/v1/jobs/123/results")
 * @returns The complete URL to use for fetch requests
 */
export function buildApiUrl(path: string): string {
  const baseUrl = getApiBaseUrl();
  return buildApiUrlWithBase(baseUrl, path);
}

/**
 * Internal helper to build a URL with an explicit base URL.
 * Exported for testing purposes.
 *
 * @internal
 */
export function buildApiUrlWithBase(baseUrl: string, path: string): string {
  if (!baseUrl) {
    return path;
  }
  // Remove trailing slash from baseUrl to avoid double slashes
  const cleanBaseUrl = baseUrl.endsWith("/") ? baseUrl.slice(0, -1) : baseUrl;
  return `${cleanBaseUrl}${path}`;
}
