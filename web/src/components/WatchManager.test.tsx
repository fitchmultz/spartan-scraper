/**
 * Purpose: Verify watch-management inspection workflows in the Automation UI.
 * Responsibilities: Assert persisted history loads from the watch list and manual checks can pivot into the saved history workflow.
 * Scope: WatchManager behavior only; network calls are mocked through props.
 * Usage: Run with `pnpm test`.
 * Invariants/Assumptions: Persisted watch history is the source of truth for detailed post-check inspection.
 */

import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";
import type {
  Watch,
  WatchCheckHistoryResponse,
  WatchCheckInspection,
  WatchCheckResult,
  WatchInput,
} from "../api";
import { WatchManager } from "./WatchManager";

function makeWatch(overrides: Partial<Watch> = {}): Watch {
  return {
    id: "watch-1",
    url: "https://example.com/pricing",
    intervalSeconds: 3600,
    enabled: true,
    createdAt: "2026-03-19T15:00:00Z",
    changeCount: 2,
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
    status: "changed",
    changed: true,
    title: "Change detected",
    message: "Spartan detected a content change for this watch.",
    diffText: "-old\n+new",
    visualChanged: false,
    actions: [
      {
        label: "Inspect this check from the CLI",
        kind: "command",
        value: "spartan watch history watch-1 --check-id check-1",
      },
    ],
    ...overrides,
  };
}

function makeHistoryResponse(
  overrides: Partial<WatchCheckHistoryResponse> = {},
): WatchCheckHistoryResponse {
  return {
    checks: [makeInspection()],
    total: 1,
    limit: 10,
    offset: 0,
    ...overrides,
  };
}

function createProps(
  overrides: Partial<ComponentProps<typeof WatchManager>> = {},
): ComponentProps<typeof WatchManager> {
  const onCreate = vi.fn<(watch: WatchInput) => Promise<void>>();
  const onUpdate = vi.fn<(id: string, watch: WatchInput) => Promise<void>>();
  const onDelete = vi.fn<(id: string) => Promise<void>>();
  const onCheck =
    vi.fn<(id: string) => Promise<WatchCheckResult | undefined>>();
  const onLoadHistory =
    vi.fn<
      (
        watchId: string,
        limit: number,
        offset: number,
      ) => Promise<WatchCheckHistoryResponse | undefined>
    >();
  const onLoadHistoryDetail =
    vi.fn<
      (
        watchId: string,
        checkId: string,
      ) => Promise<WatchCheckInspection | undefined>
    >();

  onCreate.mockResolvedValue(undefined);
  onUpdate.mockResolvedValue(undefined);
  onDelete.mockResolvedValue(undefined);
  onCheck.mockResolvedValue(undefined);
  onLoadHistory.mockResolvedValue(makeHistoryResponse());
  onLoadHistoryDetail.mockResolvedValue(makeInspection());

  return {
    watches: [makeWatch()],
    onRefresh: vi.fn(),
    onCreate,
    onUpdate,
    onDelete,
    onCheck,
    onLoadHistory,
    onLoadHistoryDetail,
    loading: false,
    ...overrides,
  };
}

describe("WatchManager", () => {
  it("loads persisted watch history from the list view", async () => {
    const user = userEvent.setup();
    const props = createProps();

    render(<WatchManager {...props} />);

    await user.click(screen.getByRole("button", { name: "History" }));

    await waitFor(() => {
      expect(props.onLoadHistory).toHaveBeenCalledWith("watch-1", 10, 0);
    });
    await waitFor(() => {
      expect(props.onLoadHistoryDetail).toHaveBeenCalledWith(
        "watch-1",
        "check-1",
      );
    });

    expect(
      await screen.findByRole("heading", {
        name: "Watch History: https://example.com/pricing",
      }),
    ).toBeInTheDocument();
    expect(screen.getAllByText("Change detected").length).toBeGreaterThan(0);
    expect(
      screen.getByText("Inspect this check from the CLI"),
    ).toBeInTheDocument();
  });

  it("can pivot from a manual check result into persisted history", async () => {
    const user = userEvent.setup();
    const props = createProps({
      onCheck: vi.fn().mockResolvedValue({
        checkId: "check-1",
        watchId: "watch-1",
        url: "https://example.com/pricing",
        checkedAt: "2026-03-19T15:05:00Z",
        changed: true,
        diffText: "-old\n+new",
      } satisfies WatchCheckResult),
    });

    render(<WatchManager {...props} />);

    await user.click(screen.getByRole("button", { name: "Check" }));

    expect(
      await screen.findByRole("heading", { name: "Change detected" }),
    ).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "View history" }));

    await waitFor(() => {
      expect(props.onLoadHistory).toHaveBeenCalledWith("watch-1", 10, 0);
    });

    expect(
      await screen.findByRole("heading", {
        name: "Watch History: https://example.com/pricing",
      }),
    ).toBeInTheDocument();
  });
});
