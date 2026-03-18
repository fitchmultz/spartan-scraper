/**
 * Purpose: Render guided retention status, capacity pressure, and cleanup controls in Settings.
 * Responsibilities: Explain disabled and pressure states, preserve manual cleanup controls, and surface safe next steps.
 * Scope: Retention settings presentation and local UI state only.
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
import { ActionEmptyState } from "./ActionEmptyState";
import { CapabilityActionList } from "./CapabilityActionList";

interface RetentionStatusPanelProps {
  health: HealthResponse | null;
  onNavigate: (path: string) => void;
  onRefreshHealth: () => Promise<unknown> | undefined;
  onCreateJob: () => void;
  onOpenAutomation: () => void;
}

interface StatusCardProps {
  label: string;
  value: string | number;
  unit?: string;
  highlight?: "normal" | "warning" | "danger";
}

interface DerivedRetentionCapability {
  status: "disabled" | "warning" | "danger" | "ok";
  title: string;
  message: string;
  actions: RecommendedAction[];
}

function StatusCard({
  label,
  value,
  unit,
  highlight = "normal",
}: StatusCardProps) {
  return (
    <div
      className={`retention-status-card retention-status-card--${highlight}`}
    >
      <div className="retention-status-card__label">{label}</div>
      <div className="retention-status-card__value">
        {value}
        {unit ? (
          <span className="retention-status-card__unit">{unit}</span>
        ) : null}
      </div>
    </div>
  );
}

function deriveRetentionCapability(
  status: RetentionStatusResponse | null,
): DerivedRetentionCapability | null {
  if (!status) {
    return null;
  }

  if (!status.enabled) {
    return {
      status: "disabled",
      title: "Automatic retention is disabled",
      message:
        "Spartan will keep completed jobs and crawl state until you enable automatic cleanup or run targeted cleanup manually. Preview first so you know the blast radius.",
      actions: [
        {
          label: "Enable retention in the environment",
          kind: "env",
          value: "RETENTION_ENABLED=true",
        },
        {
          label: "Preview cleanup from the CLI",
          kind: "command",
          value: "spartan retention cleanup --dry-run",
        },
      ],
    };
  }

  const storageRatio =
    status.maxStorageGB > 0
      ? status.storageUsedMB / 1024 / status.maxStorageGB
      : 0;
  const jobsRatio = status.maxJobs > 0 ? status.totalJobs / status.maxJobs : 0;

  if (storageRatio >= 0.9 || jobsRatio >= 0.9) {
    return {
      status: "danger",
      title: "Retention limits are close to being hit",
      message:
        "Storage or job-count pressure is high. Preview cleanup now, then run cleanup or raise limits intentionally if this growth is expected.",
      actions: [
        {
          label: "Preview cleanup from the CLI",
          kind: "command",
          value: "spartan retention cleanup --dry-run",
        },
      ],
    };
  }

  if (status.jobsEligible > 0) {
    return {
      status: "warning",
      title: "Cleanup opportunity detected",
      message: `${status.jobsEligible.toLocaleString()} job(s) already meet the current cleanup policy. Preview a cleanup run before pressure becomes urgent.`,
      actions: [
        {
          label: "Preview cleanup from the CLI",
          kind: "command",
          value: "spartan retention cleanup --dry-run",
        },
      ],
    };
  }

  return {
    status: "ok",
    title: "",
    message: "",
    actions: [],
  };
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
  const [showConfirm, setShowConfirm] = useState(false);
  const [dryRun, setDryRun] = useState(true);
  const [kind, setKind] = useState<"" | "scrape" | "crawl" | "research">("");
  const [olderThan, setOlderThan] = useState<string>("");

  const refreshStatus = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await getRetentionStatus({ baseUrl: getApiBaseUrl() });
      if (response.data) {
        setStatus(response.data);
      } else if (response.error) {
        setError(String(response.error));
      }
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Failed to fetch retention status",
      );
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
    async (nextDryRun: boolean) => {
      setCleanupLoading(true);
      setError(null);
      setCleanupResult(null);

      try {
        const request = {
          dryRun: nextDryRun,
          ...(kind ? { kind } : {}),
          ...(olderThan && !Number.isNaN(Number.parseInt(olderThan, 10))
            ? {
                olderThan: Number.parseInt(olderThan, 10),
              }
            : {}),
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
          setError(String(response.error));
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : "Cleanup failed");
      } finally {
        setCleanupLoading(false);
        setShowConfirm(false);
      }
    },
    [kind, olderThan, refreshAll],
  );

  const runPreviewCleanup = useCallback(() => {
    setDryRun(true);
    void executeCleanup(true);
  }, [executeCleanup]);

  const getStorageHighlight = () => {
    if (!status) {
      return "normal";
    }
    if (status.maxStorageGB > 0) {
      const usedGB = status.storageUsedMB / 1024;
      const ratio = usedGB / status.maxStorageGB;
      if (ratio >= 0.9) {
        return "danger";
      }
      if (ratio >= 0.75) {
        return "warning";
      }
    }
    return "normal";
  };

  const getJobsHighlight = () => {
    if (!status) {
      return "normal";
    }
    if (status.maxJobs > 0) {
      const ratio = status.totalJobs / status.maxJobs;
      if (ratio >= 0.9) {
        return "danger";
      }
      if (ratio >= 0.75) {
        return "warning";
      }
    }
    return "normal";
  };

  const retentionComponent = health?.components?.retention;
  const derivedCapability = useMemo(
    () => status?.guidance ?? deriveRetentionCapability(status),
    [status],
  );
  const capabilityStatus =
    retentionComponent?.status ?? derivedCapability?.status ?? "ok";
  const capabilityTitle = retentionComponent?.message
    ? "Retention needs attention"
    : (derivedCapability?.title ?? "");
  const capabilityMessage =
    retentionComponent?.message ?? derivedCapability?.message ?? "";
  const capabilityActions =
    retentionComponent?.actions ?? derivedCapability?.actions ?? [];

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
            Understand when cleanup is off, optional, or becoming urgent before
            you delete anything.
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

      {error ? (
        <div className="error" style={{ marginBottom: "16px" }}>
          {error}
        </div>
      ) : null}

      {loading && !status ? (
        <div>Loading retention status...</div>
      ) : status ? (
        <>
          {capabilityStatus === "disabled" ? (
            <ActionEmptyState
              eyebrow="Optional subsystem"
              title={
                retentionComponent?.message
                  ? "Retention is off by configuration"
                  : (derivedCapability?.title ??
                    "Automatic retention is disabled")
              }
              description={capabilityMessage}
              actions={[
                { label: "Preview cleanup", onClick: runPreviewCleanup },
                {
                  label: "Create job",
                  onClick: onCreateJob,
                  tone: "secondary",
                },
              ]}
            >
              <CapabilityActionList
                actions={capabilityActions}
                onNavigate={onNavigate}
                onRefresh={refreshAll}
              />
            </ActionEmptyState>
          ) : null}

          {(capabilityStatus === "warning" || capabilityStatus === "danger") &&
          capabilityMessage ? (
            <div
              className={`retention-notice ${
                capabilityStatus === "danger"
                  ? "retention-notice--warning"
                  : "retention-notice--info"
              }`}
            >
              <h4 className="retention-section-title">{capabilityTitle}</h4>
              <p className="retention-notice__copy">{capabilityMessage}</p>
              <div className="retention-notice__actions">
                <button
                  type="button"
                  className="secondary"
                  onClick={runPreviewCleanup}
                  disabled={cleanupLoading}
                >
                  {cleanupLoading ? "Running..." : "Preview cleanup"}
                </button>
                <button
                  type="button"
                  className="secondary"
                  onClick={onOpenAutomation}
                >
                  Open automation
                </button>
              </div>

              <CapabilityActionList
                actions={capabilityActions}
                onNavigate={onNavigate}
                onRefresh={refreshAll}
              />
            </div>
          ) : null}

          <div className="retention-status-grid">
            <StatusCard
              label="Retention Enabled"
              value={status.enabled ? "Yes" : "No"}
              highlight={status.enabled ? "normal" : "warning"}
            />
            <StatusCard
              label="Total Jobs"
              value={status.totalJobs.toLocaleString()}
              highlight={getJobsHighlight()}
            />
            <StatusCard
              label="Storage Used"
              value={
                status.storageUsedMB >= 1024
                  ? (status.storageUsedMB / 1024).toFixed(2)
                  : status.storageUsedMB
              }
              unit={status.storageUsedMB >= 1024 ? "GB" : "MB"}
              highlight={getStorageHighlight()}
            />
            <StatusCard
              label="Jobs Eligible for Cleanup"
              value={status.jobsEligible.toLocaleString()}
              highlight={status.jobsEligible > 0 ? "warning" : "normal"}
            />
          </div>

          <div className="retention-config-card">
            <h4 className="retention-section-title">Configuration</h4>
            <div className="retention-config-grid">
              <div className="retention-metric">
                <span className="retention-metric__label">Job Retention</span>
                <span className="retention-metric__value">
                  {status.jobRetentionDays > 0
                    ? `${status.jobRetentionDays} days`
                    : "Unlimited"}
                </span>
              </div>
              <div className="retention-metric">
                <span className="retention-metric__label">
                  Crawl State Retention
                </span>
                <span className="retention-metric__value">
                  {status.crawlStateDays > 0
                    ? `${status.crawlStateDays} days`
                    : "Unlimited"}
                </span>
              </div>
              <div className="retention-metric">
                <span className="retention-metric__label">Max Jobs</span>
                <span className="retention-metric__value">
                  {status.maxJobs > 0
                    ? status.maxJobs.toLocaleString()
                    : "Unlimited"}
                </span>
              </div>
              <div className="retention-metric">
                <span className="retention-metric__label">Max Storage</span>
                <span className="retention-metric__value">
                  {status.maxStorageGB > 0
                    ? `${status.maxStorageGB} GB`
                    : "Unlimited"}
                </span>
              </div>
            </div>
          </div>

          <div className="retention-controls">
            <h4 className="retention-section-title">Cleanup Controls</h4>

            <div className="retention-controls__toggle-row">
              <label className="retention-controls__toggle">
                <input
                  type="checkbox"
                  checked={dryRun}
                  onChange={(event) => setDryRun(event.target.checked)}
                />
                <span>Dry-run mode (preview only, no actual deletions)</span>
              </label>
            </div>

            <div className="retention-controls__grid">
              <div>
                <label
                  htmlFor="kind-select"
                  className="retention-controls__label"
                >
                  Job Kind (optional)
                </label>
                <select
                  id="kind-select"
                  value={kind}
                  onChange={(event) =>
                    setKind(
                      event.target.value as
                        | ""
                        | "scrape"
                        | "crawl"
                        | "research",
                    )
                  }
                >
                  <option value="">All kinds</option>
                  <option value="scrape">Scrape</option>
                  <option value="crawl">Crawl</option>
                  <option value="research">Research</option>
                </select>
              </div>
              <div>
                <label
                  htmlFor="older-than-input"
                  className="retention-controls__label"
                >
                  Older Than (days, optional)
                </label>
                <input
                  id="older-than-input"
                  type="number"
                  min="1"
                  value={olderThan}
                  onChange={(event) => setOlderThan(event.target.value)}
                  placeholder="Use config default"
                />
              </div>
            </div>

            <div className="retention-controls__actions">
              <button
                type="button"
                onClick={() =>
                  dryRun ? void executeCleanup(true) : setShowConfirm(true)
                }
                disabled={cleanupLoading || (!status.enabled && !dryRun)}
                className={dryRun ? "secondary" : "primary"}
              >
                {cleanupLoading
                  ? "Running..."
                  : dryRun
                    ? "Preview Cleanup"
                    : "Run Cleanup"}
              </button>
              <button
                type="button"
                onClick={() => {
                  void refreshAll();
                }}
                disabled={loading}
                className="secondary"
              >
                {loading ? "Refreshing..." : "Refresh Status"}
              </button>
              {!status.enabled && !dryRun ? (
                <span className="retention-controls__disabled-message">
                  Retention is disabled. Enable dry-run to preview.
                </span>
              ) : null}
            </div>
          </div>

          {showConfirm ? (
            <div className="retention-notice retention-notice--warning">
              <p className="retention-notice__copy">
                <strong>Warning:</strong> This will permanently delete jobs and
                their artifacts. This action cannot be undone.
              </p>
              <div className="retention-notice__actions">
                <button
                  type="button"
                  onClick={() => {
                    void executeCleanup(false);
                  }}
                  disabled={cleanupLoading}
                  className="retention-action-button retention-action-button--danger"
                >
                  {cleanupLoading ? "Running..." : "Confirm Delete"}
                </button>
                <button
                  type="button"
                  onClick={() => setShowConfirm(false)}
                  disabled={cleanupLoading}
                  className="secondary"
                >
                  Cancel
                </button>
              </div>
            </div>
          ) : null}

          {cleanupResult ? (
            <div
              className={`retention-notice ${
                cleanupResult.dryRun
                  ? "retention-notice--info"
                  : "retention-notice--success"
              }`}
            >
              <h4 className="retention-section-title retention-notice__title">
                {cleanupResult.dryRun
                  ? "Dry-Run Preview Results"
                  : "Cleanup Complete"}
              </h4>
              <div className="retention-results-grid">
                <div className="retention-metric">
                  <span className="retention-metric__label">
                    Jobs {cleanupResult.dryRun ? "Would Delete" : "Deleted"}:{" "}
                  </span>
                  <strong className="retention-metric__value">
                    {cleanupResult.jobsDeleted.toLocaleString()}
                  </strong>
                </div>
                <div className="retention-metric">
                  <span className="retention-metric__label">
                    Jobs Attempted
                  </span>
                  <strong className="retention-metric__value">
                    {cleanupResult.jobsAttempted.toLocaleString()}
                  </strong>
                </div>
                <div className="retention-metric">
                  <span className="retention-metric__label">
                    Crawl States{" "}
                    {cleanupResult.dryRun ? "Would Delete" : "Deleted"}:{" "}
                  </span>
                  <strong className="retention-metric__value">
                    {cleanupResult.crawlStatesDeleted.toLocaleString()}
                  </strong>
                </div>
                <div className="retention-metric">
                  <span className="retention-metric__label">
                    Space {cleanupResult.dryRun ? "Would Reclaim" : "Reclaimed"}
                    :{" "}
                  </span>
                  <strong className="retention-metric__value">
                    {cleanupResult.spaceReclaimedMB >= 1024
                      ? `${(cleanupResult.spaceReclaimedMB / 1024).toFixed(2)} GB`
                      : `${cleanupResult.spaceReclaimedMB} MB`}
                  </strong>
                </div>
                <div className="retention-metric">
                  <span className="retention-metric__label">Duration</span>
                  <strong className="retention-metric__value">
                    {cleanupResult.durationMs}ms
                  </strong>
                </div>
              </div>
              {cleanupResult.failedJobIDs &&
              cleanupResult.failedJobIDs.length > 0 ? (
                <div className="retention-notice__detail">
                  <span className="retention-notice__danger-text">
                    Warning: {cleanupResult.failedJobIDs.length} job(s) had
                    artifact deletion failures
                  </span>
                </div>
              ) : null}
              {cleanupResult.errors && cleanupResult.errors.length > 0 ? (
                <div className="retention-notice__detail">
                  <span className="retention-notice__danger-text">
                    Errors ({cleanupResult.errors.length}):
                  </span>
                  <ul className="retention-notice__list">
                    {cleanupResult.errors.map((cleanupError) => (
                      <li key={cleanupError}>{cleanupError}</li>
                    ))}
                  </ul>
                </div>
              ) : null}
            </div>
          ) : null}
        </>
      ) : null}
    </section>
  );
}
