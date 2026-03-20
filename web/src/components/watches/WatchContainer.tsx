/**
 * Purpose: Coordinate watch-management data loading and mutations for the automation route.
 * Responsibilities: Fetch the authoritative watch list, proxy create/update/delete/check actions to the API, and surface success/failure feedback through the shared toast layer.
 * Scope: Watch route container behavior only; form/list presentation stays inside `WatchManager`.
 * Usage: Render from the automation route wherever the watch workspace should appear.
 * Invariants/Assumptions: The server remains the source of truth for watch state, post-mutation refreshes reconcile local state, and operator-triggered checks always surface completion feedback.
 */

import { useCallback, useEffect, useState, Suspense, lazy } from "react";
import {
  listWatches,
  createWatch,
  updateWatch,
  deleteWatch,
  checkWatch,
  getWatchHistory,
  getWatchCheck,
  type Watch,
  type WatchCheckInspection,
  type WatchCheckHistoryResponse,
  type WatchInput,
} from "../../api";
import { getApiBaseUrl } from "../../lib/api-config";
import { getApiErrorMessage } from "../../lib/api-errors";
import { useToast } from "../toast";
import type { WatchPromotionSeed } from "../../types/promotion";

const WatchManager = lazy(() =>
  import("../../components/WatchManager").then((mod) => ({
    default: mod.WatchManager,
  })),
);

interface WatchContainerProps {
  promotionSeed?: WatchPromotionSeed | null;
  onClearPromotionSeed?: () => void;
  onOpenSourceJob?: (jobId: string) => void;
}

export function WatchContainer({
  promotionSeed = null,
  onClearPromotionSeed,
  onOpenSourceJob,
}: WatchContainerProps) {
  const toast = useToast();
  const [watches, setWatches] = useState<Watch[]>([]);
  const [watchesLoading, setWatchesLoading] = useState(false);

  const refreshWatches = useCallback(async () => {
    setWatchesLoading(true);
    try {
      const { data, error } = await listWatches({ baseUrl: getApiBaseUrl() });
      if (error) {
        const message = getApiErrorMessage(error, "Failed to load watches.");
        console.error("Failed to load watches:", error);
        toast.show({
          tone: "error",
          title: "Failed to load watches",
          description: message,
        });
        return;
      }
      setWatches(data?.watches || []);
    } catch (err) {
      console.error("Error loading watches:", err);
      toast.show({
        tone: "error",
        title: "Failed to load watches",
        description: getApiErrorMessage(err, "Failed to load watches."),
      });
    } finally {
      setWatchesLoading(false);
    }
  }, [toast]);

  const handleCreateWatch = useCallback(
    async (input: WatchInput) => {
      const toastId = toast.show({
        tone: "loading",
        title: "Creating watch",
        description: "Saving the new watch configuration.",
      });
      const { error } = await createWatch({
        baseUrl: getApiBaseUrl(),
        body: input,
      });
      if (error) {
        const message = getApiErrorMessage(error, "Failed to create watch.");
        toast.update(toastId, {
          tone: "error",
          title: "Failed to create watch",
          description: message,
        });
        throw error;
      }
      await refreshWatches();
      toast.update(toastId, {
        tone: "success",
        title: "Watch created",
        description: `${input.url} is now being monitored for change detection.`,
      });
    },
    [refreshWatches, toast],
  );

  const handleUpdateWatch = useCallback(
    async (id: string, input: WatchInput) => {
      const toastId = toast.show({
        tone: "loading",
        title: "Updating watch",
        description: "Saving the latest watch changes.",
      });
      const { error } = await updateWatch({
        baseUrl: getApiBaseUrl(),
        path: { id },
        body: input,
      });
      if (error) {
        const message = getApiErrorMessage(error, "Failed to update watch.");
        toast.update(toastId, {
          tone: "error",
          title: "Failed to update watch",
          description: message,
        });
        throw error;
      }
      await refreshWatches();
      toast.update(toastId, {
        tone: "success",
        title: "Watch updated",
        description: `${input.url} now reflects the latest watch settings.`,
      });
    },
    [refreshWatches, toast],
  );

  const handleDeleteWatch = useCallback(
    async (id: string) => {
      const toastId = toast.show({
        tone: "loading",
        title: "Deleting watch",
        description: "Removing the saved watch configuration.",
      });
      const { error } = await deleteWatch({
        baseUrl: getApiBaseUrl(),
        path: { id },
      });
      if (error) {
        const message = getApiErrorMessage(error, "Failed to delete watch.");
        toast.update(toastId, {
          tone: "error",
          title: "Failed to delete watch",
          description: message,
        });
        throw error;
      }
      await refreshWatches();
      toast.update(toastId, {
        tone: "success",
        title: "Watch deleted",
        description: "The selected watch has been removed.",
      });
    },
    [refreshWatches, toast],
  );

  const handleCheckWatch = useCallback(
    async (id: string): Promise<WatchCheckInspection | undefined> => {
      const toastId = toast.show({
        tone: "loading",
        title: "Running watch check",
        description:
          "Comparing the latest response against the stored baseline.",
      });
      const { data, error } = await checkWatch({
        baseUrl: getApiBaseUrl(),
        path: { id },
      });
      if (error) {
        const message = getApiErrorMessage(error, "Failed to run watch check.");
        toast.update(toastId, {
          tone: "error",
          title: "Failed to run watch check",
          description: message,
        });
        throw error;
      }
      await refreshWatches();
      const inspection = data?.check;
      toast.update(toastId, {
        tone: "success",
        title: "Watch check finished",
        description: inspection?.changed
          ? "Spartan detected a change in the watched target."
          : inspection?.baseline
            ? "Baseline recorded for future watch comparisons."
            : "No change was detected in the watched target.",
      });
      return inspection;
    },
    [refreshWatches, toast],
  );

  const handleLoadWatchHistory = useCallback(
    async (
      watchId: string,
      limit: number,
      offset: number,
    ): Promise<WatchCheckHistoryResponse | undefined> => {
      const { data, error } = await getWatchHistory({
        baseUrl: getApiBaseUrl(),
        path: { id: watchId },
        query: { limit, offset },
      });
      if (error) {
        const message = getApiErrorMessage(
          error,
          "Failed to load watch history.",
        );
        toast.show({
          tone: "error",
          title: "Failed to load watch history",
          description: message,
        });
        throw error;
      }
      return data;
    },
    [toast],
  );

  const handleLoadWatchHistoryDetail = useCallback(
    async (
      watchId: string,
      checkId: string,
    ): Promise<WatchCheckInspection | undefined> => {
      const { data, error } = await getWatchCheck({
        baseUrl: getApiBaseUrl(),
        path: { id: watchId, checkId },
      });
      if (error) {
        const message = getApiErrorMessage(
          error,
          "Failed to load watch check details.",
        );
        toast.show({
          tone: "error",
          title: "Failed to load watch check",
          description: message,
        });
        throw error;
      }
      return data?.check;
    },
    [toast],
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
          onLoadHistory={handleLoadWatchHistory}
          onLoadHistoryDetail={handleLoadWatchHistoryDetail}
          loading={watchesLoading}
          promotionSeed={promotionSeed}
          onClearPromotionSeed={onClearPromotionSeed}
          onOpenSourceJob={onOpenSourceJob}
        />
      </Suspense>
    </section>
  );
}
