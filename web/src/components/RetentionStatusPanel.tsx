/**
 * Retention Status Panel Component
 *
 * Displays retention configuration, current statistics, and provides
 * cleanup controls with dry-run preview capability.
 *
 * @module RetentionStatusPanel
 */

import { useState, useCallback, useEffect } from "react";
import {
  getRetentionStatus,
  runRetentionCleanup,
  type RetentionStatusResponse,
  type RetentionCleanupResponse,
} from "../api";
import { getApiBaseUrl } from "../lib/api-config";

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
  const getHighlightStyle = () => {
    switch (highlight) {
      case "warning":
        return { backgroundColor: "#fef3c7", borderColor: "#f59e0b" };
      case "danger":
        return { backgroundColor: "#fee2e2", borderColor: "#ef4444" };
      default:
        return {};
    }
  };

  return (
    <div
      className="status-card"
      style={{
        padding: "12px 16px",
        borderRadius: "8px",
        border: "1px solid #e5e7eb",
        backgroundColor: "#f9fafb",
        ...getHighlightStyle(),
      }}
    >
      <div style={{ fontSize: "12px", color: "#6b7280", marginBottom: "4px" }}>
        {label}
      </div>
      <div style={{ fontSize: "18px", fontWeight: 600, color: "#111827" }}>
        {value}
        {unit && (
          <span
            style={{ fontSize: "14px", fontWeight: 400, marginLeft: "4px" }}
          >
            {unit}
          </span>
        )}
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
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "repeat(auto-fit, minmax(150px, 1fr))",
              gap: "12px",
              marginBottom: "20px",
            }}
          >
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

          <div
            style={{
              backgroundColor: "#f3f4f6",
              padding: "12px 16px",
              borderRadius: "8px",
              marginBottom: "20px",
            }}
          >
            <h4 style={{ margin: "0 0 12px 0", fontSize: "14px" }}>
              Configuration
            </h4>
            <div
              style={{
                display: "grid",
                gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))",
                gap: "8px",
                fontSize: "13px",
              }}
            >
              <div>
                <span style={{ color: "#6b7280" }}>Job Retention: </span>
                <span>
                  {status.jobRetentionDays > 0
                    ? `${status.jobRetentionDays} days`
                    : "Unlimited"}
                </span>
              </div>
              <div>
                <span style={{ color: "#6b7280" }}>
                  Crawl State Retention:{" "}
                </span>
                <span>
                  {status.crawlStateDays > 0
                    ? `${status.crawlStateDays} days`
                    : "Unlimited"}
                </span>
              </div>
              <div>
                <span style={{ color: "#6b7280" }}>Max Jobs: </span>
                <span>
                  {status.maxJobs > 0
                    ? status.maxJobs.toLocaleString()
                    : "Unlimited"}
                </span>
              </div>
              <div>
                <span style={{ color: "#6b7280" }}>Max Storage: </span>
                <span>
                  {status.maxStorageGB > 0
                    ? `${status.maxStorageGB} GB`
                    : "Unlimited"}
                </span>
              </div>
            </div>
          </div>

          <div style={{ borderTop: "1px solid #e5e7eb", paddingTop: "20px" }}>
            <h4 style={{ margin: "0 0 16px 0" }}>Cleanup Controls</h4>

            <div style={{ marginBottom: "16px" }}>
              <label
                style={{
                  display: "flex",
                  alignItems: "center",
                  gap: "8px",
                  cursor: "pointer",
                }}
              >
                <input
                  type="checkbox"
                  checked={dryRun}
                  onChange={(e) => setDryRun(e.target.checked)}
                />
                <span>Dry-run mode (preview only, no actual deletions)</span>
              </label>
            </div>

            <div
              style={{
                display: "grid",
                gridTemplateColumns: "repeat(auto-fit, minmax(200px, 1fr))",
                gap: "12px",
                marginBottom: "16px",
              }}
            >
              <div>
                <label
                  htmlFor="kind-select"
                  style={{
                    display: "block",
                    fontSize: "13px",
                    marginBottom: "4px",
                  }}
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
                  style={{ width: "100%", padding: "6px 8px" }}
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
                  style={{
                    display: "block",
                    fontSize: "13px",
                    marginBottom: "4px",
                  }}
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
                  style={{ width: "100%", padding: "6px 8px" }}
                />
              </div>
            </div>

            <div style={{ display: "flex", gap: "12px", alignItems: "center" }}>
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
                <span style={{ fontSize: "13px", color: "#ef4444" }}>
                  Retention is disabled. Enable dry-run to preview.
                </span>
              )}
            </div>
          </div>

          {showConfirm && (
            <div
              style={{
                marginTop: "20px",
                padding: "16px",
                backgroundColor: "#fef3c7",
                border: "1px solid #f59e0b",
                borderRadius: "8px",
              }}
            >
              <p style={{ margin: "0 0 12px 0" }}>
                <strong>Warning:</strong> This will permanently delete jobs and
                their artifacts. This action cannot be undone.
              </p>
              <div style={{ display: "flex", gap: "12px" }}>
                <button
                  type="button"
                  onClick={handleCleanup}
                  disabled={cleanupLoading}
                  style={{ backgroundColor: "#ef4444", color: "white" }}
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
              style={{
                marginTop: "20px",
                padding: "16px",
                backgroundColor: cleanupResult.dryRun ? "#dbeafe" : "#d1fae5",
                border: `1px solid ${cleanupResult.dryRun ? "#3b82f6" : "#10b981"}`,
                borderRadius: "8px",
              }}
            >
              <h4 style={{ margin: "0 0 12px 0" }}>
                {cleanupResult.dryRun
                  ? "Dry-Run Preview Results"
                  : "Cleanup Complete"}
              </h4>
              <div
                style={{
                  display: "grid",
                  gridTemplateColumns: "repeat(auto-fit, minmax(150px, 1fr))",
                  gap: "8px",
                  fontSize: "13px",
                }}
              >
                <div>
                  <span style={{ color: "#6b7280" }}>
                    Jobs {cleanupResult.dryRun ? "Would Delete" : "Deleted"}:{" "}
                  </span>
                  <strong>{cleanupResult.jobsDeleted.toLocaleString()}</strong>
                </div>
                <div>
                  <span style={{ color: "#6b7280" }}>Jobs Attempted: </span>
                  <strong>
                    {cleanupResult.jobsAttempted.toLocaleString()}
                  </strong>
                </div>
                <div>
                  <span style={{ color: "#6b7280" }}>
                    Crawl States{" "}
                    {cleanupResult.dryRun ? "Would Delete" : "Deleted"}:{" "}
                  </span>
                  <strong>
                    {cleanupResult.crawlStatesDeleted.toLocaleString()}
                  </strong>
                </div>
                <div>
                  <span style={{ color: "#6b7280" }}>
                    Space {cleanupResult.dryRun ? "Would Reclaim" : "Reclaimed"}
                    :{" "}
                  </span>
                  <strong>
                    {cleanupResult.spaceReclaimedMB >= 1024
                      ? `${(cleanupResult.spaceReclaimedMB / 1024).toFixed(2)} GB`
                      : `${cleanupResult.spaceReclaimedMB} MB`}
                  </strong>
                </div>
                <div>
                  <span style={{ color: "#6b7280" }}>Duration: </span>
                  <strong>{cleanupResult.durationMs}ms</strong>
                </div>
              </div>
              {cleanupResult.failedJobIDs &&
                cleanupResult.failedJobIDs.length > 0 && (
                  <div style={{ marginTop: "12px", fontSize: "13px" }}>
                    <span style={{ color: "#ef4444" }}>
                      Warning: {cleanupResult.failedJobIDs.length} job(s) had
                      artifact deletion failures
                    </span>
                  </div>
                )}
              {cleanupResult.errors && cleanupResult.errors.length > 0 && (
                <div style={{ marginTop: "12px", fontSize: "13px" }}>
                  <span style={{ color: "#ef4444" }}>
                    Errors ({cleanupResult.errors.length}):
                  </span>
                  <ul style={{ margin: "4px 0 0 0", paddingLeft: "20px" }}>
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
