/**
 * Purpose: Render guided retention status, capacity pressure, and cleanup controls in Settings.
 * Responsibilities: Fetch retention status, coordinate manual cleanup actions, and compose the extracted retention guidance sections.
 * Scope: Retention settings orchestration and local UI state only; presentational subsections stay in `RetentionStatusPanelSections.tsx`.
 * Usage: Mount on the Settings route with health, navigation, refresh, and workflow callbacks.
 * Invariants/Assumptions: Retention is optional, but storage pressure and cleanup opportunities should be explicit and actionable.
 */

import { useCallback, useEffect, useMemo, useState } from "react";

import {
  getRetentionStatus,
  runRetentionCleanup,
  type HealthResponse,
  type RecommendedAction,
  type RetentionCleanupResponse,
  type RetentionStatusResponse,
} from "../api";
import { getApiBaseUrl } from "../lib/api-config";
import { getApiErrorMessage } from "../lib/api-errors";
import {
  CleanupResultNotice,
  RetentionCapabilitySurface,
  RetentionCleanupControls,
  RetentionStatusSummary,
} from "./RetentionStatusPanelSections";
import {
  type CleanupActivity,
  parseOlderThanDays,
  resolveRetentionCapability,
  type RetentionLoadErrorState,
} from "./retentionStatusPanelUtils";

interface RetentionStatusPanelProps {
  health: HealthResponse | null;
  onNavigate: (path: string) => void;
  onRefreshHealth: () => Promise<unknown> | undefined;
  onCreateJob: () => void;
  onOpenAutomation: () => void;
}

