/**
 * FeedContainer Tests
 *
 * Tests for the FeedContainer component covering:
 * - Initial load and rendering
 * - Feed CRUD operations (create, update, delete)
 * - Feed checking
 * - Items retrieval
 * - Error handling
 *
 * @module FeedContainerTests
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { FeedContainer } from "./FeedContainer";

// Mock the API module
vi.mock("../../api", () => ({
  listFeeds: vi.fn(),
  createFeed: vi.fn(),
  updateFeed: vi.fn(),
  deleteFeed: vi.fn(),
  checkFeed: vi.fn(),
  listFeedItems: vi.fn(),
}));

// Mock the api-config module
vi.mock("../../lib/api-config", () => ({
  getApiBaseUrl: vi.fn(() => "http://localhost:8080"),
}));

// Mock the FeedManager component to simplify testing
vi.mock("../../components/FeedManager", () => ({
  FeedManager: (props: {
    feeds: Array<Record<string, unknown>>;
    onRefresh: () => void;
    onCreate: (feed: Record<string, unknown>) => Promise<void>;
    onUpdate: (id: string, feed: Record<string, unknown>) => Promise<void>;
    onDelete: (id: string) => Promise<void>;
    onCheck: (id: string) => Promise<Record<string, unknown> | undefined>;
    onGetItems?: (
      id: string,
    ) => Promise<Array<Record<string, unknown>> | undefined>;
    loading?: boolean;
  }) => (
    <div data-testid="feed-manager">
      <div data-testid="feed-count">{props.feeds.length}</div>
      <div data-testid="loading-state">{props.loading ? "true" : "false"}</div>
      <button data-testid="refresh-btn" onClick={props.onRefresh} type="button">
        Refresh
      </button>
      <button
        data-testid="create-btn"
        onClick={() =>
          props.onCreate({
            url: "https://example.com/rss.xml",
            feedType: "rss",
            intervalSeconds: 3600,
            enabled: true,
            autoScrape: true,
          })
        }
        type="button"
      >
        Create
      </button>
      <button
        data-testid="update-btn"
        onClick={() =>
          props.onUpdate("feed-1", {
            url: "https://example.com/updated.xml",
            feedType: "atom",
            intervalSeconds: 7200,
            enabled: false,
            autoScrape: false,
          })
        }
        type="button"
      >
        Update
      </button>
      <button
        data-testid="delete-btn"
        onClick={() => props.onDelete("feed-1")}
        type="button"
      >
        Delete
      </button>
      <button
        data-testid="check-btn"
        onClick={async () => {
          const result = await props.onCheck("feed-1");
          return result;
        }}
        type="button"
      >
        Check
      </button>
      <button
        data-testid="items-btn"
        onClick={async () => {
          const items = await props.onGetItems?.("feed-1");
          return items;
        }}
        type="button"
      >
        Items
      </button>
    </div>
  ),
}));

import {
  listFeeds,
  createFeed,
  updateFeed,
  deleteFeed,
  checkFeed,
  listFeedItems,
} from "../../api";

describe("FeedContainer", () => {
  const mockListFeeds = listFeeds as ReturnType<typeof vi.fn>;
  const mockCreateFeed = createFeed as ReturnType<typeof vi.fn>;
  const mockUpdateFeed = updateFeed as ReturnType<typeof vi.fn>;
  const mockDeleteFeed = deleteFeed as ReturnType<typeof vi.fn>;
  const mockCheckFeed = checkFeed as ReturnType<typeof vi.fn>;
  const mockListFeedItems = listFeedItems as ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders FeedManager with initial empty state", async () => {
    mockListFeeds.mockResolvedValueOnce({ data: { feeds: [] } });

    render(<FeedContainer />);

    await waitFor(() => {
      expect(screen.getByTestId("feed-manager")).toBeInTheDocument();
    });

    expect(screen.getByTestId("feed-count").textContent).toBe("0");
    expect(screen.getByTestId("loading-state").textContent).toBe("false");
  });

  it("calls listFeeds on mount", async () => {
    mockListFeeds.mockResolvedValueOnce({ data: { feeds: [] } });

    render(<FeedContainer />);

    await waitFor(() => {
      expect(mockListFeeds).toHaveBeenCalledWith({
        baseUrl: "http://localhost:8080",
      });
    });
  });

  it("loads and displays feeds", async () => {
    const mockFeeds = [
      {
        id: "feed-1",
        url: "https://example.com/rss.xml",
        feedType: "rss",
        intervalSeconds: 3600,
        enabled: true,
        autoScrape: true,
        createdAt: "2024-01-01T00:00:00Z",
        status: "active",
      },
      {
        id: "feed-2",
        url: "https://example.com/atom.xml",
        feedType: "atom",
        intervalSeconds: 7200,
        enabled: false,
        autoScrape: false,
        createdAt: "2024-01-02T00:00:00Z",
        status: "disabled",
      },
    ];

    mockListFeeds.mockResolvedValueOnce({ data: { feeds: mockFeeds } });

    render(<FeedContainer />);

    await waitFor(() => {
      expect(screen.getByTestId("feed-count").textContent).toBe("2");
    });
  });

  it("handles listFeeds error gracefully", async () => {
    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    mockListFeeds.mockResolvedValueOnce({
      error: { message: "Network error" },
    });

    render(<FeedContainer />);

    await waitFor(() => {
      expect(consoleSpy).toHaveBeenCalledWith(
        "Failed to load feeds:",
        expect.anything(),
      );
    });

    consoleSpy.mockRestore();
  });

  it("creates feed successfully", async () => {
    const user = userEvent.setup();

    mockListFeeds.mockResolvedValue({ data: { feeds: [] } });
    mockCreateFeed.mockResolvedValue({ data: undefined });

    render(<FeedContainer />);

    await waitFor(() => {
      expect(screen.getByTestId("create-btn")).toBeInTheDocument();
    });

    await user.click(screen.getByTestId("create-btn"));

    await waitFor(() => {
      expect(mockCreateFeed).toHaveBeenCalledWith({
        baseUrl: "http://localhost:8080",
        body: {
          url: "https://example.com/rss.xml",
          feedType: "rss",
          intervalSeconds: 3600,
          enabled: true,
          autoScrape: true,
        },
      });
    });
  });

  it("updates feed successfully", async () => {
    const user = userEvent.setup();

    mockListFeeds.mockResolvedValue({ data: { feeds: [] } });
    mockUpdateFeed.mockResolvedValue({ data: undefined });

    render(<FeedContainer />);

    await waitFor(() => {
      expect(screen.getByTestId("update-btn")).toBeInTheDocument();
    });

    await user.click(screen.getByTestId("update-btn"));

    await waitFor(() => {
      expect(mockUpdateFeed).toHaveBeenCalledWith({
        baseUrl: "http://localhost:8080",
        path: { id: "feed-1" },
        body: {
          url: "https://example.com/updated.xml",
          feedType: "atom",
          intervalSeconds: 7200,
          enabled: false,
          autoScrape: false,
        },
      });
    });
  });

  it("deletes feed successfully", async () => {
    const user = userEvent.setup();

    mockListFeeds.mockResolvedValue({ data: { feeds: [] } });
    mockDeleteFeed.mockResolvedValue({ data: undefined });

    render(<FeedContainer />);

    await waitFor(() => {
      expect(screen.getByTestId("delete-btn")).toBeInTheDocument();
    });

    await user.click(screen.getByTestId("delete-btn"));

    await waitFor(() => {
      expect(mockDeleteFeed).toHaveBeenCalledWith({
        baseUrl: "http://localhost:8080",
        path: { id: "feed-1" },
      });
    });
  });

  it("checks feed and returns result", async () => {
    const user = userEvent.setup();

    const checkResult = {
      feedId: "feed-1",
      checkedAt: "2024-01-01T00:00:00Z",
      newItems: [
        {
          guid: "item-1",
          title: "New Article",
          link: "https://example.com/article",
          pubDate: "2024-01-01T00:00:00Z",
        },
      ],
      totalItems: 10,
      feedTitle: "Test Feed",
    };

    mockListFeeds.mockResolvedValue({ data: { feeds: [] } });
    mockCheckFeed.mockResolvedValue({ data: checkResult });

    render(<FeedContainer />);

    await waitFor(() => {
      expect(screen.getByTestId("check-btn")).toBeInTheDocument();
    });

    await user.click(screen.getByTestId("check-btn"));

    await waitFor(() => {
      expect(mockCheckFeed).toHaveBeenCalledWith({
        baseUrl: "http://localhost:8080",
        path: { id: "feed-1" },
      });
    });
  });

  it("gets feed items successfully", async () => {
    const user = userEvent.setup();

    const mockItems = [
      {
        guid: "item-1",
        title: "Article 1",
        link: "https://example.com/article1",
        seenAt: "2024-01-01T00:00:00Z",
      },
      {
        guid: "item-2",
        title: "Article 2",
        link: "https://example.com/article2",
        seenAt: "2024-01-02T00:00:00Z",
      },
    ];

    mockListFeeds.mockResolvedValue({ data: { feeds: [] } });
    mockListFeedItems.mockResolvedValue({ data: { items: mockItems } });

    render(<FeedContainer />);

    await waitFor(() => {
      expect(screen.getByTestId("items-btn")).toBeInTheDocument();
    });

    await user.click(screen.getByTestId("items-btn"));

    await waitFor(() => {
      expect(mockListFeedItems).toHaveBeenCalledWith({
        baseUrl: "http://localhost:8080",
        path: { id: "feed-1" },
      });
    });
  });

  it("refreshes feeds on create success", async () => {
    const user = userEvent.setup();

    mockListFeeds.mockResolvedValue({ data: { feeds: [] } });
    mockCreateFeed.mockResolvedValue({ data: undefined });

    render(<FeedContainer />);

    await waitFor(() => {
      expect(screen.getByTestId("create-btn")).toBeInTheDocument();
    });

    // Reset mock to count calls after initial load
    mockListFeeds.mockClear();

    await user.click(screen.getByTestId("create-btn"));

    await waitFor(() => {
      // Should be called for initial load + refresh after create
      expect(mockListFeeds).toHaveBeenCalled();
    });
  });

  it("refreshes feeds on delete success", async () => {
    const user = userEvent.setup();

    mockListFeeds.mockResolvedValue({ data: { feeds: [] } });
    mockDeleteFeed.mockResolvedValue({ data: undefined });

    render(<FeedContainer />);

    await waitFor(() => {
      expect(screen.getByTestId("delete-btn")).toBeInTheDocument();
    });

    mockListFeeds.mockClear();

    await user.click(screen.getByTestId("delete-btn"));

    await waitFor(() => {
      expect(mockListFeeds).toHaveBeenCalled();
    });
  });

  it("refreshes feeds on check success", async () => {
    const user = userEvent.setup();

    mockListFeeds.mockResolvedValue({ data: { feeds: [] } });
    mockCheckFeed.mockResolvedValue({
      data: {
        feedId: "feed-1",
        checkedAt: "2024-01-01T00:00:00Z",
        newItems: [],
        totalItems: 0,
      },
    });

    render(<FeedContainer />);

    await waitFor(() => {
      expect(screen.getByTestId("check-btn")).toBeInTheDocument();
    });

    mockListFeeds.mockClear();

    await user.click(screen.getByTestId("check-btn"));

    await waitFor(() => {
      expect(mockListFeeds).toHaveBeenCalled();
    });
  });
});
