/**
 * Traffic Inspector Component
 *
 * Purpose:
 * - Display captured network traffic and replay controls for a selected job.
 *
 * Responsibilities:
 * - Filter and search intercepted entries.
 * - Show request/response details for a selected entry.
 * - Launch traffic replay and surface replay results.
 *
 * Scope:
 * - Traffic inspection UI only.
 *
 * Usage:
 * - Rendered from results views when network interception data exists.
 *
 * Invariants/Assumptions:
 * - Replay always targets an existing job id.
 * - Entry selection is cleared only by explicit user action.
 * - Empty replay filters are omitted from the API request.
 */

import { useMemo, useState } from "react";

import {
  postV1JobsByIdReplay,
  type InterceptedEntry,
  type TrafficReplayResponse,
} from "../api";
import {
  formatMillisecondsAsDuration,
  truncateMiddle,
} from "../lib/formatting";
import { getDetailedHttpStatusClass } from "../lib/http-status";

import {
  buildReplayRequest,
  filterTrafficEntries,
  formatTrafficBytes,
  getResourceTypeLabel,
  getTrafficEntryKey,
  summarizeTrafficEntries,
  type TrafficFilterType,
} from "./traffic-inspector/trafficInspectorUtils";

interface TrafficInspectorProps {
  entries: InterceptedEntry[];
  jobId: string;
}

interface TrafficReplayModalProps {
  loading: boolean;
  replayResult: TrafficReplayResponse | null;
  replayError: string | null;
  replayTargetURL: string;
  replayCompare: boolean;
  replayFilterURL: string;
  replayFilterMethod: string;
  onChangeTargetURL: (value: string) => void;
  onChangeCompare: (value: boolean) => void;
  onChangeFilterURL: (value: string) => void;
  onChangeFilterMethod: (value: string) => void;
  onClose: () => void;
  onReplay: () => void;
}

