/**
 * Purpose: Verify the watch history modal presents consistent loading and empty states.
 * Responsibilities: Assert top-level history loading, detail loading, and empty history copy render clearly.
 * Scope: WatchHistoryModal presentation only.
 * Usage: Run with `pnpm test`.
 * Invariants/Assumptions: WatchManager owns data loading, while the modal must clearly communicate loading and empty persisted history states.
 */

import { render, screen } from "@testing-library/react";
import type { ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";
import type { Watch, WatchCheckInspection } from "../../api";
import { WatchHistoryModal } from "./WatchHistoryModal";

function makeWatch(overrides: Partial<Watch> = {}): Watch {
  return {
    id: "watch-1",
    url: "https://example.com/pricing",
    intervalSeconds: 3600,
    enabled: true,
    createdAt: "2026-03-19T15:00:00Z",
    changeCount: 0,
    diffFormat: "unified",
    notifyOnChange: false,
    headless: false,
    usePlaywright: false,
    status: "active",
    ...overrides,
  };
}

function makeInspection(
  overrides: Partial<WatchCheckInspection> = {},
): WatchCheckInspection {
  return {
    id: "check-1",
    watchId: "watch-1",
    url: "https://example.com/pricing",
    checkedAt: "2026-03-19T15:05:00Z",
    status: "unchanged",
    changed: false,
    title: "No change detected",
    message: "The latest check matched the baseline.",
    visualChanged: false,
    actions: [],
    ...overrides,
  };
}

function renderHistoryModal(
  overrides: Partial<ComponentProps<typeof WatchHistoryModal>> = {},
) {
  render(
    <WatchHistoryModal
      watch={makeWatch()}
      records={[makeInspection()]}
      total={1}
      limit={10}
      offset={0}
      loading={false}
      selectedCheck={makeInspection()}
      selectedCheckLoading={false}
      onClose={vi.fn()}
      onSelectCheck={vi.fn()}
      onPageChange={vi.fn()}
      {...overrides}
    />,
  );
}

describe("WatchHistoryModal", () => {
  it("renders a guided loading state while watch history is loading", () => {
    renderHistoryModal({
      loading: true,
      records: [],
      total: 0,
      selectedCheck: null,
    });

    expect(
      screen.getByRole("heading", {
        name: "Watch History: https://example.com/pricing",
      }),
    ).toBeInTheDocument();
    expect(screen.getByText("Loading watch history")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Fetching saved checks and inspection summaries for this watch.",
      ),
    ).toBeInTheDocument();
    expect(
      screen.queryByText("No watch history found yet"),
    ).not.toBeInTheDocument();
  });

  it("renders a guided empty state when no persisted watch checks exist yet", () => {
    renderHistoryModal({ records: [], total: 0, selectedCheck: null });

    expect(screen.getByText("No watch history found yet")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Run a manual check or wait for the scheduler to record one.",
      ),
    ).toBeInTheDocument();
  });

  it("renders a guided detail loading state while a selected check is loading", () => {
    renderHistoryModal({ selectedCheckLoading: true });

    expect(screen.getByText("Loading check details")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Fetching saved artifacts, diff output, and recommended next steps for this run.",
      ),
    ).toBeInTheDocument();
  });
});
