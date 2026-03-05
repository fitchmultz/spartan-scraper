/**
 * ExportScheduleContainer - Container component for export schedule management
 *
 * This component encapsulates all export schedule-related state and operations:
 * - Loading and displaying export schedules
 * - Creating, updating, and deleting export schedules
 * - Viewing export history
 * - Toggling schedule enabled state
 *
 * It does NOT handle:
 * - Job submission or results viewing
 * - Watch management
 * - Batch operations
 *
 * @module ExportScheduleContainer
 */

import { useCallback, useEffect, useState, Suspense, lazy } from "react";
import {
  listExportSchedules,
  createExportSchedule,
  getExportSchedule,
  updateExportSchedule,
  deleteExportSchedule,
  getExportScheduleHistory,
  type ExportSchedule,
  type ExportScheduleRequest,
  type ExportHistoryRecord,
} from "../../api";
import { getApiBaseUrl } from "../../lib/api-config";

const ExportScheduleManager = lazy(() =>
  import("../../components/ExportScheduleManager").then((mod) => ({
    default: mod.ExportScheduleManager,
  })),
);

export function ExportScheduleContainer() {
  const [schedules, setSchedules] = useState<ExportSchedule[]>([]);
  const [schedulesLoading, setSchedulesLoading] = useState(false);

  const refreshSchedules = useCallback(async () => {
    setSchedulesLoading(true);
    try {
      const { data, error } = await listExportSchedules({
        baseUrl: getApiBaseUrl(),
      });
      if (error) {
        console.error("Failed to load export schedules:", error);
        return;
      }
      setSchedules(data?.schedules || []);
    } catch (err) {
      console.error("Error loading export schedules:", err);
    } finally {
      setSchedulesLoading(false);
    }
  }, []);

  const handleCreateSchedule = useCallback(
    async (request: ExportScheduleRequest) => {
      const { error } = await createExportSchedule({
        baseUrl: getApiBaseUrl(),
        body: request,
      });
      if (error) throw error;
      await refreshSchedules();
    },
    [refreshSchedules],
  );

  const handleUpdateSchedule = useCallback(
    async (id: string, request: ExportScheduleRequest) => {
      const { error } = await updateExportSchedule({
        baseUrl: getApiBaseUrl(),
        path: { id },
        body: request,
      });
      if (error) throw error;
      await refreshSchedules();
    },
    [refreshSchedules],
  );

  const handleDeleteSchedule = useCallback(
    async (id: string) => {
      const { error } = await deleteExportSchedule({
        baseUrl: getApiBaseUrl(),
        path: { id },
      });
      if (error) throw error;
      await refreshSchedules();
    },
    [refreshSchedules],
  );

  const handleToggleEnabled = useCallback(
    async (id: string, enabled: boolean) => {
      // First get the current schedule
      const { data, error: getError } = await getExportSchedule({
        baseUrl: getApiBaseUrl(),
        path: { id },
      });
      if (getError) throw getError;
      if (!data) throw new Error("Schedule not found");

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
      if (updateError) throw updateError;
      await refreshSchedules();
    },
    [refreshSchedules],
  );

  const handleGetHistory = useCallback(
    async (
      id: string,
      limit = 10,
      offset = 0,
    ): Promise<{ records: ExportHistoryRecord[]; total: number }> => {
      const { data, error } = await getExportScheduleHistory({
        baseUrl: getApiBaseUrl(),
        path: { id },
        query: { limit, offset },
      });
      if (error) throw error;
      return {
        records: data?.records || [],
        total: data?.total || 0,
      };
    },
    [],
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
        />
      </Suspense>
    </section>
  );
}
