/**
 * Purpose: Render guided proxy-pool capability status in Settings.
 * Responsibilities: Merge health-driven capability guidance with detailed runtime proxy-pool stats and recovery actions.
 * Scope: Proxy-pool settings presentation only.
 * Usage: Mount on the Settings route with health, navigation, and refresh helpers.
 * Invariants/Assumptions: Proxy pooling is optional, but degraded configuration must remain actionable and self-explanatory.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  getProxyPoolStatus,
  type HealthResponse,
  type ProxyPoolStatusResponse,
  type RecommendedAction,
} from "../api";
import { getApiBaseUrl } from "../lib/api-config";
import { getApiErrorMessage } from "../lib/api-errors";
import { ActionEmptyState } from "./ActionEmptyState";
import { CapabilityActionList } from "./CapabilityActionList";
import { CapabilityLoadErrorState } from "./CapabilityLoadErrorState";

interface ProxyPoolStatusPanelProps {
  health: HealthResponse | null;
  onNavigate: (path: string) => void;
  onRefreshHealth: () => Promise<unknown> | undefined;
}

function formatTags(tags: string[] | undefined): string {
  if (!tags || tags.length === 0) {
    return "n/a";
  }
  return tags.join(", ");
}

function readStringDetail(details: unknown, key: string): string | null {
  if (!details || typeof details !== "object") {
    return null;
  }

  const value = (details as Record<string, unknown>)[key];
  return typeof value === "string" && value.trim() !== "" ? value : null;
}

export function ProxyPoolStatusPanel({
  health,
  onNavigate,
  onRefreshHealth,
}: ProxyPoolStatusPanelProps) {
  const [status, setStatus] = useState<ProxyPoolStatusResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refreshStatus = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await getProxyPoolStatus({ baseUrl: getApiBaseUrl() });
      if (response.data) {
        setStatus(response.data);
      } else if (response.error) {
        setError(
          getApiErrorMessage(
            response.error,
            "Failed to load proxy pool status",
          ),
        );
      }
    } catch (err) {
      setError(getApiErrorMessage(err, "Failed to load proxy pool status"));
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

  const proxyComponent = health?.components?.proxy_pool;
  const capabilityStatus =
    proxyComponent?.status ?? (status?.proxies.length ? "ok" : "disabled");
  const configuredPath = readStringDetail(proxyComponent?.details, "path");
  const hasLoadedProxies = (status?.proxies.length ?? 0) > 0;

  const guidance = useMemo(() => {
    switch (capabilityStatus) {
      case "disabled":
        return {
          eyebrow: "Optional subsystem",
          title: "Proxy pooling stays off by default",
          description:
            proxyComponent?.message ??
            "Core scraping works normally without a proxy pool. Add one later only when you need pooled routing across multiple proxies.",
        };
      case "degraded":
      case "error":
      case "setup_required":
        return {
          eyebrow: "Recovery guidance",
          title: "Proxy pool needs attention",
          description:
            proxyComponent?.message ??
            "Spartan found proxy-pool configuration issues. Use the recovery actions below, then re-check the pool.",
        };
      default:
        return null;
    }
  }, [capabilityStatus, proxyComponent?.message]);

  const loadErrorActions = useMemo<RecommendedAction[]>(() => {
    if (proxyComponent?.actions && proxyComponent.actions.length > 0) {
      return proxyComponent.actions;
    }

    return [
      {
        label: "Check proxy-pool status from the CLI",
        kind: "command",
        value: "spartan proxy-pool status",
      },
    ];
  }, [proxyComponent?.actions]);

  const loadErrorState = useMemo(() => {
    if (!error || status) {
      return null;
    }

    switch (capabilityStatus) {
      case "disabled":
        return {
          eyebrow: "Optional subsystem",
          title: "Unable to load proxy pool status",
          description:
            "Spartan could not confirm whether proxy pooling is still off by choice. Core scraping can continue without it while you re-check this section.",
        };
      case "degraded":
      case "error":
      case "setup_required":
        return {
          eyebrow: "Recovery guidance",
          title: "Unable to load proxy pool status",
          description:
            "Spartan could not load live proxy-pool details for this section. Use the recovery actions below, then refresh this section.",
        };
      default:
        return {
          eyebrow: "Status unavailable",
          title: "Unable to load proxy pool status",
          description:
            "Spartan could not load live proxy-pool details for this section. Refresh this section or use the CLI status command below.",
        };
    }
  }, [capabilityStatus, error, status]);

  const showLoadErrorState = Boolean(error && !loading && !status);

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
            See whether pooled proxy routing is still off by choice, needs
            attention, or is ready to inspect.
          </p>
        </div>

        <button
          type="button"
          className="secondary"
          onClick={() => {
            void refreshAll();
          }}
        >
          Refresh
        </button>
      </div>

      {error && status ? (
        <div className="error" style={{ marginBottom: 16 }}>
          {error}
        </div>
      ) : null}

      {showLoadErrorState && loadErrorState && error ? (
        <CapabilityLoadErrorState
          eyebrow={loadErrorState.eyebrow}
          title={loadErrorState.title}
          description={loadErrorState.description}
          error={error}
          actions={loadErrorActions}
          onNavigate={onNavigate}
          onRefresh={refreshAll}
        >
          {configuredPath ? (
            <div className="system-status__hint">
              <strong>Configured file</strong>
              <span>{configuredPath}</span>
            </div>
          ) : null}
        </CapabilityLoadErrorState>
      ) : null}

      {!showLoadErrorState && guidance && !hasLoadedProxies ? (
        <ActionEmptyState
          eyebrow={guidance.eyebrow}
          title={guidance.title}
          description={guidance.description}
          actions={[
            {
              label: "Refresh status",
              onClick: () => {
                void refreshAll();
              },
              tone: "secondary",
            },
          ]}
        >
          {configuredPath ? (
            <div className="system-status__hint">
              <strong>Configured file</strong>
              <span>{configuredPath}</span>
            </div>
          ) : null}

          <CapabilityActionList
            actions={proxyComponent?.actions ?? []}
            onNavigate={onNavigate}
            onRefresh={refreshAll}
          />
        </ActionEmptyState>
      ) : null}

      {guidance && hasLoadedProxies ? (
        <div className="retention-notice retention-notice--warning">
          <h4 className="retention-section-title">{guidance.title}</h4>
          <p className="retention-notice__copy">{guidance.description}</p>
          {configuredPath ? (
            <div className="retention-notice__detail">
              <strong>Configured file:</strong> <code>{configuredPath}</code>
            </div>
          ) : null}
          <CapabilityActionList
            actions={proxyComponent?.actions ?? []}
            onNavigate={onNavigate}
            onRefresh={refreshAll}
          />
        </div>
      ) : null}

      {loading && !status ? (
        <div>Loading proxy pool status...</div>
      ) : status ? (
        <>
          <div className="retention-status-grid">
            <div className="retention-status-card retention-status-card--normal">
              <div className="retention-status-card__label">Capability</div>
              <div className="retention-status-card__value">
                {capabilityStatus.replaceAll("_", " ")}
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

          {status.regions.length > 0 || status.tags.length > 0 ? (
            <div style={{ marginTop: 16, display: "grid", gap: 8 }}>
              <div>
                <strong>Available regions:</strong> {formatTags(status.regions)}
              </div>
              <div>
                <strong>Available tags:</strong> {formatTags(status.tags)}
              </div>
            </div>
          ) : null}

          {hasLoadedProxies ? (
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
          ) : null}
        </>
      ) : null}
    </section>
  );
}
