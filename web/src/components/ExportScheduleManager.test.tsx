/**
 * Tests for ExportScheduleManager.
 *
 * Verifies schedule list actions trigger the expected callbacks and that the
 * export history modal loads and paginates correctly.
 */
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";
import type {
  ExportHistoryRecord,
  ExportSchedule,
  ExportScheduleRequest,
} from "../api";
import { ExportScheduleManager } from "./ExportScheduleManager";

function makeSchedule(overrides: Partial<ExportSchedule> = {}): ExportSchedule {
  return {
    id: "schedule-1",
    name: "Nightly Export",
    enabled: true,
    created_at: "2026-03-05T00:00:00Z",
    updated_at: "2026-03-05T00:00:00Z",
    filters: {},
    export: {
      format: "json",
      destination_type: "local",
      local_path: "/tmp/exports",
    },
    ...overrides,
  };
}

function makeHistoryRecord(
  overrides: Partial<ExportHistoryRecord> = {},
): ExportHistoryRecord {
  return {
    id: "history-1",
    schedule_id: "schedule-1",
    job_id: "job-123456789abc",
    status: "success",
    destination: "/tmp/exports/run-1.json",
    exported_at: "2026-03-05T00:05:00Z",
    record_count: 3,
    export_size: 512,
    ...overrides,
  };
}

function createManagerProps(
  overrides: Partial<ComponentProps<typeof ExportScheduleManager>> = {},
): ComponentProps<typeof ExportScheduleManager> {
  const onCreate = vi.fn<(request: ExportScheduleRequest) => Promise<void>>();
  const onUpdate =
    vi.fn<(id: string, request: ExportScheduleRequest) => Promise<void>>();
  const onDelete = vi.fn<(id: string) => Promise<void>>();
  const onToggleEnabled =
    vi.fn<(id: string, enabled: boolean) => Promise<void>>();
  const onGetHistory =
    vi.fn<
      (
        id: string,
        limit?: number,
        offset?: number,
      ) => Promise<{ records: ExportHistoryRecord[]; total: number }>
    >();

  onCreate.mockResolvedValue(undefined);
  onUpdate.mockResolvedValue(undefined);
  onDelete.mockResolvedValue(undefined);
  onToggleEnabled.mockResolvedValue(undefined);
  onGetHistory.mockResolvedValue({
    records: [makeHistoryRecord()],
    total: 1,
  });

  return {
    schedules: [makeSchedule()],
    onRefresh: vi.fn(),
    onCreate,
    onUpdate,
    onDelete,
    onToggleEnabled,
    onGetHistory,
    loading: false,
    ...overrides,
  };
}

describe("ExportScheduleManager", () => {
  it("loads and displays export history when History is clicked", async () => {
    const user = userEvent.setup();
    const props = createManagerProps();

    render(<ExportScheduleManager {...props} />);

    await user.click(screen.getByRole("button", { name: "History" }));

    await waitFor(() => {
      expect(props.onGetHistory).toHaveBeenCalledWith("schedule-1", 10, 0);
    });

    expect(
      await screen.findByRole("heading", {
        name: "Export History: Nightly Export",
      }),
    ).toBeInTheDocument();
    expect(screen.getByText("success")).toBeInTheDocument();
    expect(screen.getByText("Showing 1-1 of 1")).toBeInTheDocument();
  });

  it("requests the next history page when a pagination button is clicked", async () => {
    const user = userEvent.setup();
    const props = createManagerProps({
      onGetHistory: vi
        .fn()
        .mockResolvedValueOnce({
          records: [makeHistoryRecord()],
          total: 11,
        })
        .mockResolvedValueOnce({
          records: [
            makeHistoryRecord({
              id: "history-2",
              job_id: "job-222222222222",
              destination: "/tmp/exports/run-2.json",
            }),
          ],
          total: 11,
        }),
    });

    render(<ExportScheduleManager {...props} />);

    await user.click(screen.getByRole("button", { name: "History" }));

    expect(
      await screen.findByRole("heading", {
        name: "Export History: Nightly Export",
      }),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Next" }));

    await waitFor(() => {
      expect(props.onGetHistory).toHaveBeenNthCalledWith(
        2,
        "schedule-1",
        10,
        10,
      );
    });

    expect(await screen.findByText("Showing 11-11 of 11")).toBeInTheDocument();
  });
});
