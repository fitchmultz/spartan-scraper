/**
 * Tests for development proxy configuration helpers.
 *
 * Verifies that local dev proxy routing stays separate from the browser API
 * base URL and that WebSocket proxy targets are derived consistently.
 */

import { describe, expect, it } from "vitest";
import {
  resolveApiProxyTarget,
  resolveWebSocketProxyTarget,
} from "./dev-proxy-config";

describe("resolveApiProxyTarget", () => {
  it("defaults to the local backend port when unset", () => {
    expect(resolveApiProxyTarget({})).toBe("http://localhost:8741");
  });

  it("uses DEV_API_PROXY_TARGET when provided", () => {
    expect(
      resolveApiProxyTarget({
        DEV_API_PROXY_TARGET: "http://127.0.0.1:8841",
        VITE_API_BASE_URL: "https://api.example.com",
      }),
    ).toBe("http://127.0.0.1:8841");
  });

  it("does not fall back to the browser API base URL", () => {
    expect(
      resolveApiProxyTarget({
        VITE_API_BASE_URL: "https://api.example.com",
      }),
    ).toBe("http://localhost:8741");
  });
});

describe("resolveWebSocketProxyTarget", () => {
  it("maps http targets to ws", () => {
    expect(resolveWebSocketProxyTarget("http://127.0.0.1:8841")).toBe(
      "ws://127.0.0.1:8841",
    );
  });

  it("maps https targets to wss", () => {
    expect(resolveWebSocketProxyTarget("https://api.example.com")).toBe(
      "wss://api.example.com",
    );
  });
});
