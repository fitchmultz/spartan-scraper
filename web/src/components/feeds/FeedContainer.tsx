/**
 * FeedContainer - Container component for feed monitoring functionality
 *
 * This component encapsulates all feed-related state and operations:
 * - Loading and displaying feeds
 * - Creating, updating, and deleting feeds
 * - Checking feeds for new items
 * - Viewing seen feed items
 *
 * It does NOT handle:
 * - Job submission or results viewing
 * - Watch management
 * - Batch operations
 *
 * @module FeedContainer
 */

import { useCallback, useEffect, useState, Suspense, lazy } from "react";
import {
  listFeeds,
  createFeed,
  updateFeed,
  deleteFeed,
  checkFeed,
  listFeedItems,
  type Feed,
  type FeedInput,
  type FeedCheckResult,
  type SeenFeedItem,
} from "../../api";
import { getApiBaseUrl } from "../../lib/api-config";

const FeedManager = lazy(() =>
  import("../../components/FeedManager").then((mod) => ({
    default: mod.FeedManager,
  })),
);

export function FeedContainer() {
  const [feeds, setFeeds] = useState<Feed[]>([]);
  const [feedsLoading, setFeedsLoading] = useState(false);

  const refreshFeeds = useCallback(async () => {
    setFeedsLoading(true);
    try {
      const { data, error } = await listFeeds({ baseUrl: getApiBaseUrl() });
      if (error) {
        console.error("Failed to load feeds:", error);
        return;
      }
      setFeeds(data?.feeds || []);
    } catch (err) {
      console.error("Error loading feeds:", err);
    } finally {
      setFeedsLoading(false);
    }
  }, []);

  const handleCreateFeed = useCallback(
    async (input: FeedInput) => {
      const { error } = await createFeed({
        baseUrl: getApiBaseUrl(),
        body: input,
      });
      if (error) throw error;
      await refreshFeeds();
    },
    [refreshFeeds],
  );

  const handleUpdateFeed = useCallback(
    async (id: string, input: FeedInput) => {
      const { error } = await updateFeed({
        baseUrl: getApiBaseUrl(),
        path: { id },
        body: input,
      });
      if (error) throw error;
      await refreshFeeds();
    },
    [refreshFeeds],
  );

  const handleDeleteFeed = useCallback(
    async (id: string) => {
      const { error } = await deleteFeed({
        baseUrl: getApiBaseUrl(),
        path: { id },
      });
      if (error) throw error;
      await refreshFeeds();
    },
    [refreshFeeds],
  );

  const handleCheckFeed = useCallback(
    async (id: string): Promise<FeedCheckResult | undefined> => {
      const { data, error } = await checkFeed({
        baseUrl: getApiBaseUrl(),
        path: { id },
      });
      if (error) throw error;
      await refreshFeeds();
      return data;
    },
    [refreshFeeds],
  );

  const handleGetItems = useCallback(
    async (id: string): Promise<SeenFeedItem[] | undefined> => {
      const { data, error } = await listFeedItems({
        baseUrl: getApiBaseUrl(),
        path: { id },
      });
      if (error) throw error;
      return data?.items || [];
    },
    [],
  );

  useEffect(() => {
    refreshFeeds();
  }, [refreshFeeds]);

  return (
    <section id="feeds">
      <Suspense
        fallback={
          <div className="loading-placeholder">Loading feed manager...</div>
        }
      >
        <FeedManager
          feeds={feeds}
          onRefresh={refreshFeeds}
          onCreate={handleCreateFeed}
          onUpdate={handleUpdateFeed}
          onDelete={handleDeleteFeed}
          onCheck={handleCheckFeed}
          onGetItems={handleGetItems}
          loading={feedsLoading}
        />
      </Suspense>
    </section>
  );
}