function TrafficReplayModal({
  loading,
  replayResult,
  replayError,
  replayTargetURL,
  replayCompare,
  replayFilterURL,
  replayFilterMethod,
  onChangeTargetURL,
  onChangeCompare,
  onChangeFilterURL,
  onChangeFilterMethod,
  onClose,
  onReplay,
}: TrafficReplayModalProps) {
  return (
    <div
      role="dialog"
      aria-modal="true"
      style={modalBackdropStyle}
      onClick={onClose}
      onKeyDown={(event) => {
        if (event.key === "Escape") {
          onClose();
        }
      }}
    >
      <div
        role="document"
        tabIndex={-1}
        style={modalCardStyle}
        onClick={(event) => event.stopPropagation()}
        onKeyDown={(event) => {
          if (event.key === "Escape") {
            onClose();
          }
        }}
      >
        <div style={modalHeaderStyle}>
          <h3>Replay Traffic</h3>
          <button
            type="button"
            className="secondary"
            onClick={onClose}
            style={{ padding: "4px 12px" }}
          >
            ×
          </button>
        </div>

        <p style={modalMutedTextStyle}>
          Replay captured network requests against a target URL.
        </p>

        <label htmlFor="replay-target-url">
          Target Base URL <span style={{ color: "var(--accent)" }}>*</span>
        </label>
        <input
          id="replay-target-url"
          type="url"
          value={replayTargetURL}
          onChange={(event) => onChangeTargetURL(event.target.value)}
          placeholder="https://staging.example.com"
          disabled={loading}
        />

        <label htmlFor="replay-filter-url" style={{ marginTop: 12 }}>
          URL Pattern Filter (optional)
        </label>
        <input
          id="replay-filter-url"
          value={replayFilterURL}
          onChange={(event) => onChangeFilterURL(event.target.value)}
          placeholder="**/api/**, *.json"
          disabled={loading}
        />
        <small>Comma-separated glob patterns to filter requests</small>

        <label htmlFor="replay-filter-method" style={{ marginTop: 12 }}>
          HTTP Method Filter (optional)
        </label>
        <input
          id="replay-filter-method"
          value={replayFilterMethod}
          onChange={(event) => onChangeFilterMethod(event.target.value)}
          placeholder="GET, POST"
          disabled={loading}
        />
        <small>Comma-separated HTTP methods to filter</small>

        <label style={modalCheckboxRowStyle}>
          <input
            type="checkbox"
            checked={replayCompare}
            onChange={(event) => onChangeCompare(event.target.checked)}
            disabled={loading}
          />
          Compare responses with original
        </label>

        {replayError && <div style={modalErrorStyle}>{replayError}</div>}

        {replayResult && (
          <div style={modalResultsStyle}>
            <h4>Replay Results</h4>
            <div style={modalStatsGridStyle}>
              <ReplayStat label="Total" value={replayResult.totalRequests} />
              <ReplayStat
                label="Successful"
                value={replayResult.successful}
                color="#22c55e"
              />
              <ReplayStat
                label="Failed"
                value={replayResult.failed}
                color="#ef4444"
              />
            </div>
            {replayResult.comparison && (
              <div style={{ marginTop: 12 }}>
                <div style={modalSubtleMetricStyle}>
                  Matches: {replayResult.comparison.matches} /{" "}
                  {replayResult.comparison.totalCompared}
                </div>
                <div style={modalSubtleMetricStyle}>
                  Mismatches: {replayResult.comparison.mismatches}
                </div>
              </div>
            )}
            {replayResult.durationMs && (
              <div style={{ ...modalSubtleMetricStyle, marginTop: 8 }}>
                Duration: {(replayResult.durationMs / 1000).toFixed(2)}s
              </div>
            )}
          </div>
        )}

        <div style={{ marginTop: 24, display: "flex", gap: 12 }}>
          <button
            type="button"
            onClick={onReplay}
            disabled={loading || !replayTargetURL}
          >
            {loading ? "Replaying..." : "Run Replay"}
          </button>
          <button
            type="button"
            className="secondary"
            onClick={onClose}
            disabled={loading}
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}

function ReplayStat({
  label,
  value,
  color,
}: {
  label: string;
  value: number | undefined;
  color?: string;
}) {
  return (
    <div>
      <div style={modalSubtleMetricStyle}>{label}</div>
      <div
        style={{
          fontSize: "1.5rem",
          fontWeight: 600,
          ...(color ? { color } : {}),
        }}
      >
        {value ?? 0}
      </div>
    </div>
  );
}

function TrafficEntriesTable({
  entries,
  selectedEntry,
  onSelectEntry,
}: {
  entries: InterceptedEntry[];
  selectedEntry: InterceptedEntry | null;
  onSelectEntry: (entry: InterceptedEntry) => void;
}) {
  return (
    <div className="traffic-list">
      <table className="traffic-table">
        <thead>
          <tr>
            <th>Method</th>
            <th>URL</th>
            <th>Type</th>
            <th>Status</th>
            <th>Size</th>
            <th>Duration</th>
          </tr>
        </thead>
        <tbody>
          {entries.map((entry) => (
            <tr
              key={getTrafficEntryKey(entry)}
              className={selectedEntry === entry ? "selected" : ""}
              onClick={() => onSelectEntry(entry)}
            >
              <td className="method-cell">
                <span
                  className={`method method-${entry.request?.method?.toLowerCase() || "get"}`}
                >
                  {entry.request?.method || "GET"}
                </span>
              </td>
              <td className="url-cell" title={entry.request?.url}>
                {truncateMiddle(entry.request?.url, 50, "")}
              </td>
              <td className="type-cell">
                {getResourceTypeLabel(entry.request?.resourceType)}
              </td>
              <td className="status-cell">
                {entry.response ? (
                  <span
                    className={`status ${getDetailedHttpStatusClass(entry.response.status)}`}
                  >
                    {entry.response.status}
                  </span>
                ) : (
                  <span className="status pending">-</span>
                )}
              </td>
              <td className="size-cell">
                {formatTrafficBytes(entry.response?.bodySize)}
              </td>
              <td className="duration-cell">
                {formatMillisecondsAsDuration(entry.duration)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {entries.length === 0 && (
        <div className="traffic-no-results">No matching entries found</div>
      )}
    </div>
  );
}

function HeaderRows({ headers }: { headers?: Record<string, string> }) {
  if (!headers || Object.keys(headers).length === 0) {
    return <div className="header-row">No headers captured</div>;
  }

  return (
    <div className="headers-list">
      {Object.entries(headers).map(([key, value]) => (
        <div key={key} className="header-row">
          <span className="header-name">{key}:</span>
          <span className="header-value">{value}</span>
        </div>
      ))}
    </div>
  );
}

function TrafficDetailPanel({
  entry,
  showRequestBody,
  showResponseBody,
  onToggleRequestBody,
  onToggleResponseBody,
  onClose,
}: {
  entry: InterceptedEntry;
  showRequestBody: boolean;
  showResponseBody: boolean;
  onToggleRequestBody: (value: boolean) => void;
  onToggleResponseBody: (value: boolean) => void;
  onClose: () => void;
}) {
  return (
    <div className="traffic-detail">
      <div className="traffic-detail-header">
        <h5>Request Details</h5>
        <button type="button" className="close-btn" onClick={onClose}>
          ×
        </button>
      </div>

      <div className="traffic-detail-content">
        <div className="detail-section">
          <h6>General</h6>
          <div className="detail-row">
            <span className="detail-label">URL:</span>
            <span className="detail-value url">{entry.request?.url}</span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Method:</span>
            <span className="detail-value">{entry.request?.method}</span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Resource Type:</span>
            <span className="detail-value">{entry.request?.resourceType}</span>
          </div>
          <div className="detail-row">
            <span className="detail-label">Duration:</span>
            <span className="detail-value">
              {formatMillisecondsAsDuration(entry.duration)}
            </span>
          </div>
        </div>

        <div className="detail-section">
          <h6>Request Headers</h6>
          <HeaderRows
            headers={
              entry.request?.headers as Record<string, string> | undefined
            }
          />
        </div>

        {entry.request?.body && (
          <div className="detail-section">
            <h6>
              <label className="toggle-label">
                <input
                  type="checkbox"
                  checked={showRequestBody}
                  onChange={(event) =>
                    onToggleRequestBody(event.target.checked)
                  }
                />
                Request Body ({formatTrafficBytes(entry.request.bodySize)})
              </label>
            </h6>
            {showRequestBody && (
              <pre className="body-content">{entry.request.body}</pre>
            )}
          </div>
        )}

        {entry.response && (
          <>
            <div className="detail-section">
              <h6>Response</h6>
              <div className="detail-row">
                <span className="detail-label">Status:</span>
                <span
                  className={`detail-value status ${getDetailedHttpStatusClass(entry.response.status)}`}
                >
                  {entry.response.status} {entry.response.statusText}
                </span>
              </div>
              <div className="detail-row">
                <span className="detail-label">Size:</span>
                <span className="detail-value">
                  {formatTrafficBytes(entry.response.bodySize)}
                </span>
              </div>
            </div>

            <div className="detail-section">
              <h6>Response Headers</h6>
              <HeaderRows
                headers={
                  entry.response.headers as Record<string, string> | undefined
                }
              />
            </div>

            {entry.response.body && (
              <div className="detail-section">
                <h6>
                  <label className="toggle-label">
                    <input
                      type="checkbox"
                      checked={showResponseBody}
                      onChange={(event) =>
                        onToggleResponseBody(event.target.checked)
                      }
                    />
                    Response Body ({formatTrafficBytes(entry.response.bodySize)}
                    )
                  </label>
                </h6>
                {showResponseBody && (
                  <pre className="body-content">{entry.response.body}</pre>
                )}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}

export function TrafficInspector({ entries, jobId }: TrafficInspectorProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [filterType, setFilterType] = useState<TrafficFilterType>("all");
  const [selectedEntry, setSelectedEntry] = useState<InterceptedEntry | null>(
    null,
  );
  const [showRequestBody, setShowRequestBody] = useState(true);
  const [showResponseBody, setShowResponseBody] = useState(true);
  const [showReplayModal, setShowReplayModal] = useState(false);
  const [replayTargetURL, setReplayTargetURL] = useState("");
  const [replayCompare, setReplayCompare] = useState(false);
  const [replayFilterURL, setReplayFilterURL] = useState("");
  const [replayFilterMethod, setReplayFilterMethod] = useState("");
  const [replayLoading, setReplayLoading] = useState(false);
  const [replayResult, setReplayResult] =
    useState<TrafficReplayResponse | null>(null);
  const [replayError, setReplayError] = useState<string | null>(null);

  const filteredEntries = useMemo(
    () => filterTrafficEntries(entries, filterType, searchQuery),
    [entries, filterType, searchQuery],
  );
  const stats = useMemo(() => summarizeTrafficEntries(entries), [entries]);

  const handleExportHar = () => {
    window.open(`/v1/jobs/${jobId}/results?format=har`, "_blank");
  };

  const resetReplayState = () => {
    setShowReplayModal(false);
    setReplayTargetURL("");
    setReplayCompare(false);
    setReplayFilterURL("");
    setReplayFilterMethod("");
    setReplayResult(null);
    setReplayError(null);
  };

  const handleReplay = async () => {
    if (!replayTargetURL) {
      setReplayError("Target URL is required");
      return;
    }

    setReplayLoading(true);
    setReplayError(null);
    setReplayResult(null);

    try {
      const response = await postV1JobsByIdReplay({
        path: { id: jobId },
        body: buildReplayRequest(
          jobId,
          replayTargetURL,
          replayCompare,
          replayFilterURL,
          replayFilterMethod,
        ),
      });

      if (response.data) {
        setReplayResult(response.data);
      } else if (response.error) {
        setReplayError(String(response.error) || "Replay failed");
      }
    } catch (error) {
      setReplayError(error instanceof Error ? error.message : "Replay failed");
    } finally {
      setReplayLoading(false);
    }
  };

  if (entries.length === 0) {
    return (
      <div className="traffic-inspector empty">
        <div className="traffic-inspector-header">
          <h4>Network Traffic</h4>
        </div>
        <div className="traffic-empty-state">
          <p>No network traffic captured for this job.</p>
          <p className="traffic-empty-hint">
            Traffic capture is only available when using headless mode (chromedp
            or playwright) with network interception enabled.
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="traffic-inspector">
      <div className="traffic-inspector-header">
        <h4>Network Traffic</h4>
        <div className="traffic-stats">
          <span className="traffic-stat">{stats.total} requests</span>
          <span className="traffic-stat">{stats.withResponse} responses</span>
          <span className="traffic-stat">
            {formatTrafficBytes(stats.totalSize)} total
          </span>
          <span className="traffic-stat">
            avg {formatMillisecondsAsDuration(stats.avgDuration)}
          </span>
        </div>
        <div style={{ display: "flex", gap: 8 }}>
          <button
            type="button"
            className="secondary"
            onClick={() => setShowReplayModal(true)}
          >
            Replay Traffic
          </button>
          <button type="button" className="secondary" onClick={handleExportHar}>
            Export HAR
          </button>
        </div>
      </div>

      <div className="traffic-toolbar">
        <div className="traffic-search">
          <input
            type="text"
            placeholder="Search by URL or method..."
            value={searchQuery}
            onChange={(event) => setSearchQuery(event.target.value)}
          />
          {searchQuery && (
            <button
              type="button"
              className="search-clear"
              onClick={() => setSearchQuery("")}
            >
              ×
            </button>
          )}
        </div>
        <select
          value={filterType}
          onChange={(event) =>
            setFilterType(event.target.value as TrafficFilterType)
          }
          className="traffic-filter"
        >
          <option value="all">All Types</option>
          <option value="xhr">XHR</option>
          <option value="fetch">Fetch</option>
          <option value="document">Document</option>
          <option value="script">Script</option>
          <option value="stylesheet">Stylesheet</option>
          <option value="image">Image</option>
          <option value="other">Other</option>
        </select>
      </div>

      <div className="traffic-content">
        <TrafficEntriesTable
          entries={filteredEntries}
          selectedEntry={selectedEntry}
          onSelectEntry={setSelectedEntry}
        />
        {selectedEntry && (
          <TrafficDetailPanel
            entry={selectedEntry}
            showRequestBody={showRequestBody}
            showResponseBody={showResponseBody}
            onToggleRequestBody={setShowRequestBody}
            onToggleResponseBody={setShowResponseBody}
            onClose={() => setSelectedEntry(null)}
          />
        )}
      </div>

      {showReplayModal && (
        <TrafficReplayModal
          loading={replayLoading}
          replayResult={replayResult}
          replayError={replayError}
          replayTargetURL={replayTargetURL}
          replayCompare={replayCompare}
          replayFilterURL={replayFilterURL}
          replayFilterMethod={replayFilterMethod}
          onChangeTargetURL={setReplayTargetURL}
          onChangeCompare={setReplayCompare}
          onChangeFilterURL={setReplayFilterURL}
          onChangeFilterMethod={setReplayFilterMethod}
          onClose={resetReplayState}
          onReplay={handleReplay}
        />
      )}
    </div>
  );
}

const modalBackdropStyle = {
  position: "fixed",
  top: 0,
  left: 0,
  right: 0,
  bottom: 0,
  background: "rgba(0, 0, 0, 0.7)",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  zIndex: 1000,
} as const;

const modalCardStyle = {
  background: "var(--panel-bg)",
  borderRadius: 12,
  padding: 24,
  maxWidth: 600,
  width: "90%",
  maxHeight: "80vh",
  overflow: "auto",
  outline: "none",
} as const;

const modalHeaderStyle = {
  display: "flex",
  justifyContent: "space-between",
  alignItems: "center",
  marginBottom: 16,
} as const;

const modalMutedTextStyle = {
  marginBottom: 16,
  color: "var(--text-secondary)",
} as const;

const modalCheckboxRowStyle = {
  marginTop: 12,
  display: "flex",
  alignItems: "center",
  gap: 8,
  cursor: "pointer",
} as const;

const modalErrorStyle = {
  marginTop: 16,
  padding: 12,
  borderRadius: 8,
  background: "rgba(239, 68, 68, 0.2)",
  color: "#ef4444",
} as const;

const modalResultsStyle = {
  marginTop: 16,
  padding: 16,
  borderRadius: 8,
  background: "rgba(0, 0, 0, 0.2)",
} as const;

const modalStatsGridStyle = {
  display: "grid",
  gridTemplateColumns: "repeat(3, 1fr)",
  gap: 12,
  marginTop: 12,
} as const;

const modalSubtleMetricStyle = {
  fontSize: "0.85rem",
  color: "var(--text-secondary)",
} as const;

export default TrafficInspector;
