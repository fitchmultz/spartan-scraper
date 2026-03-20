/**
 * Purpose: Coordinate export-schedule data loading and mutations for the automation route.
 * Responsibilities: Fetch the authoritative schedule list, proxy create/update/delete/toggle/history actions to the API, and surface operator-facing success/failure feedback through shared toasts.
 * Scope: Export-schedule container behavior only; modal/list presentation stays in `ExportScheduleManager`.
 * Usage: Render from the automation route with the app-level toast provider already mounted.
 * Invariants/Assumptions: The server remains the source of truth for schedules, post-mutation refreshes reconcile local state, and history-load failures should never disappear into console-only output.
 */

import { useCallback, useEffect, useState, Suspense, lazy } from "react";
import {
  listExportSchedules,
  createExportSchedule,
  getExportSchedule,
  updateExportSchedule,
  deleteExportSchedule,
  getExportScheduleHistory,
  type ComponentStatus,
  type ExportOutcomeListResponse,
  type ExportSchedule,
  type ExportScheduleRequest,
} from "../../api";
import { getApiBaseUrl } from "../../lib/api-config";
import { getApiErrorMessage } from "../../lib/api-errors";
import { useToast } from "../toast";
import type { ExportSchedulePromotionSeed } from "../../types/promotion";

const ExportScheduleManager = lazy(() =>
  import("../../components/ExportScheduleManager").then((mod) => ({
    default: mod.ExportScheduleManager,
  })),
);

interface ExportScheduleContainerProps {
  aiStatus?: ComponentStatus | null;
  promotionSeed?: ExportSchedulePromotionSeed | null;
  onClearPromotionSeed?: () => void;
  onOpenSourceJob?: (jobId: string) => void;
}

