/**
 * WatchContainer - Container component for watch management functionality
 *
 * This component encapsulates all watch-related state and operations:
 * - Loading and displaying watches
 * - Creating, updating, and deleting watches
 * - Checking watch status
 *
 * It does NOT handle:
 * - Job submission or results viewing
 * - Chain management
 * - Batch operations
 *
 * @module WatchContainer
 */

import { useCallback, useEffect, useState, Suspense, lazy } from "react";
import {
  listWatches,
  createWatch,
  updateWatch,
  deleteWatch,
  checkWatch,
  type Watch,
  type WatchInput,
  type WatchCheckResult,
} from "../../api";
import { getApiBaseUrl } from "../../lib/api-config";

const WatchManager = lazy(() =>
  import("../../components/WatchManager").then((mod) => ({
    default: mod.WatchManager,
  })),
);

export function WatchContainer() {
  const [watches, setWatches] = useState<Watch[]>([]);
  const [watchesLoading, setWatchesLoading] = useState(false);

  const refreshWatches = useCallback(async () => {
    setWatchesLoading(true);
    try {
      const { data, error } = await listWatches({ baseUrl: getApiBaseUrl() });
      if (error) {
        console.error("Failed to load watches:", error);
        return;
      }
      setWatches(data?.watches || []);
    } catch (err) {
      console.error("Error loading watches:", err);
    } finally {
      setWatchesLoading(false);
    }
  }, []);

  const handleCreateWatch = useCallback(
    async (input: WatchInput) => {
      const { error } = await createWatch({
        baseUrl: getApiBaseUrl(),
        body: input,
      });
      if (error) throw error;
      await refreshWatches();
    },
    [refreshWatches],
  );

  const handleUpdateWatch = useCallback(
    async (id: string, input: WatchInput) => {
      const { error } = await updateWatch({
        baseUrl: getApiBaseUrl(),
        path: { id },
        body: input,
      });
      if (error) throw error;
      await refreshWatches();
    },
    [refreshWatches],
  );

  const handleDeleteWatch = useCallback(
    async (id: string) => {
      const { error } = await deleteWatch({
        baseUrl: getApiBaseUrl(),
        path: { id },
      });
      if (error) throw error;
      await refreshWatches();
    },
    [refreshWatches],
  );

  const handleCheckWatch = useCallback(
    async (id: string): Promise<WatchCheckResult | undefined> => {
      const { data, error } = await checkWatch({
        baseUrl: getApiBaseUrl(),
        path: { id },
      });
      if (error) throw error;
      await refreshWatches();
      return data;
    },
    [refreshWatches],
  );

  useEffect(() => {
    refreshWatches();
  }, [refreshWatches]);

  return (
    <section id="watches">
      <Suspense
        fallback={
          <div className="loading-placeholder">Loading watch manager...</div>
        }
      >
        <WatchManager
          watches={watches}
          onRefresh={refreshWatches}
          onCreate={handleCreateWatch}
          onUpdate={handleUpdateWatch}
          onDelete={handleDeleteWatch}
          onCheck={handleCheckWatch}
          loading={watchesLoading}
        />
      </Suspense>
    </section>
  );
}
