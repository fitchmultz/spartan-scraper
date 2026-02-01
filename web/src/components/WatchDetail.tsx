/**
 * Watch Detail Component
 *
 * Displays detailed information about a single watch including configuration,
 * status, and recent check results with diff display.
 *
 * @module WatchDetail
 */

import { useState } from "react";
import type { Watch, WatchCheckResult } from "../api";

interface WatchDetailProps {
  watch: Watch;
  checkResult?: WatchCheckResult | null;
  onCheck: () => Promise<void>;
  onClose: () => void;
  loading?: boolean;
}

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

function getNextRun(watch: Watch): string {
  if (!watch.enabled) return "Disabled";
  if (!watch.lastCheckedAt) return "Due now";
  const lastChecked = new Date(watch.lastCheckedAt).getTime();
  const nextRun = lastChecked + watch.intervalSeconds * 1000;
  const now = Date.now();
  if (nextRun <= now) return "Due now";
  const diff = Math.ceil((nextRun - now) / 1000);
  if (diff < 60) return `in ${diff}s`;
  if (diff < 3600) return `in ${Math.floor(diff / 60)}m`;
  return `in ${Math.floor(diff / 3600)}h`;
}

export function WatchDetail({
  watch,
  checkResult,
  onCheck,
  onClose,
  loading,
}: WatchDetailProps) {
  const [activeTab, setActiveTab] = useState<"overview" | "diff" | "config">(
    "overview",
  );

  const renderDiff = () => {
    if (!checkResult) {
      return (
        <div
          style={{
            padding: 40,
            textAlign: "center",
            color: "var(--muted)",
          }}
        >
          <p>No check results available.</p>
          <button type="button" onClick={onCheck} disabled={loading}>
            {loading ? "Checking..." : "Run Check Now"}
          </button>
        </div>
      );
    }

    if (checkResult.error) {
      return (
        <div
          style={{
            padding: 20,
            backgroundColor: "rgba(239, 68, 68, 0.1)",
            borderRadius: 8,
            color: "#ef4444",
          }}
        >
          <strong>Check Failed:</strong> {checkResult.error}
        </div>
      );
    }

    if (!checkResult.changed) {
      return (
        <div
          style={{
            padding: 40,
            textAlign: "center",
            color: "var(--muted)",
          }}
        >
          <p style={{ fontSize: 18, marginBottom: 8 }}>✓ No Changes Detected</p>
          <p>Content hash matches the previous snapshot.</p>
          {checkResult.screenshotPath && (
            <p style={{ fontSize: 12, marginTop: 8 }}>
              Screenshot captured: <code>{checkResult.screenshotPath}</code>
            </p>
          )}
          <p style={{ fontSize: 12, marginTop: 16 }}>
            Current Hash:{" "}
            <code>{checkResult.currentHash?.slice(0, 16)}...</code>
          </p>
        </div>
      );
    }

    return (
      <div>
        <div
          className="row"
          style={{
            gap: 16,
            marginBottom: 16,
            padding: 12,
            backgroundColor: "rgba(34, 197, 94, 0.1)",
            borderRadius: 8,
          }}
        >
          <span>
            <strong>Changed:</strong>{" "}
            <span style={{ color: "#22c55e", fontWeight: 600 }}>Yes</span>
          </span>
          {checkResult.visualChanged && (
            <span>
              <strong>Visual Change:</strong>{" "}
              <span style={{ color: "#3b82f6", fontWeight: 600 }}>Yes</span>
            </span>
          )}
          <span>
            <strong>Previous:</strong>{" "}
            <code>{checkResult.previousHash?.slice(0, 16) || "N/A"}...</code>
          </span>
          <span>
            <strong>Current:</strong>{" "}
            <code>{checkResult.currentHash?.slice(0, 16)}...</code>
          </span>
        </div>

        {/* Visual Diff Section */}
        {checkResult.visualChanged && (
          <div
            style={{
              marginBottom: 16,
              padding: 16,
              backgroundColor: "rgba(59, 130, 246, 0.1)",
              borderRadius: 8,
            }}
          >
            <h4 style={{ margin: "0 0 12px 0" }}>Visual Changes Detected</h4>
            <div className="row" style={{ gap: 16, flexWrap: "wrap" }}>
              {checkResult.visualSimilarity !== undefined && (
                <span>
                  <strong>Similarity:</strong>{" "}
                  {Math.round(checkResult.visualSimilarity * 100)}%
                </span>
              )}
              {checkResult.screenshotPath && (
                <span>
                  <strong>Screenshot:</strong>{" "}
                  <code style={{ fontSize: 12 }}>
                    {checkResult.screenshotPath}
                  </code>
                </span>
              )}
              {checkResult.visualDiffPath && (
                <span>
                  <strong>Diff:</strong>{" "}
                  <code style={{ fontSize: 12 }}>
                    {checkResult.visualDiffPath}
                  </code>
                </span>
              )}
            </div>
          </div>
        )}

        {checkResult.diffHtml ? (
          <div
            // biome-ignore lint/security/noDangerouslySetInnerHtml: Diff HTML is generated server-side from trusted sources
            dangerouslySetInnerHTML={{ __html: checkResult.diffHtml }}
            style={{
              backgroundColor: "var(--bg-alt)",
              borderRadius: 8,
              overflow: "auto",
            }}
          />
        ) : checkResult.diffText ? (
          <pre
            style={{
              backgroundColor: "var(--bg-alt)",
              padding: 16,
              borderRadius: 8,
              overflow: "auto",
              fontSize: 12,
              lineHeight: 1.5,
              maxHeight: 500,
              margin: 0,
            }}
          >
            {checkResult.diffText}
          </pre>
        ) : (
          <p style={{ color: "var(--muted)" }}>
            No diff available (first check or no previous content).
          </p>
        )}
      </div>
    );
  };

  return (
    <div className="panel">
      <div
        className="row"
        style={{
          justifyContent: "space-between",
          alignItems: "flex-start",
          marginBottom: 20,
        }}
      >
        <div>
          <h2 style={{ margin: "0 0 8px 0" }}>Watch Details</h2>
          <div
            style={{
              fontSize: 14,
              color: "var(--muted)",
              wordBreak: "break-all",
            }}
          >
            {watch.url}
          </div>
        </div>
        <div className="row" style={{ gap: 8 }}>
          <button
            type="button"
            onClick={onCheck}
            disabled={loading}
            className="secondary"
          >
            {loading ? "Checking..." : "Check Now"}
          </button>
          <button type="button" onClick={onClose} className="secondary">
            Close
          </button>
        </div>
      </div>

      {/* Status Bar */}
      <div
        className="row"
        style={{
          gap: 24,
          padding: 16,
          backgroundColor: "var(--bg-alt)",
          borderRadius: 8,
          marginBottom: 20,
          flexWrap: "wrap",
        }}
      >
        <div>
          <div style={{ fontSize: 12, color: "var(--muted)", marginBottom: 4 }}>
            Status
          </div>
          <div
            style={{
              display: "flex",
              alignItems: "center",
              gap: 6,
              fontWeight: 600,
            }}
          >
            <span
              style={{
                width: 8,
                height: 8,
                borderRadius: "50%",
                backgroundColor:
                  watch.status === "active" ? "#22c55e" : "var(--muted)",
              }}
            />
            <span
              style={{
                color: watch.status === "active" ? "#22c55e" : "inherit",
              }}
            >
              {watch.status}
            </span>
          </div>
        </div>

        <div>
          <div style={{ fontSize: 12, color: "var(--muted)", marginBottom: 4 }}>
            Interval
          </div>
          <div style={{ fontWeight: 500 }}>
            {formatDuration(watch.intervalSeconds)}
          </div>
        </div>

        <div>
          <div style={{ fontSize: 12, color: "var(--muted)", marginBottom: 4 }}>
            Changes Detected
          </div>
          <div
            style={{
              fontWeight: 600,
              color: (watch.changeCount || 0) > 0 ? "var(--accent)" : "inherit",
            }}
          >
            {watch.changeCount || 0}
          </div>
        </div>

        <div>
          <div style={{ fontSize: 12, color: "var(--muted)", marginBottom: 4 }}>
            Next Run
          </div>
          <div style={{ fontWeight: 500 }}>{getNextRun(watch)}</div>
        </div>

        <div>
          <div style={{ fontSize: 12, color: "var(--muted)", marginBottom: 4 }}>
            Last Changed
          </div>
          <div style={{ fontWeight: 500 }}>
            {formatDate(watch.lastChangedAt)}
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div
        style={{
          display: "flex",
          gap: 8,
          borderBottom: "1px solid var(--stroke)",
          marginBottom: 20,
        }}
      >
        {(["overview", "diff", "config"] as const).map((tab) => (
          <button
            key={tab}
            type="button"
            onClick={() => setActiveTab(tab)}
            style={{
              padding: "10px 16px",
              background: "none",
              border: "none",
              borderBottom: `2px solid ${activeTab === tab ? "var(--accent)" : "transparent"}`,
              color: activeTab === tab ? "var(--text)" : "var(--muted)",
              fontWeight: activeTab === tab ? 600 : 400,
              cursor: "pointer",
              textTransform: "capitalize",
            }}
          >
            {tab}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      {activeTab === "overview" && (
        <div>
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "repeat(auto-fit, minmax(250px, 1fr))",
              gap: 16,
            }}
          >
            <div
              style={{
                padding: 16,
                backgroundColor: "var(--bg-alt)",
                borderRadius: 8,
              }}
            >
              <div
                style={{
                  fontSize: 12,
                  color: "var(--muted)",
                  marginBottom: 8,
                  textTransform: "uppercase",
                  letterSpacing: "0.5px",
                }}
              >
                Created
              </div>
              <div>{formatDate(watch.createdAt)}</div>
            </div>

            <div
              style={{
                padding: 16,
                backgroundColor: "var(--bg-alt)",
                borderRadius: 8,
              }}
            >
              <div
                style={{
                  fontSize: 12,
                  color: "var(--muted)",
                  marginBottom: 8,
                  textTransform: "uppercase",
                  letterSpacing: "0.5px",
                }}
              >
                Last Checked
              </div>
              <div>{formatDate(watch.lastCheckedAt)}</div>
            </div>

            <div
              style={{
                padding: 16,
                backgroundColor: "var(--bg-alt)",
                borderRadius: 8,
              }}
            >
              <div
                style={{
                  fontSize: 12,
                  color: "var(--muted)",
                  marginBottom: 8,
                  textTransform: "uppercase",
                  letterSpacing: "0.5px",
                }}
              >
                Watch ID
              </div>
              <code style={{ fontSize: 12 }}>{watch.id}</code>
            </div>

            {watch.selector && (
              <div
                style={{
                  padding: 16,
                  backgroundColor: "var(--bg-alt)",
                  borderRadius: 8,
                }}
              >
                <div
                  style={{
                    fontSize: 12,
                    color: "var(--muted)",
                    marginBottom: 8,
                    textTransform: "uppercase",
                    letterSpacing: "0.5px",
                  }}
                >
                  CSS Selector
                </div>
                <code style={{ fontSize: 12 }}>{watch.selector}</code>
              </div>
            )}
          </div>

          {checkResult && (
            <div
              style={{
                marginTop: 20,
                padding: 16,
                backgroundColor: checkResult.changed
                  ? "rgba(34, 197, 94, 0.1)"
                  : "var(--bg-alt)",
                borderRadius: 8,
              }}
            >
              <h4 style={{ margin: "0 0 12px 0" }}>Last Check Result</h4>
              <div className="row" style={{ gap: 16, flexWrap: "wrap" }}>
                <span>
                  <strong>Changed:</strong>{" "}
                  {checkResult.changed ? (
                    <span style={{ color: "#22c55e" }}>Yes</span>
                  ) : (
                    "No"
                  )}
                </span>
                <span>
                  <strong>Checked At:</strong>{" "}
                  {formatDate(checkResult.checkedAt)}
                </span>
                {checkResult.error && (
                  <span style={{ color: "#ef4444" }}>
                    <strong>Error:</strong> {checkResult.error}
                  </span>
                )}
              </div>
            </div>
          )}
        </div>
      )}

      {activeTab === "diff" && renderDiff()}

      {activeTab === "config" && (
        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fit, minmax(300px, 1fr))",
            gap: 16,
          }}
        >
          <div
            style={{
              padding: 16,
              backgroundColor: "var(--bg-alt)",
              borderRadius: 8,
            }}
          >
            <h4 style={{ margin: "0 0 12px 0" }}>Extraction Settings</h4>
            <div style={{ fontSize: 14, lineHeight: 1.6 }}>
              <div
                className="row"
                style={{ justifyContent: "space-between", marginBottom: 8 }}
              >
                <span style={{ color: "var(--muted)" }}>Mode:</span>
                <span>{watch.extractMode || "HTML (default)"}</span>
              </div>
              <div
                className="row"
                style={{ justifyContent: "space-between", marginBottom: 8 }}
              >
                <span style={{ color: "var(--muted)" }}>Headless:</span>
                <span>{watch.headless ? "Yes" : "No"}</span>
              </div>
              <div
                className="row"
                style={{ justifyContent: "space-between", marginBottom: 8 }}
              >
                <span style={{ color: "var(--muted)" }}>Use Playwright:</span>
                <span>{watch.usePlaywright ? "Yes" : "No"}</span>
              </div>
              {watch.minChangeSize !== undefined && (
                <div
                  className="row"
                  style={{ justifyContent: "space-between" }}
                >
                  <span style={{ color: "var(--muted)" }}>
                    Min Change Size:
                  </span>
                  <span>{watch.minChangeSize} bytes</span>
                </div>
              )}
            </div>
          </div>

          <div
            style={{
              padding: 16,
              backgroundColor: "var(--bg-alt)",
              borderRadius: 8,
            }}
          >
            <h4 style={{ margin: "0 0 12px 0" }}>Visual Change Detection</h4>
            <div style={{ fontSize: 14, lineHeight: 1.6 }}>
              <div
                className="row"
                style={{ justifyContent: "space-between", marginBottom: 8 }}
              >
                <span style={{ color: "var(--muted)" }}>Enabled:</span>
                <span>{watch.screenshotEnabled ? "Yes" : "No"}</span>
              </div>
              {watch.screenshotEnabled && watch.screenshotConfig && (
                <>
                  <div
                    className="row"
                    style={{ justifyContent: "space-between", marginBottom: 8 }}
                  >
                    <span style={{ color: "var(--muted)" }}>Format:</span>
                    <span>{watch.screenshotConfig.format || "png"}</span>
                  </div>
                  <div
                    className="row"
                    style={{ justifyContent: "space-between", marginBottom: 8 }}
                  >
                    <span style={{ color: "var(--muted)" }}>Full Page:</span>
                    <span>
                      {watch.screenshotConfig.fullPage ? "Yes" : "No"}
                    </span>
                  </div>
                </>
              )}
              {watch.visualDiffThreshold !== undefined && (
                <div
                  className="row"
                  style={{ justifyContent: "space-between" }}
                >
                  <span style={{ color: "var(--muted)" }}>Diff Threshold:</span>
                  <span>{watch.visualDiffThreshold}</span>
                </div>
              )}
            </div>
          </div>

          <div
            style={{
              padding: 16,
              backgroundColor: "var(--bg-alt)",
              borderRadius: 8,
            }}
          >
            <h4 style={{ margin: "0 0 12px 0" }}>Diff & Notification</h4>
            <div style={{ fontSize: 14, lineHeight: 1.6 }}>
              <div
                className="row"
                style={{ justifyContent: "space-between", marginBottom: 8 }}
              >
                <span style={{ color: "var(--muted)" }}>Diff Format:</span>
                <span style={{ textTransform: "capitalize" }}>
                  {(watch.diffFormat || "unified").replace(/-/g, " ")}
                </span>
              </div>
              <div
                className="row"
                style={{ justifyContent: "space-between", marginBottom: 8 }}
              >
                <span style={{ color: "var(--muted)" }}>Notify on Change:</span>
                <span>{watch.notifyOnChange ? "Yes" : "No"}</span>
              </div>
              {watch.webhookConfig && (
                <div
                  className="row"
                  style={{ justifyContent: "space-between" }}
                >
                  <span style={{ color: "var(--muted)" }}>Webhook URL:</span>
                  <span
                    style={{
                      fontSize: 12,
                      maxWidth: 200,
                      wordBreak: "break-all",
                    }}
                  >
                    {watch.webhookConfig.url}
                  </span>
                </div>
              )}
            </div>
          </div>

          {watch.ignorePatterns && watch.ignorePatterns.length > 0 && (
            <div
              style={{
                padding: 16,
                backgroundColor: "var(--bg-alt)",
                borderRadius: 8,
                gridColumn: "1 / -1",
              }}
            >
              <h4 style={{ margin: "0 0 12px 0" }}>Ignore Patterns</h4>
              <ul style={{ margin: 0, paddingLeft: 20 }}>
                {watch.ignorePatterns.map((pattern) => (
                  <li key={pattern}>
                    <code style={{ fontSize: 12 }}>{pattern}</code>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
