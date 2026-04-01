/**
 * Purpose: Provide reusable dev proxy config helpers for the web app.
 * Responsibilities: Define pure helpers, adapters, and small utility contracts shared across feature modules.
 * Scope: Shared helper logic only; route rendering and persistence stay elsewhere.
 * Usage: Import from adjacent modules that need the helper behavior defined here.
 * Invariants/Assumptions: Helpers should stay side-effect-light and reflect the current product contracts.
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
