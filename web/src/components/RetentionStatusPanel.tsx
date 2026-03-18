/**
 * Purpose: Render retention status, capacity, and cleanup controls in Settings.
 * Responsibilities: Fetch retention status, surface disabled-state guidance, run cleanup previews or executions, and summarize capacity pressure.
 * Scope: Retention settings presentation and local UI state only.
 * Usage: Mount on the Settings route to help operators understand and operate data-retention behavior.
 * Invariants/Assumptions: Retention is optional, disabled mode should still explain what to do next, and destructive cleanup runs remain explicitly operator-controlled.
 */

import { useState, useCallback, useEffect } from "react";
import {
  getRetentionStatus,
  runRetentionCleanup,
  type RetentionStatusResponse,
  type RetentionCleanupResponse,
} from "../api";
import { getApiBaseUrl } from "../lib/api-config";
import { ActionEmptyState } from "./ActionEmptyState";

interface StatusCardProps {
  label: string;
  value: string | number;
  unit?: string;
  highlight?: "normal" | "warning" | "danger";
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
        {unit && <span className="retention-status-card__unit">{unit}</span>}
      </div>
    </div>
  );
}

export function RetentionStatusPanel() {
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

  /**
   * Refresh retention status from the API
   */
  const refreshStatus = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await getRetentionStatus({
        baseUrl: getApiBaseUrl(),
      });
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

  // Load status on mount
  useEffect(() => {
    refreshStatus();
  }, [refreshStatus]);

  const handleCleanup = async () => {
    setCleanupLoading(true);
    setError(null);
    setCleanupResult(null);
    try {
      const request = {
        dryRun,
        ...(kind && { kind }),
        ...(olderThan &&
          !Number.isNaN(parseInt(olderThan, 10)) && {
            olderThan: parseInt(olderThan, 10),
          }),
      };
      const response = await runRetentionCleanup({
        baseUrl: getApiBaseUrl(),
        body: request,
      });
      if (response.data) {
        setCleanupResult(response.data);
        // Refresh status after cleanup (especially important for non-dry-run)
        if (!dryRun) {
          await refreshStatus();
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
  };

  const getStorageHighlight = () => {
    if (!status) return "normal";
    if (status.maxStorageGB > 0) {
      const usedGB = status.storageUsedMB / 1024;
      const ratio = usedGB / status.maxStorageGB;
      if (ratio >= 0.9) return "danger";
      if (ratio >= 0.75) return "warning";
    }
    return "normal";
  };

  const getJobsHighlight = () => {
    if (!status) return "normal";
    if (status.maxJobs > 0) {
      const ratio = status.totalJobs / status.maxJobs;
      if (ratio >= 0.9) return "danger";
      if (ratio >= 0.75) return "warning";
    }
    return "normal";
  };

  return (
    <section className="panel" id="retention">
      <h2>Data Retention</h2>

      {error && (
        <div className="error" style={{ marginBottom: "16px" }}>
          {error}
        </div>
      )}

      {loading && !status ? (
        <div>Loading retention status...</div>
      ) : status ? (
        <>
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

          {!status.enabled ? (
            <ActionEmptyState
              eyebrow="Optional subsystem"
              title="Automatic retention is disabled"
              description="Retention is optional. Enable RETENTION_ENABLED when you want Spartan to clean up old jobs and crawl state automatically."
              actions={[
                {
                  label: "Refresh status",
                  onClick: () => {
                    void refreshStatus();
                  },
                  tone: "secondary",
                },
              ]}
            />
          ) : null}

          <div className="retention-controls">
            <h4 className="retention-section-title">Cleanup Controls</h4>

            <div className="retention-controls__toggle-row">
              <label className="retention-controls__toggle">
                <input
                  type="checkbox"
                  checked={dryRun}
                  onChange={(e) => setDryRun(e.target.checked)}
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
                  onChange={(e) =>
                    setKind(
                      e.target.value as "" | "scrape" | "crawl" | "research",
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
                  onChange={(e) => setOlderThan(e.target.value)}
                  placeholder="Use config default"
                />
              </div>
            </div>

            <div className="retention-controls__actions">
              <button
                type="button"
                onClick={() =>
                  dryRun ? handleCleanup() : setShowConfirm(true)
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
                onClick={refreshStatus}
                disabled={loading}
                className="secondary"
              >
                {loading ? "Refreshing..." : "Refresh Status"}
              </button>
              {!status.enabled && !dryRun && (
                <span className="retention-controls__disabled-message">
                  Retention is disabled. Enable dry-run to preview.
                </span>
              )}
            </div>
          </div>

          {showConfirm && (
            <div className="retention-notice retention-notice--warning">
              <p className="retention-notice__copy">
                <strong>Warning:</strong> This will permanently delete jobs and
                their artifacts. This action cannot be undone.
              </p>
              <div className="retention-notice__actions">
                <button
                  type="button"
                  onClick={handleCleanup}
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
          )}

          {cleanupResult && (
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
                cleanupResult.failedJobIDs.length > 0 && (
                  <div className="retention-notice__detail">
                    <span className="retention-notice__danger-text">
                      Warning: {cleanupResult.failedJobIDs.length} job(s) had
                      artifact deletion failures
                    </span>
                  </div>
                )}
              {cleanupResult.errors && cleanupResult.errors.length > 0 && (
                <div className="retention-notice__detail">
                  <span className="retention-notice__danger-text">
                    Errors ({cleanupResult.errors.length}):
                  </span>
                  <ul className="retention-notice__list">
                    {cleanupResult.errors.map((err) => (
                      <li key={err}>{err}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          )}
        </>
      ) : null}
    </section>
  );
}
