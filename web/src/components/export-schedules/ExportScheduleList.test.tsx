/**
 * Purpose: Verify the export schedule list renders its own rows and sorting behavior.
 * Responsibilities: Assert newest-first ordering and confirm empty arrays stay header-only because empty/loading states belong to ExportScheduleManager.
 * Scope: ExportScheduleList presentation only.
 * Usage: Run with `pnpm test`.
 * Invariants/Assumptions: ExportScheduleManager owns empty/loading messaging while ExportScheduleList owns row ordering and row rendering.
 */

import { render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { ExportSchedule } from "../../api";
import { ExportScheduleList } from "./ExportScheduleList";

function makeSchedule(overrides: Partial<ExportSchedule> = {}): ExportSchedule {
  return {
    id: "schedule-1",
    name: "Older Export",
    enabled: true,
    created_at: "2026-03-18T00:00:00Z",
    updated_at: "2026-03-18T00:00:00Z",
    filters: {},
    export: {
      format: "json",
      destination_type: "local",
      local_path: "/tmp/exports",
    },
    ...overrides,
  };
}

function renderScheduleList(schedules: ExportSchedule[]) {
  render(
    <ExportScheduleList
      schedules={schedules}
      historyLoadingId={null}
      deleteConfirmId={null}
      onEdit={vi.fn()}
      onDelete={vi.fn()}
      onToggleEnabled={vi.fn()}
      onViewHistory={vi.fn()}
      onDeleteConfirm={vi.fn()}
    />,
  );
}

describe("ExportScheduleList", () => {
  it("renders schedules newest-first", () => {
    renderScheduleList([
      makeSchedule({
        id: "schedule-older",
        name: "Older Export",
        created_at: "2026-03-18T00:00:00Z",
      }),
      makeSchedule({
        id: "schedule-newer",
        name: "Newer Export",
        created_at: "2026-03-19T00:00:00Z",
      }),
    ]);

    const rows = screen.getAllByRole("row");
    expect(
      within(rows[1] as HTMLElement).getByText("Newer Export"),
    ).toBeInTheDocument();
    expect(
      within(rows[2] as HTMLElement).getByText("Older Export"),
    ).toBeInTheDocument();
  });

  it("renders only the header row when the schedule array is empty", () => {
    renderScheduleList([]);

    expect(
      screen.getByRole("columnheader", { name: "Name" }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("columnheader", { name: "Status" }),
    ).toBeInTheDocument();
    expect(screen.getAllByRole("row")).toHaveLength(1);
    expect(
      screen.queryByText("No export schedules yet"),
    ).not.toBeInTheDocument();
  });
});
