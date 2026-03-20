/**
 * Purpose: Verify the watch list renders its own table rows and sorting behavior.
 * Responsibilities: Assert newest-first ordering and confirm empty arrays stay header-only because empty/loading states belong to WatchManager.
 * Scope: WatchList presentation only.
 * Usage: Run with `pnpm test`.
 * Invariants/Assumptions: WatchManager owns empty/loading state messaging while WatchList owns row ordering and row rendering.
 */

import { render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { Watch } from "../../api";
import { WatchList } from "./WatchList";

function makeWatch(overrides: Partial<Watch> = {}): Watch {
  return {
    id: "watch-1",
    url: "https://example.com/older",
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

function renderWatchList(watches: Watch[]) {
  render(
    <WatchList
      watches={watches}
      checkingId={null}
      historyLoadingId={null}
      deleteConfirmId={null}
      onEdit={vi.fn()}
      onDelete={vi.fn()}
      onCheck={vi.fn()}
      onHistory={vi.fn()}
      onDeleteConfirm={vi.fn()}
    />,
  );
}

describe("WatchList", () => {
  it("renders watches newest-first", () => {
    renderWatchList([
      makeWatch({
        id: "watch-older",
        url: "https://example.com/older",
        createdAt: "2026-03-18T15:00:00Z",
      }),
      makeWatch({
        id: "watch-newer",
        url: "https://example.com/newer",
        createdAt: "2026-03-19T15:00:00Z",
      }),
    ]);

    const rows = screen.getAllByRole("row");
    expect(
      within(rows[1] as HTMLElement).getByText("https://example.com/newer"),
    ).toBeInTheDocument();
    expect(
      within(rows[2] as HTMLElement).getByText("https://example.com/older"),
    ).toBeInTheDocument();
  });

  it("renders only the header row when the watch array is empty", () => {
    renderWatchList([]);

    expect(
      screen.getByRole("columnheader", { name: "URL" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("columnheader", { name: "Status" }),
    ).toBeInTheDocument();
    expect(screen.getAllByRole("row")).toHaveLength(1);
    expect(
      screen.queryByText("No watches configured yet"),
    ).not.toBeInTheDocument();
  });
});
