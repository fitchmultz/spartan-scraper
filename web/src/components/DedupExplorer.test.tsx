/**
 * DedupExplorer Component Tests
 *
 * @module DedupExplorerTests
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { DedupExplorer } from "./DedupExplorer";
import * as api from "../api";

// Mock the API module
vi.mock("../api", () => ({
  findDuplicates: vi.fn(),
  getContentHistory: vi.fn(),
  getDedupStats: vi.fn(),
}));

// Mock the api-config module
vi.mock("../lib/api-config", () => ({
  getApiBaseUrl: vi.fn(() => "http://localhost:8741"),
}));

describe("DedupExplorer", () => {
  const mockStats = {
    totalIndexed: 1000,
    uniqueUrls: 800,
    uniqueJobs: 50,
    duplicatePairs: 200,
  };

  const mockDuplicates = [
    {
      jobId: "job-1",
      url: "https://example.com/page1",
      simhash: 1234567890,
      distance: 2,
      indexedAt: "2026-01-01T00:00:00Z",
    },
  ];

  const mockHistory = [
    {
      jobId: "job-1",
      simhash: 1234567890,
      indexedAt: "2026-01-01T00:00:00Z",
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(api.getDedupStats).mockReturnValue(
      Promise.resolve({
        data: mockStats,
        request: new Request("http://localhost:8741/v1/dedup/stats"),
        response: new Response(),
      }) as ReturnType<typeof api.getDedupStats>,
    );
  });

  it("renders with search tab active by default", () => {
    // Override mock to return a never-resolving promise to prevent
    // post-test state updates that trigger act() warnings
    vi.mocked(api.getDedupStats).mockReturnValue(
      new Promise(() => {}) as ReturnType<typeof api.getDedupStats>,
    );

    render(<DedupExplorer />);
    expect(screen.getByText(/find duplicates/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/simhash value/i)).toBeInTheDocument();
  });

  it("loads stats on mount", async () => {
    render(<DedupExplorer />);

    await waitFor(() => {
      expect(api.getDedupStats).toHaveBeenCalledWith({
        baseUrl: "http://localhost:8741",
      });
    });
  });

  it("switches to history tab", async () => {
    render(<DedupExplorer />);

    // Click the URL History tab button
    const tabButtons = screen.getAllByRole("button");
    const historyTab = tabButtons.find((btn) =>
      btn.textContent?.includes("URL History"),
    );
    if (historyTab) fireEvent.click(historyTab);

    await waitFor(() => {
      expect(screen.getByLabelText(/url/i)).toBeInTheDocument();
    });
  });

  it("switches to stats tab and displays statistics", async () => {
    render(<DedupExplorer />);

    // Click the Statistics tab button
    const tabButtons = screen.getAllByRole("button");
    const statsTab = tabButtons.find((btn) =>
      btn.textContent?.includes("Statistics"),
    );
    if (statsTab) fireEvent.click(statsTab);

    await waitFor(() => {
      expect(screen.getByText("1,000")).toBeInTheDocument();
    });

    expect(screen.getByText("800")).toBeInTheDocument();
    expect(screen.getByText("50")).toBeInTheDocument();
    expect(screen.getByText("200")).toBeInTheDocument();
  });

  it("calls findDuplicates API with correct params", async () => {
    vi.mocked(api.findDuplicates).mockReturnValue(
      Promise.resolve({
        data: mockDuplicates,
        request: new Request("http://localhost:8741/v1/dedup/duplicates"),
        response: new Response(),
      }) as ReturnType<typeof api.findDuplicates>,
    );

    render(<DedupExplorer />);

    // Wait for component to be fully rendered
    await waitFor(() => {
      expect(screen.getByLabelText(/simhash value/i)).toBeInTheDocument();
    });

    const simhashInput = screen.getByLabelText(/simhash value/i);
    fireEvent.change(simhashInput, { target: { value: "1234567890" } });

    // Find the action button inside the form section (not the tab)
    const formSection = screen
      .getByLabelText(/simhash value/i)
      .closest("section");
    const findButton = formSection?.querySelector(
      "button.dedup-explorer__button",
    );

    // Click the button
    if (findButton) {
      fireEvent.click(findButton);
    }

    await waitFor(() => {
      expect(api.findDuplicates).toHaveBeenCalledWith({
        baseUrl: "http://localhost:8741",
        query: { simhash: 1234567890, threshold: 3 },
      });
    });
  });

  it("calls getContentHistory API with correct params", async () => {
    vi.mocked(api.getContentHistory).mockReturnValue(
      Promise.resolve({
        data: mockHistory,
        request: new Request("http://localhost:8741/v1/dedup/history"),
        response: new Response(),
      }) as ReturnType<typeof api.getContentHistory>,
    );

    render(<DedupExplorer />);

    // Switch to history tab
    const tabButtons = screen.getAllByRole("button");
    const historyTab = tabButtons.find((btn) =>
      btn.textContent?.includes("URL History"),
    );
    if (historyTab) fireEvent.click(historyTab);

    await waitFor(() => {
      expect(screen.getByLabelText(/url/i)).toBeInTheDocument();
    });

    const urlInput = screen.getByLabelText(/url/i);
    fireEvent.change(urlInput, {
      target: { value: "https://example.com/page" },
    });

    const buttons = screen.getAllByRole("button");
    const getHistoryButton = buttons.find((btn) =>
      btn.textContent?.includes("Get History"),
    );
    if (getHistoryButton) fireEvent.click(getHistoryButton);

    await waitFor(() => {
      expect(api.getContentHistory).toHaveBeenCalledWith({
        baseUrl: "http://localhost:8741",
        query: { url: "https://example.com/page" },
      });
    });
  });

  it("adjusts threshold with slider", () => {
    // Override mock to return a never-resolving promise to prevent
    // post-test state updates that trigger act() warnings
    vi.mocked(api.getDedupStats).mockReturnValue(
      new Promise(() => {}) as ReturnType<typeof api.getDedupStats>,
    );

    render(<DedupExplorer />);

    const slider = screen.getByLabelText(/threshold/i);
    fireEvent.change(slider, { target: { value: "5" } });

    expect(screen.getByText(/threshold: 5/i)).toBeInTheDocument();
  });

  it("triggers stats refresh", async () => {
    render(<DedupExplorer />);

    // Wait for initial load
    await waitFor(() => {
      expect(api.getDedupStats).toHaveBeenCalled();
    });

    const initialCalls = vi.mocked(api.getDedupStats).mock.calls.length;

    // Click the Statistics tab button
    const tabButtons = screen.getAllByRole("button");
    const statsTab = tabButtons.find((btn) =>
      btn.textContent?.includes("Statistics"),
    );
    if (statsTab) fireEvent.click(statsTab);

    // Find and click refresh button
    await waitFor(() => {
      const buttons = screen.getAllByRole("button");
      const refreshButton = buttons.find((btn) =>
        btn.textContent?.includes("Refresh Stats"),
      );
      if (refreshButton) fireEvent.click(refreshButton);
    });

    // Should have more calls than initial
    await waitFor(() => {
      expect(vi.mocked(api.getDedupStats).mock.calls.length).toBeGreaterThan(
        initialCalls,
      );
    });
  });
});
