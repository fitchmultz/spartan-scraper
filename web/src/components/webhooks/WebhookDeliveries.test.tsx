/**
 * Purpose: Verify the webhook-delivery automation surface renders and filters persisted delivery history correctly.
 * Responsibilities: Assert list rendering, filter behavior, pagination, detail inspection, loading states, and error recovery paths.
 * Scope: `WebhookDeliveryContainer` integration behavior only.
 * Usage: Run with Vitest.
 * Invariants/Assumptions: API calls are mocked, route-local state is authoritative in the container, and detail responses refine the selected row view.
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { WebhookDeliveryContainer } from "./WebhookDeliveryContainer";

// Mock the API module
vi.mock("../../api", () => ({
  getV1WebhooksDeliveries: vi.fn(),
  getV1WebhooksDeliveriesById: vi.fn(),
}));

// Mock the api-config module
vi.mock("../../lib/api-config", () => ({
  getApiBaseUrl: vi.fn(() => "http://localhost:8080"),
}));

// Mock the WebhookDeliveries component to simplify testing
vi.mock("./WebhookDeliveries", () => ({
  WebhookDeliveries: (props: {
    deliveries: Array<Record<string, unknown>>;
    total: number;
    loading: boolean;
    filters: { jobId: string; status: string };
    limit: number;
    offset: number;
    selectedDelivery: Record<string, unknown> | null;
    detailLoading: boolean;
    onRefresh: () => void;
    onFilterChange: (filters: { jobId: string; status: string }) => void;
    onPageChange: (offset: number) => void;
    onViewDetail: (delivery: Record<string, unknown>) => void;
    onCloseDetail: () => void;
  }) => (
    <div data-testid="webhook-deliveries">
      <div data-testid="delivery-count">{props.deliveries.length}</div>
      <div data-testid="total-count">{props.total}</div>
      <div data-testid="loading-state">{props.loading ? "true" : "false"}</div>
      <div data-testid="detail-loading">
        {props.detailLoading ? "true" : "false"}
      </div>
      <div data-testid="current-job-id">{props.filters.jobId}</div>
      <div data-testid="current-status">{props.filters.status}</div>
      <div data-testid="current-offset">{props.offset}</div>
      <div data-testid="has-selected-delivery">
        {props.selectedDelivery ? "true" : "false"}
      </div>
      <button data-testid="refresh-btn" onClick={props.onRefresh} type="button">
        Refresh
      </button>
      <button
        data-testid="filter-btn"
        onClick={() =>
          props.onFilterChange({ jobId: "job-123", status: "failed" })
        }
        type="button"
      >
        Apply Filters
      </button>
      <button
        data-testid="reset-btn"
        onClick={() => props.onFilterChange({ jobId: "", status: "all" })}
        type="button"
      >
        Reset Filters
      </button>
      <button
        data-testid="next-page-btn"
        onClick={() => props.onPageChange(props.offset + props.limit)}
        type="button"
      >
        Next Page
      </button>
      <button
        data-testid="prev-page-btn"
        onClick={() =>
          props.onPageChange(Math.max(0, props.offset - props.limit))
        }
        type="button"
      >
        Previous Page
      </button>
      <button
        data-testid="view-detail-btn"
        onClick={() =>
          props.onViewDetail({
            id: "del-123",
            eventType: "job.completed",
            jobId: "job-456",
            url: "https://example.com/webhook",
            status: "delivered",
            attempts: 1,
            createdAt: "2024-01-01T00:00:00Z",
          })
        }
        type="button"
      >
        View Detail
      </button>
      <button
        data-testid="close-detail-btn"
        onClick={props.onCloseDetail}
        type="button"
      >
        Close Detail
      </button>
    </div>
  ),
}));

import {
  getV1WebhooksDeliveries,
  getV1WebhooksDeliveriesById,
} from "../../api";

describe("WebhookDeliveryContainer", () => {
  const mockGetDeliveries = getV1WebhooksDeliveries as ReturnType<typeof vi.fn>;
  const mockGetDeliveryDetail = getV1WebhooksDeliveriesById as ReturnType<
    typeof vi.fn
  >;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders WebhookDeliveries with initial empty state", async () => {
    mockGetDeliveries.mockResolvedValueOnce({
      data: { deliveries: [], total: 0 },
    });

    render(<WebhookDeliveryContainer />);

    await waitFor(() => {
      expect(screen.getByTestId("webhook-deliveries")).toBeInTheDocument();
    });

    expect(screen.getByTestId("delivery-count").textContent).toBe("0");
    expect(screen.getByTestId("total-count").textContent).toBe("0");
    expect(screen.getByTestId("loading-state").textContent).toBe("false");
  });

  it("calls getV1WebhooksDeliveries on mount", async () => {
    mockGetDeliveries.mockResolvedValueOnce({
      data: { deliveries: [], total: 0 },
    });

    render(<WebhookDeliveryContainer />);

    await waitFor(() => {
      expect(mockGetDeliveries).toHaveBeenCalledWith({
        baseUrl: "http://localhost:8080",
        query: {
          limit: 20,
          offset: 0,
        },
      });
    });
  });

  it("loads and displays deliveries", async () => {
    const mockDeliveries = [
      {
        id: "del-1",
        eventId: "evt-1",
        eventType: "job.completed",
        jobId: "job-1",
        url: "https://example.com/webhook",
        status: "delivered",
        attempts: 1,
        createdAt: "2024-01-01T00:00:00Z",
        updatedAt: "2024-01-01T00:00:01Z",
      },
      {
        id: "del-2",
        eventId: "evt-2",
        eventType: "job.failed",
        jobId: "job-2",
        url: "https://example.com/webhook",
        status: "failed",
        attempts: 3,
        createdAt: "2024-01-02T00:00:00Z",
        updatedAt: "2024-01-02T00:00:03Z",
        lastError: "Connection timeout",
      },
    ];

    mockGetDeliveries.mockResolvedValueOnce({
      data: { deliveries: mockDeliveries, total: 2 },
    });

    render(<WebhookDeliveryContainer />);

    await waitFor(() => {
      expect(screen.getByTestId("delivery-count").textContent).toBe("2");
    });

    expect(screen.getByTestId("total-count").textContent).toBe("2");
  });

  it("handles list deliveries error gracefully", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    mockGetDeliveries.mockResolvedValueOnce({
      error: { message: "Network error" },
    });

    render(<WebhookDeliveryContainer />);

    await waitFor(() => {
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining(
          "Failed to load webhook deliveries: Network error",
        ),
        expect.anything(),
      );
    });

    consoleSpy.mockRestore();
  });

  it("applies job ID filter and refreshes", async () => {
    const user = userEvent.setup();

    mockGetDeliveries.mockResolvedValue({
      data: { deliveries: [], total: 0 },
    });

    render(<WebhookDeliveryContainer />);

    await waitFor(() => {
      expect(screen.getByTestId("filter-btn")).toBeInTheDocument();
    });

    mockGetDeliveries.mockClear();

    await user.click(screen.getByTestId("filter-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("current-job-id").textContent).toBe("job-123");
      expect(screen.getByTestId("current-status").textContent).toBe("failed");
    });

    // Should reset offset to 0 when filter changes
    expect(screen.getByTestId("current-offset").textContent).toBe("0");
  });

  it("resets filters when reset is clicked", async () => {
    const user = userEvent.setup();

    mockGetDeliveries.mockResolvedValue({
      data: { deliveries: [], total: 0 },
    });

    render(<WebhookDeliveryContainer />);

    await waitFor(() => {
      expect(screen.getByTestId("reset-btn")).toBeInTheDocument();
    });

    await user.click(screen.getByTestId("reset-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("current-job-id").textContent).toBe("");
      expect(screen.getByTestId("current-status").textContent).toBe("all");
    });
  });

  it("changes page when next page is clicked", async () => {
    const user = userEvent.setup();

    mockGetDeliveries.mockResolvedValue({
      data: { deliveries: [], total: 50 },
    });

    render(<WebhookDeliveryContainer />);

    await waitFor(() => {
      expect(screen.getByTestId("next-page-btn")).toBeInTheDocument();
    });

    mockGetDeliveries.mockClear();

    await user.click(screen.getByTestId("next-page-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("current-offset").textContent).toBe("20");
    });
  });

  it("changes page when previous page is clicked", async () => {
    const user = userEvent.setup();

    mockGetDeliveries.mockResolvedValue({
      data: { deliveries: [], total: 50 },
    });

    render(<WebhookDeliveryContainer />);

    await waitFor(() => {
      expect(screen.getByTestId("next-page-btn")).toBeInTheDocument();
    });

    // Go to next page first
    await user.click(screen.getByTestId("next-page-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("current-offset").textContent).toBe("20");
    });

    // Then go back
    await user.click(screen.getByTestId("prev-page-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("current-offset").textContent).toBe("0");
    });
  });

  it("refreshes deliveries when refresh is clicked", async () => {
    const user = userEvent.setup();

    mockGetDeliveries.mockResolvedValue({
      data: { deliveries: [], total: 0 },
    });

    render(<WebhookDeliveryContainer />);

    await waitFor(() => {
      expect(screen.getByTestId("refresh-btn")).toBeInTheDocument();
    });

    mockGetDeliveries.mockClear();

    await user.click(screen.getByTestId("refresh-btn"));

    await waitFor(() => {
      expect(mockGetDeliveries).toHaveBeenCalled();
    });
  });

  it("loads delivery detail when view detail is clicked", async () => {
    const user = userEvent.setup();

    const mockDetail = {
      id: "del-123",
      eventType: "job.completed",
      jobId: "job-456",
      url: "https://example.com/webhook",
      status: "delivered",
      attempts: 1,
      createdAt: "2024-01-01T00:00:00Z",
      responseCode: 200,
      lastError: null,
    };

    mockGetDeliveries.mockResolvedValue({
      data: { deliveries: [], total: 0 },
    });

    mockGetDeliveryDetail.mockResolvedValue({
      data: mockDetail,
    });

    render(<WebhookDeliveryContainer />);

    await waitFor(() => {
      expect(screen.getByTestId("view-detail-btn")).toBeInTheDocument();
    });

    await user.click(screen.getByTestId("view-detail-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("has-selected-delivery").textContent).toBe(
        "true",
      );
    });
  });

  it("closes detail when close is clicked", async () => {
    const user = userEvent.setup();

    const mockDetail = {
      id: "del-123",
      eventType: "job.completed",
      jobId: "job-456",
      url: "https://example.com/webhook",
      status: "delivered",
      attempts: 1,
      createdAt: "2024-01-01T00:00:00Z",
    };

    mockGetDeliveries.mockResolvedValue({
      data: { deliveries: [], total: 0 },
    });

    mockGetDeliveryDetail.mockResolvedValue({
      data: mockDetail,
    });

    render(<WebhookDeliveryContainer />);

    await waitFor(() => {
      expect(screen.getByTestId("view-detail-btn")).toBeInTheDocument();
    });

    // Open detail
    await user.click(screen.getByTestId("view-detail-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("has-selected-delivery").textContent).toBe(
        "true",
      );
    });

    // Close detail
    await user.click(screen.getByTestId("close-detail-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("has-selected-delivery").textContent).toBe(
        "false",
      );
    });
  });

  it("handles detail load error gracefully", async () => {
    const user = userEvent.setup();
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});

    mockGetDeliveries.mockResolvedValue({
      data: { deliveries: [], total: 0 },
    });

    mockGetDeliveryDetail.mockResolvedValue({
      error: { message: "Not found" },
    });

    render(<WebhookDeliveryContainer />);

    await waitFor(() => {
      expect(screen.getByTestId("view-detail-btn")).toBeInTheDocument();
    });

    await user.click(screen.getByTestId("view-detail-btn"));

    await waitFor(() => {
      expect(consoleSpy).toHaveBeenCalledWith(
        expect.stringContaining("Failed to load delivery detail: Not found"),
        expect.anything(),
      );
    });

    consoleSpy.mockRestore();
  });
});
