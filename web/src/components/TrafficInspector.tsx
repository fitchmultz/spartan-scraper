/**
 * Traffic Inspector Component
 *
 * Displays and inspects captured network traffic from job results.
 * Shows a list of intercepted requests/responses with filtering,
 * search capabilities, and detailed view for individual entries.
 *
 * @module TrafficInspector
 */
import { useState, useMemo } from "react";
import type { InterceptedEntry } from "../api";

interface TrafficInspectorProps {
  entries: InterceptedEntry[];
  jobId: string;
}

type FilterType =
  | "all"
  | "xhr"
  | "fetch"
  | "document"
  | "script"
  | "stylesheet"
  | "image"
  | "other";

/**
 * Format bytes to human-readable string.
 */
function formatBytes(bytes?: number): string {
  if (bytes === undefined || bytes === null) return "-";
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${(bytes / k ** i).toFixed(1)} ${sizes[i]}`;
}

/**
 * Format duration in milliseconds to human-readable string.
 */
function formatDuration(ms?: number): string {
  if (ms === undefined || ms === null) return "-";
  if (ms < 1) return "<1ms";
  if (ms < 1000) return `${Math.round(ms)}ms`;
  return `${(ms / 1000).toFixed(2)}s`;
}

/**
 * Truncate URL for display.
 */
function truncateUrl(url: string, maxLength: number = 60): string {
  if (url.length <= maxLength) return url;
  const start = url.substring(0, maxLength / 2);
  const end = url.substring(url.length - maxLength / 2);
  return `${start}...${end}`;
}

/**
 * Get status code class for styling.
 */
function getStatusClass(status?: number): string {
  if (status === undefined || status === null) return "unknown";
  if (status >= 200 && status < 300) return "success";
  if (status >= 300 && status < 400) return "redirect";
  if (status >= 400 && status < 500) return "client-error";
  if (status >= 500) return "server-error";
  return "unknown";
}

/**
 * Get resource type icon/label.
 */
function getResourceTypeLabel(type?: string): string {
  if (!type) return "other";
  return type;
}

export function TrafficInspector({ entries, jobId }: TrafficInspectorProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [filterType, setFilterType] = useState<FilterType>("all");
  const [selectedEntry, setSelectedEntry] = useState<InterceptedEntry | null>(
    null,
  );
  const [showRequestBody, setShowRequestBody] = useState(true);
  const [showResponseBody, setShowResponseBody] = useState(true);

  const filteredEntries = useMemo(() => {
    let filtered = entries;

    // Apply resource type filter
    if (filterType !== "all") {
      filtered = filtered.filter((entry) => {
        const resourceType = entry.request?.resourceType || "other";
        if (filterType === "other") {
          return ![
            "xhr",
            "fetch",
            "document",
            "script",
            "stylesheet",
            "image",
          ].includes(resourceType);
        }
        return resourceType === filterType;
      });
    }

    // Apply search filter
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter((entry) => {
        const url = entry.request?.url?.toLowerCase() || "";
        const method = entry.request?.method?.toLowerCase() || "";
        return url.includes(query) || method.includes(query);
      });
    }

    return filtered;
  }, [entries, filterType, searchQuery]);

  const stats = useMemo(() => {
    const total = entries.length;
    const withResponse = entries.filter((e) => e.response).length;
    const totalSize = entries.reduce(
      (sum, e) => sum + (e.response?.bodySize || 0),
      0,
    );
    const avgDuration =
      total > 0
        ? entries.reduce((sum, e) => sum + (e.duration || 0), 0) / total
        : 0;

    return { total, withResponse, totalSize, avgDuration };
  }, [entries]);

  const handleExportHar = () => {
    // Trigger HAR export via the existing API endpoint
    const url = `/v1/jobs/${jobId}/results?format=har`;
    window.open(url, "_blank");
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
            {formatBytes(stats.totalSize)} total
          </span>
          <span className="traffic-stat">
            avg {formatDuration(stats.avgDuration)}
          </span>
        </div>
        <button type="button" className="secondary" onClick={handleExportHar}>
          Export HAR
        </button>
      </div>

      <div className="traffic-toolbar">
        <div className="traffic-search">
          <input
            type="text"
            placeholder="Search by URL or method..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
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
          onChange={(e) => setFilterType(e.target.value as FilterType)}
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
              {filteredEntries.map((entry, index) => (
                <tr
                  key={`${entry.request?.requestId || index}-${index}`}
                  className={selectedEntry === entry ? "selected" : ""}
                  onClick={() => setSelectedEntry(entry)}
                >
                  <td className="method-cell">
                    <span
                      className={`method method-${entry.request?.method?.toLowerCase() || "get"}`}
                    >
                      {entry.request?.method || "GET"}
                    </span>
                  </td>
                  <td className="url-cell" title={entry.request?.url}>
                    {truncateUrl(entry.request?.url || "", 50)}
                  </td>
                  <td className="type-cell">
                    {getResourceTypeLabel(entry.request?.resourceType)}
                  </td>
                  <td className="status-cell">
                    {entry.response ? (
                      <span
                        className={`status ${getStatusClass(entry.response.status)}`}
                      >
                        {entry.response.status}
                      </span>
                    ) : (
                      <span className="status pending">-</span>
                    )}
                  </td>
                  <td className="size-cell">
                    {formatBytes(entry.response?.bodySize)}
                  </td>
                  <td className="duration-cell">
                    {formatDuration(entry.duration)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {filteredEntries.length === 0 && (
            <div className="traffic-no-results">No matching entries found</div>
          )}
        </div>

        {selectedEntry && (
          <div className="traffic-detail">
            <div className="traffic-detail-header">
              <h5>Request Details</h5>
              <button
                type="button"
                className="close-btn"
                onClick={() => setSelectedEntry(null)}
              >
                ×
              </button>
            </div>

            <div className="traffic-detail-content">
              <div className="detail-section">
                <h6>General</h6>
                <div className="detail-row">
                  <span className="detail-label">URL:</span>
                  <span className="detail-value url">
                    {selectedEntry.request?.url}
                  </span>
                </div>
                <div className="detail-row">
                  <span className="detail-label">Method:</span>
                  <span className="detail-value">
                    {selectedEntry.request?.method}
                  </span>
                </div>
                <div className="detail-row">
                  <span className="detail-label">Resource Type:</span>
                  <span className="detail-value">
                    {selectedEntry.request?.resourceType}
                  </span>
                </div>
                <div className="detail-row">
                  <span className="detail-label">Duration:</span>
                  <span className="detail-value">
                    {formatDuration(selectedEntry.duration)}
                  </span>
                </div>
              </div>

              <div className="detail-section">
                <h6>Request Headers</h6>
                <div className="headers-list">
                  {selectedEntry.request?.headers &&
                    Object.entries(selectedEntry.request.headers).map(
                      ([key, value]) => (
                        <div key={key} className="header-row">
                          <span className="header-name">{key}:</span>
                          <span className="header-value">{value}</span>
                        </div>
                      ),
                    )}
                </div>
              </div>

              {selectedEntry.request?.body && (
                <div className="detail-section">
                  <h6>
                    <label className="toggle-label">
                      <input
                        type="checkbox"
                        checked={showRequestBody}
                        onChange={(e) => setShowRequestBody(e.target.checked)}
                      />
                      Request Body (
                      {formatBytes(selectedEntry.request.bodySize)})
                    </label>
                  </h6>
                  {showRequestBody && (
                    <pre className="body-content">
                      {selectedEntry.request.body}
                    </pre>
                  )}
                </div>
              )}

              {selectedEntry.response && (
                <>
                  <div className="detail-section">
                    <h6>Response</h6>
                    <div className="detail-row">
                      <span className="detail-label">Status:</span>
                      <span
                        className={`detail-value status ${getStatusClass(selectedEntry.response.status)}`}
                      >
                        {selectedEntry.response.status}{" "}
                        {selectedEntry.response.statusText}
                      </span>
                    </div>
                    <div className="detail-row">
                      <span className="detail-label">Size:</span>
                      <span className="detail-value">
                        {formatBytes(selectedEntry.response.bodySize)}
                      </span>
                    </div>
                  </div>

                  <div className="detail-section">
                    <h6>Response Headers</h6>
                    <div className="headers-list">
                      {selectedEntry.response.headers &&
                        Object.entries(selectedEntry.response.headers).map(
                          ([key, value]) => (
                            <div key={key} className="header-row">
                              <span className="header-name">{key}:</span>
                              <span className="header-value">{value}</span>
                            </div>
                          ),
                        )}
                    </div>
                  </div>

                  {selectedEntry.response.body && (
                    <div className="detail-section">
                      <h6>
                        <label className="toggle-label">
                          <input
                            type="checkbox"
                            checked={showResponseBody}
                            onChange={(e) =>
                              setShowResponseBody(e.target.checked)
                            }
                          />
                          Response Body (
                          {formatBytes(selectedEntry.response.bodySize)})
                        </label>
                      </h6>
                      {showResponseBody && (
                        <pre className="body-content">
                          {selectedEntry.response.body}
                        </pre>
                      )}
                    </div>
                  )}
                </>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default TrafficInspector;
