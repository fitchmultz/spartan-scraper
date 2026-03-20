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
  ExportInspection,
  ExportOutcomeListResponse,
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
  overrides: Partial<ExportInspection> = {},
): ExportInspection {
  return {
    id: "history-1",
    scheduleId: "schedule-1",
    jobId: "job-123456789abc",
    trigger: "schedule",
    status: "succeeded",
    title: "Export ready",
    message: "JSON export completed successfully with 3 record(s).",
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
    actions: [
      {
        label: "Inspect export from the CLI",
        kind: "command",
        value: "spartan export --inspect-id history-1",
      },
    ],
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
      ) => Promise<ExportOutcomeListResponse>
    >();

  onCreate.mockResolvedValue(undefined);
  onUpdate.mockResolvedValue(undefined);
  onDelete.mockResolvedValue(undefined);
  onToggleEnabled.mockResolvedValue(undefined);
  onGetHistory.mockResolvedValue({
    exports: [makeHistoryRecord()],
    total: 1,
    limit: 10,
    offset: 0,
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
  it("opens a promotion-seeded export schedule draft with source context", async () => {
    render(
      <ExportScheduleManager
        {...createManagerProps({ schedules: [] })}
        promotionSeed={{
          kind: "export-schedule",
          source: {
            jobId: "job-123",
            jobKind: "scrape",
            jobStatus: "succeeded",
            label: "Source URL",
            value: "https://example.com/pricing",
          },
          formData: {
            name: "verified-export",
            enabled: true,
            filterJobKinds: ["scrape"],
            filterJobStatus: ["succeeded"],
            filterTags: "",
            filterHasResults: true,
            format: "md",
            destinationType: "local",
            pathTemplate: "exports/{kind}/{job_id}.{format}",
            localPath: "exports/{kind}/{job_id}.{format}",
            webhookUrl: "",
            maxRetries: 3,
            baseDelayMs: 1000,
            transformExpression: "",
            transformLanguage: "jmespath",
            shapeTopLevelFields: "",
            shapeNormalizedFields: "",
            shapeEvidenceFields: "",
            shapeSummaryFields: "",
            shapeFieldLabels: "",
            shapeEmptyValue: "",
            shapeMultiValueJoin: "",
            shapeMarkdownTitle: "",
          },
          seededFormat: "md",
          carriedForward: [
            "A scrape filter scoped to future successful jobs like this one.",
          ],
          remainingDecisions: ["Confirm the destination before saving."],
          unsupportedCarryForward: [
            "Export schedules automate future matching completed jobs; they do not rerun this source job on a cadence.",
          ],
        }}
      />,
    );

    expect(
      await screen.findByRole("heading", { name: /create export schedule/i }),
    ).toBeInTheDocument();
    expect(screen.getByDisplayValue("verified-export")).toBeInTheDocument();
    expect(
      screen.getAllByText(/do not rerun this source job on a cadence/i).length,
    ).toBeGreaterThan(0);
  });

  it("loads and displays guided export history when History is clicked", async () => {
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
    expect(screen.getByText("Export ready")).toBeInTheDocument();
    expect(screen.getByText("succeeded")).toBeInTheDocument();
    expect(screen.getByText("Showing 1-1 of 1")).toBeInTheDocument();
    expect(screen.getByText("Inspect export from the CLI")).toBeInTheDocument();
  });

  it("requests the next history page when a pagination button is clicked", async () => {
    const user = userEvent.setup();
    const props = createManagerProps({
      onGetHistory: vi
        .fn()
        .mockResolvedValueOnce({
          exports: [makeHistoryRecord()],
          total: 11,
          limit: 10,
          offset: 0,
        })
        .mockResolvedValueOnce({
          exports: [
            makeHistoryRecord({
              id: "history-2",
              jobId: "job-222222222222",
              destination: "/tmp/exports/run-2.json",
              artifact: {
                format: "json",
                filename: "run-2.json",
                contentType: "application/json",
                recordCount: 1,
                size: 256,
              },
            }),
          ],
          total: 11,
          limit: 10,
          offset: 10,
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
