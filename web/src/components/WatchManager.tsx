/**
 * Watch Manager Component
 *
 * Provides UI for managing content change monitoring watches. Supports creating,
 * editing, deleting, and manually checking watches. Displays watch status,
 * change counts, and next run times.
 *
 * @module WatchManager
 */

import { useState, useCallback, useMemo } from "react";
import type { Watch, WatchInput, WatchCheckResult } from "../api";

interface WatchManagerProps {
  watches: Watch[];
  onRefresh: () => void;
  onCreate: (watch: WatchInput) => Promise<void>;
  onUpdate: (id: string, watch: WatchInput) => Promise<void>;
  onDelete: (id: string) => Promise<void>;
  onCheck: (id: string) => Promise<WatchCheckResult | undefined>;
  loading?: boolean;
}

interface WatchFormData {
  url: string;
  selector: string;
  intervalSeconds: number;
  enabled: boolean;
  diffFormat: "unified" | "html-side-by-side" | "html-inline";
  notifyOnChange: boolean;
  webhookUrl: string;
  webhookSecret: string;
  headless: boolean;
  usePlaywright: boolean;
  extractMode: "" | "text" | "html" | "markdown";
  minChangeSize: string;
  ignorePatterns: string;
}

const defaultFormData: WatchFormData = {
  url: "",
  selector: "",
  intervalSeconds: 3600,
  enabled: true,
  diffFormat: "unified",
  notifyOnChange: false,
  webhookUrl: "",
  webhookSecret: "",
  headless: false,
  usePlaywright: false,
  extractMode: "",
  minChangeSize: "",
  ignorePatterns: "",
};

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`;
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`;
  return `${Math.floor(seconds / 86400)}d`;
}

function formatDate(dateStr: string | undefined): string {
  if (!dateStr) return "Never";
  const date = new Date(dateStr);
  return date.toLocaleString();
}

function watchToFormData(watch: Watch): WatchFormData {
  return {
    url: watch.url,
    selector: watch.selector || "",
    intervalSeconds: watch.intervalSeconds,
    enabled: watch.enabled ?? true,
    diffFormat: (watch.diffFormat || "unified") as WatchFormData["diffFormat"],
    notifyOnChange: watch.notifyOnChange ?? false,
    webhookUrl: watch.webhookConfig?.url || "",
    webhookSecret: watch.webhookConfig?.secret || "",
    headless: watch.headless ?? false,
    usePlaywright: watch.usePlaywright ?? false,
    extractMode: (watch.extractMode as WatchFormData["extractMode"]) || "",
    minChangeSize: watch.minChangeSize?.toString() || "",
    ignorePatterns: watch.ignorePatterns?.join("\n") || "",
  };
}

function formDataToWatchInput(data: WatchFormData): WatchInput {
  const input: WatchInput = {
    url: data.url,
    intervalSeconds: data.intervalSeconds,
    enabled: data.enabled,
    diffFormat: data.diffFormat,
    notifyOnChange: data.notifyOnChange,
    headless: data.headless,
    usePlaywright: data.usePlaywright,
  };

  if (data.selector) input.selector = data.selector;
  if (data.extractMode) input.extractMode = data.extractMode;
  if (data.minChangeSize)
    input.minChangeSize = parseInt(data.minChangeSize, 10);
  if (data.ignorePatterns.trim()) {
    input.ignorePatterns = data.ignorePatterns
      .split("\n")
      .filter((p) => p.trim());
  }
  if (data.webhookUrl && data.notifyOnChange) {
    input.webhookConfig = {
      url: data.webhookUrl,
      secret: data.webhookSecret || undefined,
    };
  }

  return input;
}

