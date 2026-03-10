/**
 * Feed Manager Component
 *
 * Provides UI for managing RSS/Atom feed monitoring. Supports creating,
 * editing, deleting, and manually checking feeds. Displays feed status,
 * seen items, and check results.
 *
 * @module FeedManager
 */

import { useState, useCallback, useMemo } from "react";
import type { Feed, FeedInput, FeedCheckResult, SeenFeedItem } from "../api";
import { formatDateTime, formatSecondsAsDuration } from "../lib/formatting";
import { getFeedStatusTone } from "../lib/status-display";
import { StatusPill } from "./StatusPill";

interface FeedManagerProps {
  feeds: Feed[];
  onRefresh: () => void;
  onCreate: (feed: FeedInput) => Promise<void>;
  onUpdate: (id: string, feed: FeedInput) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
  onCheck: (id: string) => Promise<FeedCheckResult | undefined>;
  onGetItems?: (id: string) => Promise<SeenFeedItem[] | undefined>;
  loading?: boolean;
}

interface FeedFormData {
  url: string;
  feedType: "auto" | "rss" | "atom";
  intervalSeconds: number;
  enabled: boolean;
  autoScrape: boolean;
}

const defaultFormData: FeedFormData = {
  url: "",
  feedType: "auto",
  intervalSeconds: 3600,
  enabled: true,
  autoScrape: true,
};

function feedToFormData(feed: Feed): FeedFormData {
  return {
    url: feed.url,
    feedType: (feed.feedType as FeedFormData["feedType"]) || "auto",
    intervalSeconds: feed.intervalSeconds,
    enabled: feed.enabled ?? true,
    autoScrape: feed.autoScrape ?? true,
  };
}

function formDataToFeedInput(data: FeedFormData): FeedInput {
  return {
    url: data.url,
    feedType: data.feedType,
    intervalSeconds: data.intervalSeconds,
    enabled: data.enabled,
    autoScrape: data.autoScrape,
  };
}

