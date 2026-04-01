/**
 * Purpose: Verify job status behavior with automated regression coverage.
 * Responsibilities: Define focused test cases, fixtures, and assertions for the module under test.
 * Scope: Automated test coverage only; production logic stays in the adjacent source modules.
 * Usage: Run through the repo test entrypoints or the feature-local test command.
 * Invariants/Assumptions: Tests should describe the current contract clearly and remain deterministic under local CI settings.
 */

import { describe, expect, it } from "vitest";

import { getJobStatusBadgeClass, getJobStatusIcon } from "./job-status";

describe("getJobStatusBadgeClass", () => {
  it("maps in-progress states to the running badge", () => {
    expect(getJobStatusBadgeClass("queued")).toBe("running");
    expect(getJobStatusBadgeClass("running")).toBe("running");
    expect(getJobStatusBadgeClass("pending")).toBe("running");
  });

  it("maps terminal states to success and failure badges", () => {
    expect(getJobStatusBadgeClass("succeeded")).toBe("success");
    expect(getJobStatusBadgeClass("ready")).toBe("success");
    expect(getJobStatusBadgeClass("failed")).toBe("failed");
    expect(getJobStatusBadgeClass("canceled")).toBe("failed");
  });

  it("falls back to a neutral badge class", () => {
    expect(getJobStatusBadgeClass()).toBe("");
  });
});

describe("getJobStatusIcon", () => {
  it("maps known statuses to stable icons", () => {
    expect(getJobStatusIcon("queued")).toBe("⏳");
    expect(getJobStatusIcon("running")).toBe("▶️");
    expect(getJobStatusIcon("succeeded")).toBe("✅");
    expect(getJobStatusIcon("failed")).toBe("❌");
    expect(getJobStatusIcon("canceled")).toBe("⏹️");
  });

  it("falls back to a neutral icon", () => {
    expect(getJobStatusIcon()).toBe("📄");
  });
});
