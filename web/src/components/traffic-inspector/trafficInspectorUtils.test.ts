/**
 * trafficInspectorUtils.test
 *
 * Purpose:
 * - Verify the traffic-inspector helper functions keep filtering and replay behavior stable.
 *
 * Responsibilities:
 * - Cover byte formatting, filtering, and summary aggregation.
 * - Confirm entry key generation remains stable.
 * - Lock in replay request normalization.
 *
 * Scope:
 * - Unit tests for pure traffic-inspector helpers only.
 *
 * Usage:
 * - Run via Vitest as part of frontend validation.
 *
 * Invariants/Assumptions:
 * - Test fixtures mirror the generated intercepted-entry API shape.
 * - Empty replay filters should never appear in request payloads.
 */

import { describe, expect, it } from "vitest";

import type { InterceptedEntry } from "../../api";

import {
  buildReplayRequest,
  filterTrafficEntries,
  formatTrafficBytes,
  getTrafficEntryKey,
  summarizeTrafficEntries,
} from "./trafficInspectorUtils";

const entries: InterceptedEntry[] = [
  {
    duration: 100,
    request: {
      requestId: "1",
      method: "GET",
      resourceType: "xhr",
      url: "https://example.com/api/data",
    },
    response: {
      status: 200,
      bodySize: 512,
      timestamp: "2025-01-01T00:00:00Z",
    },
  },
  {
    duration: 300,
    request: {
      method: "POST",
      resourceType: "other",
      url: "https://example.com/upload",
      timestamp: "2025-01-01T00:01:00Z",
    },
  },
];

describe("trafficInspectorUtils", () => {
  it("formats bytes for display", () => {
    expect(formatTrafficBytes()).toBe("-");
    expect(formatTrafficBytes(0)).toBe("0 B");
    expect(formatTrafficBytes(1536)).toBe("1.5 KB");
  });

  it("filters by resource type and search query", () => {
    expect(filterTrafficEntries(entries, "xhr", "").length).toBe(1);
    expect(filterTrafficEntries(entries, "other", "").length).toBe(1);
    expect(filterTrafficEntries(entries, "all", "upload").length).toBe(1);
  });

  it("summarizes entry totals", () => {
    expect(summarizeTrafficEntries(entries)).toEqual({
      total: 2,
      withResponse: 1,
      totalSize: 512,
      avgDuration: 200,
    });
  });

  it("builds stable entry keys", () => {
    expect(getTrafficEntryKey(entries[0])).toBe("1");
    expect(getTrafficEntryKey(entries[1])).toContain(
      "https://example.com/upload",
    );
  });

  it("builds replay requests with trimmed filter values", () => {
    expect(
      buildReplayRequest(
        "job-1",
        "https://staging.example.com",
        true,
        "**/api/**, *.json",
        "GET, POST",
      ),
    ).toEqual({
      jobId: "job-1",
      targetBaseUrl: "https://staging.example.com",
      compareResponses: true,
      filter: {
        urlPatterns: ["**/api/**", "*.json"],
        methods: ["GET", "POST"],
      },
    });
  });
});