export function FeedManager({
  feeds,
  onRefresh,
  onCreate,
  onUpdate,
  onDelete,
  onCheck,
  onGetItems,
  loading,
}: FeedManagerProps) {
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [formData, setFormData] = useState<FeedFormData>(defaultFormData);
  const [formError, setFormError] = useState<string | null>(null);
  const [formSubmitting, setFormSubmitting] = useState(false);
  const [checkResult, setCheckResult] = useState<FeedCheckResult | null>(null);
  const [checkingId, setCheckingId] = useState<string | null>(null);
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);
  const [itemsModalFeedId, setItemsModalFeedId] = useState<string | null>(null);
  const [feedItems, setFeedItems] = useState<SeenFeedItem[]>([]);
  const [itemsLoading, setItemsLoading] = useState(false);

  const sortedFeeds = useMemo(() => {
    return [...feeds].sort((a, b) => {
      return (
        new Date(b.createdAt || 0).getTime() -
        new Date(a.createdAt || 0).getTime()
      );
    });
  }, [feeds]);

  const handleCreateClick = useCallback(() => {
    setFormData(defaultFormData);
    setEditingId(null);
    setFormError(null);
    setShowForm(true);
  }, []);

  const handleEditClick = useCallback((feed: Feed) => {
    setFormData(feedToFormData(feed));
    setEditingId(feed.id);
    setFormError(null);
    setShowForm(true);
  }, []);

  const handleCloseForm = useCallback(() => {
    setShowForm(false);
    setEditingId(null);
    setFormError(null);
  }, []);

  const handleSubmit = useCallback(async () => {
    if (!formData.url.trim()) {
      setFormError("URL is required");
      return;
    }

    if (formData.intervalSeconds < 60) {
      setFormError("Interval must be at least 60 seconds");
      return;
    }

    setFormSubmitting(true);
    setFormError(null);

    try {
      const input = formDataToFeedInput(formData);
      if (editingId) {
        await onUpdate(editingId, input);
      } else {
        await onCreate(input);
      }
      setShowForm(false);
      setEditingId(null);
      onRefresh();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Failed to save feed");
    } finally {
      setFormSubmitting(false);
    }
  }, [formData, editingId, onCreate, onUpdate, onRefresh]);

  const handleCheck = useCallback(
    async (id: string) => {
      setCheckingId(id);
      setCheckResult(null);
      try {
        const result = await onCheck(id);
        if (result) {
          setCheckResult(result);
        }
      } catch (err) {
        console.error("Feed check failed:", err);
      } finally {
        setCheckingId(null);
      }
    },
    [onCheck],
  );

  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await onDelete(id);
        setDeleteConfirmId(null);
        onRefresh();
      } catch (err) {
        console.error("Failed to delete feed:", err);
      }
    },
    [onDelete, onRefresh],
  );

  const handleViewItems = useCallback(
    async (id: string) => {
      setItemsModalFeedId(id);
      setItemsLoading(true);
      setFeedItems([]);
      try {
        const items = await onGetItems?.(id);
        if (items) {
          setFeedItems(items);
        }
      } catch (err) {
        console.error("Failed to load feed items:", err);
      } finally {
        setItemsLoading(false);
      }
    },
    [onGetItems],
  );

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold">Feed Monitoring</h2>
          <p className="text-sm text-gray-500 dark:text-gray-400">
            Monitor RSS/Atom feeds and automatically create scrape jobs for new
            items
          </p>
        </div>
        <button
          type="button"
          onClick={handleCreateClick}
          className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors"
        >
          Add Feed
        </button>
      </div>

      {/* Check Result */}
      {checkResult && (
        <div className="bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg p-4">
          <div className="flex items-center justify-between">
            <h3 className="font-medium text-green-900 dark:text-green-100">
              Check Complete
            </h3>
            <button
              type="button"
              onClick={() => setCheckResult(null)}
              className="text-green-600 dark:text-green-400 hover:text-green-800"
            >
              Dismiss
            </button>
          </div>
          <p className="text-sm text-green-700 dark:text-green-300 mt-1">
            Found {checkResult.newItems.length} new items out of{" "}
            {checkResult.totalItems} total
          </p>
          {checkResult.feedTitle && (
            <p className="text-xs text-green-600 dark:text-green-400 mt-1">
              Feed: {checkResult.feedTitle}
            </p>
          )}
          {checkResult.newItems.length > 0 && (
            <ul className="mt-2 space-y-1">
              {checkResult.newItems.slice(0, 5).map((item) => (
                <li
                  key={item.guid}
                  className="text-sm text-green-700 dark:text-green-300 truncate"
                >
                  - {item.title}
                </li>
              ))}
              {checkResult.newItems.length > 5 && (
                <li className="text-xs text-green-600 dark:text-green-400">
                  ...and {checkResult.newItems.length - 5} more
                </li>
              )}
            </ul>
          )}
        </div>
      )}

      {/* Feed List */}
      <div className="bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 overflow-hidden">
        {loading ? (
          <div className="p-8 text-center text-gray-500">Loading feeds...</div>
        ) : sortedFeeds.length === 0 ? (
          <div className="p-8 text-center text-gray-500">
            <p>No feeds configured.</p>
            <p className="text-sm mt-1">
              Add a feed to start monitoring RSS/Atom sources.
            </p>
          </div>
        ) : (
          <div className="divide-y divide-gray-200 dark:divide-gray-700">
            {sortedFeeds.map((feed) => (
              <div
                key={feed.id}
                className="p-4 hover:bg-gray-50 dark:hover:bg-gray-700/50 transition-colors"
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <h3 className="font-medium truncate">{feed.url}</h3>
                      <StatusPill
                        label={feed.status ?? "unknown"}
                        tone={getFeedStatusTone(feed.status)}
                      />
                      {feed.autoScrape && (
                        <span className="px-2 py-0.5 text-xs rounded-full bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200">
                          auto-scrape
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-4 mt-1 text-sm text-gray-500 dark:text-gray-400">
                      <span>Type: {feed.feedType}</span>
                      <span>
                        Interval:{" "}
                        {formatSecondsAsDuration(feed.intervalSeconds)}
                      </span>
                      <span>
                        Last checked:{" "}
                        {formatDateTime(feed.lastCheckedAt, "Never")}
                      </span>
                    </div>
                    {feed.lastError && (
                      <p className="text-sm text-red-600 dark:text-red-400 mt-1">
                        Error: {feed.lastError}
                      </p>
                    )}
                  </div>
                  <div className="flex items-center gap-2 ml-4">
                    <button
                      type="button"
                      onClick={() => handleCheck(feed.id)}
                      disabled={checkingId === feed.id}
                      className="px-3 py-1 text-sm bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded transition-colors disabled:opacity-50"
                    >
                      {checkingId === feed.id ? "Checking..." : "Check"}
                    </button>
                    {onGetItems && (
                      <button
                        type="button"
                        onClick={() => handleViewItems(feed.id)}
                        className="px-3 py-1 text-sm bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded transition-colors"
                      >
                        Items
                      </button>
                    )}
                    <button
                      type="button"
                      onClick={() => handleEditClick(feed)}
                      className="px-3 py-1 text-sm bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 rounded transition-colors"
                    >
                      Edit
                    </button>
                    <button
                      type="button"
                      onClick={() => setDeleteConfirmId(feed.id)}
                      className="px-3 py-1 text-sm bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-300 hover:bg-red-200 dark:hover:bg-red-900/50 rounded transition-colors"
                    >
                      Delete
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Add/Edit Modal */}
      {showForm && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-md w-full p-6">
            <h3 className="text-lg font-semibold mb-4">
              {editingId ? "Edit Feed" : "Add Feed"}
            </h3>

            {formError && (
              <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded text-red-700 dark:text-red-300 text-sm">
                {formError}
              </div>
            )}

            <div className="space-y-4">
              <div>
                <label
                  htmlFor="feed-url"
                  className="block text-sm font-medium mb-1"
                >
                  Feed URL
                </label>
                <input
                  id="feed-url"
                  type="url"
                  value={formData.url}
                  onChange={(e) =>
                    setFormData({ ...formData, url: e.target.value })
                  }
                  placeholder="https://example.com/rss.xml"
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700"
                />
              </div>

              <div>
                <label
                  htmlFor="feed-type"
                  className="block text-sm font-medium mb-1"
                >
                  Feed Type
                </label>
                <select
                  id="feed-type"
                  value={formData.feedType}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      feedType: e.target.value as FeedFormData["feedType"],
                    })
                  }
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700"
                >
                  <option value="auto">Auto-detect</option>
                  <option value="rss">RSS 2.0</option>
                  <option value="atom">Atom 1.0</option>
                </select>
              </div>

              <div>
                <label
                  htmlFor="feed-interval"
                  className="block text-sm font-medium mb-1"
                >
                  Check Interval (seconds)
                </label>
                <input
                  id="feed-interval"
                  type="number"
                  min={60}
                  value={formData.intervalSeconds}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      intervalSeconds: parseInt(e.target.value, 10) || 3600,
                    })
                  }
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700"
                />
                <p className="text-xs text-gray-500 mt-1">
                  Minimum: 60 seconds
                </p>
              </div>

              <div className="flex items-center gap-4">
                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={formData.enabled}
                    onChange={(e) =>
                      setFormData({ ...formData, enabled: e.target.checked })
                    }
                    className="rounded"
                  />
                  <span className="text-sm">Enabled</span>
                </label>

                <label className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    checked={formData.autoScrape}
                    onChange={(e) =>
                      setFormData({ ...formData, autoScrape: e.target.checked })
                    }
                    className="rounded"
                  />
                  <span className="text-sm">Auto-scrape new items</span>
                </label>
              </div>
            </div>

            <div className="flex justify-end gap-3 mt-6">
              <button
                type="button"
                onClick={handleCloseForm}
                className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleSubmit}
                disabled={formSubmitting}
                className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors disabled:opacity-50"
              >
                {formSubmitting ? "Saving..." : editingId ? "Update" : "Add"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Delete Confirmation */}
      {deleteConfirmId && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-sm w-full p-6">
            <h3 className="text-lg font-semibold mb-2">Delete Feed?</h3>
            <p className="text-gray-600 dark:text-gray-400 text-sm mb-4">
              This will permanently delete this feed and all its seen items.
              This action cannot be undone.
            </p>
            <div className="flex justify-end gap-3">
              <button
                type="button"
                onClick={() => setDeleteConfirmId(null)}
                className="px-4 py-2 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 rounded-lg transition-colors"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => handleDelete(deleteConfirmId)}
                className="px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 transition-colors"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Items Modal */}
      {itemsModalFeedId && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center p-4 z-50">
          <div className="bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-2xl w-full max-h-[80vh] flex flex-col">
            <div className="p-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
              <h3 className="text-lg font-semibold">Feed Items</h3>
              <button
                type="button"
                onClick={() => setItemsModalFeedId(null)}
                className="text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
              >
                Close
              </button>
            </div>
            <div className="flex-1 overflow-auto p-4">
              {itemsLoading ? (
                <div className="text-center text-gray-500 py-8">
                  Loading items...
                </div>
              ) : feedItems.length === 0 ? (
                <div className="text-center text-gray-500 py-8">
                  No items seen yet.
                </div>
              ) : (
                <div className="space-y-3">
                  {feedItems.map((item) => (
                    <div
                      key={item.guid}
                      className="p-3 bg-gray-50 dark:bg-gray-700/50 rounded-lg"
                    >
                      <h4 className="font-medium truncate">{item.title}</h4>
                      <a
                        href={item.link}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-sm text-blue-600 dark:text-blue-400 hover:underline truncate block"
                      >
                        {item.link}
                      </a>
                      <p className="text-xs text-gray-500 mt-1">
                        Seen: {formatDateTime(item.seenAt, "Never")}
                      </p>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
