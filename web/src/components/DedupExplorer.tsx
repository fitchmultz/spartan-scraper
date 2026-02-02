/**
 * Dedup Explorer Component
 *
 * Provides a UI for exploring cross-job content deduplication data.
 * Allows searching for duplicates by URL or simhash, viewing content history,
 * and analyzing deduplication statistics.
 *
 * @module DedupExplorer
 */

import { useState, useCallback, useEffect } from "react";

interface DuplicateMatch {
  jobId: string;
  url: string;
  simhash: number;
  distance: number;
  indexedAt: string;
}

interface ContentEntry {
  jobId: string;
  simhash: number;
  indexedAt: string;
}

interface DedupStats {
  totalIndexed: number;
  uniqueUrls: number;
  uniqueJobs: number;
  duplicatePairs: number;
}

interface DedupExplorerProps {
  apiBaseUrl?: string;
}

export function DedupExplorer({ apiBaseUrl = "" }: DedupExplorerProps) {
  const [activeTab, setActiveTab] = useState<"search" | "history" | "stats">(
    "search",
  );
  const [url, setUrl] = useState("");
  const [simhash, setSimhash] = useState("");
  const [threshold, setThreshold] = useState(3);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [duplicates, setDuplicates] = useState<DuplicateMatch[]>([]);
  const [history, setHistory] = useState<ContentEntry[]>([]);
  const [stats, setStats] = useState<DedupStats | null>(null);

  const fetchDuplicates = useCallback(async () => {
    if (!simhash) {
      setError("Please enter a simhash value");
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await fetch(
        `${apiBaseUrl}/v1/dedup/duplicates?simhash=${simhash}&threshold=${threshold}`,
      );

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || "Failed to fetch duplicates");
      }

      const data = await response.json();
      setDuplicates(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "An error occurred");
    } finally {
      setLoading(false);
    }
  }, [simhash, threshold, apiBaseUrl]);

  const fetchHistory = useCallback(async () => {
    if (!url) {
      setError("Please enter a URL");
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const response = await fetch(
        `${apiBaseUrl}/v1/dedup/history?url=${encodeURIComponent(url)}`,
      );

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || "Failed to fetch history");
      }

      const data = await response.json();
      setHistory(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "An error occurred");
    } finally {
      setLoading(false);
    }
  }, [url, apiBaseUrl]);

  const fetchStats = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`${apiBaseUrl}/v1/dedup/stats`);

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.error || "Failed to fetch stats");
      }

      const data = await response.json();
      setStats(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : "An error occurred");
    } finally {
      setLoading(false);
    }
  }, [apiBaseUrl]);

  // Load stats on initial mount
  useEffect(() => {
    fetchStats();
  }, [fetchStats]);

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString();
  };

  const getDistanceColor = (distance: number) => {
    if (distance === 0) return "#22c55e"; // exact match - green
    if (distance <= 3) return "#3b82f6"; // near duplicate - blue
    if (distance <= 8) return "#f59e0b"; // similar - amber
    return "#6b7280"; // distinct - gray
  };

  const getDistanceLabel = (distance: number) => {
    if (distance === 0) return "Exact Match";
    if (distance <= 3) return "Near Duplicate";
    if (distance <= 8) return "Similar";
    return "Distinct";
  };

  return (
    <div className="dedup-explorer">
      <header className="dedup-explorer__header">
        <h2>Content Deduplication</h2>
        <p className="dedup-explorer__subtitle">
          Explore cross-job content fingerprints and find duplicate pages
        </p>
      </header>

      <nav className="dedup-explorer__tabs">
        <button
          type="button"
          className={`dedup-explorer__tab ${activeTab === "search" ? "active" : ""}`}
          onClick={() => setActiveTab("search")}
        >
          Find Duplicates
        </button>
        <button
          type="button"
          className={`dedup-explorer__tab ${activeTab === "history" ? "active" : ""}`}
          onClick={() => setActiveTab("history")}
        >
          URL History
        </button>
        <button
          type="button"
          className={`dedup-explorer__tab ${activeTab === "stats" ? "active" : ""}`}
          onClick={() => {
            setActiveTab("stats");
            fetchStats();
          }}
        >
          Statistics
        </button>
      </nav>

      {error && (
        <div className="dedup-explorer__error">
          <span className="dedup-explorer__error-icon">⚠️</span>
          {error}
        </div>
      )}

      {activeTab === "search" && (
        <section className="dedup-explorer__section">
          <div className="dedup-explorer__form">
            <div className="dedup-explorer__field">
              <label htmlFor="simhash">Simhash Value</label>
              <input
                id="simhash"
                type="text"
                value={simhash}
                onChange={(e) => setSimhash(e.target.value)}
                placeholder="Enter simhash (e.g., 1234567890)"
                className="dedup-explorer__input"
              />
            </div>
            <div className="dedup-explorer__field">
              <label htmlFor="threshold">
                Threshold: {threshold} ({getDistanceLabel(threshold)})
              </label>
              <input
                id="threshold"
                type="range"
                min="0"
                max="16"
                value={threshold}
                onChange={(e) => setThreshold(parseInt(e.target.value, 10))}
                className="dedup-explorer__slider"
              />
              <div className="dedup-explorer__threshold-labels">
                <span>Exact (0)</span>
                <span>Near (3)</span>
                <span>Similar (8)</span>
                <span>Any (16)</span>
              </div>
            </div>
            <button
              type="button"
              onClick={fetchDuplicates}
              disabled={loading}
              className="dedup-explorer__button"
            >
              {loading ? "Searching..." : "Find Duplicates"}
            </button>
          </div>

          {duplicates.length > 0 && (
            <div className="dedup-explorer__results">
              <h3>Found {duplicates.length} Duplicate(s)</h3>
              <div className="dedup-explorer__list">
                {duplicates.map((dup) => (
                  <div
                    key={`${dup.jobId}-${dup.url}`}
                    className="dedup-explorer__item"
                  >
                    <div className="dedup-explorer__item-header">
                      <span
                        className="dedup-explorer__distance"
                        style={{ color: getDistanceColor(dup.distance) }}
                      >
                        {dup.distance} bits
                      </span>
                      <span className="dedup-explorer__label">
                        {getDistanceLabel(dup.distance)}
                      </span>
                    </div>
                    <div className="dedup-explorer__url">{dup.url}</div>
                    <div className="dedup-explorer__meta">
                      <span>Job: {dup.jobId}</span>
                      <span>Indexed: {formatDate(dup.indexedAt)}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {duplicates.length === 0 && !loading && simhash && (
            <div className="dedup-explorer__empty">
              No duplicates found for simhash {simhash} with threshold{" "}
              {threshold}
            </div>
          )}
        </section>
      )}

      {activeTab === "history" && (
        <section className="dedup-explorer__section">
          <div className="dedup-explorer__form">
            <div className="dedup-explorer__field">
              <label htmlFor="url">URL</label>
              <input
                id="url"
                type="text"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                placeholder="Enter URL to check history"
                className="dedup-explorer__input"
              />
            </div>
            <button
              type="button"
              onClick={fetchHistory}
              disabled={loading}
              className="dedup-explorer__button"
            >
              {loading ? "Loading..." : "Get History"}
            </button>
          </div>

          {history.length > 0 && (
            <div className="dedup-explorer__results">
              <h3>Content History ({history.length} entries)</h3>
              <div className="dedup-explorer__list">
                {history.map((entry) => (
                  <div
                    key={`${entry.jobId}-${entry.indexedAt}`}
                    className="dedup-explorer__item"
                  >
                    <div className="dedup-explorer__item-header">
                      <span className="dedup-explorer__simhash">
                        Simhash: {entry.simhash}
                      </span>
                    </div>
                    <div className="dedup-explorer__meta">
                      <span>Job: {entry.jobId}</span>
                      <span>Indexed: {formatDate(entry.indexedAt)}</span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}

          {history.length === 0 && !loading && url && (
            <div className="dedup-explorer__empty">
              No history found for URL: {url}
            </div>
          )}
        </section>
      )}

      {activeTab === "stats" && (
        <section className="dedup-explorer__section">
          {stats && (
            <div className="dedup-explorer__stats">
              <div className="dedup-explorer__stat-card">
                <div className="dedup-explorer__stat-value">
                  {stats.totalIndexed.toLocaleString()}
                </div>
                <div className="dedup-explorer__stat-label">Total Indexed</div>
              </div>
              <div className="dedup-explorer__stat-card">
                <div className="dedup-explorer__stat-value">
                  {stats.uniqueUrls.toLocaleString()}
                </div>
                <div className="dedup-explorer__stat-label">Unique URLs</div>
              </div>
              <div className="dedup-explorer__stat-card">
                <div className="dedup-explorer__stat-value">
                  {stats.uniqueJobs.toLocaleString()}
                </div>
                <div className="dedup-explorer__stat-label">Unique Jobs</div>
              </div>
              <div className="dedup-explorer__stat-card">
                <div className="dedup-explorer__stat-value">
                  {stats.duplicatePairs.toLocaleString()}
                </div>
                <div className="dedup-explorer__stat-label">Duplicate URLs</div>
              </div>
            </div>
          )}

          {stats && stats.totalIndexed > 0 && (
            <div className="dedup-explorer__stats-detail">
              <h3>Deduplication Rate</h3>
              <div className="dedup-explorer__progress">
                <div
                  className="dedup-explorer__progress-bar"
                  style={{
                    width: `${(stats.duplicatePairs / stats.uniqueUrls) * 100}%`,
                  }}
                />
              </div>
              <p>
                {(stats.duplicatePairs / stats.uniqueUrls) * 100}% of unique
                URLs appear in multiple jobs
              </p>
            </div>
          )}

          <button
            type="button"
            onClick={fetchStats}
            disabled={loading}
            className="dedup-explorer__button dedup-explorer__button--secondary"
          >
            {loading ? "Refreshing..." : "Refresh Stats"}
          </button>
        </section>
      )}
    </div>
  );
}
