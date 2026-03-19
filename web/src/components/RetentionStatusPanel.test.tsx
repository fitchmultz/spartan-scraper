/**
 * Purpose: Verify retention settings render guided explanations, cleanup controls, and preview outcomes.
 * Responsibilities: Assert disabled/opportunity guidance, dry-run behavior, destructive confirmation, and result rendering.
 * Scope: RetentionStatusPanel behavior only.
 * Usage: Run with Vitest as part of the web test suite.
 * Invariants/Assumptions: Retention is optional, preview remains the safe default, and operators should understand the next action before deleting data.
 */

import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "../api";
import { RetentionStatusPanel } from "./RetentionStatusPanel";

vi.mock("../api", async () => {
  const actual = await vi.importActual<Record<string, unknown>>("../api");
  return {
    ...actual,
    getRetentionStatus: vi.fn(),
    runRetentionCleanup: vi.fn(),
    postV1DiagnosticsAiCheck: vi.fn(),
    postV1DiagnosticsBrowserCheck: vi.fn(),
    postV1DiagnosticsProxyPoolCheck: vi.fn(),
  };
});

vi.mock("../lib/api-config", () => ({
  getApiBaseUrl: vi.fn(() => "http://localhost:8741"),
}));

describe("RetentionStatusPanel", () => {
  const mockStatus = {
    enabled: true,
    jobRetentionDays: 30,
    crawlStateDays: 90,
    maxJobs: 10000,
    maxStorageGB: 10,
    totalJobs: 5000,
    jobsEligible: 100,
    storageUsedMB: 5120,
    guidance: {
      status: "warning" as const,
      title: "Cleanup opportunity detected",
      message:
        "100 job(s) already meet the current cleanup policy. Preview a cleanup run before pressure becomes urgent.",
      actions: [
        {
          label: "Preview cleanup from the CLI",
          kind: "command" as const,
          value: "spartan retention cleanup --dry-run",
        },
      ],
    },
  };

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.getRetentionStatus).mockResolvedValue({
      data: mockStatus,
      request: new Request("http://localhost:8741/v1/retention/status"),
      response: new Response(),
    });
  });

  function renderPanel() {
    return render(
      <RetentionStatusPanel
        health={null}
        onNavigate={vi.fn()}
        onRefreshHealth={vi.fn()}
        onCreateJob={vi.fn()}
        onOpenAutomation={vi.fn()}
      />,
    );
  }

  it("renders loading state initially", () => {
    vi.mocked(api.getRetentionStatus).mockReturnValue(
      new Promise(() => {}) as ReturnType<typeof api.getRetentionStatus>,
    );

    renderPanel();
    expect(screen.getByText(/loading retention status/i)).toBeInTheDocument();
  });

  it("displays guided retention status after loading", async () => {
    renderPanel();

    expect(
      await screen.findByText("Cleanup opportunity detected"),
    ).toBeInTheDocument();
    expect(screen.getByText("5,000")).toBeInTheDocument();
    expect(screen.getByText("Yes")).toBeInTheDocument();
    expect(screen.getByText("100")).toBeInTheDocument();
    expect(
      screen.getByText("spartan retention cleanup --dry-run"),
    ).toBeInTheDocument();
  });

  it("shows disabled guidance when retention is off", async () => {
    vi.mocked(api.getRetentionStatus).mockResolvedValue({
      data: {
        ...mockStatus,
        enabled: false,
        guidance: {
          status: "disabled" as const,
          title: "Automatic retention is disabled",
          message:
            "Spartan will keep completed jobs and crawl state until you enable automatic cleanup or run targeted cleanup manually. Preview first so you understand the blast radius.",
          actions: [
            {
              label: "Enable retention in the environment",
              kind: "env" as const,
              value: "RETENTION_ENABLED=true",
            },
          ],
        },
      },
      request: new Request("http://localhost:8741/v1/retention/status"),
      response: new Response(),
    });

    renderPanel();

    expect(
      await screen.findByText("Automatic retention is disabled"),
    ).toBeInTheDocument();
    expect(screen.getByText("RETENTION_ENABLED=true")).toBeInTheDocument();
  });

  it("keeps disabled retention health quiet if health starts reporting retention", async () => {
    vi.mocked(api.getRetentionStatus).mockResolvedValue({
      data: {
        ...mockStatus,
        enabled: false,
        guidance: {
          status: "disabled" as const,
          title: "Automatic retention is disabled",
          message:
            "Spartan will keep completed jobs and crawl state until you enable automatic cleanup or run targeted cleanup manually. Preview first so you understand the blast radius.",
          actions: [],
        },
      },
      request: new Request("http://localhost:8741/v1/retention/status"),
      response: new Response(),
    });

    render(
      <RetentionStatusPanel
        health={{
          status: "ok",
          version: "test",
          components: {
            retention: {
              status: "disabled",
              message: "Retention is intentionally off.",
            },
          },
          notices: [],
        }}
        onNavigate={vi.fn()}
        onRefreshHealth={vi.fn()}
        onCreateJob={vi.fn()}
        onOpenAutomation={vi.fn()}
      />,
    );

    expect(
      await screen.findByText("Automatic retention is disabled"),
    ).toBeInTheDocument();
    expect(
      screen.queryByText("Retention needs attention"),
    ).not.toBeInTheDocument();
  });

  it("surfaces degraded retention health explicitly when retention health is present", async () => {
    render(
      <RetentionStatusPanel
        health={{
          status: "degraded",
          version: "test",
          components: {
            retention: {
              status: "degraded",
              message: "Retention metadata could not be refreshed.",
              actions: [
                {
                  label: "Refresh retention diagnostics",
                  kind: "copy",
                  value: "spartan retention status",
                },
              ],
            },
          },
          notices: [],
        }}
        onNavigate={vi.fn()}
        onRefreshHealth={vi.fn()}
        onCreateJob={vi.fn()}
        onOpenAutomation={vi.fn()}
      />,
    );

    expect(
      await screen.findByText("Retention needs attention"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("Retention metadata could not be refreshed."),
    ).toBeInTheDocument();
    expect(screen.getByText("spartan retention status")).toBeInTheDocument();
  });

  it("toggles dry-run mode", async () => {
    renderPanel();

    await waitFor(() => {
      expect(
        screen.getAllByRole("button", { name: /preview cleanup/i }).length,
      ).toBeGreaterThan(0);
    });

    const dryRunCheckbox = screen.getByLabelText(
      /dry-run mode/i,
    ) as HTMLInputElement;
    expect(dryRunCheckbox).toBeChecked();

    fireEvent.click(dryRunCheckbox);
    expect(dryRunCheckbox).not.toBeChecked();
  });

  it("calls cleanup API with dry-run on preview", async () => {
    vi.mocked(api.runRetentionCleanup).mockResolvedValue({
      data: {
        jobsDeleted: 10,
        jobsAttempted: 10,
        crawlStatesDeleted: 0,
        spaceReclaimedMB: 100,
        durationMs: 1000,
        dryRun: true,
      },
      request: new Request("http://localhost:8741/v1/retention/cleanup"),
      response: new Response(),
    });

    renderPanel();

    await waitFor(() => {
      expect(
        screen.getAllByRole("button", { name: /preview cleanup/i }).length,
      ).toBeGreaterThan(0);
    });

    fireEvent.click(
      screen.getAllByRole("button", { name: /preview cleanup/i })[0],
    );

    await waitFor(() => {
      expect(api.runRetentionCleanup).toHaveBeenCalledWith({
        baseUrl: "http://localhost:8741",
        body: { dryRun: true },
      });
    });
  });

  it("shows confirmation dialog for non-dry-run cleanup", async () => {
    renderPanel();

    await waitFor(() => {
      expect(screen.getByLabelText(/dry-run mode/i)).toBeInTheDocument();
    });

    fireEvent.click(screen.getByLabelText(/dry-run mode/i));
    fireEvent.click(screen.getByText(/run cleanup/i));

    await waitFor(() => {
      expect(screen.getByText(/warning/i)).toBeInTheDocument();
      expect(screen.getByText(/confirm delete/i)).toBeInTheDocument();
    });
  });

  it("displays cleanup results", async () => {
    vi.mocked(api.runRetentionCleanup).mockResolvedValue({
      data: {
        jobsDeleted: 50,
        jobsAttempted: 50,
        crawlStatesDeleted: 10,
        spaceReclaimedMB: 2048,
        durationMs: 2500,
        dryRun: true,
        failedJobIDs: [],
        errors: [],
      },
      request: new Request("http://localhost:8741/v1/retention/cleanup"),
      response: new Response(),
    });

    renderPanel();

    await waitFor(() => {
      expect(
        screen.getAllByRole("button", { name: /preview cleanup/i }).length,
      ).toBeGreaterThan(0);
    });

    fireEvent.click(
      screen.getAllByRole("button", { name: /preview cleanup/i })[0],
    );

    await waitFor(() => {
      expect(screen.getByText(/dry-run preview results/i)).toBeInTheDocument();
      expect(screen.getByText(/jobs would delete/i)).toBeInTheDocument();
      expect(screen.getByText("2.00 GB")).toBeInTheDocument();
    });

    expect(
      document.querySelector(".retention-notice--info"),
    ).toBeInTheDocument();
  });

  it("refreshes status when refresh button clicked", async () => {
    renderPanel();

    await waitFor(() => {
      expect(
        screen.getAllByRole("button", { name: /refresh status/i }).length,
      ).toBeGreaterThan(0);
    });

    fireEvent.click(
      screen.getAllByRole("button", { name: /refresh status/i })[0],
    );

    await waitFor(() => {
      expect(api.getRetentionStatus).toHaveBeenCalledTimes(2);
    });
  });
});
