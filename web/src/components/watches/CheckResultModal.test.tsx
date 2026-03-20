/**
 * Purpose: Verify manual watch-check results render trustworthy failed-state guidance.
 * Responsibilities: Assert failed checks show their error block, hide diff output, and keep recovery actions visible.
 * Scope: CheckResultModal presentation only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Failed watch checks should never look like unchanged or diffable successes.
 */

import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { WatchCheckInspection } from "../../api";
import { CheckResultModal } from "./CheckResultModal";

function makeInspection(
  overrides: Partial<WatchCheckInspection> = {},
): WatchCheckInspection {
  return {
    id: "check-1",
    watchId: "watch-1",
    url: "http://127.0.0.1:1",
    checkedAt: "2026-03-20T12:00:00Z",
    status: "failed",
    changed: false,
    title: "Check failed",
    message: "fetch failed: dial tcp 127.0.0.1:1: connect: connection refused",
    error: "fetch failed: dial tcp 127.0.0.1:1: connect: connection refused",
    visualChanged: false,
    diffText: "--- previous\n+++ current",
    actions: [
      {
        label: "Open watch automation workspace",
        kind: "route",
        value: "/automation/watches",
      },
    ],
    ...overrides,
  };
}

describe("CheckResultModal", () => {
  it("renders failed watch checks with an error block and no diff", () => {
    render(
      <CheckResultModal
        inspection={makeInspection()}
        onClose={vi.fn()}
        onOpenHistory={vi.fn()}
      />,
    );

    expect(screen.getByText("Error")).toBeInTheDocument();
    expect(screen.getAllByText(/connection refused/i).length).toBeGreaterThan(
      0,
    );
    expect(screen.queryByText("Diff")).not.toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: "Open watch automation workspace" }),
    ).toBeInTheDocument();
  });
});
