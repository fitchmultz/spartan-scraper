/**
 * Metrics Dashboard Component
 *
 * Displays real-time performance metrics including request rates, success rates,
 * response times, fetcher usage breakdown, and rate limiter status.
 *
 * @module MetricsDashboard
 */

import type { MetricsResponse } from "../api";

interface MetricsDashboardProps {
  metrics: MetricsResponse | null;
  connectionState: "connected" | "disconnected" | "reconnecting" | "polling";
}

interface MetricCardProps {
  label: string;
  value: string | number;
  unit?: string;
  trend?: "up" | "down" | "neutral";
  color?: "default" | "success" | "warning" | "error";
}

function MetricCard({
  label,
  value,
  unit,
  trend,
  color = "default",
}: MetricCardProps) {
  const getTrendIcon = () => {
    switch (trend) {
      case "up":
        return "↑";
      case "down":
        return "↓";
      default:
        return null;
    }
  };

  const getColorClass = () => {
    switch (color) {
      case "success":
        return "metric-card--success";
      case "warning":
        return "metric-card--warning";
      case "error":
        return "metric-card--error";
      default:
        return "";
    }
  };

  return (
    <div className={`metric-card ${getColorClass()}`}>
      <div className="metric-card__label">{label}</div>
      <div className="metric-card__value">
        {value}
        {unit && <span className="metric-card__unit">{unit}</span>}
        {trend && (
          <span className={`metric-card__trend metric-card__trend--${trend}`}>
            {getTrendIcon()}
          </span>
        )}
      </div>
    </div>
  );
}

interface RateLimitIndicatorProps {
  status: {
    host?: string;
    qps?: number;
    burst?: number;
    tokens?: number;
    lastRequest?: number;
  };
}

function RateLimitIndicator({ status }: RateLimitIndicatorProps) {
  // Calculate health based on tokens remaining
  const burst = status.burst ?? 1;
  const tokens = status.tokens ?? 0;
  const tokenRatio = tokens / burst;
  let health: "healthy" | "throttling" | "limited" = "healthy";
  let healthLabel = "Healthy";

  if (tokenRatio <= 0) {
    health = "limited";
    healthLabel = "Rate Limited";
  } else if (tokenRatio < 0.5) {
    health = "throttling";
    healthLabel = "Throttling";
  }

  return (
    <div className={`rate-limit-indicator rate-limit-indicator--${health}`}>
      <div className="rate-limit-indicator__host">
        {status.host ?? "unknown"}
      </div>
      <div className="rate-limit-indicator__details">
        <span className="rate-limit-indicator__qps">
          {(status.qps ?? 0).toFixed(1)} req/s
        </span>
        <span className="rate-limit-indicator__burst">Burst: {burst}</span>
      </div>
      <div className="rate-limit-indicator__tokens">
        <div
          className="rate-limit-indicator__token-bar"
          style={{ width: `${Math.min(tokenRatio * 100, 100)}%` }}
        />
      </div>
      <div className="rate-limit-indicator__status">{healthLabel}</div>
    </div>
  );
}

interface FetcherUsageBarProps {
  usage: {
    http?: number;
    chromedp?: number;
    playwright?: number;
  };
}

function FetcherUsageBar({ usage }: FetcherUsageBarProps) {
  const total =
    (usage.http || 0) + (usage.chromedp || 0) + (usage.playwright || 0);

  if (total === 0) {
    return (
      <div className="fetcher-usage-bar fetcher-usage-bar--empty">No data</div>
    );
  }

  const httpPct = ((usage.http || 0) / total) * 100;
  const cdpPct = ((usage.chromedp || 0) / total) * 100;
  const pwPct = ((usage.playwright || 0) / total) * 100;

  return (
    <div className="fetcher-usage-bar">
      <div className="fetcher-usage-bar__segments">
        {httpPct > 0 && (
          <div
            className="fetcher-usage-bar__segment fetcher-usage-bar__segment--http"
            style={{ width: `${httpPct}%` }}
            title={`HTTP: ${httpPct.toFixed(1)}% (${usage.http})`}
          />
        )}
        {cdpPct > 0 && (
          <div
            className="fetcher-usage-bar__segment fetcher-usage-bar__segment--chromedp"
            style={{ width: `${cdpPct}%` }}
            title={`Chromedp: ${cdpPct.toFixed(1)}% (${usage.chromedp})`}
          />
        )}
        {pwPct > 0 && (
          <div
            className="fetcher-usage-bar__segment fetcher-usage-bar__segment--playwright"
            style={{ width: `${pwPct}%` }}
            title={`Playwright: ${pwPct.toFixed(1)}% (${usage.playwright})`}
          />
        )}
      </div>
      <div className="fetcher-usage-bar__legend">
        <div className="fetcher-usage-bar__legend-item">
          <span className="fetcher-usage-bar__legend-color fetcher-usage-bar__legend-color--http" />
          <span>HTTP: {usage.http || 0}</span>
        </div>
        <div className="fetcher-usage-bar__legend-item">
          <span className="fetcher-usage-bar__legend-color fetcher-usage-bar__legend-color--chromedp" />
          <span>Chromedp: {usage.chromedp || 0}</span>
        </div>
        <div className="fetcher-usage-bar__legend-item">
          <span className="fetcher-usage-bar__legend-color fetcher-usage-bar__legend-color--playwright" />
          <span>Playwright: {usage.playwright || 0}</span>
        </div>
      </div>
    </div>
  );
}

