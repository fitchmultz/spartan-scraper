/**
 * Shared job status helper tests.
 *
 * Purpose:
 * - Verify the shared job status presentation helpers stay consistent.
 *
 * Responsibilities:
 * - Cover badge-class mappings for job and dependency statuses.
 * - Cover icon mappings for known and unknown job states.
 *
 * Scope:
 * - Unit tests for web/src/lib/job-status.ts only.
 *
 * Usage:
 * - Run through Vitest as part of the web test suite.
 *
 * Invariants/Assumptions:
 * - Waiting states should share the running badge style.
 * - Unknown states should fall back to neutral presentation values.
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
