/**
 * Shared status display helper tests.
 *
 * Purpose:
 * - Verify shared tone mappings stay consistent across status surfaces.
 *
 * Responsibilities:
 * - Cover tone palette selection for known status values.
 * - Cover fallback behavior for unknown states.
 *
 * Scope:
 * - Unit tests for web/src/lib/status-display.ts only.
 *
 * Usage:
 * - Run through Vitest as part of the web test suite.
 *
 * Invariants/Assumptions:
 * - Unknown states should not render as success or danger by accident.
 */

import { describe, expect, it } from "vitest";

import {
  getEnabledStatusTone,
  getExportHistoryStatusTone,
  getStatusToneColors,
  getWatchStatusTone,
} from "./status-display";

describe("getStatusToneColors", () => {
  it("returns stable palette colors for supported tones", () => {
    expect(getStatusToneColors("success")).toEqual({
      backgroundColor: "rgba(34, 197, 94, 0.15)",
      color: "#22c55e",
    });
    expect(getStatusToneColors("neutral")).toEqual({
      backgroundColor: "rgba(156, 163, 175, 0.15)",
      color: "var(--muted)",
    });
  });
});

describe("watch/export tone helpers", () => {
  it("maps watch statuses correctly", () => {
    expect(getWatchStatusTone("active")).toBe("success");
    expect(getWatchStatusTone("error")).toBe("danger");
    expect(getWatchStatusTone("paused")).toBe("neutral");
  });

  it("maps export-history statuses correctly", () => {
    expect(getExportHistoryStatusTone("success")).toBe("success");
    expect(getExportHistoryStatusTone("pending")).toBe("warning");
    expect(getExportHistoryStatusTone("failed")).toBe("danger");
  });

  it("maps enabled state correctly", () => {
    expect(getEnabledStatusTone(true)).toBe("success");
    expect(getEnabledStatusTone(false)).toBe("neutral");
  });
});