export function WatchManager({
  watches,
  onRefresh,
  onCreate,
  onUpdate,
  onDelete,
  onCheck,
  loading,
}: WatchManagerProps) {
  const [showForm, setShowForm] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [formData, setFormData] = useState<WatchFormData>(defaultFormData);
  const [formError, setFormError] = useState<string | null>(null);
  const [formSubmitting, setFormSubmitting] = useState(false);
  const [checkResult, setCheckResult] = useState<WatchCheckResult | null>(null);
  const [checkingId, setCheckingId] = useState<string | null>(null);
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null);

  const sortedWatches = useMemo(() => {
    return [...watches].sort((a, b) => {
      return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
    });
  }, [watches]);

  const handleCreateClick = useCallback(() => {
    setFormData(defaultFormData);
    setEditingId(null);
    setFormError(null);
    setShowForm(true);
  }, []);

  const handleEditClick = useCallback((watch: Watch) => {
    setFormData(watchToFormData(watch));
    setEditingId(watch.id);
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
      const input = formDataToWatchInput(formData);
      if (editingId) {
        await onUpdate(editingId, input);
      } else {
        await onCreate(input);
      }
      setShowForm(false);
      setEditingId(null);
      onRefresh();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Failed to save watch");
    } finally {
      setFormSubmitting(false);
    }
  }, [formData, editingId, onCreate, onUpdate, onRefresh]);

  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await onDelete(id);
        setDeleteConfirmId(null);
        onRefresh();
      } catch (err) {
        console.error("Failed to delete watch:", err);
      }
    },
    [onDelete, onRefresh],
  );

  const handleCheck = useCallback(
    async (watch: Watch) => {
      setCheckingId(watch.id);
      setCheckResult(null);
      try {
        const result = await onCheck(watch.id);
        if (result) {
          setCheckResult(result);
        }
      } catch (err) {
        console.error("Check failed:", err);
      } finally {
        setCheckingId(null);
      }
    },
    [onCheck],
  );

  return (
    <div className="panel">
      <div
        className="row"
        style={{
          justifyContent: "space-between",
          alignItems: "center",
          marginBottom: 16,
        }}
      >
        <h2 style={{ margin: 0 }}>Watch Monitoring</h2>
        <div className="row" style={{ gap: 8 }}>
          <button
            type="button"
            onClick={onRefresh}
            disabled={loading}
            className="secondary"
          >
            {loading ? "Loading..." : "Refresh"}
          </button>
          <button type="button" onClick={handleCreateClick}>
            + Add Watch
          </button>
        </div>
      </div>

      {watches.length === 0 && !loading ? (
        <div
          style={{
            textAlign: "center",
            padding: "40px 20px",
            color: "var(--muted)",
          }}
        >
          <p>No watches configured yet.</p>
          <p>Click "Add Watch" to start monitoring content changes.</p>
        </div>
      ) : (
        <div className="watch-list">
          <table style={{ width: "100%", borderCollapse: "collapse" }}>
            <thead>
              <tr style={{ borderBottom: "1px solid var(--stroke)" }}>
                <th style={{ textAlign: "left", padding: "8px 12px" }}>URL</th>
                <th style={{ textAlign: "left", padding: "8px 12px" }}>
                  Status
                </th>
                <th style={{ textAlign: "left", padding: "8px 12px" }}>
                  Interval
                </th>
                <th style={{ textAlign: "left", padding: "8px 12px" }}>
                  Changes
                </th>
                <th style={{ textAlign: "left", padding: "8px 12px" }}>
                  Last Checked
                </th>
                <th style={{ textAlign: "right", padding: "8px 12px" }}>
                  Actions
                </th>
              </tr>
            </thead>
            <tbody>
              {sortedWatches.map((watch) => (
                <tr
                  key={watch.id}
                  style={{ borderBottom: "1px solid var(--stroke)" }}
                >
                  <td style={{ padding: "12px" }}>
                    <div style={{ fontWeight: 500 }}>{watch.url}</div>
                    {watch.selector && (
                      <div
                        style={{
                          fontSize: 12,
                          color: "var(--muted)",
                          marginTop: 2,
                        }}
                      >
                        Selector: {watch.selector}
                      </div>
                    )}
                  </td>
                  <td style={{ padding: "12px" }}>
                    <span
                      style={{
                        display: "inline-flex",
                        alignItems: "center",
                        gap: 6,
                        padding: "4px 10px",
                        borderRadius: 12,
                        fontSize: 12,
                        fontWeight: 500,
                        backgroundColor:
                          watch.status === "active"
                            ? "rgba(34, 197, 94, 0.15)"
                            : "rgba(156, 163, 175, 0.15)",
                        color:
                          watch.status === "active"
                            ? "#22c55e"
                            : "var(--muted)",
                      }}
                    >
                      <span
                        style={{
                          width: 6,
                          height: 6,
                          borderRadius: "50%",
                          backgroundColor:
                            watch.status === "active"
                              ? "#22c55e"
                              : "var(--muted)",
                        }}
                      />
                      {watch.status}
                    </span>
                  </td>
                  <td style={{ padding: "12px" }}>
                    {formatDuration(watch.intervalSeconds)}
                  </td>
                  <td style={{ padding: "12px" }}>
                    <span
                      style={{
                        fontWeight: 600,
                        color:
                          (watch.changeCount || 0) > 0
                            ? "var(--accent)"
                            : "inherit",
                      }}
                    >
                      {watch.changeCount || 0}
                    </span>
                  </td>
                  <td style={{ padding: "12px", fontSize: 13 }}>
                    {formatDate(watch.lastCheckedAt)}
                  </td>
                  <td style={{ padding: "12px", textAlign: "right" }}>
                    <div
                      className="row"
                      style={{ gap: 6, justifyContent: "flex-end" }}
                    >
                      <button
                        type="button"
                        onClick={() => handleCheck(watch)}
                        disabled={checkingId === watch.id}
                        className="secondary"
                        style={{ padding: "6px 12px", fontSize: 12 }}
                        title="Check now"
                      >
                        {checkingId === watch.id ? "Checking..." : "Check"}
                      </button>
                      <button
                        type="button"
                        onClick={() => handleEditClick(watch)}
                        className="secondary"
                        style={{ padding: "6px 12px", fontSize: 12 }}
                      >
                        Edit
                      </button>
                      {deleteConfirmId === watch.id ? (
                        <>
                          <button
                            type="button"
                            onClick={() => handleDelete(watch.id)}
                            style={{
                              padding: "6px 12px",
                              fontSize: 12,
                              backgroundColor: "#ef4444",
                            }}
                          >
                            Confirm
                          </button>
                          <button
                            type="button"
                            onClick={() => setDeleteConfirmId(null)}
                            className="secondary"
                            style={{ padding: "6px 12px", fontSize: 12 }}
                          >
                            Cancel
                          </button>
                        </>
                      ) : (
                        <button
                          type="button"
                          onClick={() => setDeleteConfirmId(watch.id)}
                          className="secondary"
                          style={{ padding: "6px 12px", fontSize: 12 }}
                        >
                          Delete
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Check Result Modal */}
      {checkResult && (
        <div
          style={{
            position: "fixed",
            inset: 0,
            backgroundColor: "rgba(0, 0, 0, 0.7)",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            zIndex: 1000,
            padding: 20,
          }}
        >
          <div
            className="panel"
            style={{
              maxWidth: 700,
              width: "100%",
              maxHeight: "80vh",
              overflow: "auto",
            }}
          >
            <div
              className="row"
              style={{
                justifyContent: "space-between",
                alignItems: "center",
                marginBottom: 16,
              }}
            >
              <h3 style={{ margin: 0 }}>Check Result</h3>
              <button
                type="button"
                onClick={() => setCheckResult(null)}
                className="secondary"
              >
                Close
              </button>
            </div>

            <div style={{ marginBottom: 16 }}>
              <div className="row" style={{ gap: 16, marginBottom: 8 }}>
                <span>
                  <strong>Changed:</strong>{" "}
                  <span
                    style={{
                      color: checkResult.changed ? "#22c55e" : "var(--muted)",
                      fontWeight: 600,
                    }}
                  >
                    {checkResult.changed ? "Yes" : "No"}
                  </span>
                </span>
                <span>
                  <strong>Checked At:</strong>{" "}
                  {formatDate(checkResult.checkedAt)}
                </span>
              </div>
              {checkResult.error && (
                <div
                  style={{
                    padding: 12,
                    backgroundColor: "rgba(239, 68, 68, 0.1)",
                    borderRadius: 8,
                    color: "#ef4444",
                    marginTop: 8,
                  }}
                >
                  <strong>Error:</strong> {checkResult.error}
                </div>
              )}
            </div>

            {checkResult.diffText && (
              <div>
                <h4 style={{ marginBottom: 8 }}>Diff</h4>
                <pre
                  style={{
                    backgroundColor: "var(--bg-alt)",
                    padding: 16,
                    borderRadius: 8,
                    overflow: "auto",
                    fontSize: 12,
                    lineHeight: 1.5,
                    maxHeight: 300,
                  }}
                >
                  {checkResult.diffText}
                </pre>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Create/Edit Form Modal */}
      {showForm && (
        <div
          style={{
            position: "fixed",
            inset: 0,
            backgroundColor: "rgba(0, 0, 0, 0.7)",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            zIndex: 1000,
            padding: 20,
          }}
        >
          <div
            className="panel"
            style={{
              maxWidth: 600,
              width: "100%",
              maxHeight: "90vh",
              overflow: "auto",
            }}
          >
            <div
              className="row"
              style={{
                justifyContent: "space-between",
                alignItems: "center",
                marginBottom: 16,
              }}
            >
              <h3 style={{ margin: 0 }}>
                {editingId ? "Edit Watch" : "Create Watch"}
              </h3>
              <button
                type="button"
                onClick={handleCloseForm}
                className="secondary"
              >
                Cancel
              </button>
            </div>

            {formError && (
              <div
                style={{
                  padding: 12,
                  backgroundColor: "rgba(239, 68, 68, 0.1)",
                  borderRadius: 8,
                  color: "#ef4444",
                  marginBottom: 16,
                }}
              >
                {formError}
              </div>
            )}

            <form
              onSubmit={(e) => {
                e.preventDefault();
                handleSubmit();
              }}
            >
              <div style={{ marginBottom: 16 }}>
                <label
                  htmlFor="watch-url"
                  style={{ display: "block", marginBottom: 4 }}
                >
                  URL <span style={{ color: "#ef4444" }}>*</span>
                </label>
                <input
                  id="watch-url"
                  type="url"
                  value={formData.url}
                  onChange={(e) =>
                    setFormData((prev) => ({ ...prev, url: e.target.value }))
                  }
                  placeholder="https://example.com/page"
                  required
                  style={{ width: "100%" }}
                />
              </div>

              <div style={{ marginBottom: 16 }}>
                <label
                  htmlFor="watch-selector"
                  style={{ display: "block", marginBottom: 4 }}
                >
                  CSS Selector (optional)
                </label>
                <input
                  id="watch-selector"
                  type="text"
                  value={formData.selector}
                  onChange={(e) =>
                    setFormData((prev) => ({
                      ...prev,
                      selector: e.target.value,
                    }))
                  }
                  placeholder=".content, #main, article"
                  style={{ width: "100%" }}
                />
                <small style={{ color: "var(--muted)" }}>
                  Extract only content matching this selector
                </small>
              </div>

              <div className="row" style={{ gap: 16, marginBottom: 16 }}>
                <div style={{ flex: 1 }}>
                  <label
                    htmlFor="watch-interval"
                    style={{ display: "block", marginBottom: 4 }}
                  >
                    Check Interval (seconds){" "}
                    <span style={{ color: "#ef4444" }}>*</span>
                  </label>
                  <input
                    id="watch-interval"
                    type="number"
                    min={60}
                    step={60}
                    value={formData.intervalSeconds}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        intervalSeconds: parseInt(e.target.value, 10) || 60,
                      }))
                    }
                    required
                    style={{ width: "100%" }}
                  />
                  <small style={{ color: "var(--muted)" }}>
                    Minimum 60 seconds (
                    {formatDuration(formData.intervalSeconds)})
                  </small>
                </div>

                <div style={{ flex: 1 }}>
                  <label
                    htmlFor="watch-diff-format"
                    style={{ display: "block", marginBottom: 4 }}
                  >
                    Diff Format
                  </label>
                  <select
                    id="watch-diff-format"
                    value={formData.diffFormat}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        diffFormat: e.target
                          .value as WatchFormData["diffFormat"],
                      }))
                    }
                    style={{ width: "100%" }}
                  >
                    <option value="unified">Unified Text</option>
                    <option value="html-side-by-side">HTML Side-by-Side</option>
                    <option value="html-inline">HTML Inline</option>
                  </select>
                </div>
              </div>

              <div className="row" style={{ gap: 16, marginBottom: 16 }}>
                <div style={{ flex: 1 }}>
                  <label
                    htmlFor="watch-extract-mode"
                    style={{ display: "block", marginBottom: 4 }}
                  >
                    Extract Mode
                  </label>
                  <select
                    id="watch-extract-mode"
                    value={formData.extractMode}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        extractMode: e.target
                          .value as WatchFormData["extractMode"],
                      }))
                    }
                    style={{ width: "100%" }}
                  >
                    <option value="">HTML (default)</option>
                    <option value="text">Plain Text</option>
                    <option value="markdown">Markdown</option>
                  </select>
                </div>

                <div style={{ flex: 1 }}>
                  <label
                    htmlFor="watch-min-size"
                    style={{ display: "block", marginBottom: 4 }}
                  >
                    Min Change Size (bytes)
                  </label>
                  <input
                    id="watch-min-size"
                    type="number"
                    min={0}
                    value={formData.minChangeSize}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        minChangeSize: e.target.value,
                      }))
                    }
                    placeholder="0"
                    style={{ width: "100%" }}
                  />
                </div>
              </div>

              <div style={{ marginBottom: 16 }}>
                <label
                  htmlFor="watch-ignore-patterns"
                  style={{ display: "block", marginBottom: 4 }}
                >
                  Ignore Patterns (one regex per line)
                </label>
                <textarea
                  id="watch-ignore-patterns"
                  value={formData.ignorePatterns}
                  onChange={(e) =>
                    setFormData((prev) => ({
                      ...prev,
                      ignorePatterns: e.target.value,
                    }))
                  }
                  placeholder="\\d{4}-\\d{2}-\\d{2}  # Date patterns&#10;timestamp: \\d+  # Timestamps"
                  rows={3}
                  style={{
                    width: "100%",
                    fontFamily: "monospace",
                    fontSize: 12,
                  }}
                />
              </div>

              <div
                className="row"
                style={{ gap: 16, marginBottom: 16, alignItems: "center" }}
              >
                <label
                  style={{ display: "flex", alignItems: "center", gap: 8 }}
                >
                  <input
                    type="checkbox"
                    checked={formData.enabled}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        enabled: e.target.checked,
                      }))
                    }
                  />
                  Enabled
                </label>

                <label
                  style={{ display: "flex", alignItems: "center", gap: 8 }}
                >
                  <input
                    type="checkbox"
                    checked={formData.headless}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        headless: e.target.checked,
                      }))
                    }
                  />
                  Use Headless Browser
                </label>

                <label
                  style={{ display: "flex", alignItems: "center", gap: 8 }}
                >
                  <input
                    type="checkbox"
                    checked={formData.usePlaywright}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        usePlaywright: e.target.checked,
                      }))
                    }
                  />
                  Use Playwright
                </label>
              </div>

              <div style={{ marginBottom: 16 }}>
                <label
                  style={{ display: "flex", alignItems: "center", gap: 8 }}
                >
                  <input
                    type="checkbox"
                    checked={formData.notifyOnChange}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        notifyOnChange: e.target.checked,
                      }))
                    }
                  />
                  Send Webhook Notification on Change
                </label>
              </div>

              {formData.notifyOnChange && (
                <div
                  style={{
                    marginBottom: 16,
                    padding: 16,
                    backgroundColor: "var(--bg-alt)",
                    borderRadius: 8,
                  }}
                >
                  <div style={{ marginBottom: 12 }}>
                    <label
                      htmlFor="watch-webhook-url"
                      style={{ display: "block", marginBottom: 4 }}
                    >
                      Webhook URL
                    </label>
                    <input
                      id="watch-webhook-url"
                      type="url"
                      value={formData.webhookUrl}
                      onChange={(e) =>
                        setFormData((prev) => ({
                          ...prev,
                          webhookUrl: e.target.value,
                        }))
                      }
                      placeholder="https://hooks.example.com/webhook"
                      style={{ width: "100%" }}
                    />
                  </div>
                  <div>
                    <label
                      htmlFor="watch-webhook-secret"
                      style={{ display: "block", marginBottom: 4 }}
                    >
                      Webhook Secret (optional)
                    </label>
                    <input
                      id="watch-webhook-secret"
                      type="password"
                      value={formData.webhookSecret}
                      onChange={(e) =>
                        setFormData((prev) => ({
                          ...prev,
                          webhookSecret: e.target.value,
                        }))
                      }
                      placeholder="secret-for-hmac-signature"
                      style={{ width: "100%" }}
                    />
                  </div>
                </div>
              )}

              <div
                className="row"
                style={{ gap: 8, justifyContent: "flex-end" }}
              >
                <button
                  type="button"
                  onClick={handleCloseForm}
                  className="secondary"
                  disabled={formSubmitting}
                >
                  Cancel
                </button>
                <button type="submit" disabled={formSubmitting}>
                  {formSubmitting
                    ? "Saving..."
                    : editingId
                      ? "Update Watch"
                      : "Create Watch"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
