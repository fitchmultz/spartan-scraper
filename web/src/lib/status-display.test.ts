/**
 * Purpose: Verify status display behavior with automated regression coverage.
 * Responsibilities: Define focused test cases, fixtures, and assertions for the module under test.
 * Scope: Automated test coverage only; production logic stays in the adjacent source modules.
 * Usage: Run through the repo test entrypoints or the feature-local test command.
 * Invariants/Assumptions: Tests should describe the current contract clearly and remain deterministic under local CI settings.
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
