/**
 * RetentionStatusPanel Component Tests
 *
 * Tests cover:
 * - Status display and formatting
 * - Dry-run toggle behavior
 * - Confirmation dialog for destructive actions
 * - Cleanup result display
 *
 * @module RetentionStatusPanelTests
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { RetentionStatusPanel } from "./RetentionStatusPanel";
import * as api from "../api";

// Mock the API module
vi.mock("../api", () => ({
  getRetentionStatus: vi.fn(),
  runRetentionCleanup: vi.fn(),
}));

// Mock the api-config module
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
  };

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.getRetentionStatus).mockResolvedValue({
      data: mockStatus,
      request: new Request("http://localhost:8741/v1/retention/status"),
      response: new Response(),
    });
  });

  it("renders loading state initially", () => {
    // Override mock to return a never-resolving promise to prevent
    // post-test state updates that trigger act() warnings
    vi.mocked(api.getRetentionStatus).mockReturnValue(
      new Promise(() => {}) as ReturnType<typeof api.getRetentionStatus>,
    );

    render(<RetentionStatusPanel />);
    expect(screen.getByText(/loading retention status/i)).toBeInTheDocument();
  });

  it("displays retention status after loading", async () => {
    render(<RetentionStatusPanel />);

    await waitFor(() => {
      expect(screen.getByText("5,000")).toBeInTheDocument(); // Total jobs
    });

    expect(screen.getByText("Yes")).toBeInTheDocument(); // Enabled
    expect(screen.getByText("100")).toBeInTheDocument(); // Eligible
  });

  it("shows warning when retention is disabled", async () => {
    vi.mocked(api.getRetentionStatus).mockResolvedValue({
      data: { ...mockStatus, enabled: false },
      request: new Request("http://localhost:8741/v1/retention/status"),
      response: new Response(),
    });

    render(<RetentionStatusPanel />);

    await waitFor(() => {
      expect(screen.getByText("No")).toBeInTheDocument();
    });
  });

  it("toggles dry-run mode", async () => {
    render(<RetentionStatusPanel />);

    await waitFor(() => {
      expect(screen.getByText(/preview cleanup/i)).toBeInTheDocument();
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

    render(<RetentionStatusPanel />);

    await waitFor(() => {
      expect(screen.getByText(/preview cleanup/i)).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText(/preview cleanup/i));

    await waitFor(() => {
      expect(api.runRetentionCleanup).toHaveBeenCalledWith({
        baseUrl: "http://localhost:8741",
        body: { dryRun: true },
      });
    });
  });

  it("shows confirmation dialog for non-dry-run cleanup", async () => {
    render(<RetentionStatusPanel />);

    await waitFor(() => {
      expect(screen.getByLabelText(/dry-run mode/i)).toBeInTheDocument();
    });

    // Uncheck dry-run
    fireEvent.click(screen.getByLabelText(/dry-run mode/i));

    // Click run cleanup
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

    render(<RetentionStatusPanel />);

    await waitFor(() => {
      expect(screen.getByText(/preview cleanup/i)).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText(/preview cleanup/i));

    await waitFor(() => {
      expect(screen.getByText(/dry-run preview results/i)).toBeInTheDocument();
      expect(screen.getByText(/jobs would delete/i)).toBeInTheDocument();
      expect(screen.getByText("2.00 GB")).toBeInTheDocument();
    });

    expect(
      document.querySelector(".retention-notice--info"),
    ).toBeInTheDocument();
  });

  it("renders configuration in a theme-aware card", async () => {
    render(<RetentionStatusPanel />);

    await waitFor(() => {
      expect(screen.getByText(/configuration/i)).toBeInTheDocument();
    });

    expect(
      document.querySelector(".retention-config-card"),
    ).toBeInTheDocument();
    expect(screen.getByText(/job retention/i)).toBeInTheDocument();
    expect(screen.getByText("30 days")).toBeInTheDocument();
  });

  it("refreshes status when refresh button clicked", async () => {
    render(<RetentionStatusPanel />);

    await waitFor(() => {
      expect(screen.getByText(/refresh status/i)).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText(/refresh status/i));

    await waitFor(() => {
      expect(api.getRetentionStatus).toHaveBeenCalledTimes(2);
    });
  });
});
