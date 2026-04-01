/**
 * Purpose: Verify dev proxy config behavior with automated regression coverage.
 * Responsibilities: Define focused test cases, fixtures, and assertions for the module under test.
 * Scope: Automated test coverage only; production logic stays in the adjacent source modules.
 * Usage: Run through the repo test entrypoints or the feature-local test command.
 * Invariants/Assumptions: Tests should describe the current contract clearly and remain deterministic under local CI settings.
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