export function MetricsDashboard({
  metrics,
  connectionState,
}: MetricsDashboardProps) {
  const getConnectionStatus = () => {
    switch (connectionState) {
      case "connected":
        return { label: "Live", className: "status--connected" };
      case "reconnecting":
        return { label: "Reconnecting...", className: "status--reconnecting" };
      case "polling":
        return { label: "Polling", className: "status--polling" };
      default:
        return { label: "Disconnected", className: "status--disconnected" };
    }
  };

  const connectionStatus = getConnectionStatus();

  // Format numbers for display
  const formatNumber = (num?: number, decimals = 1) => {
    if (num === undefined || num === null) return "--";
    return num.toFixed(decimals);
  };

  const formatInteger = (num?: number) => {
    if (num === undefined || num === null) return "--";
    return num.toLocaleString();
  };

  // Determine success rate color
  const getSuccessRateColor = (rate?: number): MetricCardProps["color"] => {
    if (rate === undefined || rate === null) return "default";
    if (rate >= 95) return "success";
    if (rate >= 80) return "warning";
    return "error";
  };

  return (
    <section className="metrics-dashboard" data-tour="metrics">
      <div className="metrics-dashboard__header">
        <h3>Performance Metrics</h3>
        <div
          className={`metrics-dashboard__status ${connectionStatus.className}`}
        >
          {connectionStatus.label}
        </div>
      </div>

      {!metrics ? (
        <div className="metrics-dashboard__empty">Waiting for metrics...</div>
      ) : (
        <>
          <div className="metrics-dashboard__grid">
            <MetricCard
              label="Requests/sec"
              value={formatNumber(metrics.requestsPerSec)}
              trend={
                metrics.requestsPerSec && metrics.requestsPerSec > 0
                  ? "up"
                  : "neutral"
              }
            />
            <MetricCard
              label="Success Rate"
              value={formatNumber(metrics.successRate)}
              unit="%"
              color={getSuccessRateColor(metrics.successRate)}
            />
            <MetricCard
              label="Avg Response Time"
              value={formatNumber(metrics.avgResponseTimeMs)}
              unit="ms"
            />
            <MetricCard
              label="Active Requests"
              value={formatInteger(metrics.activeRequests)}
            />
            <MetricCard
              label="Total Requests"
              value={formatInteger(metrics.totalRequests)}
            />
            <MetricCard
              label="Job Throughput"
              value={formatNumber(metrics.jobThroughputPerMin)}
              unit="/min"
            />
            <MetricCard
              label="Avg Job Duration"
              value={formatNumber(metrics.avgJobDurationMs)}
              unit="ms"
            />
          </div>

          {metrics.fetcherUsage && (
            <div className="metrics-dashboard__section">
              <h4>Fetcher Usage</h4>
              <FetcherUsageBar usage={metrics.fetcherUsage} />
            </div>
          )}

          {metrics.rateLimitStatus && metrics.rateLimitStatus.length > 0 && (
            <div className="metrics-dashboard__section">
              <h4>Rate Limit Status</h4>
              <div className="metrics-dashboard__rate-limits">
                {metrics.rateLimitStatus.map((status) => (
                  <RateLimitIndicator key={status.host} status={status} />
                ))}
              </div>
            </div>
          )}
        </>
      )}
    </section>
  );
}