export function RetentionStatusPanel({
  health,
  onNavigate,
  onRefreshHealth,
  onCreateJob,
  onOpenAutomation,
}: RetentionStatusPanelProps) {
  const [status, setStatus] = useState<RetentionStatusResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [cleanupLoading, setCleanupLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [cleanupResult, setCleanupResult] =
    useState<RetentionCleanupResponse | null>(null);
  const [cleanupActivity, setCleanupActivity] =
    useState<CleanupActivity | null>(null);
  const [showConfirm, setShowConfirm] = useState(false);
  const [dryRun, setDryRun] = useState(true);
  const [kind, setKind] = useState<CleanupActivity["kind"]>("");
  const [olderThan, setOlderThan] = useState("");

  const refreshStatus = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await getRetentionStatus({ baseUrl: getApiBaseUrl() });
      if (response.data) {
        setStatus(response.data);
      } else if (response.error) {
        setError(
          getApiErrorMessage(
            response.error,
            "Failed to fetch retention status",
          ),
        );
      }
    } catch (err) {
      setError(getApiErrorMessage(err, "Failed to fetch retention status"));
    } finally {
      setLoading(false);
    }
  }, []);

  const refreshAll = useCallback(async () => {
    await Promise.all([refreshStatus(), Promise.resolve(onRefreshHealth())]);
  }, [onRefreshHealth, refreshStatus]);

  useEffect(() => {
    void refreshStatus();
  }, [refreshStatus]);

  const executeCleanup = useCallback(
    async (
      nextDryRun: boolean,
      origin: CleanupActivity["origin"] = "controls",
    ) => {
      const olderThanDays = parseOlderThanDays(olderThan);
      const forcePreview = nextDryRun && status?.enabled === false;

      setCleanupLoading(true);
      setError(null);
      setCleanupResult(null);
      setCleanupActivity({
        origin,
        dryRun: nextDryRun,
        kind,
        olderThanDays,
        force: forcePreview,
      });

      try {
        const request = {
          dryRun: nextDryRun,
          ...(forcePreview ? { force: true } : {}),
          ...(kind ? { kind } : {}),
          ...(olderThanDays !== null ? { olderThan: olderThanDays } : {}),
        };

        const response = await runRetentionCleanup({
          baseUrl: getApiBaseUrl(),
          body: request,
        });

        if (response.data) {
          setCleanupResult(response.data);
          if (!nextDryRun) {
            await refreshAll();
          }
        } else if (response.error) {
          setError(getApiErrorMessage(response.error, "Cleanup failed"));
        }
      } catch (err) {
        setError(getApiErrorMessage(err, "Cleanup failed"));
      } finally {
        setCleanupLoading(false);
        setShowConfirm(false);
      }
    },
    [kind, olderThan, refreshAll, status?.enabled],
  );

  const runPreviewCleanup = useCallback(() => {
    setDryRun(true);
    void executeCleanup(true, "banner");
  }, [executeCleanup]);

  const getStorageHighlight = () => {
    if (!status) {
      return "normal" as const;
    }
    if (status.maxStorageGB > 0) {
      const usedGB = status.storageUsedMB / 1024;
      const ratio = usedGB / status.maxStorageGB;
      if (ratio >= 0.9) {
        return "danger" as const;
      }
      if (ratio >= 0.75) {
        return "warning" as const;
      }
    }
    return "normal" as const;
  };

  const getJobsHighlight = () => {
    if (!status) {
      return "normal" as const;
    }
    if (status.maxJobs > 0) {
      const ratio = status.totalJobs / status.maxJobs;
      if (ratio >= 0.9) {
        return "danger" as const;
      }
      if (ratio >= 0.75) {
        return "warning" as const;
      }
    }
    return "normal" as const;
  };

  const retentionComponent = health?.components?.retention;

  const capability = useMemo(
    () => resolveRetentionCapability(health, status),
    [health, status],
  );

  const loadErrorActions = useMemo<RecommendedAction[]>(() => {
    if (retentionComponent?.actions && retentionComponent.actions.length > 0) {
      return retentionComponent.actions;
    }

    return [
      {
        label: "Check retention status from the CLI",
        kind: "command",
        value: "spartan retention status",
      },
      {
        label: "Preview cleanup from the CLI",
        kind: "command",
        value: "spartan retention cleanup --dry-run",
      },
    ];
  }, [retentionComponent?.actions]);

  const loadErrorState = useMemo<RetentionLoadErrorState | null>(() => {
    if (!error || status) {
      return null;
    }

    switch (retentionComponent?.status) {
      case "disabled":
        return {
          eyebrow: "Optional subsystem",
          title: "Unable to load retention status",
          description:
            "Spartan could not confirm current retention metadata for this section. Automatic cleanup is optional, but status checks and previews are temporarily unavailable until this check succeeds.",
        };
      case "degraded":
      case "error":
      case "setup_required":
        return {
          eyebrow: "Recovery guidance",
          title: "Unable to load retention status",
          description:
            "Spartan could not load retention metadata for this section. Use the recovery actions below, then refresh this section.",
        };
      default:
        return {
          eyebrow: "Status unavailable",
          title: "Unable to load retention status",
          description:
            "Spartan could not load retention metadata for this section. Refresh this section or use the CLI commands below.",
        };
    }
  }, [error, retentionComponent?.status, status]);

  return (
    <section className="panel" id="retention">
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "center",
          gap: 12,
          marginBottom: 16,
        }}
      >
        <div>
          <h2 style={{ marginBottom: 4 }}>Data Retention</h2>
          <p style={{ margin: 0, opacity: 0.8 }}>
            See whether automatic cleanup is still off by choice, available to
            preview, or worth reviewing because storage is growing.
          </p>
        </div>

        <button
          type="button"
          className="secondary"
          onClick={() => {
            void refreshAll();
          }}
          disabled={loading}
        >
          {loading ? "Refreshing..." : "Refresh Status"}
        </button>
      </div>

      {error && status ? (
        <div className="error" style={{ marginBottom: "16px" }}>
          {error}
        </div>
      ) : null}

      <RetentionCapabilitySurface
        loading={loading}
        error={error}
        status={status}
        capability={capability}
        cleanupActivity={cleanupActivity}
        cleanupLoading={cleanupLoading}
        cleanupResult={cleanupResult}
        loadErrorState={loadErrorState}
        loadErrorActions={loadErrorActions}
        onNavigate={onNavigate}
        onRefresh={refreshAll}
        onPreviewCleanup={runPreviewCleanup}
        onCreateJob={onCreateJob}
        onOpenAutomation={onOpenAutomation}
      />

      {status ? (
        <>
          <RetentionStatusSummary
            status={status}
            jobsHighlight={getJobsHighlight()}
            storageHighlight={getStorageHighlight()}
          />

          <RetentionCleanupControls
            status={status}
            dryRun={dryRun}
            kind={kind}
            olderThan={olderThan}
            loading={loading}
            cleanupLoading={cleanupLoading}
            showConfirm={showConfirm}
            error={error}
            cleanupActivity={cleanupActivity}
            cleanupResult={cleanupResult}
            onChangeDryRun={setDryRun}
            onChangeKind={setKind}
            onChangeOlderThan={setOlderThan}
            onPreviewOrRun={() =>
              dryRun
                ? void executeCleanup(true, "controls")
                : setShowConfirm(true)
            }
            onRefresh={refreshAll}
            onConfirmDelete={() => {
              void executeCleanup(false, "controls");
            }}
            onCancelConfirm={() => setShowConfirm(false)}
          />

          <CleanupResultNotice cleanupResult={cleanupResult} />
        </>
      ) : null}
    </section>
  );
}