export function ExportScheduleContainer({
  aiStatus = null,
  promotionSeed = null,
  onClearPromotionSeed,
  onOpenSourceJob,
}: ExportScheduleContainerProps) {
  const toast = useToast();
  const [schedules, setSchedules] = useState<ExportSchedule[]>([]);
  const [schedulesLoading, setSchedulesLoading] = useState(false);

  const refreshSchedules = useCallback(async () => {
    setSchedulesLoading(true);
    try {
      const { data, error } = await listExportSchedules({
        baseUrl: getApiBaseUrl(),
      });
      if (error) {
        const message = getApiErrorMessage(
          error,
          "Failed to load export schedules.",
        );
        console.error("Failed to load export schedules:", error);
        toast.show({
          tone: "error",
          title: "Failed to load export schedules",
          description: message,
        });
        return;
      }
      setSchedules(data?.schedules || []);
    } catch (err) {
      console.error("Error loading export schedules:", err);
      toast.show({
        tone: "error",
        title: "Failed to load export schedules",
        description: getApiErrorMessage(
          err,
          "Failed to load export schedules.",
        ),
      });
    } finally {
      setSchedulesLoading(false);
    }
  }, [toast]);

  const handleCreateSchedule = useCallback(
    async (request: ExportScheduleRequest) => {
      const toastId = toast.show({
        tone: "loading",
        title: request.name
          ? `Creating ${request.name}`
          : "Creating export schedule",
        description: "Saving the new recurring export configuration.",
      });
      const { error } = await createExportSchedule({
        baseUrl: getApiBaseUrl(),
        body: request,
      });
      if (error) {
        const message = getApiErrorMessage(
          error,
          "Failed to create export schedule.",
        );
        toast.update(toastId, {
          tone: "error",
          title: "Failed to create export schedule",
          description: message,
        });
        throw error;
      }
      await refreshSchedules();
      toast.update(toastId, {
        tone: "success",
        title: "Export schedule created",
        description: `${request.name} is now ready to run automatically.`,
      });
    },
    [refreshSchedules, toast],
  );

  const handleUpdateSchedule = useCallback(
    async (id: string, request: ExportScheduleRequest) => {
      const toastId = toast.show({
        tone: "loading",
        title: request.name
          ? `Updating ${request.name}`
          : "Updating export schedule",
        description: "Saving the latest export schedule changes.",
      });
      const { error } = await updateExportSchedule({
        baseUrl: getApiBaseUrl(),
        path: { id },
        body: request,
      });
      if (error) {
        const message = getApiErrorMessage(
          error,
          "Failed to update export schedule.",
        );
        toast.update(toastId, {
          tone: "error",
          title: "Failed to update export schedule",
          description: message,
        });
        throw error;
      }
      await refreshSchedules();
      toast.update(toastId, {
        tone: "success",
        title: "Export schedule updated",
        description: `${request.name} now reflects the latest settings.`,
      });
    },
    [refreshSchedules, toast],
  );

  const handleDeleteSchedule = useCallback(
    async (id: string) => {
      const toastId = toast.show({
        tone: "loading",
        title: "Deleting export schedule",
        description: "Removing the recurring export configuration.",
      });
      const { error } = await deleteExportSchedule({
        baseUrl: getApiBaseUrl(),
        path: { id },
      });
      if (error) {
        const message = getApiErrorMessage(
          error,
          "Failed to delete export schedule.",
        );
        toast.update(toastId, {
          tone: "error",
          title: "Failed to delete export schedule",
          description: message,
        });
        throw error;
      }
      await refreshSchedules();
      toast.update(toastId, {
        tone: "success",
        title: "Export schedule deleted",
        description: "The selected recurring export has been removed.",
      });
    },
    [refreshSchedules, toast],
  );

  const handleToggleEnabled = useCallback(
    async (id: string, enabled: boolean) => {
      const toastId = toast.show({
        tone: "loading",
        title: enabled
          ? "Enabling export schedule"
          : "Disabling export schedule",
        description: "Saving the updated schedule state.",
      });
      // First get the current schedule
      const { data, error: getError } = await getExportSchedule({
        baseUrl: getApiBaseUrl(),
        path: { id },
      });
      if (getError) {
        const message = getApiErrorMessage(
          getError,
          "Failed to load export schedule.",
        );
        toast.update(toastId, {
          tone: "error",
          title: "Failed to update export schedule",
          description: message,
        });
        throw getError;
      }
      if (!data) {
        const message = "Schedule not found";
        toast.update(toastId, {
          tone: "error",
          title: "Failed to update export schedule",
          description: message,
        });
        throw new Error(message);
      }

      // Update only the enabled field
      const { error: updateError } = await updateExportSchedule({
        baseUrl: getApiBaseUrl(),
        path: { id },
        body: {
          name: data.name,
          enabled,
          filters: data.filters,
          export: data.export,
          retry: data.retry,
        },
      });
      if (updateError) {
        const message = getApiErrorMessage(
          updateError,
          "Failed to update export schedule.",
        );
        toast.update(toastId, {
          tone: "error",
          title: "Failed to update export schedule",
          description: message,
        });
        throw updateError;
      }
      await refreshSchedules();
      toast.update(toastId, {
        tone: "success",
        title: enabled ? "Export schedule enabled" : "Export schedule disabled",
        description: `${data.name} has been ${enabled ? "enabled" : "disabled"}.`,
      });
    },
    [refreshSchedules, toast],
  );

  const handleGetHistory = useCallback(
    async (
      id: string,
      limit = 10,
      offset = 0,
    ): Promise<ExportOutcomeListResponse> => {
      const { data, error } = await getExportScheduleHistory({
        baseUrl: getApiBaseUrl(),
        path: { id },
        query: { limit, offset },
      });
      if (error) {
        const message = getApiErrorMessage(
          error,
          "Failed to load export history.",
        );
        toast.show({
          tone: "error",
          title: "Failed to load export history",
          description: message,
        });
        throw error;
      }
      return {
        exports: data?.exports || [],
        total: data?.total || 0,
        limit: data?.limit || limit,
        offset: data?.offset || offset,
      };
    },
    [toast],
  );

  useEffect(() => {
    refreshSchedules();
  }, [refreshSchedules]);

  return (
    <section id="export-schedules">
      <Suspense
        fallback={
          <div className="loading-placeholder">
            Loading export schedule manager...
          </div>
        }
      >
        <ExportScheduleManager
          schedules={schedules}
          onRefresh={refreshSchedules}
          onCreate={handleCreateSchedule}
          onUpdate={handleUpdateSchedule}
          onDelete={handleDeleteSchedule}
          onToggleEnabled={handleToggleEnabled}
          onGetHistory={handleGetHistory}
          loading={schedulesLoading}
          aiStatus={aiStatus}
          promotionSeed={promotionSeed}
          onClearPromotionSeed={onClearPromotionSeed}
          onOpenSourceJob={onOpenSourceJob}
        />
      </Suspense>
    </section>
  );
}
