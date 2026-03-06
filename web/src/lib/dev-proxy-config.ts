/**
 * Development proxy configuration helpers.
 *
 * Keeps Vite dev-server proxy target resolution separate from browser-exposed
 * API base URL configuration so local development stays same-origin.
 */

export function resolveApiProxyTarget(env: Record<string, string>): string {
  return env.DEV_API_PROXY_TARGET || "http://localhost:8741";
}

export function resolveWebSocketProxyTarget(target: string): string {
  if (target.startsWith("https://")) {
    return `wss://${target.slice("https://".length)}`;
  }
  if (target.startsWith("http://")) {
    return `ws://${target.slice("http://".length)}`;
  }
  return target;
}
