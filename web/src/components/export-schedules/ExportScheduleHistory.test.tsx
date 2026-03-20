/**
 * Purpose: Verify the export history modal presents consistent loading and empty states.
 * Responsibilities: Assert top-level history loading and empty persisted-history copy render clearly.
 * Scope: ExportScheduleHistory presentation only.
 * Usage: Run with `pnpm test`.
 * Invariants/Assumptions: ExportScheduleManager owns data loading, while the modal must clearly communicate loading and empty persisted history states.
 */

import { render, screen } from "@testing-library/react";
import type { ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";
import type { ExportInspection } from "../../api";
import { ExportScheduleHistory } from "./ExportScheduleHistory";

function makeHistoryRecord(
  overrides: Partial<ExportInspection> = {},
): ExportInspection {
  return {
    id: "history-1",
    scheduleId: "schedule-1",
    jobId: "job-123",
    trigger: "schedule",
    status: "succeeded",
    title: "Export ready",
    message: "JSON export completed successfully.",
    destination: "/tmp/exports/run-1.json",
    exportedAt: "2026-03-05T00:05:00Z",
    completedAt: "2026-03-05T00:05:01Z",
    retryCount: 0,
    request: { format: "json" },
    artifact: {
      format: "json",
      filename: "run-1.json",
      contentType: "application/json",
      recordCount: 3,
      size: 512,
    },
    actions: [],
    ...overrides,
  };
}

function renderHistory(
  overrides: Partial<ComponentProps<typeof ExportScheduleHistory>> = {},
) {
  render(
    <ExportScheduleHistory
      scheduleName="Nightly Export"
      records={[makeHistoryRecord()]}
      total={1}
      limit={10}
      offset={0}
      loading={false}
      onClose={vi.fn()}
      onPageChange={vi.fn()}
      {...overrides}
    />,
  );
}

describe("ExportScheduleHistory", () => {
  it("renders a guided loading state while export history is loading", () => {
    renderHistory({ loading: true, records: [], total: 0 });

    expect(
      screen.getByRole("heading", { name: "Export History: Nightly Export" }),
    ).toBeInTheDocument();
    expect(screen.getByText("Loading export history")).toBeInTheDocument();
    expect(
      screen.getByText("Fetching recent export outcomes for this schedule."),
    ).toBeInTheDocument();
    expect(
      screen.queryByText("No export history found"),
    ).not.toBeInTheDocument();
  });

  it("renders a guided empty state when no persisted exports exist yet", () => {
    renderHistory({ records: [], total: 0 });

    expect(screen.getByText("No export history found")).toBeInTheDocument();
    expect(
      screen.getByText(
        "History will appear when jobs matching this schedule are exported.",
      ),
    ).toBeInTheDocument();
  });

  it("renders failure recovery actions with correct action semantics", () => {
    renderHistory({
      records: [
        makeHistoryRecord({
          status: "failed",
          title: "Export failed",
          message: "Webhook delivery timed out.",
          failure: {
            category: "timeout",
            summary: "Webhook delivery timed out.",
            retryable: true,
            terminal: true,
          },
          actions: [
            {
              label: "Review export automation settings",
              kind: "route",
              value: "/automation/exports",
            },
            {
              label: "Retry export from the CLI",
              kind: "command",
              value: "spartan export --job-id job-123 --format json",
            },
          ],
        }),
      ],
    });

    expect(screen.getByText("Timeout issue")).toBeInTheDocument();
    expect(
      screen.getByText("This outcome looks retryable."),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", {
        name: "Review export automation settings",
      }),
    ).toBeInTheDocument();
    expect(
      screen.getByLabelText("Copy Retry export from the CLI"),
    ).toBeInTheDocument();
  });
});
