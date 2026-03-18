/**
 * Purpose: Render proxy-pool status and optional-subsystem guidance in Settings.
 * Responsibilities: Fetch the current proxy-pool status, summarize pool and per-proxy health, and explain the disabled state without treating it as a startup failure.
 * Scope: Proxy-pool settings presentation only.
 * Usage: Mount on the Settings route to help operators understand whether proxy pooling is configured.
 * Invariants/Assumptions: Missing proxy-pool config is a valid optional state and should remain actionable instead of alarming.
 */

import { useCallback, useEffect, useState } from "react";
import { getProxyPoolStatus, type ProxyPoolStatusResponse } from "../api";
import { getApiBaseUrl } from "../lib/api-config";
import { ActionEmptyState } from "./ActionEmptyState";

function formatTags(tags: string[] | undefined): string {
  if (!tags || tags.length === 0) {
    return "n/a";
  }
  return tags.join(", ");
}

export function ProxyPoolStatusPanel() {
  const [status, setStatus] = useState<ProxyPoolStatusResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refreshStatus = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await getProxyPoolStatus({
        baseUrl: getApiBaseUrl(),
      });
      if (response.data) {
        setStatus(response.data);
      } else if (response.error) {
        setError(String(response.error));
      }
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to fetch proxy pool status",
      );
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refreshStatus();
  }, [refreshStatus]);

  return (
    <section className="panel" id="proxy-pool">
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
          <h2 style={{ marginBottom: 4 }}>Proxy Pool</h2>
          <p style={{ margin: 0, opacity: 0.8 }}>
            Inspect the currently loaded proxy pool and per-proxy health/runtime
            stats.
          </p>
        </div>
        <button type="button" className="secondary" onClick={refreshStatus}>
          Refresh
        </button>
      </div>

      {error && (
        <div className="error" style={{ marginBottom: 16 }}>
          {error}
        </div>
      )}

      {loading && !status ? (
        <div>Loading proxy pool status...</div>
      ) : status ? (
        <>
          <div className="retention-status-grid">
            <div className="retention-status-card retention-status-card--normal">
              <div className="retention-status-card__label">Status</div>
              <div className="retention-status-card__value">
                {status.total_proxies > 0 ? "Loaded" : "Disabled"}
              </div>
            </div>
            <div className="retention-status-card retention-status-card--normal">
              <div className="retention-status-card__label">Strategy</div>
              <div className="retention-status-card__value">
                {status.strategy}
              </div>
            </div>
            <div className="retention-status-card retention-status-card--normal">
              <div className="retention-status-card__label">Total Proxies</div>
              <div className="retention-status-card__value">
                {status.total_proxies}
              </div>
            </div>
            <div className="retention-status-card retention-status-card--normal">
              <div className="retention-status-card__label">
                Healthy Proxies
              </div>
              <div className="retention-status-card__value">
                {status.healthy_proxies}
              </div>
            </div>
            <div className="retention-status-card retention-status-card--normal">
              <div className="retention-status-card__label">Regions</div>
              <div className="retention-status-card__value">
                {status.regions.length}
              </div>
            </div>
            <div className="retention-status-card retention-status-card--normal">
              <div className="retention-status-card__label">Tags</div>
              <div className="retention-status-card__value">
                {status.tags.length}
              </div>
            </div>
          </div>

          {(status.regions.length > 0 || status.tags.length > 0) && (
            <div style={{ marginTop: 16, display: "grid", gap: 8 }}>
              <div>
                <strong>Available regions:</strong> {formatTags(status.regions)}
              </div>
              <div>
                <strong>Available tags:</strong> {formatTags(status.tags)}
              </div>
            </div>
          )}

          {status.proxies.length === 0 ? (
            <ActionEmptyState
              eyebrow="Optional subsystem"
              title="Proxy pooling is disabled"
              description="Spartan does not need a proxy pool for normal startup. Configure PROXY_POOL_FILE only when you want pooled routing across multiple proxies."
              actions={[
                {
                  label: "Refresh status",
                  onClick: refreshStatus,
                  tone: "secondary",
                },
              ]}
            />
          ) : (
            <div style={{ display: "grid", gap: 12, marginTop: 16 }}>
              {status.proxies.map((proxy) => (
                <div key={proxy.id} className="retention-config-card">
                  <div
                    style={{
                      display: "flex",
                      justifyContent: "space-between",
                      alignItems: "baseline",
                      gap: 12,
                      marginBottom: 12,
                    }}
                  >
                    <h4 className="retention-section-title">{proxy.id}</h4>
                    <span>{proxy.is_healthy ? "Healthy" : "Unhealthy"}</span>
                  </div>
                  <div className="retention-config-grid">
                    <div className="retention-metric">
                      <span className="retention-metric__label">Region</span>
                      <span className="retention-metric__value">
                        {proxy.region || "n/a"}
                      </span>
                    </div>
                    <div className="retention-metric">
                      <span className="retention-metric__label">Tags</span>
                      <span className="retention-metric__value">
                        {formatTags(proxy.tags)}
                      </span>
                    </div>
                    <div className="retention-metric">
                      <span className="retention-metric__label">Requests</span>
                      <span className="retention-metric__value">
                        {proxy.request_count}
                      </span>
                    </div>
                    <div className="retention-metric">
                      <span className="retention-metric__label">Successes</span>
                      <span className="retention-metric__value">
                        {proxy.success_count}
                      </span>
                    </div>
                    <div className="retention-metric">
                      <span className="retention-metric__label">Failures</span>
                      <span className="retention-metric__value">
                        {proxy.failure_count}
                      </span>
                    </div>
                    <div className="retention-metric">
                      <span className="retention-metric__label">
                        Success Rate
                      </span>
                      <span className="retention-metric__value">
                        {proxy.success_rate.toFixed(2)}%
                      </span>
                    </div>
                    <div className="retention-metric">
                      <span className="retention-metric__label">
                        Avg Latency
                      </span>
                      <span className="retention-metric__value">
                        {proxy.avg_latency_ms} ms
                      </span>
                    </div>
                    <div className="retention-metric">
                      <span className="retention-metric__label">
                        Consecutive Fails
                      </span>
                      <span className="retention-metric__value">
                        {proxy.consecutive_fails}
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </>
      ) : null}
    </section>
  );
}
